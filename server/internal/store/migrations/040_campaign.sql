-- The «return» campaign (docs/superpowers/specs/2026-11-08-return-campaign-design.md): a one-off message to learners
-- who never reached lesson 02, in three waves, with two button styles and a group that is never written to.
--
-- campaign_sends: one row per person in the campaign's frozen audience. kind is 'sent' (gets a message) or 'holdout'
-- (never written to, kept for the comparison). ord is the person's place in the random order the waves take them in.
-- wave is 0 until a wave takes the person, style (0 = A, 1 = C) and token (the click token in the link) are set then.
-- sent_at stays empty until the message has actually gone out, clicked_at until the first press on its button.
CREATE TABLE IF NOT EXISTS campaign_sends (
	id         {{.AutoID}},
	campaign   TEXT NOT NULL,
	user_id    TEXT NOT NULL,
	channel    TEXT NOT NULL,
	segment    INTEGER NOT NULL,
	kind       TEXT NOT NULL,
	ord        BIGINT NOT NULL,
	wave       INTEGER NOT NULL DEFAULT 0,
	style      INTEGER NOT NULL DEFAULT 0,
	token      TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	sent_at    TEXT NOT NULL DEFAULT '',
	clicked_at TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS campaign_sends_user ON campaign_sends (campaign, user_id);
CREATE INDEX IF NOT EXISTS campaign_sends_token ON campaign_sends (token);

-- campaign_waves: the opening of a wave in one channel, with what it was decided on (the leading style and its share, in
-- thousandths), so the results can say why a wave looked the way it did.
CREATE TABLE IF NOT EXISTS campaign_waves (
	campaign     TEXT NOT NULL,
	wave         INTEGER NOT NULL,
	channel      TEXT NOT NULL,
	opened_at    TEXT NOT NULL,
	leader       INTEGER NOT NULL,
	leader_share INTEGER NOT NULL,
	PRIMARY KEY (campaign, wave, channel)
);

-- email_optout: a person who asked not to get study e-mails. Telegram has bot_prefs, this is the e-mail side.
CREATE TABLE IF NOT EXISTS email_optout (
	user_id    TEXT NOT NULL PRIMARY KEY,
	created_at TEXT NOT NULL
);

-- A queued Telegram message can be a photo with the text as its caption: photo is the image address, empty for plain text.
ALTER TABLE bot_outbox ADD COLUMN photo TEXT NOT NULL DEFAULT '';
