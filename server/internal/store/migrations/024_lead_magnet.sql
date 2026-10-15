-- Lead magnets handed out by the Telegram bot behind a channel-subscription gate
-- (first one: the "Top-100 words" PDF). One row per (magnet, Telegram user): when they
-- first followed the link, whether they were already in the channel at that moment
-- (so "subscribed because of this" can be told apart from "was already ours"), and
-- whether/when the file went out. Counting is done on demand from these rows.
-- (023 is taken by the bot opt-out work that is still in progress.)
CREATE TABLE IF NOT EXISTS lead_magnet_claims (
	slug         TEXT    NOT NULL,
	tg_id        TEXT    NOT NULL,
	source       TEXT    NOT NULL DEFAULT '',
	was_member   INTEGER NOT NULL DEFAULT 0,
	started_at   TEXT    NOT NULL,
	delivered_at TEXT,
	deliveries   INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (slug, tg_id)
);

-- Telegram's file_id for an uploaded magnet, keyed by the file's sha256: the first send
-- uploads the PDF, every later one sends the id (instant, and no upload through the proxy).
CREATE TABLE IF NOT EXISTS lead_magnet_files (
	slug       TEXT PRIMARY KEY,
	sha        TEXT NOT NULL,
	file_id    TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
