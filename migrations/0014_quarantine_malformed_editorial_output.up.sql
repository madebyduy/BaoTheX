-- Never expose raw/truncated model JSON as reader-facing copy.
UPDATE content_items
SET status='needs_review', updated_at=now()
WHERE status='ready'
  AND (
    trim(COALESCE(translated_title,title,''))=''
    OR trim(COALESCE(summary,'')) ~ '^\{'
    OR trim(COALESCE(summary,'')) ~ '^```'
  );
