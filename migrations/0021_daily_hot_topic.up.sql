-- The newsroom commits to one story per day rather than drafting whatever
-- happens to clear a threshold each hour. These columns record why a cluster
-- won, so an editor can see the reasoning instead of trusting a bare number.
ALTER TABLE analysis_candidates
  ADD COLUMN IF NOT EXISTS controversy_score REAL NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS action_score      REAL NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS velocity_6h       INT  NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS heat_terms        JSONB NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS picked_for_date   DATE;

-- At most one champion per day. A partial unique index both enforces that and
-- makes "have we already picked today?" a cheap lookup.
CREATE UNIQUE INDEX IF NOT EXISTS analysis_candidates_daily_pick_idx
  ON analysis_candidates (picked_for_date)
  WHERE picked_for_date IS NOT NULL;
