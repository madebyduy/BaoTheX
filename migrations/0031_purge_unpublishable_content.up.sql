-- Delete content that can never reach a reader.
--
-- Three groups, each unpublishable for a settled reason rather than a temporary
-- one. Nothing here is a judgement about quality; it is about whether a path to
-- the front page still exists.
--
--   1. Foreign articles and videos whose translation window has closed. The
--      publish gate (0006) requires a Vietnamese edition, the scheduler
--      abandons anything past LLM_TRANSLATE_MAX_AGE rather than queue it, and
--      no rescore reopens that door. They have been sitting in 'processing' and
--      'needs_review' waiting for a translation that is not coming.
--   2. Podcasts. The surface was retired in 0015 and the feeds were switched off
--      in 0030; 648 episodes were ingested into a section that no longer exists.
--   3. Research. Europe PMC was disabled in 0024 as out of scope for a sports
--      paper, and the /nghien-cuu page — which nothing linked to — is gone.
--
-- Deliberately NOT touched: everything 'ready'. 471 articles and 330 videos are
-- live and stay live, including the older ones, because a published archive is
-- not junk. Also untouched are the 623 articles still in 'needs_review' that are
-- Vietnamese or still inside the window — they are a review-queue problem, not a
-- deletion problem, and conflating the two is how you delete a backlog you meant
-- to work through.
--
-- Verified by dry run before writing: this removes 1,340 items and touches zero
-- items belonging to a cluster behind an analysis draft (clusters 324 and 525
-- lose nothing). Every dependent table cascades — bodies, topics, entities,
-- subtype rows, cluster items, saves, reading history.

CREATE TEMP TABLE purge_ids ON COMMIT DROP AS
SELECT c.id
FROM content_items c
WHERE
  (c.language <> 'vi' AND c.translated_title IS NULL
   AND c.status IN ('processing', 'needs_review')
   AND c.published_at < now() - interval '36 hours')
  OR c.type = 'podcast'
  OR c.type = 'research';

-- Cancel queued work for the doomed rows first. A job whose content disappears
-- mid-flight fails on a lookup and burns its retries reporting it.
UPDATE jobs
SET status = 'dead', finished_at = now(), last_error = 'content purged by migration 0031'
WHERE status IN ('pending', 'failed')
  AND (payload->>'content_id')::bigint IN (SELECT id FROM purge_ids);

DELETE FROM content_items WHERE id IN (SELECT id FROM purge_ids);

-- Clusters that existed only to group the deleted. story_cluster_items cascades,
-- so these are now empty shells; analysis_candidates.cluster_id is ON DELETE
-- SET NULL, so a candidate pointing at one survives with a null cluster rather
-- than vanishing — which is why only genuinely empty clusters go.
DELETE FROM story_clusters sc
WHERE NOT EXISTS (
  SELECT 1 FROM story_cluster_items sci WHERE sci.cluster_id = sc.id);

-- Candidates whose cluster is gone can never be drafted or reviewed again.
DELETE FROM analysis_candidates WHERE cluster_id IS NULL;
