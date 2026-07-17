-- Zero-budget sports companion: curated/free-provider events, private desks,
-- predictions and first-party product analytics. All additions are optional;
-- the existing news pipeline continues to work without provider credentials.

CREATE TABLE sports (
    id          BIGSERIAL PRIMARY KEY,
    slug        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO sports (slug, name) VALUES
  ('football', 'Bóng đá'),
  ('basketball', 'Bóng rổ'),
  ('tennis', 'Tennis'),
  ('badminton', 'Cầu lông'),
  ('formula-1', 'Formula 1'),
  ('esports', 'Thể thao điện tử')
ON CONFLICT (slug) DO UPDATE SET name=EXCLUDED.name, enabled=TRUE;

CREATE TABLE sports_competitions (
    id           BIGSERIAL PRIMARY KEY,
    sport_id     BIGINT NOT NULL REFERENCES sports(id) ON DELETE CASCADE,
    slug         TEXT NOT NULL UNIQUE,
    name         TEXT NOT NULL,
    country      TEXT,
    data_source  TEXT NOT NULL DEFAULT 'baothex',
    external_id  TEXT,
    coverage     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (data_source, external_id)
);

CREATE TABLE sports_events (
    id              BIGSERIAL PRIMARY KEY,
    sport_id        BIGINT NOT NULL REFERENCES sports(id) ON DELETE RESTRICT,
    competition_id  BIGINT REFERENCES sports_competitions(id) ON DELETE SET NULL,
    title           TEXT NOT NULL,
    home_name       TEXT,
    away_name       TEXT,
    starts_at       TIMESTAMPTZ NOT NULL,
    status          TEXT NOT NULL DEFAULT 'scheduled'
                    CHECK (status IN ('scheduled','live','finished','postponed','cancelled')),
    home_score      TEXT,
    away_score      TEXT,
    result_details  JSONB NOT NULL DEFAULT '{}'::jsonb,
    data_source     TEXT NOT NULL DEFAULT 'baothex',
    external_id     TEXT,
    data_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    freshness       TEXT NOT NULL DEFAULT 'manual'
                    CHECK (freshness IN ('live','delayed','scheduled','manual','stale')),
    is_manual       BOOLEAN NOT NULL DEFAULT TRUE,
    manual_locked   BOOLEAN NOT NULL DEFAULT FALSE,
    coverage        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX sports_events_provider_id_idx
  ON sports_events (data_source, external_id) WHERE external_id IS NOT NULL;
CREATE INDEX sports_events_date_idx ON sports_events (starts_at DESC);
CREATE INDEX sports_events_sport_idx ON sports_events (sport_id, starts_at DESC);

CREATE TABLE event_content_links (
    event_id    BIGINT NOT NULL REFERENCES sports_events(id) ON DELETE CASCADE,
    content_id  BIGINT NOT NULL REFERENCES content_items(id) ON DELETE CASCADE,
    relation    TEXT NOT NULL DEFAULT 'coverage',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, content_id)
);

CREATE TABLE user_event_follows (
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_id   BIGINT NOT NULL REFERENCES sports_events(id) ON DELETE CASCADE,
    remind_at  TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, event_id)
);

CREATE TABLE user_source_mutes (
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_id  BIGINT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, source_id)
);

CREATE TABLE user_dashboard_layouts (
    user_id    BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    layout     JSONB NOT NULL DEFAULT '["today","catch_up","schedule","favorites","following","read_later","listen_later","predictions"]'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE story_cluster_follows (
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cluster_id BIGINT NOT NULL REFERENCES story_clusters(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, cluster_id)
);

CREATE TABLE story_cluster_reads (
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cluster_id BIGINT NOT NULL REFERENCES story_clusters(id) ON DELETE CASCADE,
    last_read_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, cluster_id)
);

CREATE TABLE predictions (
    id              BIGSERIAL PRIMARY KEY,
    event_id        BIGINT REFERENCES sports_events(id) ON DELETE SET NULL,
    kind            TEXT NOT NULL DEFAULT 'poll' CHECK (kind IN ('winner','score','player','quiz','poll')),
    question        TEXT NOT NULL,
    options         JSONB NOT NULL,
    correct_option  TEXT,
    deadline        TIMESTAMPTZ NOT NULL,
    status          TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','settled','cancelled')),
    points          INT NOT NULL DEFAULT 10 CHECK (points >= 0 AND points <= 1000),
    created_by      BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    settled_at      TIMESTAMPTZ
);
CREATE INDEX predictions_open_idx ON predictions (deadline) WHERE status='open';

CREATE TABLE prediction_answers (
    prediction_id BIGINT NOT NULL REFERENCES predictions(id) ON DELETE CASCADE,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    answer        TEXT NOT NULL,
    is_correct    BOOLEAN,
    points_earned INT NOT NULL DEFAULT 0,
    answered_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (prediction_id, user_id)
);

CREATE TABLE product_events (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT REFERENCES users(id) ON DELETE SET NULL,
    client_id   TEXT,
    event_name  TEXT NOT NULL,
    properties  JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX product_events_user_time_idx ON product_events (user_id, occurred_at DESC);
CREATE INDEX product_events_name_time_idx ON product_events (event_name, occurred_at DESC);
