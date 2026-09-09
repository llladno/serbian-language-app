CREATE TABLE IF NOT EXISTS users (
	name       TEXT PRIMARY KEY,
	created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS srs_cards (
	user_name     TEXT NOT NULL,
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
	PRIMARY KEY (user_name, card_id)
);
CREATE TABLE IF NOT EXISTS reviews (
	id          {{.AutoID}},
	user_name   TEXT NOT NULL,
	card_id     TEXT NOT NULL,
	grade       INTEGER NOT NULL,
	reviewed_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS attempts (
	id           {{.AutoID}},
	user_name    TEXT NOT NULL,
	exercise_id  TEXT NOT NULL,
	lesson       TEXT NOT NULL,
	block        TEXT NOT NULL,
	answer       TEXT NOT NULL,
	correct      INTEGER NOT NULL,
	attempted_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS lesson_progress (
	user_name    TEXT NOT NULL,
	lesson       TEXT NOT NULL,
	status       TEXT NOT NULL,
	started_at   TEXT,
	completed_at TEXT,
	PRIMARY KEY (user_name, lesson)
);
CREATE TABLE IF NOT EXISTS lesson_step_progress (
	user_name    TEXT NOT NULL,
	lesson       TEXT NOT NULL,
	step         TEXT NOT NULL,
	status       TEXT NOT NULL,
	completed_at TEXT,
	PRIMARY KEY (user_name, lesson, step)
);
CREATE INDEX IF NOT EXISTS reviews_user ON reviews(user_name);
CREATE INDEX IF NOT EXISTS attempts_user ON attempts(user_name);
