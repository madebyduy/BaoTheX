-- Remove only the keywords this migration added, leaving each section's original
-- vocabulary intact.
--
-- The re-classification jobs are deliberately not undone. They are ordinary
-- process_content jobs whose results are indistinguishable from normal pipeline
-- output, and a down migration that deleted topic assignments would take the
-- pre-existing ones with them.

UPDATE topics SET keywords = (
  SELECT coalesce(array_agg(k), '{}') FROM unnest(keywords) k
  WHERE k <> ALL (ARRAY['bóng rổ', 'fiba', 'vba', 'giải bóng rổ', 'nba finals'])
) WHERE slug = 'bong-ro';

UPDATE topics SET keywords = (
  SELECT coalesce(array_agg(k), '{}') FROM unnest(keywords) k
  WHERE k <> ALL (ARRAY['bóng đá', 'ngoại hạng anh', 'cúp c1', 'europa league',
    'ligue 1', 'chuyển nhượng', 'vòng loại world cup', 'giải vô địch quốc gia'])
) WHERE slug = 'bong-da-quoc-te';

UPDATE topics SET keywords = (
  SELECT coalesce(array_agg(k), '{}') FROM unnest(keywords) k
  WHERE k <> ALL (ARRAY['quyền anh', 'đấm bốc', 'bơi lội', 'điền kinh', 'xe đạp',
    'bóng chuyền', 'võ thuật', 'đấu vật', 'cử tạ', 'bắn cung', 'taekwondo',
    'karate', 'judo', 'bóng bàn', 'đua thuyền'])
) WHERE slug = 'the-thao-khac';

UPDATE topics SET keywords = (
  SELECT coalesce(array_agg(k), '{}') FROM unnest(keywords) k
  WHERE k <> ALL (ARRAY['công thức 1', 'chặng đua', 'đường đua', 'tay đua', 'giải đua'])
) WHERE slug = 'f1-the-thao-motor';

UPDATE topics SET keywords = (
  SELECT coalesce(array_agg(k), '{}') FROM unnest(keywords) k
  WHERE k <> ALL (ARRAY['asiad', 'huy chương vàng', 'đoàn thể thao việt nam',
    'tuyển quốc gia', 'thể thao thành tích cao'])
) WHERE slug = 'the-thao-viet-nam';

UPDATE topics SET keywords = (
  SELECT coalesce(array_agg(k), '{}') FROM unnest(keywords) k
  WHERE k <> ALL (ARRAY['v.league', 'hagl', 'hà nội fc', 'công an hà nội',
    'nam định', 'thể công', 'kim sang-sik', 'park hang-seo', 'u23 việt nam'])
) WHERE slug = 'bong-da-viet-nam';
