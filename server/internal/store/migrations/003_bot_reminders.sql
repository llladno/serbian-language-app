-- Migration 003: per-account state for the Telegram bot's reminder sweep
-- (internal/api RunReminderSweep) — dedup so "all done today" and "you've
-- been away" nudges each send at most once per occurrence.

CREATE TABLE bot_reminders (
	user_id          TEXT PRIMARY KEY,
	all_done_date    TEXT,
	inactive_sent_at TEXT
);
