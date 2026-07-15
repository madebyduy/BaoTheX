-- Invalid audio must not be made public again on rollback. Only clear the
-- migration marker so operators can inspect and regenerate it explicitly.
UPDATE audio_briefs
SET error = NULL, updated_at = now()
WHERE status = 'failed'
  AND error = 'invalidated: contains non-article or untranslated content';
