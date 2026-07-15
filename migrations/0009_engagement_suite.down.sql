UPDATE sources SET enabled = FALSE WHERE kind = 'youtube';
ALTER TABLE notification_preferences DROP COLUMN IF EXISTS push_enabled;
ALTER TABLE notification_preferences DROP COLUMN IF EXISTS audio_enabled;
DROP TABLE IF EXISTS push_subscriptions;
DROP TABLE IF EXISTS video_briefs;
DROP TABLE IF EXISTS audio_briefs;
DROP TABLE IF EXISTS payment_orders;
DROP TABLE IF EXISTS user_subscriptions;
