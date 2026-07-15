UPDATE sources SET enabled=TRUE WHERE kind='podcast_rss';
UPDATE content_items SET status='needs_review', updated_at=now() WHERE type='podcast' AND status='hidden';
