# BaoTheX

BaoTheX — báo thể thao tổng hợp, theo dõi tin nóng trong ngày từ các nguồn báo chí và video đáng tin cậy cho người Việt.

## Backend

Sports newsroom: aggregates important daily sports news from Vietnamese and
international publications and YouTube. Users follow sports, people and sources,
read sourced summaries, save content and receive a personalised Telegram brief.

Two intakes are built and switched off: **podcasts** (retired in migration 0015,
feeds disabled in 0030 — the surface is gone, so the fetchers were still filling
a section that no longer exists) and **Europe PMC research** (disabled in 0024 —
protein-synthesis papers are scholarship, not sports news). Their code, tables
and 45 archived papers remain; only the intake stopped.

**Stack:** Go (API + worker) · PostgreSQL 16 (DB + queue + full-text search) ·
Telegram Bot API · Docker Compose. The **Next.js frontend is built separately**
(see [Frontend](#frontend)).

---

## Repository layout

```
.
├── apps/
│   ├── api/            # Go: HTTP API server (main.go + Dockerfile)
│   ├── worker/         # Go: scheduler + job pool + fetchers + telegram sender
│   └── web/            # Next.js App Router frontend (app/)
├── internal/
│   ├── config/         # env config loader
│   ├── logging/        # slog setup
│   ├── domain/         # core types (content, user, topic, source, entity, job)
│   ├── postgres/       # pgx pool + repositories (one file per aggregate)
│   ├── auth/           # Argon2id passwords + session tokens
│   ├── ingest/         # fetchers (rss, youtube, europepmc, podcast) + normalize/dedup
│   ├── process/        # classify, entity extraction, summarize (LLM), score
│   ├── feed/           # personalized ranker + homepage builder (50/30/20)
│   ├── telegram/       # bot client, digest builder, webhook handler
│   ├── jobs/           # queue helpers, worker pool, scheduler, handlers
│   └── httpapi/        # server, middleware, respond, one handler file per area
├── migrations/         # SQL migrations (golang-migrate format)
├── deploy/             # docker-compose.yml + Caddyfile
├── Makefile
└── .env.example
```

The design keeps a clean seam between layers so it extends easily:

- **New source type** → add a fetcher in `internal/ingest`, a `source_kind`
  enum value, and a case in `jobs/scheduler.go` + `jobs/handlers.go`.
- **New content type** → add a `content_type` enum value, a subtype table +
  repo methods, and a case in `ingest/dedup.go` (`Store`).
- **New job kind** → add a constant in `domain/job.go`, a handler in
  `jobs/handlers.go`, and register it in `Handlers.Register()`.
- **Python worker (v0.5+)** → polls the same `jobs` table; no broker needed.

---

## Quick start (local, Docker)

```bash
cp .env.example .env
# Edit .env: set SESSION_SECRET (required) and, when ready, YOUTUBE_API_KEY,
# TELEGRAM_BOT_TOKEN, LLM_API_KEY. DB_PASSWORD is used by docker-compose.

make up            # builds api + worker, runs migrations, starts everything
make logs          # tail api + worker
```

- API: <http://localhost:8080/api/v1> — health at `/healthz`, readiness at `/readyz`.
- Postgres: `localhost:5432` (`repwire` / `DB_PASSWORD`).

Migrations run automatically via the `migrate` service before api/worker start.
Seed data (topics, entities, MVP sources) is applied by `0002_seed`.

### Create an admin user

```bash
# Register through the API first, then promote:
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"a-strong-password"}'

make admin EMAIL=you@example.com
```

Without Docker, use the idempotent seed command. Keep these values only in
`.env` or your deployment secret manager, never in Git:

```bash
export ADMIN_EMAIL='admin@baothex.vn'
export ADMIN_PASSWORD='a-long-random-password'
export ADMIN_NAME='Toà soạn BaoTheX'
go run ./tools/seed-admin
```

The command creates the account when missing, promotes it to `admin`, and
rotates its password when run again.

---

## Quick start (local, without Docker)

Needs a running PostgreSQL 16 with the `pg_trgm`, `unaccent`, `citext`
extensions available.

```bash
export DATABASE_URL='postgres://repwire:repwire@localhost:5432/repwire?sslmode=disable'
export SESSION_SECRET='dev-secret-change-me'

make migrate-up        # applies migrations via Dockerised migrate CLI
make run-api           # terminal 1
make run-worker        # terminal 2
```

To apply one migration directly to Supabase/PostgreSQL without Docker, load
`.env` into the process and run:

```bash
go run ./tools/apply-migration migrations/0009_engagement_suite.up.sql
```

---

## How it works

1. **Scheduler** (worker) scans `sources` every minute and enqueues `fetch_*`
   jobs for any past their `fetch_interval` (deduped per source).
2. **Fetchers** pull raw items (RSS / YouTube Data API v3 / Europe PMC / podcast
   RSS), which are **normalised** (canonical URL, tracking params stripped) and
   **deduped** in three tiers (url_hash → identifier → title similarity).
3. **process_content** classifies topics (rule-based keywords), extracts
   entities (alias matching), and scores the item.
4. **summarize** (gated by `base_score >= LLM_SCORE_THRESHOLD` and a daily USD
   budget) calls the LLM API to produce a paraphrased summary, or the fixed
   8-section research breakdown. Items below the gate go straight to `ready`.
5. **Feed / homepage** blends 50% general + 30% personal + 20% discovery.
6. **Analysis desk** ranks story clusters by heat (sources, velocity, followers,
   controversy) and commits to `EDITORIAL_PICKS_PER_DAY` of them, spaced three
   hours apart from `EDITORIAL_START_HOUR`. Each pick becomes a sourced draft for
   a human to review; nothing here publishes itself.
7. **Telegram** sends a personalised Daily Brief on a timezone-aware schedule,
   plus follow alerts when a topic you follow breaks news — rationed by a
   six-hour cooldown and your quiet hours, on top of the usual anti-spam floors.
8. **Morning audio** renders a narrated brief from the highest-ranked Vietnamese
   stories; it is cached and reused.

The job queue is pure PostgreSQL (`FOR UPDATE SKIP LOCKED`), with exponential
backoff, a dead-letter state, and a reaper for crashed workers. No Redis/RabbitMQ.

---

## Configuration

All configuration is via environment variables — see [`.env.example`](.env.example)
for the full list with descriptions. Required: `DATABASE_URL`, `SESSION_SECRET`.

To enable each capability, set:

| Capability          | Variables |
|---------------------|-----------|
| YouTube ingestion   | `YOUTUBE_API_KEY` |
| LLM summaries       | `LLM_API_KEY` (+ optional `LLM_MODEL`, `LLM_BASE_URL`, `LLM_DAILY_BUDGET_USD`, and `LLM_INPUT_USD_PER_MTOK` / `LLM_OUTPUT_USD_PER_MTOK` — the budget is computed from these, so set them to match your provider, or to 0 on a free tier) |
| Telegram digests    | `TELEGRAM_BOT_TOKEN`, `TELEGRAM_BOT_USERNAME`, `TELEGRAM_WEBHOOK_SECRET` |
| Local Telegram dev  | `TELEGRAM_POLLING=true` |
| Morning audio       | `TTS_API_KEY`, `TTS_MODEL`, `TTS_VOICE`, `MEDIA_STORAGE_DIR` |
| Web Push / PWA      | `WEB_PUSH_PUBLIC_KEY`, `WEB_PUSH_PRIVATE_KEY`, `WEB_PUSH_SUBJECT` |
| Premium / SePay     | `SEPAY_MERCHANT`, `SEPAY_SECRET_KEY`, `SEPAY_IPN_SECRET_KEY` |
| Analysis desk       | `EDITORIAL_START_HOUR` (default 9), `EDITORIAL_PICKS_PER_DAY` (default 3) |

Everything runs without these — those pipelines simply stay idle until configured.

### Telegram webhook

Point the bot at your public URL once deployed:

```

For localhost, set `TELEGRAM_POLLING=true`; the worker then uses `getUpdates`
and no public tunnel is required. Disable polling when the production webhook
is active.
https://api.telegram.org/bot<TOKEN>/setWebhook?url=<PUBLIC_BASE_URL>/api/v1/telegram/webhook&secret_token=<TELEGRAM_WEBHOOK_SECRET>
```

---

## API

Base path `/api/v1`, JSON, cookie-based auth. Success responses use
`{"data": ..., "meta": {...}}`; errors use `{"error": {"code","message"}}`.
See `internal/httpapi/server.go` for the full route table (public content,
research, videos, topics, entities, search, auth, feed, follows, saves,
telegram, and admin endpoints).

---

## Frontend

The Next.js app lives at `apps/web`. Run it locally with:

```bash
cd apps/web
npm install
npm run dev
```

It includes an installable PWA, Web Push controls, Telegram settings, the
morning audio edition, curated YouTube links and Premium checkout.

For deployment, set `DOMAIN`, `PUBLIC_BASE_URL`, database credentials and all
secrets in the server environment, then run:

```bash
docker compose -f deploy/docker-compose.yml --env-file .env up -d --build
```

The production stack now includes Next.js, API, worker, migrations, shared
audio/video storage and Caddy HTTPS. The existing Supabase database can be used
by setting `DATABASE_URL`; PostgreSQL does not need to be exposed publicly.

For the current Supabase-based production setup, follow
[`deploy/PRODUCTION.md`](deploy/PRODUCTION.md) and use
`deploy/docker-compose.prod.yml`.

For the CI/CD architecture, Cloudflare deployment, Windows pull-based updater,
rollback procedure, and interview talking points, see
[`deploy/DELIVERY_PIPELINE.md`](deploy/DELIVERY_PIPELINE.md).

The API already emits CORS headers for `CORS_ORIGINS`, so the web app can call
it directly during development.

---

## Roadmap

See the master spec. Current backend covers v0.1–v0.4 (ingestion, processing,
summaries, personalization, Telegram). v0.5 (embeddings/clustering via a Python
worker on the same `jobs` table) is intentionally deferred.
