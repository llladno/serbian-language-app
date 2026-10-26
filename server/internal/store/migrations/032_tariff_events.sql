-- Where the paid tariffs were shown to a learner, and where they went on from there: the raw log behind the
-- admin's «Воронка тарифов». One row per showing (kind 'view') or press of the way to the tariffs (kind
-- 'click') in one of the places that sell them (source: the profile banner, the repeat-limit window, the
-- window of a paid level). Nothing here is counted twice within a few seconds (RecordTariffEvent), and who is
-- unique and how many times is worked out from the rows. The admin reads it through its read-only role.
CREATE TABLE IF NOT EXISTS tariff_events (
	id         {{.AutoID}},
	user_id    TEXT NOT NULL,
	kind       TEXT NOT NULL,
	source     TEXT NOT NULL,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS tariff_events_user_idx ON tariff_events (user_id, kind, source, created_at);
CREATE INDEX IF NOT EXISTS tariff_events_time_idx ON tariff_events (created_at);
