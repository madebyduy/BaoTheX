CREATE TABLE IF NOT EXISTS story_clusters (
    id BIGSERIAL PRIMARY KEY,
    representative_title TEXT NOT NULL,
    primary_content_id BIGINT REFERENCES content_items(id) ON DELETE SET NULL,
    verification_status TEXT NOT NULL DEFAULT 'rumor'
        CHECK (verification_status IN ('rumor','verifying','confirmed')),
    source_count INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS story_cluster_items (
    cluster_id BIGINT NOT NULL REFERENCES story_clusters(id) ON DELETE CASCADE,
    content_id BIGINT NOT NULL UNIQUE REFERENCES content_items(id) ON DELETE CASCADE,
    similarity REAL NOT NULL DEFAULT 1,
    added_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (cluster_id, content_id)
);

CREATE INDEX IF NOT EXISTS story_clusters_updated_idx ON story_clusters(updated_at DESC);
CREATE INDEX IF NOT EXISTS story_cluster_items_cluster_idx ON story_cluster_items(cluster_id);
