-- The Telegram nudge for learners who stopped using «Повторить».
--
-- repeat_state: when a learner last opened «Повторить» (for every learner, including those with
-- unlimited repeats, who get no repeat_sessions row), and whether they pressed «Не про повторения»
-- under the nudge. The opt-out lives here and not in bot_prefs so that the settings screen and
-- /stop keep their three switches.
--
-- repeat_nudges: one row per nudge written for a learner. kind is 'sent' for a real message and
-- 'control' for a learner in the half that is never written to, kept so the two halves can be
-- compared. opened_at is the first time the learner opened «Повторить» after the row, empty until
-- then.
CREATE TABLE IF NOT EXISTS repeat_state (
	user_id        TEXT PRIMARY KEY,
	last_opened_at TEXT NOT NULL DEFAULT '',
	nudge_off      INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS repeat_nudges (
	id         {{.AutoID}},
	user_id    TEXT NOT NULL,
	variant    INTEGER NOT NULL,
	kind       TEXT NOT NULL,
	created_at TEXT NOT NULL,
	opened_at  TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS repeat_nudges_user ON repeat_nudges (user_id, id);
