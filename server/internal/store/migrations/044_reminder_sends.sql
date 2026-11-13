-- One row per study reminder the bot sends, with which of its texts it was, so the texts can be compared by how many
-- people come back after them (the return is read from reviews and attempts after sent_at, not stored here).
--
-- kind is the message ('inactive': the «давно не заглядывали» nudge), variant names the text: 'n1'..'n5' are the five
-- with a name or @nick in them, 'p1'..'p5' the same five without (nothing to address the person by). The text itself is
-- in bot_outbox. bot_reminders keeps only the latest send per person, this table keeps them all.
-- Its role needs GRANT SELECT, INSERT on the table, by hand on the production database.
CREATE TABLE IF NOT EXISTS reminder_sends (
	id      {{.AutoID}},
	user_id TEXT NOT NULL,
	kind    TEXT NOT NULL,
	variant TEXT NOT NULL,
	sent_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS reminder_sends_kind ON reminder_sends (kind, variant, sent_at);
