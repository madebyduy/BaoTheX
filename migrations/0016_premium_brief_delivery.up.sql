-- Premium members can opt into the two fixed editorial appointments.  These
-- deliveries are kept separate from the user's ordinary text digest so a
-- morning/evening audio cannot be sent twice.
ALTER TABLE notification_preferences
  ADD COLUMN IF NOT EXISTS evening_brief_enabled BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE IF NOT EXISTS audio_brief_deliveries (
  id             BIGSERIAL PRIMARY KEY,
  audio_brief_id BIGINT NOT NULL REFERENCES audio_briefs(id) ON DELETE CASCADE,
  user_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  telegram_message_id BIGINT,
  sent_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  error          TEXT,
  UNIQUE (audio_brief_id, user_id)
);

CREATE INDEX IF NOT EXISTS audio_brief_deliveries_user_idx
  ON audio_brief_deliveries (user_id, sent_at DESC);
