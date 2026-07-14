DELETE FROM sources  WHERE kind IN ('rss','youtube','europepmc');
DELETE FROM entities WHERE slug IN
  ('jeff-nippard','stronger-by-science','examine','renaissance-periodization','barbell-medicine','mr-olympia','ipf');
DELETE FROM topics   WHERE slug IN
  ('hypertrophy','training-volume','training-frequency','progressive-overload','technique','protein','creatine',
   'supplements','nutrition','sleep','recovery','injury','cardio','fat-loss','muscle-gain','strength',
   'bodybuilding','powerlifting','calisthenics','crossfit');
