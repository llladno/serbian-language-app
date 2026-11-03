-- What the tariff events say more than "where": which level the window was about, which tariff was pressed, which
-- error stopped an order. Free text of a few words ("phase:2", "level", "level_cooldown:level"), written by the
-- app and cleaned on the way in (RecordTariffEventDetail). The admin's «Воронка тарифов» groups by it.
ALTER TABLE tariff_events ADD COLUMN detail TEXT NOT NULL DEFAULT '';
