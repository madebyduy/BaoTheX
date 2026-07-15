-- Engagement suite: Premium billing, daily audio/video briefs and Web Push.

CREATE TABLE user_subscriptions (
    user_id            BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    plan               TEXT NOT NULL DEFAULT 'premium',
    status             TEXT NOT NULL DEFAULT 'inactive' CHECK (status IN ('inactive','pending','active','expired','cancelled')),
    provider           TEXT NOT NULL DEFAULT 'sepay',
    current_period_end TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE payment_orders (
    id                   BIGSERIAL PRIMARY KEY,
    user_id              BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invoice_number       TEXT NOT NULL UNIQUE,
    amount_vnd           BIGINT NOT NULL CHECK (amount_vnd > 0),
    status               TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','paid','cancelled','failed')),
    provider             TEXT NOT NULL DEFAULT 'sepay',
    provider_order_id    TEXT,
    provider_transaction TEXT UNIQUE,
    paid_at              TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ON payment_orders (user_id, created_at DESC);

CREATE TABLE audio_briefs (
    id               BIGSERIAL PRIMARY KEY,
    brief_date       DATE NOT NULL UNIQUE,
    title            TEXT NOT NULL,
    script           TEXT NOT NULL,
    audio_url        TEXT,
    duration_seconds INT,
    content_ids      BIGINT[] NOT NULL DEFAULT '{}',
    status           TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','rendering','ready','failed')),
    error            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE video_briefs (
    id               BIGSERIAL PRIMARY KEY,
    brief_date       DATE NOT NULL UNIQUE,
    title            TEXT NOT NULL,
    script           TEXT NOT NULL,
    video_url        TEXT,
    thumbnail_url    TEXT,
    duration_seconds INT,
    content_ids      BIGINT[] NOT NULL DEFAULT '{}',
    youtube_video_id TEXT,
    status           TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','rendering','ready','published','failed')),
    error            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE push_subscriptions (
    endpoint    TEXT PRIMARY KEY,
    user_id     BIGINT REFERENCES users(id) ON DELETE CASCADE,
    p256dh      TEXT NOT NULL,
    auth        TEXT NOT NULL,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ON push_subscriptions (user_id);

ALTER TABLE notification_preferences
  ADD COLUMN audio_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN push_enabled BOOLEAN NOT NULL DEFAULT FALSE;

-- The worker can now fall back to public channel RSS when no YouTube API key
-- is present, so approved channels are safe to enable.
UPDATE sources SET enabled = TRUE, last_error = NULL, consecutive_failures = 0
WHERE kind = 'youtube';
