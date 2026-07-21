ALTER TABLE analysis_candidates
  ADD COLUMN IF NOT EXISTS progress_stage TEXT,
  ADD COLUMN IF NOT EXISTS progress_current INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS progress_total INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS retry_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS checkpoint_claims JSONB;

-- Repair candidates left spinning by a job whose final context had already
-- expired before MarkFailed tried to write its state.
UPDATE analysis_candidates ac
SET status='failed',
    progress_stage='failed',
    last_error=COALESCE(ac.last_error,'Job da dung truoc khi cap nhat trang thai'),
    updated_at=now()
WHERE ac.status='drafting'
  AND NOT EXISTS (
    SELECT 1 FROM jobs j
    WHERE j.kind IN ('generate_cluster_analysis','generate_article_perspective')
      AND (j.payload->>'cluster_id')::bigint=ac.cluster_id
      AND j.status IN ('pending','running')
  );
