-- Strict home-feed preference. The default remains the full newsroom.
ALTER TABLE notification_preferences
  ADD COLUMN IF NOT EXISTS feed_following_only BOOLEAN NOT NULL DEFAULT FALSE;

-- A broad Vietnamese fitness topic lets readers follow one clean channel while
-- the classifier can still attach narrower topics such as hypertrophy/protein.
INSERT INTO topics (slug, name, description, category, keywords) VALUES
  ('the-hinh', 'Thể hình & tập luyện',
   'Thể hình, gym, sức mạnh, dinh dưỡng, bodybuilding và khoa học tập luyện.',
   'sport',
   ARRAY[
     'thể hình','tập gym','phòng gym','tăng cơ','giảm mỡ','cơ bắp','bodybuilding',
     'bodybuilder','fitness','strength training','resistance training','hypertrophy',
     'muscle growth','powerlifting','strongman','weightlifting','crossfit','workout',
     'protein','creatine','mr olympia','classic physique','ifbb'
   ])
ON CONFLICT (slug) DO UPDATE SET
  name=EXCLUDED.name,
  description=EXCLUDED.description,
  category=EXCLUDED.category,
  keywords=EXCLUDED.keywords;

-- Only feeds checked as reachable are enabled. Vietnamese publishers keep their
-- original language; international feeds still pass translation/editorial gates.
INSERT INTO sources
  (kind, name, homepage_url, feed_url, quality, default_lang, enabled, fetch_interval)
SELECT v.kind::source_kind, v.name, v.homepage_url, v.feed_url, v.quality,
       v.default_lang, v.enabled, v.fetch_interval::interval
FROM (VALUES
  ('rss', 'Tuổi Trẻ Thể thao', 'https://tuoitre.vn/the-thao.htm',
   'https://tuoitre.vn/rss/the-thao.rss', 5::smallint, 'vi', TRUE, '20 minutes'),
  ('rss', 'VietnamNet Thể thao', 'https://vietnamnet.vn/the-thao',
   'https://vietnamnet.vn/rss/the-thao.rss', 5::smallint, 'vi', TRUE, '20 minutes'),
  ('rss', 'Dân trí Thể thao', 'https://dantri.com.vn/the-thao.htm',
   'https://dantri.com.vn/rss/the-thao.rss', 4::smallint, 'vi', TRUE, '20 minutes'),
  ('rss', 'BBC Sport', 'https://www.bbc.com/sport',
   'https://feeds.bbci.co.uk/sport/rss.xml', 5::smallint, 'en', TRUE, '20 minutes'),
  ('rss', 'Guardian Football', 'https://www.theguardian.com/football',
   'https://www.theguardian.com/football/rss', 5::smallint, 'en', TRUE, '20 minutes'),
  ('rss', 'Muscle & Fitness', 'https://www.muscleandfitness.com/',
   'https://www.muscleandfitness.com/feed/', 4::smallint, 'en', TRUE, '60 minutes'),
  ('rss', 'Generation Iron', 'https://generationiron.com/',
   'https://generationiron.com/feed/', 4::smallint, 'en', TRUE, '60 minutes'),
  ('rss', 'Fitness Volt', 'https://fitnessvolt.com/',
   'https://fitnessvolt.com/feed/', 4::smallint, 'en', TRUE, '60 minutes'),
  ('rss', 'Men''s Health Fitness', 'https://www.menshealth.com/fitness/',
   'https://www.menshealth.com/rss/all.xml/', 4::smallint, 'en', TRUE, '90 minutes'),
  ('rss', 'StrengthLog', 'https://www.strengthlog.com/',
   'https://www.strengthlog.com/feed/', 4::smallint, 'en', TRUE, '90 minutes'),
  ('rss', 'BOXROX', 'https://www.boxrox.com/',
   'https://www.boxrox.com/feed/', 3::smallint, 'en', TRUE, '90 minutes')
) AS v(kind, name, homepage_url, feed_url, quality, default_lang, enabled, fetch_interval)
WHERE NOT EXISTS (SELECT 1 FROM sources s WHERE s.feed_url = v.feed_url);
