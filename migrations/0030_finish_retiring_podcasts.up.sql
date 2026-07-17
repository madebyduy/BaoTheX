-- Finish the retirement that migration 0015 started.
--
-- 0015 removed the podcast surface: it hid every ready episode and killed the
-- podcast jobs in flight. What it did not do was switch off the feeds, so the
-- scheduler kept enqueuing fetch_podcast every thirty minutes and the pipeline
-- kept ingesting, classifying and scoring episodes into a section that no longer
-- exists. Two months later that is 647 rows — 370 parked in 'processing', 266 in
-- 'needs_review', 11 that made it back to 'ready' and were visible in feeds and
-- search despite the surface being gone.
--
-- It also skewed the newsroom's own diagnostics: ESPN Daily, BBC Football Daily
-- and Guardian Football Weekly sat at the top of the "unclassified content"
-- table, which is what a podcast looks like when nothing is meant to read it.
--
-- Retiring a surface means retiring its intake. This does that.

-- 1. Stop the intake.
UPDATE sources
SET enabled = FALSE,
    last_error = 'podcast surface retired (migrations 0015, 0030)'
WHERE kind = 'podcast_rss';

-- 2. Cancel the work already queued for it. Only pending jobs: anything running
--    will finish on its own and its output is handled by step 3.
UPDATE jobs
SET status = 'dead', finished_at = now(), last_error = 'podcast surface retired'
WHERE kind = 'fetch_podcast' AND status = 'pending';

-- 3. Hide the episodes that arrived after 0015 ran. 'hidden' rather than
--    deleted: they cost nothing where they are, and a decision to drop a surface
--    is not a reason to destroy two months of ingested audio metadata that a
--    future podcast section — or an operator asking what happened here — would
--    want back.
UPDATE content_items
SET status = 'hidden', updated_at = now()
WHERE type = 'podcast' AND status <> 'hidden';
