-- The currency has a name now: пёрышки, not the placeholder "монеты" that
-- migration 013 seeded. Renaming is a data edit, not a schema change, so it
-- gets its own version rather than a rewrite of 013 — a developer database
-- that already ran 013 has to end up with the same words as a fresh one.
--
-- Only rows still holding 013's defaults are touched: a name typed into the
-- admin panel is somebody's decision and must survive this migration.
UPDATE economy_settings SET value = 'пёрышко', updated_at = '2026-10-06T00:00:00Z'
	WHERE key = 'currency_name_one' AND value = 'монета';
UPDATE economy_settings SET value = 'пёрышка', updated_at = '2026-10-06T00:00:00Z'
	WHERE key = 'currency_name_few' AND value = 'монеты';
UPDATE economy_settings SET value = 'пёрышек', updated_at = '2026-10-06T00:00:00Z'
	WHERE key = 'currency_name_many' AND value = 'монет';
