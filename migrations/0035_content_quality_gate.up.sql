-- Persist the deterministic quality gate separately from editorial status.
-- Existing content remains pending until it is reprocessed; this avoids
-- pretending that historical rows passed checks which did not exist yet.
ALTER TABLE content_items
    ADD COLUMN quality_state TEXT NOT NULL DEFAULT 'pending'
        CHECK (quality_state IN ('pending', 'passed', 'review')),
    ADD COLUMN quality_flags TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN quality_checked_at TIMESTAMPTZ;

CREATE INDEX content_items_quality_review_idx
    ON content_items (quality_checked_at DESC)
    WHERE quality_state = 'review';
