-- A personal discount: the owner picks a learner in the admin panel, a percent and for how long it holds. While it is
-- on, that learner's tariffs screen and their order are priced with it, and the screen says so.
--
-- The admin panel writes this table itself (like broadcasts and bot_outbox), so the id is made there (a UUID) and needs
-- no sequence. Its role needs GRANT SELECT, INSERT, UPDATE on the table, by hand on the production database.
-- expires_at is UTC RFC 3339 text, so it compares like time. canceled_at stays empty until the owner takes the discount
-- back. A discount is spent by the first order of that learner paid with it: payments.discount_id names the discount an
-- order was priced with, and a discount with a succeeded payment is no longer offered.
CREATE TABLE IF NOT EXISTS personal_discounts (
	id          TEXT NOT NULL PRIMARY KEY,
	user_id     TEXT NOT NULL,
	percent     INTEGER NOT NULL,
	note        TEXT NOT NULL DEFAULT '',
	created_at  TEXT NOT NULL,
	expires_at  TEXT NOT NULL,
	canceled_at TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS personal_discounts_user ON personal_discounts (user_id);

ALTER TABLE payments ADD COLUMN discount_id TEXT NOT NULL DEFAULT '';
