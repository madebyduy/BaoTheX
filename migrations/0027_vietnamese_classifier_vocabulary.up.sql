-- Give every section a Vietnamese vocabulary.
--
-- process.Classify matches keywords against an article's ORIGINAL title and
-- body, before any translation runs. Half the sections shipped with English-only
-- keyword lists, which is a strange thing for a Vietnamese sports paper: "Bóng
-- rổ" could recognise `basketball`, `nba`, `wnba` and `euroleague`, but not the
-- words "bóng rổ". A Vietnamese source reporting a VBA final matched nothing and
-- was filed under no section at all. 1,424 articles currently carry no topic,
-- and the sections readers see are correspondingly thin.
--
-- Keywords are compared with a plain lowercase substring test (strings.Contains),
-- so every phrase added below has to be one that cannot occur inside an
-- unrelated word. That rules out short fragments: "gôn" sits inside "gông",
-- "bơi" inside "bơi móc". Where a term is only safe as part of a longer phrase,
-- the longer phrase is what is listed.
--
-- Existing keywords are preserved and merged, never replaced, so a section that
-- already recognises its English vocabulary keeps it.

-- Bóng rổ — could not read its own name.
UPDATE topics SET keywords = (SELECT array_agg(DISTINCT k) FROM unnest(keywords || ARRAY[
  'bóng rổ', 'fiba', 'vba', 'giải bóng rổ', 'nba finals'
]) k) WHERE slug = 'bong-ro';

-- Bóng đá quốc tế — nine keywords, none of them the word "bóng đá".
UPDATE topics SET keywords = (SELECT array_agg(DISTINCT k) FROM unnest(keywords || ARRAY[
  'bóng đá', 'ngoại hạng anh', 'cúp c1', 'europa league', 'ligue 1',
  'chuyển nhượng', 'vòng loại world cup', 'giải vô địch quốc gia'
]) k) WHERE slug = 'bong-da-quoc-te';

-- Các môn thể thao khác — the catch-all section, entirely in English.
UPDATE topics SET keywords = (SELECT array_agg(DISTINCT k) FROM unnest(keywords || ARRAY[
  'quyền anh', 'đấm bốc', 'bơi lội', 'điền kinh', 'xe đạp', 'bóng chuyền',
  'võ thuật', 'đấu vật', 'cử tạ', 'bắn cung', 'taekwondo', 'karate', 'judo',
  'bóng bàn', 'đua thuyền'
]) k) WHERE slug = 'the-thao-khac';

-- F1 & thể thao motor.
UPDATE topics SET keywords = (SELECT array_agg(DISTINCT k) FROM unnest(keywords || ARRAY[
  'công thức 1', 'chặng đua', 'đường đua', 'tay đua', 'giải đua'
]) k) WHERE slug = 'f1-the-thao-motor';

-- Thể thao Việt Nam.
UPDATE topics SET keywords = (SELECT array_agg(DISTINCT k) FROM unnest(keywords || ARRAY[
  'asiad', 'huy chương vàng', 'đoàn thể thao việt nam', 'tuyển quốc gia',
  'thể thao thành tích cao'
]) k) WHERE slug = 'the-thao-viet-nam';

-- Bóng đá Việt Nam.
UPDATE topics SET keywords = (SELECT array_agg(DISTINCT k) FROM unnest(keywords || ARRAY[
  'v.league', 'hagl', 'hà nội fc', 'công an hà nội', 'nam định', 'thể công',
  'kim sang-sik', 'park hang-seo', 'u23 việt nam'
]) k) WHERE slug = 'bong-da-viet-nam';

-- Re-file the articles the old vocabulary could not read. Classification is
-- rule-based and costs no LLM call, and for a Vietnamese article that now
-- matches a section, jobs.handleProcess ends by promoting it to 'ready' — which
-- is the point: 339 Vietnamese articles are sitting in needs_review whose only
-- defect was that no section could read them.
--
-- Deliberately excludes status='ready'. handleProcess is not idempotent: it
-- resets status to 'processing' on entry and then re-derives it from scratch,
-- with no check for work already done. Re-running it over a live foreign article
-- would either park it back to 'processing' when it scores under
-- LLM_TRANSLATE_MIN_SCORE — taking a published story off the site — or re-queue
-- a translation for a body that is already translated, spending free-tier quota
-- to reproduce a result we hold. Both are silent. Until that handler is made
-- safe to re-run, this only touches content that is already invisible, where the
-- worst case is that it stays invisible.
INSERT INTO jobs (kind, payload, dedup_key, priority, run_at, max_attempts)
SELECT 'process_content', jsonb_build_object('content_id', c.id), 'process:' || c.id, 1, now(), 5
FROM content_items c
WHERE NOT EXISTS (SELECT 1 FROM content_topics ct WHERE ct.content_id = c.id)
  AND c.status IN ('needs_review', 'processing', 'discovered')
ON CONFLICT (dedup_key) WHERE dedup_key IS NOT NULL AND status IN ('pending','running')
DO NOTHING;
