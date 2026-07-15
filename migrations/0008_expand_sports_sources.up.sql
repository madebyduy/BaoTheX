-- Ten additional public sports RSS publishers. We ingest feed metadata,
-- preserve attribution and always link readers back to the original article.
INSERT INTO sources (kind, name, homepage_url, feed_url, quality, default_lang, enabled, fetch_interval) VALUES
  ('rss', 'DW Sports', 'https://www.dw.com/en/sports/s-8171', 'https://rss.dw.com/rdf/rss-en-sports', 5, 'en', TRUE, '30 minutes'),
  ('rss', 'France 24 Sports', 'https://www.france24.com/en/sport/', 'https://www.france24.com/en/sport/rss', 5, 'en', TRUE, '30 minutes'),
  ('rss', 'CBC Sports', 'https://www.cbc.ca/sports', 'https://www.cbc.ca/cmlink/rss-sports', 5, 'en', TRUE, '30 minutes'),
  ('rss', 'ABC Grandstand Sport', 'https://www.abc.net.au/news/sport/', 'https://www.abc.net.au/news/feed/2942460/rss.xml', 5, 'en', TRUE, '30 minutes'),
  ('rss', 'RTE Sport', 'https://www.rte.ie/sport/', 'https://www.rte.ie/feeds/rss/?index=/sport/', 5, 'en', TRUE, '30 minutes'),
  ('rss', 'CBS Sports', 'https://www.cbssports.com/', 'https://www.cbssports.com/rss/headlines/', 4, 'en', TRUE, '30 minutes'),
  ('rss', 'NPR Sports', 'https://www.npr.org/sections/sports/', 'https://feeds.npr.org/1055/rss.xml', 5, 'en', TRUE, '45 minutes'),
  ('rss', 'The Hindu Sport', 'https://www.thehindu.com/sport/', 'https://www.thehindu.com/sport/feeder/default.rss', 4, 'en', TRUE, '30 minutes'),
  ('rss', 'Cyclingnews', 'https://www.cyclingnews.com/', 'https://www.cyclingnews.com/feeds/all/', 4, 'en', TRUE, '45 minutes'),
  ('rss', 'Motorsport F1', 'https://www.motorsport.com/f1/', 'https://www.motorsport.com/rss/f1/news/', 4, 'en', TRUE, '45 minutes')
ON CONFLICT DO NOTHING;

-- Public podcast RSS requires no provider API key.
INSERT INTO sources (kind, name, homepage_url, feed_url, quality, default_lang, enabled, fetch_interval) VALUES
  ('podcast_rss', 'BBC Football Daily', 'https://www.bbc.co.uk/programmes/p02nrsln', 'https://podcasts.files.bbci.co.uk/p02nrsln.rss', 5, 'en', TRUE, '120 minutes'),
  ('podcast_rss', 'Guardian Football Weekly', 'https://www.theguardian.com/football/series/footballweekly', 'https://www.theguardian.com/football/series/footballweekly/podcast.xml', 5, 'en', TRUE, '120 minutes'),
  ('podcast_rss', 'ESPN Daily', 'https://www.espn.com/espnradio/podcast/archive/_/id/2406595', 'https://feeds.megaphone.fm/ESP8348692127', 5, 'en', TRUE, '120 minutes')
ON CONFLICT DO NOTHING;

-- Ready for activation after YOUTUBE_API_KEY is configured. Keeping these
-- disabled prevents repeated failed jobs and accidental quota waste.
INSERT INTO sources (kind, name, homepage_url, feed_url, quality, default_lang, enabled, fetch_interval) VALUES
  ('youtube', 'UEFA (YouTube)', 'https://www.youtube.com/@UEFA', '@UEFA', 5, 'en', FALSE, '60 minutes'),
  ('youtube', 'Premier League (YouTube)', 'https://www.youtube.com/@premierleague', '@premierleague', 5, 'en', FALSE, '60 minutes'),
  ('youtube', 'ESPN (YouTube)', 'https://www.youtube.com/@ESPN', '@ESPN', 5, 'en', FALSE, '60 minutes'),
  ('youtube', 'Sky Sports Premier League (YouTube)', 'https://www.youtube.com/@skysportspremierleague', '@skysportspremierleague', 5, 'en', FALSE, '60 minutes'),
  ('youtube', 'Olympics (YouTube)', 'https://www.youtube.com/@Olympics', '@Olympics', 5, 'en', FALSE, '60 minutes')
ON CONFLICT DO NOTHING;
