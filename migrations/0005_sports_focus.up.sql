-- BaoTheX is now a general sports newsroom. Old fitness feeds stay in the
-- database for history, but are disabled so the daily feed is sports-first.
UPDATE sources SET enabled = FALSE
WHERE kind IN ('rss', 'youtube', 'europepmc', 'podcast_rss');
UPDATE content_items c SET status = 'hidden'
FROM sources s WHERE c.source_id = s.id AND s.enabled = FALSE;

INSERT INTO topics (slug, name, description, category, keywords) VALUES
  ('bong-da-viet-nam', 'Bóng đá Việt Nam', 'V-League, đội tuyển và các giải đấu trong nước.', 'sport', ARRAY['bóng đá việt nam','v-league','đội tuyển việt nam','đội tuyển nữ việt nam','futsal việt nam','việt nam']),
  ('bong-da-quoc-te', 'Bóng đá quốc tế', 'Premier League, Champions League và các giải đấu lớn.', 'sport', ARRAY['football','soccer','premier league','champions league','la liga','serie a','bundesliga','world cup','euro']),
  ('bong-ro', 'Bóng rổ', 'NBA, WNBA và bóng rổ quốc tế.', 'sport', ARRAY['basketball','nba','wnba','euroleague']),
  ('tennis', 'Tennis', 'ATP, WTA và các giải Grand Slam.', 'sport', ARRAY['tennis','atp','wta','wimbledon','roland garros','us open','australian open']),
  ('the-thao-viet-nam', 'Thể thao Việt Nam', 'Tin tức vận động viên, SEA Games và các môn Olympic.', 'sport', ARRAY['thể thao việt nam','sea games','olympic','vận động viên','huy chương']),
  ('f1-the-thao-motor', 'F1 & thể thao motor', 'Formula 1, MotoGP và đua xe tốc độ.', 'sport', ARRAY['formula 1','f1','motogp','grand prix','đua xe']),
  ('the-thao-dien-tu', 'Thể thao điện tử', 'Esports và các giải đấu game chuyên nghiệp.', 'sport', ARRAY['esports','liên quân','valorant','league of legends','dota 2','cs2']),
  ('the-thao-khac', 'Các môn thể thao khác', 'Golf, boxing, MMA, cycling và nhiều bộ môn khác.', 'sport', ARRAY['golf','boxing','mma','ufc','cycling','volleyball','swimming','athletics'])
ON CONFLICT (slug) DO UPDATE SET name=EXCLUDED.name, description=EXCLUDED.description, category=EXCLUDED.category, keywords=EXCLUDED.keywords;

-- Vietnamese sources: VnExpress publishes a dedicated sports RSS page; the
-- other feeds are section RSS endpoints from the publications themselves.
INSERT INTO sources (kind, name, homepage_url, feed_url, quality, default_lang, enabled, fetch_interval) VALUES
  ('rss', 'VnExpress Thể thao', 'https://vnexpress.net/the-thao', 'https://vnexpress.net/rss/the-thao.rss', 5, 'vi', TRUE, '15 minutes'),
  ('rss', 'Thanh Niên Bóng đá quốc tế', 'https://thanhnien.vn/the-thao', 'https://thanhnien.vn/rss/the-thao/bong-da-quoc-te.rss', 5, 'vi', TRUE, '15 minutes'),
  ('rss', 'Soha Thể thao Việt Nam', 'https://soha.vn/the-thao.htm', 'https://soha.vn/rss/the-thao/viet-nam.rss', 4, 'vi', TRUE, '20 minutes'),
  ('rss', 'BBC Sport Football', 'https://www.bbc.com/sport/football', 'https://newsrss.bbc.co.uk/rss/sportonline_uk_edition/football/rss.xml', 5, 'en', TRUE, '15 minutes'),
  ('rss', 'The Guardian Sport', 'https://www.theguardian.com/sport', 'https://www.theguardian.com/uk/sport/rss', 4, 'en', TRUE, '20 minutes'),
  ('rss', 'Le Monde Sports', 'https://www.lemonde.fr/en/sports/', 'https://www.lemonde.fr/en/sports/rss_full.xml', 4, 'en', TRUE, '30 minutes'),
  ('rss', 'ESPN News', 'https://www.espn.com', 'https://www.espn.com/espn/rss/news', 4, 'en', TRUE, '20 minutes'),
  ('rss', 'Sky Sports Football', 'https://www.skysports.com/football', 'https://www.skysports.com/rss/12040', 4, 'en', TRUE, '15 minutes')
ON CONFLICT DO NOTHING;

-- YouTube channels are optional and only fetch when YOUTUBE_API_KEY exists.
INSERT INTO sources (kind, name, homepage_url, feed_url, quality, default_lang, enabled, fetch_interval) VALUES
  ('youtube', 'FIFA (YouTube)', 'https://www.youtube.com/@FIFA', '@FIFA', 5, 'en', TRUE, '60 minutes'),
  ('youtube', 'NBA (YouTube)', 'https://www.youtube.com/@NBA', '@NBA', 5, 'en', TRUE, '60 minutes'),
  ('youtube', 'VFF Channel', 'https://www.youtube.com/@VFFChannel', '@VFFChannel', 5, 'vi', TRUE, '60 minutes')
ON CONFLICT DO NOTHING;
