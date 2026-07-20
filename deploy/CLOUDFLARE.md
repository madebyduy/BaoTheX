# Deploy frontend to Cloudflare Workers

The Next.js frontend is packaged with the official Cloudflare OpenNext adapter.
Deploy from `apps/web`; the Go API and worker remain on the existing backend.

## First rollout: preview only

1. In Cloudflare, create or connect a Worker to the GitHub repository.
2. Set the project root directory to `apps/web`.
3. Add these build variables (they are compiled into the frontend):
   - `NEXT_PUBLIC_API_URL=https://<public-api-host>`
   - `NEXT_PUBLIC_SITE_URL=https://<preview-or-production-host>`
4. Use `npm run build:cloudflare` as the build command.
5. Use `npm run cf:upload` as the deploy command. This uploads a new Worker
   version without sending production traffic to it.

Cloudflare preview builds for non-production branches use version uploads by
default. Validate the generated preview URL before promoting a version.

For a local authenticated preview upload:

```powershell
cd apps/web
$env:NEXT_PUBLIC_API_URL = "https://<public-api-host>"
$env:NEXT_PUBLIC_SITE_URL = "https://<preview-host>"
npm run deploy:preview
```

## Promote after validation

For the initial production release, change the Workers Builds deploy command to
`npm run cf:deploy`, or deploy an already-built version from the Cloudflare
dashboard. A direct authenticated deploy from this repository is:

```powershell
cd apps/web
npm run deploy
```

Before enabling browser login or account actions, add the preview and production
frontend origins to the backend's comma-separated `CORS_ORIGINS`. Also ensure the
API's session cookie domain and HTTPS settings match the public frontend/API
domains.

Keep the preview/upload step as the default for branches. Only the production
branch should run the production deploy command.
