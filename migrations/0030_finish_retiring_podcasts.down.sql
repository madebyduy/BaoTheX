-- Re-enable the podcast feeds. This restores the intake, not the surface: the
-- /podcast route was deleted in the same change that retired it, so episodes
-- ingested after this rolls back have nowhere to appear. Reviving podcasts for
-- real means building the section first.
UPDATE sources SET enabled = TRUE, last_error = NULL WHERE kind = 'podcast_rss';

-- Episodes are left hidden. 0015 hid them and this migration's up direction hid
-- the stragglers; un-hiding here would put content into feeds that the surface
-- above cannot show, which is the state this pair of migrations exists to end.
