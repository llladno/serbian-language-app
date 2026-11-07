-- The first-lesson tour with the swallow (docs/superpowers/specs/2026-11-04-onboarding-design.md). One row per learner
-- who has answered the tour: 'postponed' (put off, never shown by itself again) or 'done' (seen to the end). No row
-- means the tour is still due once the first lesson is finished. status only ever moves postponed -> done.
CREATE TABLE IF NOT EXISTS onboarding (
	user_id    TEXT NOT NULL PRIMARY KEY,
	status     TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

-- The tour is for new accounts: everyone who exists now counts as having put it off, so it does not appear out of
-- nowhere in the middle of their course. The quest below still lets them take it, and the 5 seeds, from «Задания».
INSERT INTO onboarding (user_id, status, updated_at)
SELECT id, 'postponed', '2026-11-04T00:00:00Z' FROM users;

-- Quest «Знакомство с приложением». Only on a database the seed has already run on (a fresh one gets it from the seed,
-- and inserting here as well would duplicate it), and only if it is missing, so a reward the admin has edited is never
-- touched. Same rule as 037.
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'onboarding_done', 1, '', 'Знакомство с приложением', 'Ласточка покажет, где что', 5, 1, 5, '2026-11-04T00:00:00Z', '2026-11-04T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'onboarding_done');
