-- Two fixed daily appointments: 06:00 and 20:00.
ALTER TABLE audio_briefs
  ADD COLUMN IF NOT EXISTS edition TEXT NOT NULL DEFAULT 'morning'
  CHECK (edition IN ('morning', 'evening'));

ALTER TABLE audio_briefs DROP CONSTRAINT IF EXISTS audio_briefs_brief_date_key;
CREATE UNIQUE INDEX IF NOT EXISTS audio_briefs_date_edition_idx
  ON audio_briefs (brief_date, edition);

-- Editorial analysis desk. The scheduler proposes clusters, but a human must
-- explicitly request a draft and publish it through the existing review gate.
CREATE TABLE IF NOT EXISTS analysis_candidates (
  id                  BIGSERIAL PRIMARY KEY,
  cluster_id          BIGINT NOT NULL UNIQUE REFERENCES story_clusters(id) ON DELETE CASCADE,
  score               REAL NOT NULL DEFAULT 0,
  source_count        INT NOT NULL DEFAULT 0,
  high_quality_sources INT NOT NULL DEFAULT 0,
  velocity_24h        INT NOT NULL DEFAULT 0,
  heat_score          REAL NOT NULL DEFAULT 0,
  follower_weight     INT NOT NULL DEFAULT 0,
  status              TEXT NOT NULL DEFAULT 'proposed'
    CHECK (status IN ('proposed','drafting','needs_review','published','dismissed','failed')),
  consensus           JSONB NOT NULL DEFAULT '[]'::jsonb,
  conflicts           JSONB NOT NULL DEFAULT '[]'::jsonb,
  unique_claims       JSONB NOT NULL DEFAULT '[]'::jsonb,
  open_questions      JSONB NOT NULL DEFAULT '[]'::jsonb,
  draft_content_id    BIGINT UNIQUE REFERENCES content_items(id) ON DELETE SET NULL,
  last_error          TEXT,
  proposed_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  selected_at         TIMESTAMPTZ,
  generated_at        TIMESTAMPTZ,
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS analysis_candidates_queue_idx
  ON analysis_candidates (status, score DESC, updated_at DESC);

INSERT INTO sources (kind, name, homepage_url, quality, default_lang, enabled, fetch_interval)
SELECT 'manual', 'Góc nhìn BaoTheX', '/goc-nhin', 5, 'vi', FALSE, interval '100 years'
WHERE NOT EXISTS (SELECT 1 FROM sources WHERE kind='manual' AND name='Góc nhìn BaoTheX');

CREATE OR REPLACE FUNCTION sync_analysis_candidate_status() RETURNS trigger AS $$
BEGIN
  IF NEW.status = 'ready' AND OLD.status IS DISTINCT FROM 'ready' THEN
    UPDATE analysis_candidates
    SET status='published', updated_at=now()
    WHERE draft_content_id=NEW.id;
  END IF;
  RETURN NEW;
END $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_analysis_candidate_status ON content_items;
CREATE TRIGGER trg_analysis_candidate_status
AFTER UPDATE OF status ON content_items
FOR EACH ROW EXECUTE FUNCTION sync_analysis_candidate_status();
