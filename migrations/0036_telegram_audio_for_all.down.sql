UPDATE jobs
SET kind = 'send_premium_audio_brief',
    dedup_key = replace(dedup_key, 'audio-delivery:', 'premium-audio:')
WHERE kind = 'send_audio_brief';
