-- Public feeds contain only complete articles. Existing YouTube records are
-- safe to expose as direct links even when they have no transcript.
UPDATE content_items c
SET status = 'needs_review', updated_at = now()
WHERE c.status = 'ready'
  AND c.type = 'article'
  AND (
    NOT EXISTS (
      SELECT 1 FROM content_bodies b
      WHERE b.content_id = c.id
        AND array_length(regexp_split_to_array(trim(b.original_body), '\s+'), 1) >= 120
    )
    OR (
      c.language <> 'vi'
      AND NOT EXISTS (
        SELECT 1 FROM content_bodies b
        WHERE b.content_id = c.id
          AND b.translation_status = 'ready'
          AND length(trim(COALESCE(b.vietnamese_body, ''))) >= 400
      )
    )
  );

UPDATE content_items c
SET status = 'ready', updated_at = now()
WHERE c.type = 'video'
  AND c.status IN ('processing', 'needs_review')
  AND EXISTS (
    SELECT 1 FROM videos v
    WHERE v.content_id = c.id AND COALESCE(v.youtube_id, '') <> ''
  );
