-- Podcast feeds are first-class sources, so the scheduler can fetch them
-- without requiring a separate manual job.
INSERT INTO sources (kind, name, homepage_url, feed_url, quality, default_lang, fetch_interval) VALUES
  ('podcast_rss', 'Huberman Lab', 'https://hubermanlab.com', 'https://feeds.megaphone.fm/hubermanlab', 4, 'en', '180 minutes'),
  ('podcast_rss', 'Mind Pump', 'https://mindpumppodcast.com', 'https://feeds.megaphone.fm/mindpump', 3, 'en', '180 minutes')
ON CONFLICT DO NOTHING;
