-- Keep "The hinh & tap luyen" from swallowing ordinary team-sports injury,
-- recovery, and training-camp stories. Those words are common in football
-- coverage, but they are not enough to mean gym/bodybuilding coverage.

UPDATE topics SET keywords=ARRAY[
  'thể hình','tập gym','phòng gym','tăng cơ','giảm mỡ','cơ bắp',
  'bodybuilding','bodybuilder','strength training','resistance training',
  'hypertrophy','muscle growth','powerlifting','strongman','weightlifting',
  'crossfit','workout','protein','creatine','mr olympia','classic physique','ifbb'
] WHERE slug='the-hinh';

DELETE FROM content_topics ct
USING topics t, content_items c, sources s
WHERE ct.topic_id=t.id
  AND t.slug='the-hinh'
  AND c.id=ct.content_id
  AND s.id=c.source_id
  AND NOT (
    s.name ~* '(muscle|fitness|generation iron|strengthlog|boxrox|barbend|athlean|jeremy ethier|squat university|picturefit|hypertrophy|renaissance periodization|jeff nippard)'
    OR coalesce(s.homepage_url,'') ~* '(muscleandfitness|fitnessvolt|menshealth.com/fitness|generationiron|strengthlog|boxrox|barbend|athlean|jeremyethier|squatuniversity|picturefit|hypertrophy|renaissanceperiodization|jeffnippard)'
    OR concat_ws(' ', c.title, c.excerpt, c.summary) ~* '(thể hình|tập gym|phòng gym|tăng cơ|giảm mỡ|cơ bắp|bodybuilding|bodybuilder|strength training|resistance training|hypertrophy|muscle growth|powerlifting|strongman|weightlifting|crossfit|workout|protein|creatine|mr olympia|classic physique|ifbb)'
  );
