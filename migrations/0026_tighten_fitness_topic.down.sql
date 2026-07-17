-- Restore the broader folded fitness vocabulary from 0024.
UPDATE topics SET keywords = (
  SELECT array_agg(DISTINCT kw)
  FROM (
    SELECT unnest(keywords) AS kw FROM topics WHERE slug='the-hinh'
    UNION
    SELECT unnest(ARRAY[
      'injury','pain','rehab','tendon','joint','prehab','tendinopathy',
      'recovery','deload','fatigue','overtraining','doms','soreness',
      'cardio','conditioning','vo2max','endurance','aerobic','zone 2',
      'nutrition','diet','calories','macros','deficit','surplus','meal timing',
      'sleep','sleep quality','sleep deprivation','circadian',
      'technique','form','range of motion','tempo','lifting form','execution',
      'training volume','sets per week','junk volume','volume landmark','weekly sets',
      'training frequency','frequency','times per week','full body','split routine',
      'progressive overload','overload','load progression','double progression'
    ]) AS kw
  ) merged
) WHERE slug='the-hinh';
