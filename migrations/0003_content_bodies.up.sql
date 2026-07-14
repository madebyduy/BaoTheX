CREATE TABLE content_bodies (
    content_id BIGINT PRIMARY KEY REFERENCES content_items(id) ON DELETE CASCADE,
    original_language TEXT NOT NULL DEFAULT 'en',
    original_body TEXT NOT NULL,
    vietnamese_body TEXT,
    translation_status TEXT NOT NULL DEFAULT 'pending',
    translated_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX content_bodies_translation_status_idx ON content_bodies (translation_status);
