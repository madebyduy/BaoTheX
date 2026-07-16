-- Anonymous per-device "thích" reactions. client_id is a random id stored in
-- the reader's browser, so counts are deduped per device without requiring login.
CREATE TABLE IF NOT EXISTS content_reactions (
    content_id BIGINT NOT NULL REFERENCES content_items(id) ON DELETE CASCADE,
    client_id  TEXT   NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (content_id, client_id)
);

CREATE INDEX IF NOT EXISTS content_reactions_content_idx ON content_reactions(content_id);
