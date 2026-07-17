-- Repair the feeds that fetch_rss / fetch_youtube retired.
--
-- A source is switched off automatically after three consecutive failures, so
-- each of these went quiet on its own and stayed quiet: nothing re-enables a
-- source once the URL behind it has moved. Every replacement below was fetched
-- with the crawler's own User-Agent before being written here.
--
-- Resetting consecutive_failures matters as much as the URL. Leaving it at 5
-- would let a single unrelated hiccup trip the three-strike rule immediately and
-- switch the source straight back off.

-- BBC Sport Football: newsrss.bbc.co.uk is long gone and now answers with a
-- certificate for akamai's edge, which is why this failed as a TLS error rather
-- than a 404. The live feed is on feeds.bbci.co.uk, where BBC Sport — a source
-- that never broke — has been pointing all along. Verified: HTTP 200, 89 items.
UPDATE sources SET
  feed_url = 'https://feeds.bbci.co.uk/sport/football/rss.xml',
  enabled = TRUE, consecutive_failures = 0, last_error = NULL, etag = NULL, last_modified = NULL
WHERE name = 'BBC Sport Football';

-- CBC Sports: /cmlink/rss-sports now answers 301. The redirect target is a
-- /webfeed/ path. Verified: HTTP 200, 20 items.
UPDATE sources SET
  feed_url = 'https://www.cbc.ca/webfeed/rss/rss-sports',
  enabled = TRUE, consecutive_failures = 0, last_error = NULL, etag = NULL, last_modified = NULL
WHERE name = 'CBC Sports';

-- VFF: the handle @VFFChannel does not exist — youtube.com/@VFFChannel is a 404
-- and the fetcher reported exactly that for ten attempts. The federation's
-- channel is @VFFOfficial. uploads_playlist_id is cleared because it caches a
-- resolution of the old handle.
UPDATE sources SET
  feed_url = '@VFFOfficial',
  enabled = TRUE, consecutive_failures = 0, last_error = NULL, uploads_playlist_id = NULL
WHERE name = 'VFF Channel';

-- ESPN's YouTube channel was switched off while YOUTUBE_API_KEY was unset — the
-- failures were ours, not the channel's, and the key is configured now.
UPDATE sources SET
  enabled = TRUE, consecutive_failures = 0, last_error = NULL
WHERE name = 'ESPN (YouTube)';

-- BWF Badminton is switched off, and this is the one case where the feed is not
-- repairable: bwfbadminton.com/feed/ answers 200 with an empty channel — no
-- items, nothing to parse, no error to report. It has fetched cleanly 161 times
-- and delivered nothing, which is why "Cầu lông" reads as a classification
-- failure and is not one. Badminton Asia stays enabled: it publishes roughly
-- weekly rather than never, and the 72-hour ingest window is what makes it look
-- idle. Badminton Planet, the one genuinely live badminton feed, is untouched.
--
-- The section will fill from the general Vietnamese desks instead, which do
-- cover badminton and which migration 0027 taught the classifier to read.
UPDATE sources SET
  enabled = FALSE,
  last_error = 'feed returns an empty channel; disabled by migration 0029'
WHERE name = 'BWF Badminton';
