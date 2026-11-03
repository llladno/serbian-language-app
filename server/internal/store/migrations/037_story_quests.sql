-- Quests «Пройти 1 / 4 / 10 историй». A story counts once its learner has listened to the end and answered
-- (story_progress.completed_at). Only on a database the seed has already run on (a fresh one gets them from the seed,
-- and inserting here as well would duplicate them), and only the rungs that are missing, so a reward the admin has
-- edited is never touched. Same rule as 022.
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'stories_completed', 1, '', 'Пройти 1 историю', '', 5, 1, 24, '2026-11-02T00:00:00Z', '2026-11-02T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'stories_completed' AND target = 1);
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'stories_completed', 4, '', 'Пройти 4 истории', '', 10, 1, 25, '2026-11-02T00:00:00Z', '2026-11-02T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'stories_completed' AND target = 4);
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'stories_completed', 10, '', 'Пройти 10 историй', '', 20, 1, 26, '2026-11-02T00:00:00Z', '2026-11-02T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'stories_completed' AND target = 10);
