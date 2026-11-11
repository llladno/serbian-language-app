-- A price change that is announced ahead of time (the Black Friday notice of 2026-11).
--
-- The next prices sit beside the live ones, with two moments: announce_at, from which the app shows the countdown,
-- and change_at, when the next prices become the live ones. The switch is made by the server (Store.ApplyDuePriceChanges,
-- called whenever a price is read), not by a deploy, so the timer on the screen is the real one. After the switch the
-- next_* columns and both moments are emptied. next_price_kopecks = 0 means nothing is scheduled.
--
-- The public offer (landing/app/pages/oferta.vue, п. 4.1) and the landing quote the live prices: they have to be
-- changed on the day of the switch (docs/PRICE_CHANGE_2026-11-16.md).
ALTER TABLE tariffs ADD COLUMN next_price_kopecks       BIGINT NOT NULL DEFAULT 0;
ALTER TABLE tariffs ADD COLUMN next_first_price_kopecks BIGINT NOT NULL DEFAULT 0;
ALTER TABLE tariffs ADD COLUMN next_old_price_kopecks   BIGINT NOT NULL DEFAULT 0;
ALTER TABLE tariffs ADD COLUMN announce_at              TEXT   NOT NULL DEFAULT '';
ALTER TABLE tariffs ADD COLUMN change_at                TEXT   NOT NULL DEFAULT '';

-- «Уровень»: 399 ₽ for everybody, with no separate first-purchase price (were 199 and 299), «Год»: 1 999 ₽ with no struck-through
-- price (was 1 000 ₽, struck 2 000 ₽). Announced from the start of 10 November in Belgrade (23:00 UTC on the 9th), so
-- until the end of Sunday 15 November in Belgrade (23:00 UTC) the old prices are half of the new ones: a real discount,
-- because the new prices take over on their own at that moment.
UPDATE tariffs SET next_price_kopecks = 39900, next_first_price_kopecks = 0, next_old_price_kopecks = 0,
       announce_at = '2026-11-09T23:00:00Z', change_at = '2026-11-15T23:00:00Z'
 WHERE code = 'level';
UPDATE tariffs SET next_price_kopecks = 199900, next_first_price_kopecks = 0, next_old_price_kopecks = 0,
       announce_at = '2026-11-09T23:00:00Z', change_at = '2026-11-15T23:00:00Z'
 WHERE code = 'year';
