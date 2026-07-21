ALTER TABLE analysis_candidates
  DROP COLUMN IF EXISTS checkpoint_claims,
  DROP COLUMN IF EXISTS retry_at,
  DROP COLUMN IF EXISTS progress_total,
  DROP COLUMN IF EXISTS progress_current,
  DROP COLUMN IF EXISTS progress_stage;
