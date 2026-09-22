-- Migration 007: bot_messages — editable Telegram bot reply templates,
-- edited from ucimo-content-admin's "Бот" → "Сообщения" page. Seeded with
-- default copy by migrate007 (migration_hooks.go) so there's always
-- something to show/edit. See
-- docs/superpowers/specs/2026-09-22-bot-messages-design.md.

CREATE TABLE IF NOT EXISTS bot_messages (
	key        TEXT PRIMARY KEY,
	text       TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
