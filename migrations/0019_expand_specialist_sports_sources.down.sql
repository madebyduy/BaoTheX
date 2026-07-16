DELETE FROM sources WHERE feed_url IN (
  'https://dotesports.com/feed',
  'https://bwfbadminton.com/feed/',
  'https://badmintonasia.org/feed/',
  'https://www.badmintonplanet.com/feed',
  'https://www.tennis365.com/feed',
  'https://tennishead.net/feed/',
  'https://www.ubitennis.net/feed/',
  '@BarBend',
  '@lolesports',
  '@valorantesports',
  '@ESLCS',
  '@ATPTour',
  '@WTA',
  '@tennistv'
);

DELETE FROM topics WHERE slug='cau-long';
