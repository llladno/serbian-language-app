-- Migration 010: notifications — in-app notifications composed in
-- ucimo-content-admin's "Уведомления" page, read by the bell dropdown in
-- serbian-app's AppNav. notification_recipients materializes one row per
-- targeted user at send time (including "all" — future registrations are
-- not retroactively included). See
-- docs/superpowers/specs/2026-09-23-notifications-design.md.
--
-- Only ucimo-content-admin ever INSERTs into these two tables (a direct
-- Postgres write via ProdDbService) — serbian-app's own Go code only reads
-- and updates read_at.

CREATE TABLE IF NOT EXISTS notifications (
	id         {{.AutoID}},
	text       TEXT NOT NULL,
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS notification_recipients (
	id              {{.AutoID}},
	notification_id INTEGER NOT NULL,
	user_id         TEXT NOT NULL,
	read_at         TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS notification_recipients_user_idx
	ON notification_recipients (user_id, read_at);
