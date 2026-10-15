-- Paying rubles for the paid levels (spec: docs/superpowers/specs/2026-10-14-tariffs-design.md).
--
-- Two one-off tariffs, no auto-renew:
--   level  the next level the learner has not opened, forever
--   year   every level for period_days, new ones included
--
-- A year is not a row in `products` (those are priced in the in-app currency):
-- it is an entitlement with an end date, kind 'all_access'. A level bought for
-- rubles is the same `phase_unlock` entitlement the currency shop grants, so
-- nothing that already reads entitlements has to learn about money.
ALTER TABLE user_entitlements ADD COLUMN expires_at TEXT NOT NULL DEFAULT '';

CREATE TABLE tariffs (
	code                TEXT PRIMARY KEY,
	title               TEXT NOT NULL,
	price_kopecks       BIGINT NOT NULL,
	first_price_kopecks BIGINT NOT NULL DEFAULT 0,
	old_price_kopecks   BIGINT NOT NULL DEFAULT 0,
	period_days         INTEGER NOT NULL DEFAULT 0,
	active              INTEGER NOT NULL DEFAULT 1,
	updated_at          TEXT NOT NULL
);

INSERT INTO tariffs (code, title, price_kopecks, first_price_kopecks, old_price_kopecks, period_days, active, updated_at)
VALUES ('level', 'Уровень', 29900, 24900, 0, 0, 1, '2026-10-15T00:00:00Z');
INSERT INTO tariffs (code, title, price_kopecks, first_price_kopecks, old_price_kopecks, period_days, active, updated_at)
VALUES ('year', 'Год — все уровни', 100000, 0, 200000, 365, 1, '2026-10-15T00:00:00Z');

-- One row per attempt to pay. The level a `level` order opens is fixed when the
-- order is made (granted_phase), not when the money arrives: two quick orders
-- would otherwise both resolve to the same "next" level and charge twice for it.
CREATE TABLE payments (
	id                  {{.AutoID}},
	user_id             TEXT NOT NULL,
	tariff_code         TEXT NOT NULL,
	granted_phase       TEXT NOT NULL DEFAULT '',
	amount_kopecks      BIGINT NOT NULL,
	provider            TEXT NOT NULL,
	provider_payment_id TEXT NOT NULL DEFAULT '',
	status              TEXT NOT NULL,
	receipt_contact     TEXT NOT NULL DEFAULT '',
	test                INTEGER NOT NULL DEFAULT 0,
	created_at          TEXT NOT NULL,
	paid_at             TEXT NOT NULL DEFAULT ''
);
CREATE INDEX payments_user_idx ON payments (user_id, status);
CREATE UNIQUE INDEX payments_provider_idx ON payments (provider, provider_payment_id) WHERE provider_payment_id <> '';
