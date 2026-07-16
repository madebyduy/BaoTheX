-- Postgres cannot remove a value from an enum. Rebuilding entity_kind without
-- 'club'/'national_team' would mean rewriting every entities row and dropping
-- the column's dependants, and it would fail anyway while 0023's rows still use
-- them. Rolling back 0023 (which deletes those rows) is the meaningful undo;
-- the two unused enum labels are harmless.
SELECT 1;
