-- A click on the button under a «Повторить» message. The button opens /repeat?n=<token> and the server
-- sees that request, so a click is told apart from the learner opening the app by themselves, which
-- opened_at (set by GET /api/repeat) cannot do. token is random per message. clicked_at is the first
-- such request, empty until then.
ALTER TABLE repeat_nudges ADD COLUMN token TEXT NOT NULL DEFAULT '';
ALTER TABLE repeat_nudges ADD COLUMN clicked_at TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS repeat_nudges_token ON repeat_nudges (token);
