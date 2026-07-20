-- Broaden football and French sports coverage with the publishers requested by
-- the newsroom. The feeds remain ordinary RSS sources, so their content still
-- passes the existing translation, verification and editorial gates.
INSERT INTO sources
  (kind, name, homepage_url, feed_url, quality, default_lang, enabled, fetch_interval)
SELECT v.kind::source_kind, v.name, v.homepage_url, v.feed_url, v.quality,
       v.default_lang, TRUE, v.fetch_interval::interval
FROM (VALUES
  ('rss', 'The Sun Sport', 'https://www.thesun.co.uk/sport/',
   'https://www.thesun.co.uk/sport/feed/', 3::smallint, 'en', '30 minutes'),
  ('rss', 'RMC Sport', 'https://rmcsport.bfmtv.com/',
   'https://rmcsport.bfmtv.com/rss/fil-sport/', 4::smallint, 'fr', '30 minutes')
) AS v(kind, name, homepage_url, feed_url, quality, default_lang, fetch_interval)
WHERE NOT EXISTS (SELECT 1 FROM sources s WHERE s.feed_url=v.feed_url);
