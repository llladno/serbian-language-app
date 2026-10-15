-- Which kinds of Telegram messages an account has agreed to receive.
--
-- Until now the reminder sweep wrote to every account with a linked Telegram
-- and there was no way to say "stop". Three categories, so that a person who
-- wants no nudges can still hear about their paid year ending:
--   learning  study reminders (all done today, come back)        default on
--   billing   reminders about a paid tariff (year ending)         default on
--   news      new levels, offers, announcements                   default OFF
-- News is advertising in the legal sense, so it needs a yes, not a missing no.
-- Service messages (sign-in, a payment received, access opened) are not in
-- this table: they answer something the person just did and always go out.
--
-- No row means "the defaults": nothing is written for an account that never
-- touched its settings.
CREATE TABLE bot_prefs (
	user_id    TEXT PRIMARY KEY,
	learning   INTEGER NOT NULL DEFAULT 1,
	billing    INTEGER NOT NULL DEFAULT 1,
	news       INTEGER NOT NULL DEFAULT 0,
	updated_at TEXT NOT NULL
);
