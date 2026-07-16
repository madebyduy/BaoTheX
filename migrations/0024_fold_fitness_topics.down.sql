-- Recreate the micro-topics as empty shells and re-enable the research feed.
--
-- This is a partial undo and deliberately so: which article was tagged
-- "Hypertrophy" versus "Protein" is not recoverable once both were folded into
-- one section, and the keyword lists cannot be un-merged. The topics come back
-- so the schema and any hard-coded slugs resolve; their article assignments do
-- not.
INSERT INTO topics (slug, name, category) VALUES
  ('bodybuilding', 'Bodybuilding', 'discipline'),
  ('calisthenics', 'Calisthenics', 'discipline'),
  ('cardio', 'Cardio', 'discipline'),
  ('creatine', 'Creatine', 'knowledge'),
  ('crossfit', 'CrossFit', 'discipline'),
  ('fat-loss', 'Fat Loss', 'goal'),
  ('hypertrophy', 'Hypertrophy', 'goal'),
  ('injury', 'Injury', 'knowledge'),
  ('muscle-gain', 'Muscle Gain', 'goal'),
  ('nutrition', 'Nutrition', 'knowledge'),
  ('powerlifting', 'Powerlifting', 'discipline'),
  ('progressive-overload', 'Progressive Overload', 'knowledge'),
  ('protein', 'Protein', 'knowledge'),
  ('recovery', 'Recovery', 'knowledge'),
  ('sleep', 'Sleep', 'knowledge'),
  ('strength', 'Strength', 'goal'),
  ('supplements', 'Supplements', 'knowledge'),
  ('technique', 'Technique', 'knowledge'),
  ('training-frequency', 'Training Frequency', 'knowledge'),
  ('training-volume', 'Training Volume', 'knowledge')
ON CONFLICT (slug) DO NOTHING;

UPDATE sources SET enabled = TRUE WHERE kind = 'europepmc';
