-- Internal currency that comes with a tariff (spec: docs/superpowers/specs/2026-10-14-tariffs-design.md,
-- "Bonus currency"). Paying for a tariff credits this many зёрнышки once, in the same
-- transaction that grants the tariff, as a `topup` ledger entry keyed `payment:<id>`.
-- 0 means no bonus. It is a column, not a constant, so the amounts can be changed
-- (or switched off) with one UPDATE, and the admin can read what was promised.
ALTER TABLE tariffs ADD COLUMN bonus_currency BIGINT NOT NULL DEFAULT 0;

UPDATE tariffs SET bonus_currency = 1000  WHERE code = 'level';
UPDATE tariffs SET bonus_currency = 15000 WHERE code = 'year';
