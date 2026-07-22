-- Audio editions are now a Telegram benefit for every linked account rather
-- than a Premium-only delivery. Rename queued/history jobs without touching
-- their delivery claims, which already prevent duplicates per brief and user.
UPDATE jobs
SET kind = 'send_audio_brief',
    dedup_key = replace(dedup_key, 'premium-audio:', 'audio-delivery:')
WHERE kind = 'send_premium_audio_brief';
