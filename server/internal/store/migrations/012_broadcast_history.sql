-- Migration 012: broadcasts — groups bot_outbox rows from a single admin
-- Telegram broadcast under a stable id, so ucimo-content-admin can list
-- past broadcasts and compute which current candidates weren't targeted
-- by a given one (e.g. registered after it went out). notifications/
-- notification_recipients need no schema change — they already have this
-- shape. See docs/superpowers/specs/2026-09-28-broadcast-history-design.md.
--
-- broadcast_id is nullable and unconstrained (no REFERENCES — same
-- convention as notification_recipients.notification_id in migration
-- 010): reminders and /start-flow replies keep it NULL, only rows
-- inserted by ucimo-content-admin's broadcast composer set it.

CREATE TABLE IF NOT EXISTS broadcasts (
	id         {{.AutoID}},
	text       TEXT NOT NULL,
	created_at TEXT NOT NULL
);

ALTER TABLE bot_outbox ADD COLUMN broadcast_id BIGINT;
CREATE INDEX IF NOT EXISTS bot_outbox_broadcast_idx ON bot_outbox (broadcast_id);
