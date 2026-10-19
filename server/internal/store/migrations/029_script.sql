-- The Serbian script a learner reads the course in: 'latin' (the way the course is
-- written) or 'cyrillic' (shown by converting at render time, the content is never
-- rewritten). An account setting rather than a browser one, so it follows the
-- learner between the browser and the Telegram Mini App, which keep separate local
-- storage. See docs/superpowers/specs/2026-10-19-cyrillic-switch-design.md.
ALTER TABLE users ADD COLUMN script TEXT NOT NULL DEFAULT 'latin';
