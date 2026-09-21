-- Migration 005: user-submitted support messages (bugs, wishes, questions),
-- written from the profile page's support card. Read-only from the admin
-- panel's ProdDbService — no status/workflow column, just a list.

CREATE TABLE support_messages (
	id         {{.AutoID}},
	user_id    TEXT NOT NULL,
	message    TEXT NOT NULL,
	created_at TEXT NOT NULL
);
