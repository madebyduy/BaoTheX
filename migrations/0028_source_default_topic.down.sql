-- Dropping the column takes the mappings with it. The topic assignments already
-- made through the fallback are left in place: they are indistinguishable from
-- keyword-derived rows, and deleting by topic would take the keyword ones too.
ALTER TABLE sources DROP COLUMN IF EXISTS default_topic_id;
