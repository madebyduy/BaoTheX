-- Fold RepWire's strength-training taxonomy into one section.
--
-- BaoTheX is a sports newspaper, but its topic list still reads like a gym
-- research index: twenty of its thirty topics were Hypertrophy, Protein,
-- Creatine, Progressive Overload, Training Volume and friends. Those are
-- knowledge tags for a fitness-science aggregator, not sections of a newspaper.
-- Training coverage stays — as one section, "Thể hình & tập luyện" — because
-- lifting is a sport; the micro-topics underneath it are what go.
--
-- Order matters here. The keywords must move BEFORE the topics are dropped, or
-- process.Classify loses the vocabulary it uses to recognise training content
-- at all and the surviving section would quietly stop being populated.

-- 1. Inherit every micro-topic's keywords so classification keeps working.
UPDATE topics dest SET keywords = (
  SELECT array_agg(DISTINCT kw)
  FROM (
    SELECT unnest(dest.keywords) AS kw
    UNION
    SELECT unnest(src.keywords) FROM topics src WHERE src.slug IN (
      'bodybuilding','calisthenics','cardio','creatine','crossfit','fat-loss',
      'hypertrophy','injury','muscle-gain','nutrition','powerlifting',
      'progressive-overload','protein','recovery','sleep','strength',
      'supplements','technique','training-frequency','training-volume'
    )
  ) merged
)
WHERE dest.slug = 'the-hinh';

-- 2. Re-point existing article assignments at the surviving section. An article
--    tagged both Hypertrophy and Protein would collide on the (content_id,
--    topic_id) primary key, hence DO NOTHING.
INSERT INTO content_topics (content_id, topic_id, confidence, is_primary)
SELECT ct.content_id, dest.id, max(ct.confidence), bool_or(ct.is_primary)
FROM content_topics ct
JOIN topics src ON src.id = ct.topic_id
CROSS JOIN (SELECT id FROM topics WHERE slug = 'the-hinh') dest
WHERE src.slug IN (
  'bodybuilding','calisthenics','cardio','creatine','crossfit','fat-loss',
  'hypertrophy','injury','muscle-gain','nutrition','powerlifting',
  'progressive-overload','protein','recovery','sleep','strength',
  'supplements','technique','training-frequency','training-volume'
)
GROUP BY ct.content_id, dest.id
ON CONFLICT (content_id, topic_id) DO NOTHING;

-- 3. Drop the micro-topics. content_topics and user follows cascade.
DELETE FROM topics WHERE slug IN (
  'bodybuilding','calisthenics','cardio','creatine','crossfit','fat-loss',
  'hypertrophy','injury','muscle-gain','nutrition','powerlifting',
  'progressive-overload','protein','recovery','sleep','strength',
  'supplements','technique','training-frequency','training-volume'
);

-- 4. Retire the research feed. EuropePMC delivers protein-synthesis papers,
--    which is scholarship rather than sports news — the one clearly non-sports
--    source in the pool. Disabled rather than deleted so its 45 existing
--    articles keep their source row and stay readable.
UPDATE sources SET enabled = FALSE WHERE kind = 'europepmc';
