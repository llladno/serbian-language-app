-- Migration 006: UTM link tracking — first-touch attribution on users, plus
-- a raw visit log. Read-only from ucimo-content-admin's ProdDbService, the
-- same pattern as support_messages (migration 005). See
-- docs/superpowers/specs/2026-09-22-utm-link-tracking-design.md.

ALTER TABLE users ADD COLUMN utm_source TEXT;
ALTER TABLE users ADD COLUMN utm_medium TEXT;
ALTER TABLE users ADD COLUMN utm_campaign TEXT;
ALTER TABLE users ADD COLUMN utm_content TEXT;

CREATE TABLE link_visits (
	id           {{.AutoID}},
	utm_source   TEXT,
	utm_medium   TEXT,
	utm_campaign TEXT,
	utm_content  TEXT,
	created_at   TEXT NOT NULL
);
