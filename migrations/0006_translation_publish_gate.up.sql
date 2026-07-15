ALTER TABLE content_items ADD COLUMN IF NOT EXISTS translated_title TEXT;

-- Foreign-language stories stay out of every public feed until a Vietnamese
-- title and body have both been persisted.
UPDATE content_items c
SET status = 'processing'
WHERE c.language <> 'vi'
  AND c.status = 'ready'
  AND NOT EXISTS (
    SELECT 1 FROM content_bodies b
    WHERE b.content_id = c.id
      AND b.translation_status = 'ready'
      AND b.vietnamese_body IS NOT NULL
      AND length(trim(b.vietnamese_body)) > 0
  );

UPDATE content_bodies b
SET translation_status = 'pending', updated_at = now()
FROM content_items c
WHERE c.id = b.content_id
  AND c.language <> 'vi'
  AND b.translation_status <> 'ready';

INSERT INTO jobs (kind, payload, dedup_key, priority, run_at, max_attempts)
SELECT 'translate', jsonb_build_object('content_id', c.id), 'translate:' || c.id, 2, now(), 5
FROM content_items c
JOIN content_bodies b ON b.content_id = c.id
WHERE c.language <> 'vi'
  AND c.status = 'processing'
  AND length(trim(b.original_body)) > 0
  AND b.translation_status <> 'ready'
ON CONFLICT (dedup_key) WHERE dedup_key IS NOT NULL AND status IN ('pending','running')
DO NOTHING;
