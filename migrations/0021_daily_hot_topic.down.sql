DROP INDEX IF EXISTS analysis_candidates_daily_pick_idx;

ALTER TABLE analysis_candidates
  DROP COLUMN IF EXISTS controversy_score,
  DROP COLUMN IF EXISTS action_score,
  DROP COLUMN IF EXISTS velocity_6h,
  DROP COLUMN IF EXISTS heat_terms,
  DROP COLUMN IF EXISTS picked_for_date;
