-- A level is still worth 50 / 70 / 90, but most of it now arrives lesson by
-- lesson (see seedLessonRewards): the level quest keeps only the finishing
-- bonus. The lesson rows themselves are written by the seed, which is the part
-- that knows the course. This migration only moves the three numbers the seed
-- already wrote on databases seeded before the split.
--
-- Conditional on the old amount, like 017. A reward edited in the admin panel
-- is somebody's decision and must survive.
UPDATE quests SET reward = 10, updated_at = '2026-10-06T00:00:00Z'
	WHERE kind = 'phase_completed' AND param = '1' AND reward = 50;
UPDATE quests SET reward = 19, updated_at = '2026-10-06T00:00:00Z'
	WHERE kind = 'phase_completed' AND param = '2' AND reward = 70;
UPDATE quests SET reward = 30, updated_at = '2026-10-06T00:00:00Z'
	WHERE kind = 'phase_completed' AND param = '3' AND reward = 90;
