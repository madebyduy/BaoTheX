-- ============================================================================
-- RepWire — seed data: topics, entities, MVP sources.
-- Idempotent: safe to re-run (ON CONFLICT DO NOTHING).
-- ============================================================================

-- ============ TOPICS ============
INSERT INTO topics (slug, name, category, keywords) VALUES
  ('hypertrophy',          'Hypertrophy',          'knowledge',  ARRAY['hypertrophy','muscle growth','muscle mass','muscle building','tang co','mps','muscle protein synthesis']),
  ('training-volume',      'Training Volume',      'knowledge',  ARRAY['training volume','sets per week','junk volume','volume landmark','weekly sets']),
  ('training-frequency',   'Training Frequency',   'knowledge',  ARRAY['training frequency','frequency','times per week','full body','split routine']),
  ('progressive-overload', 'Progressive Overload', 'knowledge',  ARRAY['progressive overload','overload','load progression','double progression']),
  ('technique',            'Technique',            'knowledge',  ARRAY['technique','form','range of motion','tempo','lifting form','execution']),
  ('protein',              'Protein',              'knowledge',  ARRAY['protein','protein intake','dietary protein','leucine','whey','casein']),
  ('creatine',             'Creatine',             'knowledge',  ARRAY['creatine','creatine monohydrate','creatine loading']),
  ('supplements',          'Supplements',          'knowledge',  ARRAY['supplement','supplements','caffeine','beta alanine','citrulline','pre workout']),
  ('nutrition',            'Nutrition',            'knowledge',  ARRAY['nutrition','diet','calories','macros','deficit','surplus','meal timing']),
  ('sleep',                'Sleep',                'knowledge',  ARRAY['sleep','sleep quality','sleep deprivation','circadian']),
  ('recovery',             'Recovery',             'knowledge',  ARRAY['recovery','deload','fatigue','overtraining','doms','soreness']),
  ('injury',               'Injury',               'knowledge',  ARRAY['injury','pain','rehab','tendon','joint','prehab','tendinopathy']),
  ('cardio',               'Cardio',               'knowledge',  ARRAY['cardio','conditioning','vo2max','endurance','aerobic','zone 2']),
  ('fat-loss',             'Fat Loss',             'goal',       ARRAY['fat loss','weight loss','cutting','giam mo','body recomposition']),
  ('muscle-gain',          'Muscle Gain',          'goal',       ARRAY['muscle gain','bulking','lean bulk','mass gain']),
  ('strength',             'Strength',             'goal',       ARRAY['strength','1rm','maximal strength','powerlifting total','max strength']),
  ('bodybuilding',         'Bodybuilding',         'discipline', ARRAY['bodybuilding','physique','mr olympia','ifbb','classic physique']),
  ('powerlifting',         'Powerlifting',         'discipline', ARRAY['powerlifting','squat','bench press','deadlift','ipf','powerlifting meet']),
  ('calisthenics',         'Calisthenics',         'discipline', ARRAY['calisthenics','bodyweight','pull up','muscle up','street workout']),
  ('crossfit',             'CrossFit',             'discipline', ARRAY['crossfit','wod','functional fitness','crossfit games'])
ON CONFLICT (slug) DO NOTHING;

-- ============ ENTITIES ============
INSERT INTO entities (slug, name, kind, aliases, official_links) VALUES
  ('jeff-nippard',        'Jeff Nippard',        'creator',     ARRAY['Jeff Nippard','Nippard'],            '[{"type":"youtube","url":"https://youtube.com/@JeffNippard"}]'::jsonb),
  ('stronger-by-science', 'Stronger by Science', 'publication', ARRAY['Stronger by Science','SBS','Greg Nuckols'], '[{"type":"website","url":"https://strongerbyscience.com"}]'::jsonb),
  ('examine',             'Examine.com',         'publication', ARRAY['Examine','Examine.com'],             '[{"type":"website","url":"https://examine.com"}]'::jsonb),
  ('renaissance-periodization','Renaissance Periodization','publication', ARRAY['Renaissance Periodization','RP','Mike Israetel'], '[{"type":"youtube","url":"https://youtube.com/@RenaissancePeriodization"}]'::jsonb),
  ('barbell-medicine',    'Barbell Medicine',    'publication', ARRAY['Barbell Medicine','Jordan Feigenbaum','Austin Baraki'], '[{"type":"website","url":"https://barbellmedicine.com"}]'::jsonb),
  ('mr-olympia',          'Mr. Olympia',         'competition', ARRAY['Mr Olympia','Mr. Olympia','Olympia'], '[]'::jsonb),
  ('ipf',                 'International Powerlifting Federation', 'federation', ARRAY['IPF','International Powerlifting Federation'], '[]'::jsonb)
ON CONFLICT (slug) DO NOTHING;

-- ============ SOURCES ============
-- RSS (5 MVP sites). feed_url values are best-effort defaults; adjust in admin.
INSERT INTO sources (kind, name, homepage_url, feed_url, quality, fetch_interval) VALUES
  ('rss', 'Stronger by Science',        'https://strongerbyscience.com',   'https://strongerbyscience.com/feed/',        5, '60 minutes'),
  ('rss', 'Barbell Medicine',           'https://barbellmedicine.com',     'https://barbellmedicine.com/blog/feed/',     5, '120 minutes'),
  ('rss', 'Examine',                    'https://examine.com',             'https://examine.com/rss/',                   4, '120 minutes'),
  ('rss', 'Renaissance Periodization',  'https://rpstrength.com',          'https://rpstrength.com/blogs/articles.atom', 4, '120 minutes'),
  ('rss', 'Bret Contreras',             'https://bretcontreras.com',       'https://bretcontreras.com/feed/',            3, '180 minutes')
ON CONFLICT DO NOTHING;

-- YouTube (feed_url = channel id / handle; resolved to uploads playlist on first fetch).
INSERT INTO sources (kind, name, homepage_url, feed_url, quality, fetch_interval) VALUES
  ('youtube', 'Jeff Nippard (YouTube)',              'https://youtube.com/@JeffNippard',              '@JeffNippard',              5, '60 minutes'),
  ('youtube', 'Renaissance Periodization (YouTube)', 'https://youtube.com/@RenaissancePeriodization', '@RenaissancePeriodization', 4, '60 minutes')
ON CONFLICT DO NOTHING;

-- Europe PMC (5 MVP queries). feed_url holds the query string.
INSERT INTO sources (kind, name, feed_url, quality, fetch_interval) VALUES
  ('europepmc', 'PMC: Resistance training & hypertrophy', 'resistance training AND (hypertrophy OR muscle growth)',       4, '360 minutes'),
  ('europepmc', 'PMC: Creatine & performance',            'creatine AND (strength OR performance)',                       4, '360 minutes'),
  ('europepmc', 'PMC: Protein & MPS',                     'protein intake AND muscle protein synthesis',                  4, '360 minutes'),
  ('europepmc', 'PMC: Volume & frequency',                '(training volume OR training frequency) AND resistance',       4, '360 minutes'),
  ('europepmc', 'PMC: Sleep & recovery',                  'sleep AND (recovery OR athletic performance)',                 4, '360 minutes')
ON CONFLICT DO NOTHING;
