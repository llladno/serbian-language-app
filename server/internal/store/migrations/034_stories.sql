-- Stories: short listening stories (three passes, quick questions, final questions). See docs/STORIES_PLAN.md.
-- A story is one row. Everything the player needs (chapters, sentences with audio files and word marks, scenes,
-- quizzes) lives in `content` as JSON, so the admin can fix a sentence without a migration. `checks` keeps the
-- result of scripts/check_story.py for the admin. Files are referenced by URL (S3 in production).
CREATE TABLE IF NOT EXISTS stories (
	slug        TEXT PRIMARY KEY,
	title_sr    TEXT NOT NULL,
	title_ru    TEXT NOT NULL,
	level       TEXT NOT NULL DEFAULT '',
	tier        TEXT NOT NULL,
	is_demo     INTEGER NOT NULL DEFAULT 0,
	status      TEXT NOT NULL DEFAULT 'draft',
	sort_order  INTEGER NOT NULL DEFAULT 0,
	cover_url   TEXT NOT NULL DEFAULT '',
	duration_ms INTEGER NOT NULL DEFAULT 0,
	content     TEXT NOT NULL,
	checks      TEXT NOT NULL DEFAULT '',
	created_at  TEXT NOT NULL,
	updated_at  TEXT NOT NULL
);

-- One row per learner and story: how far they got and whether the reward was paid.
CREATE TABLE IF NOT EXISTS story_progress (
	user_id      TEXT NOT NULL,
	slug         TEXT NOT NULL,
	pass_reached INTEGER NOT NULL DEFAULT 0,
	completed_at TEXT NOT NULL DEFAULT '',
	quiz_score   INTEGER NOT NULL DEFAULT 0,
	quiz_total   INTEGER NOT NULL DEFAULT 0,
	reward_paid  INTEGER NOT NULL DEFAULT 0,
	updated_at   TEXT NOT NULL,
	PRIMARY KEY (user_id, slug)
);

-- Raw events behind the stories metrics (completion of the demo, second-story purchase, passes, questions).
CREATE TABLE IF NOT EXISTS story_events (
	id         {{.AutoID}},
	user_id    TEXT NOT NULL,
	slug       TEXT NOT NULL,
	kind       TEXT NOT NULL,
	detail     TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS story_events_story_idx ON story_events (slug, kind, created_at);
CREATE INDEX IF NOT EXISTS story_events_user_idx ON story_events (user_id, created_at);
