-- The "subscribe to the channel" quest has to be checked against a channel, and
-- migration 013 seeded that setting empty, which hides the quest altogether
-- (nobody could ever finish it). Production's channel is @ucimosrb, so it is
-- filled in here rather than left as a step to remember after the deploy.
--
-- Only an empty value is filled in, like 017 and 018. A channel typed into the
-- admin panel, or set by hand on a developer database, is somebody's decision
-- and must survive this migration.
UPDATE economy_settings SET value = '@ucimosrb', updated_at = '2026-10-08T00:00:00Z'
	WHERE key = 'telegram_channel' AND value = '';
