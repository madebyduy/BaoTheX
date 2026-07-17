-- Restore the dead URLs and the disabled state. This is a faithful rollback, not
-- a sensible configuration: every URL below is known broken.
UPDATE sources SET
  feed_url = 'https://newsrss.bbc.co.uk/rss/sportonline_uk_edition/football/rss.xml',
  enabled = FALSE, etag = NULL, last_modified = NULL
WHERE name = 'BBC Sport Football';

UPDATE sources SET
  feed_url = 'https://www.cbc.ca/cmlink/rss-sports',
  enabled = FALSE, etag = NULL, last_modified = NULL
WHERE name = 'CBC Sports';

UPDATE sources SET
  feed_url = '@VFFChannel', enabled = FALSE, uploads_playlist_id = NULL
WHERE name = 'VFF Channel';

UPDATE sources SET enabled = FALSE WHERE name = 'ESPN (YouTube)';

UPDATE sources SET enabled = TRUE, last_error = NULL WHERE name = 'BWF Badminton';
