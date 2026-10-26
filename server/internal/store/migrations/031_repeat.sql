-- "Повторить": an attempt remembers when the answer was only almost right (a one-letter slip the
-- checker forgave), and a learner's daily batch is stored so the same fifteen exercises come back
-- until they are done. Old attempts get typo = 0: the slip was never recorded.
ALTER TABLE attempts ADD COLUMN typo INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS repeat_sessions (
	id           {{.AutoID}},
	user_id      TEXT NOT NULL,
	day          TEXT NOT NULL,
	exercise_ids TEXT NOT NULL,
	-- the newest attempt of the learner when the batch started: answers count as part of the batch
	-- only after it (attempt ids, not clock times, so two answers in one second stay in order)
	after_attempt BIGINT NOT NULL DEFAULT 0,
	started_at   TEXT NOT NULL,
	UNIQUE (user_id, day)
);
