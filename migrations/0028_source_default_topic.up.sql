-- Let a source's beat file the stories its headlines don't.
--
-- process.Classify reads only the words in an article. That works for "Premier
-- League" or "bóng rổ", and fails for the way sports journalism actually writes
-- headlines: "Williams under fire as Carlos Sainz is getting nowhere" is F1 and
-- never says F1; "McIlroy makes slow start to Open challenge" is golf and never
-- says golf; "Messi bất tử, Argentina khiến trái tim người Anh tan vỡ" is
-- football and never says bóng đá. Real headlines carry proper nouns, not
-- category words, and no keyword list ever catches up with them — 1,427 articles
-- currently sit in no section at all.
--
-- The signal that was there the whole time is the masthead. A story from
-- Cyclingnews is cycling. A video from the NBA channel is basketball. This adds
-- that as a fallback only: keyword matches still win, and a source only gets a
-- default when its beat is genuinely narrow.
--
-- The general desks are deliberately left NULL, and that is the load-bearing
-- half of this migration. CBS Sports, ESPN News, BBC Sport, RTE Sport, The
-- Guardian Sport, VnExpress/Tuổi Trẻ/Dân trí/VietnamNet/Soha Thể thao, The Hindu
-- Sport, ABC Grandstand, France 24, Le Monde, NPR and DW all cover every sport
-- there is. So does ESPN Daily, despite the football-heavy company it keeps in
-- the podcast list — it is ESPN's general news show and runs NFL, NBA and MLB.
-- Handing any of them a default would file basketball under football with the
-- same confidence as a real match, which is worse than filing nothing: a wrong
-- section is a lie a reader acts on, an empty one is merely a gap.

ALTER TABLE sources ADD COLUMN IF NOT EXISTS default_topic_id BIGINT
  REFERENCES topics(id) ON DELETE SET NULL;

COMMENT ON COLUMN sources.default_topic_id IS
  'Section to file an article under when keyword classification finds nothing. '
  'Only for single-beat sources; NULL for general sports desks.';

-- Football-only desks, podcasts and channels.
UPDATE sources SET default_topic_id = (SELECT id FROM topics WHERE slug='bong-da-quoc-te')
WHERE name IN (
  'BBC Football Daily', 'Guardian Football Weekly', 'Guardian Football',
  'Sky Sports Football', 'BBC Sport Football', 'Thanh Niên Bóng đá quốc tế',
  'FIFA (YouTube)', 'Premier League (YouTube)', 'UEFA (YouTube)',
  'Sky Sports Premier League (YouTube)'
);

-- Basketball.
UPDATE sources SET default_topic_id = (SELECT id FROM topics WHERE slug='bong-ro')
WHERE name IN ('NBA (YouTube)');

-- Motorsport.
UPDATE sources SET default_topic_id = (SELECT id FROM topics WHERE slug='f1-the-thao-motor')
WHERE name IN ('Motorsport F1');

-- Tennis.
UPDATE sources SET default_topic_id = (SELECT id FROM topics WHERE slug='tennis')
WHERE name IN (
  'Tennis365', 'Tennishead', 'Ubitennis', 'WTA (YouTube)',
  'ATP Tour (YouTube)', 'Tennis TV (YouTube)'
);

-- Esports.
UPDATE sources SET default_topic_id = (SELECT id FROM topics WHERE slug='the-thao-dien-tu')
WHERE name IN (
  'Dot Esports', 'VALORANT Champions Tour (YouTube)', 'LoL Esports (YouTube)',
  'ESL Counter-Strike (YouTube)'
);

-- Badminton. The section has one article to its name, which reads as a
-- classification failure and is not: these three feeds have produced five items
-- between them. Wiring the default costs nothing and means the fifth one lands
-- somewhere, but this section needs a live source, not a better classifier.
UPDATE sources SET default_topic_id = (SELECT id FROM topics WHERE slug='cau-long')
WHERE name IN ('Badminton Planet', 'Badminton Asia', 'BWF Badminton');

-- Cycling and the multi-sport Olympic channel go to the catch-all section,
-- which is where those sports live in this paper's taxonomy.
UPDATE sources SET default_topic_id = (SELECT id FROM topics WHERE slug='the-thao-khac')
WHERE name IN ('Cyclingnews', 'Olympics (YouTube)');

-- Training / gym feeds. Matched by name rather than listed, because migration
-- 0026 already keeps this roster as a regex and duplicating it as a literal list
-- is how the two drift apart.
UPDATE sources SET default_topic_id = (SELECT id FROM topics WHERE slug='the-hinh')
WHERE name ~* '(muscle|fitness|generation iron|strengthlog|boxrox|barbend|athlean|jeremy ethier|squat university|picturefit|hypertrophy|renaissance periodization|jeff nippard|barbell medicine|stronger by science)';

-- Re-file the orphans that now have a masthead to fall back on. Only content
-- that is already invisible is touched: handleProcess rebuilds status from
-- scratch, and although it no longer discards a stored translation, re-running
-- it over a live article is still a bigger claim than this migration needs.
INSERT INTO jobs (kind, payload, dedup_key, priority, run_at, max_attempts)
SELECT 'process_content', jsonb_build_object('content_id', c.id), 'process:' || c.id, 1, now(), 5
FROM content_items c
JOIN sources s ON s.id = c.source_id
WHERE s.default_topic_id IS NOT NULL
  AND c.status IN ('needs_review', 'processing', 'discovered')
  AND NOT EXISTS (SELECT 1 FROM content_topics ct WHERE ct.content_id = c.id)
ON CONFLICT (dedup_key) WHERE dedup_key IS NOT NULL AND status IN ('pending','running')
DO NOTHING;
