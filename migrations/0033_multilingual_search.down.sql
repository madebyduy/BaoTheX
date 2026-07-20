CREATE OR REPLACE FUNCTION content_tsv_update() RETURNS trigger AS $$
BEGIN
  NEW.search_tsv :=
      setweight(to_tsvector('english', unaccent(coalesce(NEW.title,''))), 'A')
   || setweight(to_tsvector('english', unaccent(coalesce(NEW.excerpt,''))), 'B')
   || setweight(to_tsvector('english', unaccent(coalesce(NEW.summary,''))), 'C');
  NEW.updated_at := now();
  RETURN NEW;
END $$ LANGUAGE plpgsql;

UPDATE content_items SET updated_at = updated_at;
