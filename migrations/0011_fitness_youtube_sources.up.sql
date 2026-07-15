-- Curated evidence-led fitness channels. The fetcher uses uploads playlists
-- (two cheap API calls per refresh) and stores playable YouTube links.
INSERT INTO sources
  (kind, name, homepage_url, feed_url, quality, default_lang, enabled, fetch_interval)
SELECT v.kind::source_kind, v.name, v.homepage_url, v.feed_url, v.quality,
       v.default_lang, v.enabled, v.fetch_interval::interval
FROM (VALUES
  ('youtube', 'ATHLEAN-X (YouTube)', 'https://www.youtube.com/@athleanx', '@athleanx', 4::smallint, 'en', TRUE, '120 minutes'),
  ('youtube', 'Jeremy Ethier (YouTube)', 'https://www.youtube.com/@JeremyEthier', '@JeremyEthier', 4::smallint, 'en', TRUE, '120 minutes'),
  ('youtube', 'Squat University (YouTube)', 'https://www.youtube.com/@SquatUniversity', '@SquatUniversity', 4::smallint, 'en', TRUE, '120 minutes'),
  ('youtube', 'PictureFit (YouTube)', 'https://www.youtube.com/@PictureFit', '@PictureFit', 4::smallint, 'en', TRUE, '120 minutes'),
  ('youtube', 'House of Hypertrophy (YouTube)', 'https://www.youtube.com/@HouseofHypertrophy', '@HouseofHypertrophy', 4::smallint, 'en', TRUE, '120 minutes')
) AS v(kind, name, homepage_url, feed_url, quality, default_lang, enabled, fetch_interval)
WHERE NOT EXISTS (SELECT 1 FROM sources s WHERE s.feed_url = v.feed_url);
