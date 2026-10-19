-- One-time feedback survey shown after a learner's third finished lesson (or on the
-- next finished lesson for those who were already past it). One row per learner, ever:
-- the row appears the moment the survey is first offered (status 'shown'), so a closed tab
-- never brings it back, and becomes 'answered' or 'dismissed'. List-valued answers are
-- JSON arrays of option codes, and the admin panel reads this table through its read-only role.
CREATE TABLE IF NOT EXISTS survey_responses (
	user_id      TEXT NOT NULL PRIMARY KEY,
	status       TEXT NOT NULL,
	lessons_done INTEGER NOT NULL DEFAULT 0,
	rating       INTEGER,
	likes        TEXT NOT NULL DEFAULT '[]',
	pains        TEXT NOT NULL DEFAULT '[]',
	pain_text    TEXT NOT NULL DEFAULT '',
	purpose      TEXT NOT NULL DEFAULT '',
	wish         TEXT NOT NULL DEFAULT '',
	reward       INTEGER NOT NULL DEFAULT 0,
	shown_at     TEXT NOT NULL,
	answered_at  TEXT
);

INSERT INTO economy_settings (key, value, updated_at) VALUES
	('survey_reward', '10', '2026-10-19T00:00:00Z')
ON CONFLICT (key) DO NOTHING;
