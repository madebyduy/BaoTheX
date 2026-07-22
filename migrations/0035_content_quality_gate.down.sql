DROP INDEX IF EXISTS content_items_quality_review_idx;

ALTER TABLE content_items
    DROP COLUMN IF EXISTS quality_checked_at,
    DROP COLUMN IF EXISTS quality_flags,
    DROP COLUMN IF EXISTS quality_state;
