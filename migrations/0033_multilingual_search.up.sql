-- Search is reader-facing Vietnamese even when the source article is English.
-- The old English dictionary omitted translated_title entirely, so translated
-- stories could be visible on the site but undiscoverable by their visible name.
CREATE OR REPLACE FUNCTION content_tsv_update() RETURNS trigger AS $$
BEGIN
  NEW.search_tsv :=
      setweight(to_tsvector('simple', unaccent(coalesce(NEW.translated_title, NEW.title, ''))), 'A')
   || setweight(to_tsvector('simple', unaccent(coalesce(NEW.title, ''))), 'A')
   || setweight(to_tsvector('simple', unaccent(coalesce(NEW.excerpt, ''))), 'B')
   || setweight(to_tsvector('simple', unaccent(coalesce(NEW.summary, ''))), 'C');
  NEW.updated_at := now();
  RETURN NEW;
END $$ LANGUAGE plpgsql;

-- Re-fire the trigger for existing rows so the migration takes effect
-- immediately instead of only after future editorial updates.
UPDATE content_items SET updated_at = updated_at;
