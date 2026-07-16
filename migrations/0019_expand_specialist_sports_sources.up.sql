-- Expand specialist coverage with feeds verified reachable on 2026-07-15.
-- International content still passes the Vietnamese translation/editorial gate
-- before it can become public.

INSERT INTO topics (slug, name, description, category, keywords) VALUES
  ('cau-long', 'Cầu lông',
   'BWF World Tour, cầu lông châu Á, các giải quốc tế và vận động viên Việt Nam.',
   'sport',
   ARRAY[
     'cầu lông','badminton','bwf','bwf world tour','thomas cup','uber cup','sudirman cup',
     'all england','viktor axelsen','shi yu qi','an se young','nguyễn thùy linh','nguyen thuy linh'
   ])
ON CONFLICT (slug) DO UPDATE SET
  name=EXCLUDED.name,
  description=EXCLUDED.description,
  category=EXCLUDED.category,
  keywords=EXCLUDED.keywords;

UPDATE topics SET keywords=ARRAY[
  'esports','thể thao điện tử','liên minh huyền thoại','league of legends','lol esports',
  'valorant','valorant champions tour','counter-strike','counter strike','cs2','dota 2',
  'pubg','mobile legends','liên quân','arena of valor','vct','lck','lec','lpl','msi'
] WHERE slug='the-thao-dien-tu';

UPDATE topics SET keywords=ARRAY[
  'tennis','quần vợt','atp','wta','atp tour','wta tour','wimbledon','roland garros',
  'french open','us open','australian open','davis cup','billie jean king cup','grand slam'
] WHERE slug='tennis';

UPDATE topics SET keywords=ARRAY[
  'thể hình','tập gym','phòng gym','tăng cơ','giảm mỡ','cơ bắp','bodybuilding','bodybuilder',
  'fitness','strength training','resistance training','hypertrophy','muscle growth','powerlifting',
  'strongman','weightlifting','crossfit','workout','protein','creatine','mr olympia','classic physique','ifbb'
] WHERE slug='the-hinh';

INSERT INTO sources
  (kind, name, homepage_url, feed_url, quality, default_lang, enabled, fetch_interval)
SELECT v.kind::source_kind, v.name, v.homepage_url, v.feed_url, v.quality,
       v.default_lang, TRUE, v.fetch_interval::interval
FROM (VALUES
  ('rss', 'Dot Esports', 'https://dotesports.com/',
   'https://dotesports.com/feed', 4::smallint, 'en', '20 minutes'),
  ('rss', 'BWF Badminton', 'https://bwfbadminton.com/',
   'https://bwfbadminton.com/feed/', 5::smallint, 'en', '30 minutes'),
  ('rss', 'Badminton Asia', 'https://badmintonasia.org/',
   'https://badmintonasia.org/feed/', 4::smallint, 'en', '30 minutes'),
  ('rss', 'Badminton Planet', 'https://www.badmintonplanet.com/',
   'https://www.badmintonplanet.com/feed', 3::smallint, 'en', '30 minutes'),
  ('rss', 'Tennis365', 'https://www.tennis365.com/',
   'https://www.tennis365.com/feed', 4::smallint, 'en', '20 minutes'),
  ('rss', 'Tennishead', 'https://tennishead.net/',
   'https://tennishead.net/feed/', 4::smallint, 'en', '30 minutes'),
  ('rss', 'Ubitennis', 'https://www.ubitennis.net/',
   'https://www.ubitennis.net/feed/', 4::smallint, 'en', '30 minutes'),
  ('youtube', 'BarBend (YouTube)', 'https://www.youtube.com/@BarBend',
   '@BarBend', 4::smallint, 'en', '90 minutes'),
  ('youtube', 'LoL Esports (YouTube)', 'https://www.youtube.com/@lolesports',
   '@lolesports', 5::smallint, 'en', '60 minutes'),
  ('youtube', 'VALORANT Champions Tour (YouTube)', 'https://www.youtube.com/@valorantesports',
   '@valorantesports', 5::smallint, 'en', '60 minutes'),
  ('youtube', 'ESL Counter-Strike (YouTube)', 'https://www.youtube.com/@ESLCS',
   '@ESLCS', 5::smallint, 'en', '60 minutes'),
  ('youtube', 'ATP Tour (YouTube)', 'https://www.youtube.com/@ATPTour',
   '@ATPTour', 5::smallint, 'en', '90 minutes'),
  ('youtube', 'WTA (YouTube)', 'https://www.youtube.com/@WTA',
   '@WTA', 5::smallint, 'en', '90 minutes'),
  ('youtube', 'Tennis TV (YouTube)', 'https://www.youtube.com/@tennistv',
   '@tennistv', 5::smallint, 'en', '90 minutes')
) AS v(kind, name, homepage_url, feed_url, quality, default_lang, fetch_interval)
WHERE NOT EXISTS (SELECT 1 FROM sources s WHERE s.feed_url=v.feed_url);

