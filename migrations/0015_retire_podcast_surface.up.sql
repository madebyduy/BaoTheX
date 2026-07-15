-- BaoTheX now prioritizes its own Vietnamese morning/evening audio editions.
-- Keep imported podcast records for audit/history, but stop fetching and publishing them.
UPDATE sources SET enabled=FALSE WHERE kind='podcast_rss';
UPDATE content_items SET status='hidden', updated_at=now() WHERE type='podcast' AND status='ready';
UPDATE jobs
SET status='dead', finished_at=now(), last_error='podcast surface retired'
WHERE kind='fetch_podcast' AND status='pending';
