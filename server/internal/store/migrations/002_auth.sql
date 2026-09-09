-- Migration 002: real auth schema. Completed by the migrate002 hook in
-- migration_hooks.go — this .sql only creates the *_new staging tables.
--
-- The hook fills those from the legacy name-keyed rows, swaps them into
-- place, and only then creates identities/sessions/email_tokens: they
-- reference users(id), so on SQLite they must not exist while
-- DROP TABLE users runs.

CREATE TABLE users_new (
	id         TEXT PRIMARY KEY,
	name       TEXT NOT NULL,
	created_at TEXT NOT NULL
);

CREATE TABLE srs_cards_new (
	user_id       TEXT NOT NULL,
	card_id       TEXT NOT NULL,
	kind          TEXT NOT NULL,
	ref_id        TEXT NOT NULL,
	ease          REAL NOT NULL DEFAULT 2.5,
	interval_days INTEGER NOT NULL DEFAULT 0,
	reps          INTEGER NOT NULL DEFAULT 0,
	lapses        INTEGER NOT NULL DEFAULT 0,
	state         TEXT NOT NULL DEFAULT 'new',
	due           TEXT,
	updated_at    TEXT NOT NULL,
	PRIMARY KEY (user_id, card_id)
);
CREATE TABLE reviews_new (
	id          {{.AutoID}},
	user_id     TEXT NOT NULL,
	card_id     TEXT NOT NULL,
	grade       INTEGER NOT NULL,
	reviewed_at TEXT NOT NULL
);
CREATE TABLE attempts_new (
	id           {{.AutoID}},
	user_id      TEXT NOT NULL,
	exercise_id  TEXT NOT NULL,
	lesson       TEXT NOT NULL,
	block        TEXT NOT NULL,
	answer       TEXT NOT NULL,
	correct      INTEGER NOT NULL,
	attempted_at TEXT NOT NULL
);
CREATE TABLE lesson_progress_new (
	user_id      TEXT NOT NULL,
	lesson       TEXT NOT NULL,
	status       TEXT NOT NULL,
	started_at   TEXT,
	completed_at TEXT,
	PRIMARY KEY (user_id, lesson)
);
CREATE TABLE lesson_step_progress_new (
	user_id      TEXT NOT NULL,
	lesson       TEXT NOT NULL,
	step         TEXT NOT NULL,
	status       TEXT NOT NULL,
	completed_at TEXT,
	PRIMARY KEY (user_id, lesson, step)
);
