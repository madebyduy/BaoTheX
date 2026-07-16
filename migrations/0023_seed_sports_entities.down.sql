-- Remove only the sports entities this migration seeded. content_entities rows
-- cascade on delete, so extraction links go with them; the seven original
-- strength-training entities are left untouched.
DELETE FROM entities WHERE kind IN ('club', 'national_team')
   OR slug IN (
  'lionel-messi','cristiano-ronaldo','kylian-mbappe','erling-haaland',
  'jude-bellingham','vinicius-junior','mohamed-salah','kevin-de-bruyne',
  'harry-kane','bukayo-saka','lamine-yamal','rodri','robert-lewandowski',
  'neymar','declan-rice','pep-guardiola','jurgen-klopp','carlo-ancelotti',
  'mikel-arteta','thomas-tuchel','lionel-scaloni','premier-league','la-liga',
  'serie-a','bundesliga','ligue-1','champions-league','europa-league',
  'world-cup','copa-america','fifa','uefa','afc','v-league','vff','vpf',
  'quang-hai','tien-linh','van-lam','cong-phuong','van-hau','hoang-duc',
  'atp','wta','wimbledon','roland-garros','us-open-tennis','australian-open',
  'novak-djokovic','carlos-alcaraz','jannik-sinner','rafael-nadal',
  'iga-swiatek','aryna-sabalenka','coco-gauff','nba','vba','lebron-james',
  'stephen-curry','nikola-jokic','giannis-antetokounmpo','luka-doncic',
  'formula-1','max-verstappen','lewis-hamilton','charles-leclerc',
  'lando-norris','league-of-legends','valorant','counter-strike','dota-2',
  'vcs','worlds-lol','bwf','nguyen-thuy-linh','nguyen-tien-minh','olympics',
  'sea-games','asiad','nguyen-thi-oanh','nguyen-huy-hoang'
);
