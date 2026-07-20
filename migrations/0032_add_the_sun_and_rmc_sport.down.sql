DELETE FROM sources
WHERE feed_url IN (
  'https://www.thesun.co.uk/sport/feed/',
  'https://rmcsport.bfmtv.com/rss/fil-sport/'
);
