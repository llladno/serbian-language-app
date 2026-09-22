-- Migration 008: bot_outbox — priority send queue for every bot-originated
-- message (webhook replies, reminders, admin broadcasts from
-- ucimo-content-admin's "Бот" → "Рассылки" page), drained by
-- internal/outbox.ProcessNext at a rate-limited pace so Bot API limits are
-- never hit. See docs/superpowers/specs/2026-09-22-bot-messages-design.md.
--
-- chat_id is TEXT (not a numeric type), matching the existing convention
-- for Telegram ids (identities.provider_uid) elsewhere in this schema.

CREATE TABLE IF NOT EXISTS bot_outbox (
	id            {{.AutoID}},
	chat_id       TEXT NOT NULL,
	text          TEXT NOT NULL,
	button_label  TEXT NOT NULL DEFAULT '',
	button_type   TEXT NOT NULL DEFAULT '',
	button_target TEXT NOT NULL DEFAULT '',
	priority      SMALLINT NOT NULL,
	status        TEXT NOT NULL DEFAULT 'pending',
	error         TEXT NOT NULL DEFAULT '',
	created_at    TEXT NOT NULL,
	sent_at       TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS bot_outbox_pending_idx ON bot_outbox (status, priority, created_at);
