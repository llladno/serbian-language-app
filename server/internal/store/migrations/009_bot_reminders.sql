-- Migration 009: per-account state for the Telegram bot's reminder sweep
-- (internal/api RunReminderSweep) — dedup so "all done today" and "you've
-- been away" nudges each send at most once per occurrence.
--
-- Originally numbered 003, but that version was already taken (and applied
-- in prod on 2026-09-14, three days before this file was written) by the
-- unrelated migrate003 hook in migration_hooks.go (junk-account cleanup /
-- legacy Telegram linking). Reusing 003 meant the migration runner saw
-- version 3 already recorded in schema_migrations and silently skipped this
-- file's CREATE TABLE forever — the table never existed in prod, causing
-- every reminder sweep to fail with "relation bot_reminders does not exist".
-- Renumbered to 009 (next free slot after 008_bot_outbox.sql) to fix.

CREATE TABLE bot_reminders (
	user_id          TEXT PRIMARY KEY,
	all_done_date    TEXT,
	inactive_sent_at TEXT
);
