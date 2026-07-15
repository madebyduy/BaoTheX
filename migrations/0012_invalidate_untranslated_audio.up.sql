-- Never keep serving an audio edition that contains non-article or
-- untranslated foreign content. The scheduler will create a replacement with
-- the stricter Vietnamese-only selection gate.
UPDATE audio_briefs ab
SET status = 'failed',
    error = 'invalidated: contains non-article or untranslated content',
    updated_at = now()
WHERE ab.status = 'ready'
  AND EXISTS (
    SELECT 1
    FROM unnest(ab.content_ids) AS selected(content_id)
    JOIN content_items c ON c.id = selected.content_id
    LEFT JOIN content_bodies b ON b.content_id = c.id
    WHERE c.type <> 'article'
       OR (
         c.language <> 'vi'
         AND (
           b.translation_status IS DISTINCT FROM 'ready'
           OR length(trim(COALESCE(b.vietnamese_body, ''))) < 400
         )
       )
  );
