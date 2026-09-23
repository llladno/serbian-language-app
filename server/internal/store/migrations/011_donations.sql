-- Migration 011: donations received via the Tribute webhook (see
-- docs/superpowers/specs/2026-09-23-donations-design.md). Read-only from the
-- admin panel's ProdDbService, same as support_messages — no workflow state.

CREATE TABLE donations (
	id                 {{.AutoID}},
	user_id            TEXT,
	telegram_user_id   TEXT NOT NULL,
	telegram_username  TEXT,
	amount_minor_units BIGINT NOT NULL,
	currency           TEXT NOT NULL,
	event_type         TEXT NOT NULL,
	tribute_event_id   TEXT NOT NULL,
	raw_payload        TEXT NOT NULL,
	created_at         TEXT NOT NULL
);
CREATE UNIQUE INDEX donations_tribute_event_id_idx ON donations(tribute_event_id);
