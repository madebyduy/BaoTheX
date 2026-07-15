DROP TRIGGER IF EXISTS trg_analysis_candidate_status ON content_items;
DROP FUNCTION IF EXISTS sync_analysis_candidate_status();
DROP TABLE IF EXISTS analysis_candidates;
DROP INDEX IF EXISTS audio_briefs_date_edition_idx;
ALTER TABLE audio_briefs DROP COLUMN IF EXISTS edition;
ALTER TABLE audio_briefs ADD CONSTRAINT audio_briefs_brief_date_key UNIQUE (brief_date);
