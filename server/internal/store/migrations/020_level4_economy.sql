-- Level 4 became "Падежи в жизни" and the prices are now derived from what the
-- free levels pay (see the comment on SeedEconomyDefaults):
--   Level 4 = 50 + 70 + 90 + 20 (Telegram) = 230
--   Level 5 = 230 + 110 (what Level 4 pays) + 25 + 10 = 375
-- Streak quests change to 3 days -> 20 and 7 days -> 60, and Level 4 starts
-- paying for its lessons (4 / 8 coins, with a 22 coin bonus for finishing).
--
-- Like 017-019, every UPDATE is conditional on the old seeded value: a price,
-- title or reward changed in the admin panel is somebody's decision and must
-- survive. Every INSERT is guarded twice: only on a database the seed has
-- already run on (a fresh database gets all of this from the seed itself, and
-- inserting here as well would duplicate it), and only if the row is missing.

UPDATE products SET price = 230, updated_at = '2026-10-08T00:00:00Z'
	WHERE kind = 'phase_unlock' AND ref = '4' AND price = 500;
UPDATE products SET price = 375, updated_at = '2026-10-08T00:00:00Z'
	WHERE kind = 'phase_unlock' AND ref = '5' AND price = 1000;
UPDATE products SET title = 'Уровень 4 — Падежи в жизни',
		description = 'Открывает уровень 4: падежи, прилагательные, местоимения',
		updated_at = '2026-10-08T00:00:00Z'
	WHERE kind = 'phase_unlock' AND ref = '4' AND title = 'Уровень 4 — Мнения и жизнь';
UPDATE products SET title = 'Уровень 5 — Жизнь на сербском',
		description = 'Открывает уровень 5: чувства, звонки, работа, документы',
		updated_at = '2026-10-08T00:00:00Z'
	WHERE kind = 'phase_unlock' AND ref = '5' AND title = 'Уровень 5 — Уверенно';

UPDATE quests SET reward = 20, updated_at = '2026-10-08T00:00:00Z'
	WHERE kind = 'telegram_subscribed' AND reward = 15;
UPDATE quests SET reward = 60, updated_at = '2026-10-08T00:00:00Z'
	WHERE kind = 'streak_days' AND target = 7 AND reward = 10;

INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'streak_days', 3, '', 'Стрик 3 дня', 'Занимайся 3 дня подряд', 20, 1, 58, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'streak_days' AND target = 3 AND param = '');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'phase_completed', 100, '4', 'Уровень 4 на 100%', 'Пройди все уроки четвёртого уровня', 22, 1, 73, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'phase_completed' AND target = 100 AND param = '4');

-- Lesson rewards for Level 4: 4 coins, 8 on every fourth lesson. Only on a database
-- whose lesson rewards were seeded before Level 4 paid (marker economy_lessons_seeded).
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '42', 'Урок 42', '', 4, 1, 143, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '42');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '43', 'Урок 43', '', 4, 1, 144, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '43');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '44', 'Урок 44', '', 4, 1, 145, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '44');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '45', 'Урок 45', '', 8, 1, 146, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '45');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '46', 'Урок 46', '', 4, 1, 147, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '46');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '47', 'Урок 47', '', 4, 1, 148, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '47');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '48', 'Урок 48', '', 4, 1, 149, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '48');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '49', 'Урок 49', '', 8, 1, 150, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '49');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '50', 'Урок 50', '', 4, 1, 151, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '50');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '51', 'Урок 51', '', 4, 1, 152, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '51');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '52', 'Урок 52', '', 4, 1, 153, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '52');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '53', 'Урок 53', '', 8, 1, 154, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '53');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '54', 'Урок 54', '', 4, 1, 155, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '54');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '55', 'Урок 55', '', 4, 1, 156, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '55');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '56', 'Урок 56', '', 4, 1, 157, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '56');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '57', 'Урок 57', '', 8, 1, 158, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '57');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '58', 'Урок 58', '', 4, 1, 159, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '58');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '59', 'Урок 59', '', 4, 1, 160, '2026-10-08T00:00:00Z', '2026-10-08T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '59');
