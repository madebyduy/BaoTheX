ALTER TABLE notification_preferences DROP COLUMN IF EXISTS feed_following_only;

UPDATE sources SET enabled=FALSE
WHERE feed_url IN (
  'https://tuoitre.vn/rss/the-thao.rss',
  'https://vietnamnet.vn/rss/the-thao.rss',
  'https://dantri.com.vn/rss/the-thao.rss',
  'https://feeds.bbci.co.uk/sport/rss.xml',
  'https://www.theguardian.com/football/rss',
  'https://www.muscleandfitness.com/feed/',
  'https://generationiron.com/feed/',
  'https://fitnessvolt.com/feed/',
  'https://www.menshealth.com/rss/all.xml/',
  'https://www.strengthlog.com/feed/',
  'https://www.boxrox.com/feed/'
);
