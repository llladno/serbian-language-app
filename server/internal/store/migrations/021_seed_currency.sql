-- The currency is now called зёрнышки (sunflower seeds), not пёрышки. Same as
-- 017: only rows still holding the previous default are touched, so a name
-- typed into the admin panel is somebody's decision and survives this migration.
UPDATE economy_settings SET value = 'зёрнышко', updated_at = '2026-10-09T00:00:00Z'
	WHERE key = 'currency_name_one' AND value = 'пёрышко';
UPDATE economy_settings SET value = 'зёрнышка', updated_at = '2026-10-09T00:00:00Z'
	WHERE key = 'currency_name_few' AND value = 'пёрышка';
UPDATE economy_settings SET value = 'зёрнышек', updated_at = '2026-10-09T00:00:00Z'
	WHERE key = 'currency_name_many' AND value = 'пёрышек';
