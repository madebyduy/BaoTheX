# BaoTheX delivery pipeline

This document describes the portfolio-ready delivery process for BaoTheX. It
separates untrusted cloud CI from the Windows machine that temporarily hosts the
backend.

## Architecture

```text
pull request / push
        |
        v
GitHub-hosted CI
  - gofmt, vet, race tests, build
  - ESLint, TypeScript, Next.js build
        |
        +---- push to main ----> Cloudflare Workers deployment (frontend)
        |
        +---- Windows pull agent checks origin/main every 2 minutes
                                  |
                                  v
                           isolated release clone
                           test -> build -> probe
                                  |
                           cut-over or rollback
```

The repository is public, so it deliberately does **not** use a GitHub
self-hosted runner. A public-repository workflow must never be allowed to execute
arbitrary code on a personal workstation.

## Quality gates

The workflow in `.github/workflows/ci.yml` runs on pull requests, pushes to
`main`, and manual dispatches.

| Area             | Required checks                                                         |
| ---------------- | ----------------------------------------------------------------------- |
| Go backend       | `gofmt`, `go vet`, race-enabled tests, API/worker builds                |
| Next.js frontend | locked install, ESLint with zero warnings, TypeScript, production build |

Recommended `main` branch protection:

1. Require a pull request before merging.
2. Require `Backend quality gate` and `Frontend quality gate`.
3. Require the branch to be up to date before merging.
4. Block force pushes and branch deletion.
5. Do not allow workflow runs from fork pull requests to access production
   secrets.

## Frontend continuous deployment

`.github/workflows/deploy-frontend.yml` deploys the OpenNext bundle only after
the CI workflow succeeds for a push to `main`. A manual dispatch is also
available for an intentional redeploy.

Create a GitHub environment named `production`, then configure:

### Environment secrets

- `CLOUDFLARE_ACCOUNT_ID`
- `CLOUDFLARE_API_TOKEN`

Create a narrowly scoped Cloudflare API token with Workers Scripts edit access
for the BaoTheX account. Never use the global API key.

### Environment variables

- `NEXT_PUBLIC_API_URL` — currently the active tunnel URL
- `NEXT_PUBLIC_SITE_URL` — currently
  `https://baothex-web.universeapd.workers.dev`

The Quick Tunnel URL changes when the tunnel is recreated. Update
`NEXT_PUBLIC_API_URL` and re-run `Deploy frontend` after that happens. A named
Cloudflare Tunnel and `api.baothex.vn` remove this manual step.

Cloudflare retains Worker versions. Roll back the frontend from Workers & Pages
→ `baothex-web` → Deployments, or deploy a known-good Git commit again.

## Backend pull deployment on Windows

The Windows updater polls `origin/main`; GitHub never opens an inbound connection
to the machine. Each commit is materialized as an immutable Git worktree under
`%LOCALAPPDATA%\BaoTheX\releases\<sha>`.

For every new commit it:

1. fetches `origin/main`;
2. verifies that the GitHub `CI` workflow succeeded for that exact commit;
3. runs the full Go test suite again on the deployment machine;
4. builds trimmed API and worker binaries;
5. boots the candidate API on port `18081` and probes `/healthz`;
6. stops the previously managed release;
7. starts the candidate on port `8081`;
8. rolls back to the previous release if the final health-check fails.

The updater never modifies the developer workspace.

### First installation

Commit and push the pipeline files before installing. Stop the manually started
API on port `8081`, then open PowerShell as the same Windows user that will run
the services:

```powershell
cd C:\Users\MKT\Desktop\BaoTheX
.\deploy\windows\Install-BaoTheXUpdater.ps1
Start-ScheduledTask -TaskName "BaoTheX Pull Deploy"
```

The worker is off by default so a portfolio preview cannot unexpectedly consume
LLM or ingestion quotas. Enable it only when the budgets are configured:

```powershell
.\deploy\windows\Install-BaoTheXUpdater.ps1 -EnableWorker
```

Runtime locations:

- configuration: `%LOCALAPPDATA%\BaoTheX\config\deployment.json`
- releases: `%LOCALAPPDATA%\BaoTheX\releases`
- process state: `%LOCALAPPDATA%\BaoTheX\state\deployment.json`
- logs: `%LOCALAPPDATA%\BaoTheX\logs`

The task uses the existing ignored `.env` file. Secrets remain outside Git and
must never be printed by deployment logs.

### Operations

```powershell
# Run a deployment check immediately
Start-ScheduledTask -TaskName "BaoTheX Pull Deploy"

# Inspect the last task result
Get-ScheduledTaskInfo -TaskName "BaoTheX Pull Deploy"

# Follow API errors
Get-Content "$env:LOCALAPPDATA\BaoTheX\logs\api-*.err.log" -Tail 100

# Remove automation but preserve releases for recovery
.\deploy\windows\Remove-BaoTheXUpdater.ps1
```

Do not delete a release referenced by
`%LOCALAPPDATA%\BaoTheX\state\deployment.json`.

## Database migrations

Database migrations are intentionally not applied by the workstation updater.
For a junior portfolio, explain this as a safety boundary: schema changes should
use the expand-and-contract pattern, be reviewed separately, backed up, and
applied before code that depends on them. Automating irreversible SQL from every
push would make rollback misleading because application rollback cannot undo
data loss.

## Interview explanation

A concise explanation:

> I split CI from deployment. GitHub-hosted runners validate formatting, static
> analysis, race tests, types and production builds. The frontend deploys to a
> versioned Cloudflare Worker only from main. Because the repository is public,
> I avoided a self-hosted runner on my PC and built a pull-based Windows agent.
> It creates immutable releases, probes candidates on a second port, and rolls
> back automatically if the cut-over health-check fails. Secrets and runtime
> state never enter Git.

Trade-offs worth mentioning honestly:

- Quick Tunnel has no uptime guarantee and changes URL after restart.
- The Windows pull agent is appropriate for a free portfolio environment, not a
  high-availability production cluster.
- The next production step is a named tunnel or small VM, managed secrets,
  metrics/alerts, and a reviewed migration job.
