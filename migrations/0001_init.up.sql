-- ============================================================================
-- RepWire — initial schema
-- ============================================================================

-- ============ EXTENSIONS ============
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS citext;

-- ============ SOURCES ============
CREATE TYPE source_kind AS ENUM ('rss','youtube','europepmc','podcast_rss','sitemap','manual');

CREATE TABLE sources (
    id                   BIGSERIAL PRIMARY KEY,
    kind                 source_kind NOT NULL,
    name                 TEXT NOT NULL,
    homepage_url         TEXT,
    feed_url             TEXT,               -- rss url / channel id / pmc query
    quality              SMALLINT NOT NULL DEFAULT 3 CHECK (quality BETWEEN 1 AND 5),
    default_lang         TEXT NOT NULL DEFAULT 'en',
    enabled              BOOLEAN NOT NULL DEFAULT TRUE,
    fetch_interval       INTERVAL NOT NULL DEFAULT '30 minutes',
    -- Conditional GET state for RSS/podcast fetchers.
    etag                 TEXT,
    last_modified        TEXT,
    -- YouTube: resolved uploads playlist id, cached so we skip channels.list.
    uploads_playlist_id  TEXT,
    last_fetched_at      TIMESTAMPTZ,
    last_error           TEXT,
    consecutive_failures INT NOT NULL DEFAULT 0,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ON sources (enabled, last_fetched_at);

-- ============ CONTENT ============
CREATE TYPE content_type   AS ENUM ('article','research','video','podcast','announcement','event');
-- 'needs_review' added on top of the spec's set so low-confidence items land in admin.
CREATE TYPE content_status AS ENUM ('discovered','fetching','processing','ready','failed','hidden','needs_review');

CREATE TABLE content_items (
    id              BIGSERIAL PRIMARY KEY,
    source_id       BIGINT NOT NULL REFERENCES sources(id) ON DELETE RESTRICT,
    type            content_type   NOT NULL,
    status          content_status NOT NULL DEFAULT 'discovered',
    title           TEXT NOT NULL,
    canonical_url   TEXT NOT NULL,
    url_hash        TEXT NOT NULL,
    title_hash      TEXT,
    image_url       TEXT,
    excerpt         TEXT,
    summary         TEXT,
    key_points      JSONB NOT NULL DEFAULT '[]'::jsonb,
    language        TEXT NOT NULL DEFAULT 'en',
    published_at    TIMESTAMPTZ,
    discovered_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    base_score      REAL NOT NULL DEFAULT 0,
    editorial_boost REAL NOT NULL DEFAULT 0,
    final_score     REAL GENERATED ALWAYS AS (base_score + editorial_boost) STORED,
    view_count      INT  NOT NULL DEFAULT 0,
    save_count      INT  NOT NULL DEFAULT 0,
    search_tsv      TSVECTOR,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uniq_url_hash UNIQUE (url_hash)
);
CREATE INDEX ON content_items (status, published_at DESC);
CREATE INDEX ON content_items (type, published_at DESC) WHERE status = 'ready';
CREATE INDEX ON content_items (final_score DESC, published_at DESC) WHERE status = 'ready';
CREATE INDEX ON content_items USING GIN (search_tsv);
CREATE INDEX ON content_items USING GIN (title gin_trgm_ops);

CREATE FUNCTION content_tsv_update() RETURNS trigger AS $$
BEGIN
  NEW.search_tsv :=
      setweight(to_tsvector('english', unaccent(coalesce(NEW.title,''))), 'A')
   || setweight(to_tsvector('english', unaccent(coalesce(NEW.excerpt,''))), 'B')
   || setweight(to_tsvector('english', unaccent(coalesce(NEW.summary,''))), 'C');
  NEW.updated_at := now();
  RETURN NEW;
END $$ LANGUAGE plpgsql;

CREATE TRIGGER trg_content_tsv BEFORE INSERT OR UPDATE ON content_items
FOR EACH ROW EXECUTE FUNCTION content_tsv_update();

-- ============ SUBTYPE TABLES ============
CREATE TABLE articles (
    content_id BIGINT PRIMARY KEY REFERENCES content_items(id) ON DELETE CASCADE,
    author     TEXT,
    word_count INT
);

CREATE TYPE study_type AS ENUM
  ('meta_analysis','systematic_review','rct','cohort','cross_sectional','case_study','narrative_review','other');

CREATE TABLE research_papers (
    content_id        BIGINT PRIMARY KEY REFERENCES content_items(id) ON DELETE CASCADE,
    doi               TEXT,
    pmid              TEXT,
    pmcid             TEXT,
    journal           TEXT,
    authors           JSONB NOT NULL DEFAULT '[]'::jsonb,
    abstract          TEXT,
    study_type        study_type NOT NULL DEFAULT 'other',
    is_human          BOOLEAN,
    is_open_access    BOOLEAN NOT NULL DEFAULT FALSE,
    full_text_url     TEXT,
    sample_size       INT,
    population        TEXT,       -- "24 trained males, 20-35y"
    duration_weeks    INT,
    sex               TEXT,       -- male | female | mixed
    training_status   TEXT,       -- untrained | recreational | trained | athlete
    -- Research breakdown (AI + admin)
    bd_question       TEXT,
    bd_participants   TEXT,
    bd_intervention   TEXT,
    bd_findings       JSONB NOT NULL DEFAULT '[]'::jsonb,  -- 3-5 y
    bd_not_proven     TEXT,
    bd_limitations    JSONB NOT NULL DEFAULT '[]'::jsonb,
    bd_practical      TEXT,
    funding_note      TEXT,
    published_year    INT
);
CREATE UNIQUE INDEX ON research_papers (doi) WHERE doi IS NOT NULL;
CREATE UNIQUE INDEX ON research_papers (pmid) WHERE pmid IS NOT NULL;
CREATE INDEX ON research_papers (study_type, published_year DESC);

CREATE TABLE videos (
    content_id     BIGINT PRIMARY KEY REFERENCES content_items(id) ON DELETE CASCADE,
    youtube_id     TEXT NOT NULL UNIQUE,
    channel_id     TEXT NOT NULL,
    channel_title  TEXT NOT NULL,
    duration_sec   INT,
    thumbnail_url  TEXT,
    description    TEXT,
    has_transcript BOOLEAN NOT NULL DEFAULT FALSE,
    transcript     TEXT,
    timeline       JSONB NOT NULL DEFAULT '[]'::jsonb,  -- [{t:123,label:"..."}]
    yt_views       BIGINT,
    yt_likes       BIGINT
);

CREATE TABLE podcast_episodes (
    content_id   BIGINT PRIMARY KEY REFERENCES content_items(id) ON DELETE CASCADE,
    show_name    TEXT NOT NULL,
    episode_guid TEXT NOT NULL UNIQUE,
    audio_url    TEXT,
    duration_sec INT,
    show_notes   TEXT,
    transcript   TEXT
);

-- ============ TOPICS ============
CREATE TABLE topics (
    id             BIGSERIAL PRIMARY KEY,
    slug           TEXT NOT NULL UNIQUE,          -- hypertrophy, creatine, fat-loss
    name           TEXT NOT NULL,
    description    TEXT,
    category       TEXT,                          -- goal | discipline | knowledge
    keywords       TEXT[] NOT NULL DEFAULT '{}',  -- rule-based classify
    follower_count INT NOT NULL DEFAULT 0
);

CREATE TABLE content_topics (
    content_id BIGINT REFERENCES content_items(id) ON DELETE CASCADE,
    topic_id   BIGINT REFERENCES topics(id)        ON DELETE CASCADE,
    confidence REAL NOT NULL DEFAULT 1.0,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (content_id, topic_id)
);
CREATE INDEX ON content_topics (topic_id);

CREATE TABLE topic_relations (
    topic_id    BIGINT REFERENCES topics(id) ON DELETE CASCADE,
    related_id  BIGINT REFERENCES topics(id) ON DELETE CASCADE,
    PRIMARY KEY (topic_id, related_id)
);

-- ============ ENTITIES ============
CREATE TYPE entity_kind AS ENUM
  ('athlete','coach','researcher','creator','channel','podcast','competition','federation','brand','publication');

CREATE TABLE entities (
    id             BIGSERIAL PRIMARY KEY,
    slug           TEXT NOT NULL UNIQUE,
    name           TEXT NOT NULL,
    kind           entity_kind NOT NULL,
    bio            TEXT,
    avatar_url     TEXT,
    expertise      TEXT[] NOT NULL DEFAULT '{}',
    official_links JSONB NOT NULL DEFAULT '[]'::jsonb,  -- [{type:"youtube",url:"..."}]
    aliases        TEXT[] NOT NULL DEFAULT '{}',        -- match in text
    follower_count INT NOT NULL DEFAULT 0
);
CREATE INDEX ON entities USING GIN (aliases);

CREATE TABLE content_entities (
    content_id BIGINT REFERENCES content_items(id) ON DELETE CASCADE,
    entity_id  BIGINT REFERENCES entities(id)      ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'mentioned',   -- author | mentioned | subject
    PRIMARY KEY (content_id, entity_id, role)
);
CREATE INDEX ON content_entities (entity_id);

-- Content links (v0.5): video X analyses paper Y
CREATE TABLE content_links (
    from_content_id BIGINT REFERENCES content_items(id) ON DELETE CASCADE,
    to_content_id   BIGINT REFERENCES content_items(id) ON DELETE CASCADE,
    relation        TEXT NOT NULL,   -- discusses | same_study | followup | duplicate_of
    PRIMARY KEY (from_content_id, to_content_id, relation)
);

-- ============ USERS ============
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         CITEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT,
    role          TEXT NOT NULL DEFAULT 'user',   -- user | admin
    goals         TEXT[] NOT NULL DEFAULT '{}',   -- muscle_gain, fat_loss, strength, health, competition
    timezone      TEXT NOT NULL DEFAULT 'Asia/Ho_Chi_Minh',
    onboarded_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ON sessions (user_id);

-- ============ FOLLOWS ============
CREATE TABLE user_topic_follows (
    user_id          BIGINT REFERENCES users(id)  ON DELETE CASCADE,
    topic_id         BIGINT REFERENCES topics(id) ON DELETE CASCADE,
    in_feed          BOOLEAN NOT NULL DEFAULT TRUE,
    in_telegram      BOOLEAN NOT NULL DEFAULT TRUE,
    highlights_only  BOOLEAN NOT NULL DEFAULT FALSE,
    priority         SMALLINT NOT NULL DEFAULT 0,  -- 0 normal, 1 high
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, topic_id)
);

CREATE TABLE user_entity_follows (
    user_id         BIGINT REFERENCES users(id)    ON DELETE CASCADE,
    entity_id       BIGINT REFERENCES entities(id) ON DELETE CASCADE,
    in_feed         BOOLEAN NOT NULL DEFAULT TRUE,
    in_telegram     BOOLEAN NOT NULL DEFAULT TRUE,
    highlights_only BOOLEAN NOT NULL DEFAULT FALSE,
    priority        SMALLINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, entity_id)
);

CREATE TABLE user_source_follows (
    user_id    BIGINT REFERENCES users(id)   ON DELETE CASCADE,
    source_id  BIGINT REFERENCES sources(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, source_id)
);

CREATE TABLE user_topic_mutes (
    user_id  BIGINT REFERENCES users(id)  ON DELETE CASCADE,
    topic_id BIGINT REFERENCES topics(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, topic_id)
);

-- ============ SAVE / HISTORY ============
CREATE TABLE collections (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, name)
);

CREATE TABLE saved_items (
    user_id       BIGINT REFERENCES users(id)         ON DELETE CASCADE,
    content_id    BIGINT REFERENCES content_items(id) ON DELETE CASCADE,
    collection_id BIGINT REFERENCES collections(id)   ON DELETE SET NULL,
    note          TEXT,
    saved_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, content_id)
);
CREATE INDEX ON saved_items (user_id, saved_at DESC);

CREATE TABLE reading_history (
    user_id    BIGINT REFERENCES users(id)         ON DELETE CASCADE,
    content_id BIGINT REFERENCES content_items(id) ON DELETE CASCADE,
    read_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, content_id)
);
CREATE INDEX ON reading_history (user_id, read_at DESC);

CREATE TABLE hidden_items (
    user_id    BIGINT REFERENCES users(id)         ON DELETE CASCADE,
    content_id BIGINT REFERENCES content_items(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, content_id)
);

-- ============ TELEGRAM ============
CREATE TABLE telegram_connections (
    user_id   BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    chat_id   BIGINT NOT NULL UNIQUE,
    username  TEXT,
    link_code TEXT,           -- one-time code for /start <code>
    linked_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- Pending link codes (created before the /start handshake completes).
CREATE TABLE telegram_link_codes (
    code       TEXT PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ON telegram_link_codes (user_id);

CREATE TABLE notification_preferences (
    user_id         BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    daily_enabled   BOOLEAN NOT NULL DEFAULT TRUE,
    daily_hour      SMALLINT NOT NULL DEFAULT 7,     -- user timezone
    daily_days      SMALLINT[] NOT NULL DEFAULT '{1,2,3,4,5,6,7}',
    daily_max_items SMALLINT NOT NULL DEFAULT 5 CHECK (daily_max_items BETWEEN 3 AND 7),
    weekly_research BOOLEAN NOT NULL DEFAULT TRUE,
    weekly_dow      SMALLINT NOT NULL DEFAULT 1,     -- Monday
    follow_alerts   BOOLEAN NOT NULL DEFAULT TRUE,
    highlights_only BOOLEAN NOT NULL DEFAULT FALSE,
    quiet_start     SMALLINT NOT NULL DEFAULT 22,
    quiet_end       SMALLINT NOT NULL DEFAULT 7,
    content_types   content_type[] NOT NULL DEFAULT '{article,research,video}'
);

CREATE TYPE digest_kind AS ENUM ('daily','weekly_research','follow_alert');

CREATE TABLE digest_deliveries (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind        digest_kind NOT NULL,
    content_ids BIGINT[] NOT NULL,
    sent_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Materialised send date (UTC) so the anti-double-send index can be a plain,
    -- immutable column index. Set at insert via DEFAULT.
    sent_date   DATE NOT NULL DEFAULT ((now() AT TIME ZONE 'UTC')::date),
    message_id  BIGINT,
    error       TEXT
);
CREATE INDEX ON digest_deliveries (user_id, kind, sent_at DESC);
-- Anti double-send: 1 daily / user / day.
CREATE UNIQUE INDEX uniq_daily_per_day
  ON digest_deliveries (user_id, kind, sent_date)
  WHERE kind = 'daily' AND error IS NULL;

-- ============ JOBS ============
CREATE TYPE job_status AS ENUM ('pending','running','done','failed','dead');

CREATE TABLE jobs (
    id           BIGSERIAL PRIMARY KEY,
    kind         TEXT NOT NULL,          -- fetch_rss, fetch_youtube, fetch_pmc,
                                         -- process_content, summarize, classify,
                                         -- send_daily, send_weekly
    payload      JSONB NOT NULL DEFAULT '{}'::jsonb,
    dedup_key    TEXT,                   -- prevent duplicate enqueue
    status       job_status NOT NULL DEFAULT 'pending',
    priority     SMALLINT NOT NULL DEFAULT 0,
    run_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    attempts     SMALLINT NOT NULL DEFAULT 0,
    max_attempts SMALLINT NOT NULL DEFAULT 5,
    locked_by    TEXT,
    locked_at    TIMESTAMPTZ,
    last_error   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at  TIMESTAMPTZ
);
CREATE UNIQUE INDEX ON jobs (dedup_key) WHERE dedup_key IS NOT NULL AND status IN ('pending','running');
CREATE INDEX ON jobs (status, run_at, priority DESC);

-- ============ LLM USAGE (cost tracking) ============
CREATE TABLE llm_usage (
    id           BIGSERIAL PRIMARY KEY,
    day          DATE NOT NULL DEFAULT (now()::date),
    model        TEXT NOT NULL,
    input_tokens INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    cost_usd     NUMERIC(10,5) NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ON llm_usage (day);
