-- Migration 013: in-app currency. See
-- docs/superpowers/specs/2026-09-28-currency-design.md.
--
-- No REFERENCES anywhere, deliberately: ucimo-content-admin writes several of
-- these tables directly through its own connection (same convention as
-- notification_recipients in 010 and bot_outbox.broadcast_id in 012).
--
-- There is no balance column and there must never be one: a balance is always
-- SUM(currency_ledger.amount), which is what makes a second writer safe.

ALTER TABLE users ADD COLUMN timezone TEXT NOT NULL DEFAULT 'Europe/Belgrade';

-- Makes "is this the first attempt at this exercise" cheap: that check gates
-- every action counted toward the daily goal.
CREATE INDEX IF NOT EXISTS attempts_user_exercise ON attempts (user_id, exercise_id);

CREATE TABLE IF NOT EXISTS currency_ledger (
	id              {{.AutoID}},
	user_id         TEXT NOT NULL,
	amount          BIGINT NOT NULL,
	kind            TEXT NOT NULL,
	ref             TEXT NOT NULL DEFAULT '',
	idempotency_key TEXT NOT NULL,
	comment         TEXT NOT NULL DEFAULT '',
	created_by      TEXT NOT NULL DEFAULT '',
	created_at      TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS currency_ledger_idem ON currency_ledger (idempotency_key);
CREATE INDEX IF NOT EXISTS currency_ledger_user ON currency_ledger (user_id, id);

CREATE TABLE IF NOT EXISTS user_daily_activity (
	user_id TEXT NOT NULL,
	day     TEXT NOT NULL,
	actions INTEGER NOT NULL DEFAULT 0,
	goal    INTEGER NOT NULL,
	PRIMARY KEY (user_id, day)
);

CREATE TABLE IF NOT EXISTS streak_repairs (
	user_id    TEXT NOT NULL,
	day        TEXT NOT NULL,
	created_at TEXT NOT NULL,
	PRIMARY KEY (user_id, day)
);

CREATE TABLE IF NOT EXISTS user_answer_streak (
	user_id    TEXT PRIMARY KEY,
	current    INTEGER NOT NULL DEFAULT 0,
	best       INTEGER NOT NULL DEFAULT 0,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS quests (
	id          {{.AutoID}},
	kind        TEXT NOT NULL,
	target      INTEGER NOT NULL DEFAULT 0,
	param       TEXT NOT NULL DEFAULT '',
	title       TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	reward      BIGINT NOT NULL,
	active      INTEGER NOT NULL DEFAULT 1,
	sort_order  INTEGER NOT NULL DEFAULT 0,
	created_at  TEXT NOT NULL,
	updated_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS quest_claims (
	quest_id   BIGINT NOT NULL,
	user_id    TEXT NOT NULL,
	claimed_at TEXT NOT NULL,
	PRIMARY KEY (quest_id, user_id)
);

CREATE TABLE IF NOT EXISTS products (
	id               {{.AutoID}},
	kind             TEXT NOT NULL,
	ref              TEXT NOT NULL,
	title            TEXT NOT NULL,
	description      TEXT NOT NULL DEFAULT '',
	price            BIGINT NOT NULL,
	discount_percent INTEGER NOT NULL DEFAULT 0,
	discount_from    TEXT NOT NULL DEFAULT '',
	discount_to      TEXT NOT NULL DEFAULT '',
	grant_qty        INTEGER NOT NULL DEFAULT 1,
	active           INTEGER NOT NULL DEFAULT 1,
	sort_order       INTEGER NOT NULL DEFAULT 0,
	created_at       TEXT NOT NULL,
	updated_at       TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS products_kind_ref ON products (kind, ref);

CREATE TABLE IF NOT EXISTS promo_codes (
	id               {{.AutoID}},
	code             TEXT NOT NULL,
	discount_percent INTEGER NOT NULL,
	scope            TEXT NOT NULL DEFAULT 'all',
	valid_from       TEXT NOT NULL DEFAULT '',
	valid_to         TEXT NOT NULL DEFAULT '',
	max_redemptions  INTEGER NOT NULL DEFAULT 0,
	max_per_user     INTEGER NOT NULL DEFAULT 1,
	active           INTEGER NOT NULL DEFAULT 1,
	created_at       TEXT NOT NULL,
	updated_at       TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS promo_codes_code ON promo_codes (code);

CREATE TABLE IF NOT EXISTS promo_code_products (
	promo_code_id BIGINT NOT NULL,
	product_id    BIGINT NOT NULL,
	PRIMARY KEY (promo_code_id, product_id)
);

CREATE TABLE IF NOT EXISTS promo_redemptions (
	id            {{.AutoID}},
	promo_code_id BIGINT NOT NULL,
	user_id       TEXT NOT NULL,
	product_id    BIGINT NOT NULL,
	ledger_id     BIGINT NOT NULL,
	created_at    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS promo_redemptions_code ON promo_redemptions (promo_code_id);

CREATE TABLE IF NOT EXISTS user_entitlements (
	user_id    TEXT NOT NULL,
	kind       TEXT NOT NULL,
	ref        TEXT NOT NULL,
	qty        BIGINT NOT NULL DEFAULT 1,
	updated_at TEXT NOT NULL,
	PRIMARY KEY (user_id, kind, ref)
);

CREATE TABLE IF NOT EXISTS economy_settings (
	key        TEXT PRIMARY KEY,
	value      TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

INSERT INTO economy_settings (key, value, updated_at) VALUES
	('currency_name_one',          'монета',              '2026-09-29T00:00:00Z'),
	('currency_name_few',          'монеты',              '2026-09-29T00:00:00Z'),
	('currency_name_many',         'монет',               '2026-09-29T00:00:00Z'),
	('daily_goal',                 '10',                  '2026-09-29T00:00:00Z'),
	('streak_drip',                '[[1,1],[30,2],[100,3]]', '2026-09-29T00:00:00Z'),
	('streak_repair_window_hours', '48',                  '2026-09-29T00:00:00Z'),
	('telegram_channel',           '',                    '2026-09-29T00:00:00Z')
ON CONFLICT (key) DO NOTHING;
