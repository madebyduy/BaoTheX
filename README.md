# BaoTheX

BaoTheX — nền tảng tổng hợp bài viết, nghiên cứu, video và podcast fitness cho người Việt.

## Backend

Fitness information hub: aggregates and explains new content about gym, fitness
and training science from articles, research, YouTube and podcasts. Users follow
topics / people / sources, read sourced summaries, save content and receive a
personalised Telegram brief.

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
│   └── web/            # Next.js (added later — not in this repo yet)
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
6. **Telegram** sends a personalised Daily Brief and Weekly Research digest via
   timezone-aware scheduling, with strict anti-spam thresholds.

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
| LLM summaries       | `LLM_API_KEY` (+ optional `LLM_MODEL`, `LLM_BASE_URL`, `LLM_DAILY_BUDGET_USD`) |
| Telegram digests    | `TELEGRAM_BOT_TOKEN`, `TELEGRAM_BOT_USERNAME`, `TELEGRAM_WEBHOOK_SECRET` |

Everything runs without these — those pipelines simply stay idle until configured.

### Telegram webhook

Point the bot at your public URL once deployed:

```
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

The Next.js app lives at `apps/web` and is **built later**. When it exists:

1. Add a `web` service to `deploy/docker-compose.yml` (build `apps/web/Dockerfile`,
   env `API_URL=http://api:8080`, port `3000`).
2. Uncomment the `caddy` service + `web` route in `deploy/Caddyfile`.

The API already emits CORS headers for `CORS_ORIGINS`, so the web app can call
it directly during development.

---

## Roadmap

See the master spec. Current backend covers v0.1–v0.4 (ingestion, processing,
summaries, personalization, Telegram). v0.5 (embeddings/clustering via a Python
worker on the same `jobs` table) is intentionally deferred.
