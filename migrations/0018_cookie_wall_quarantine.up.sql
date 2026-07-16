-- Remove previously published consent/interstitial pages. Keep attribution and
-- canonical URL so editors can inspect the source, but clear all AI-derived copy.
WITH blocked AS (
  SELECT content_id
  FROM content_bodies
  WHERE (
    (
      lower(original_body) LIKE '%this blog is currently unavailable%'
      OR lower(original_body) LIKE '%this live blog is currently unavailable%'
      OR lower(COALESCE(vietnamese_body,'')) LIKE '%blog này hiện không khả dụng%'
    )
    AND (
      lower(original_body) LIKE '%allow cookies once%'
      OR lower(original_body) LIKE '%unable to verify if you have consented%'
      OR lower(COALESCE(vietnamese_body,'')) LIKE '%cho phép cookie một lần%'
      OR lower(COALESCE(vietnamese_body,'')) LIKE '%không thể xác minh xem bạn đã đồng ý%'
    )
  )
), cleaned AS (
  UPDATE content_bodies b SET
    original_body='',
    vietnamese_body=NULL,
    translation_status='blocked',
    translated_at=NULL,
    updated_at=now()
  FROM blocked x
  WHERE b.content_id=x.content_id
  RETURNING b.content_id
)
UPDATE content_items c SET
  status='needs_review',
  translated_title=NULL,
  summary=NULL,
  key_points='{}',
  updated_at=now()
FROM cleaned x
WHERE c.id=x.content_id;
