-- Level 2 becomes the first paid level, and the prices of every later level are
-- raised so that free play stops paying for them (see the comment on
-- SeedEconomyDefaults for the arithmetic). The course map now has six levels:
--
--   phase "1"   Level 1  free
--   phase "1b"  Level 2  40      (the lessons of Level 1 pay for it, no quests needed)
--   phase "2"   Level 3  150     (lessons and ordinary quests pay for it, no streak needed)
--   phase "3"   Level 4  260     (takes a streak, and a streak is enough)
--   phase "4"   Level 5  370     (was 230, more than a perfect learner has when they get there)
--   phase "5"   Level 6  480     (was 375, the same)
--
-- Every lesson of Levels 1 and 2 now pays, the short ones ("02b") included: 2
-- coins, 4 on every fourth lesson, and a completion bonus of 10 / 14, which makes
-- the levels worth 52 / 70.
--
-- Like 017-020, every UPDATE is conditional on the old seeded value, so a price,
-- title or reward changed in the admin panel is somebody's decision and survives.
-- Every INSERT is guarded twice: only on a database the seed has already run on
-- (a fresh database gets all of this from the seed itself) and only if the row is
-- missing.
--
-- Nobody loses a level they have started: a learner with progress in a level that
-- used to be free is given its unlock, so the new prices apply to levels they have
-- not begun.

-- Products. Levels 4 and 5 of the old numbering are now Levels 5 and 6.
UPDATE products SET price = 370, updated_at = '2026-10-22T00:00:00Z'
	WHERE kind = 'phase_unlock' AND ref = '4' AND price = 230;
UPDATE products SET title = 'Уровень 5 — Падежи в жизни',
		description = 'Открывает уровень 5: падежи, прилагательные, местоимения', updated_at = '2026-10-22T00:00:00Z'
	WHERE kind = 'phase_unlock' AND ref = '4' AND title = 'Уровень 4 — Падежи в жизни';
UPDATE products SET sort_order = 13, updated_at = '2026-10-22T00:00:00Z'
	WHERE kind = 'phase_unlock' AND ref = '4' AND sort_order = 10;
UPDATE products SET price = 480, updated_at = '2026-10-22T00:00:00Z'
	WHERE kind = 'phase_unlock' AND ref = '5' AND price = 375;
UPDATE products SET title = 'Уровень 6 — Жизнь на сербском',
		description = 'Открывает уровень 6: чувства, звонки, работа, документы', updated_at = '2026-10-22T00:00:00Z'
	WHERE kind = 'phase_unlock' AND ref = '5' AND title = 'Уровень 5 — Жизнь на сербском';
UPDATE products SET sort_order = 14, updated_at = '2026-10-22T00:00:00Z'
	WHERE kind = 'phase_unlock' AND ref = '5' AND sort_order = 11;
INSERT INTO products (kind, ref, title, description, price, discount_percent, discount_from, discount_to, grant_qty, active, sort_order, created_at, updated_at)
SELECT 'phase_unlock', '1b', 'Уровень 2 — Числа и магазин', 'Открывает уровень 2: семья, числа, цены, время, магазин', 40, 0, '', '', 1, 1, 9, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_seeded')
	  AND NOT EXISTS (SELECT 1 FROM products WHERE kind = 'phase_unlock' AND ref = '1b');
INSERT INTO products (kind, ref, title, description, price, discount_percent, discount_from, discount_to, grant_qty, active, sort_order, created_at, updated_at)
SELECT 'phase_unlock', '2', 'Уровень 3 — Быт', 'Открывает уровень 3: повседневность, квартира, еда, город', 150, 0, '', '', 1, 1, 10, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_seeded')
	  AND NOT EXISTS (SELECT 1 FROM products WHERE kind = 'phase_unlock' AND ref = '2');
INSERT INTO products (kind, ref, title, description, price, discount_percent, discount_from, discount_to, grant_qty, active, sort_order, created_at, updated_at)
SELECT 'phase_unlock', '3', 'Уровень 4 — Связная речь', 'Открывает уровень 4: прошлое, планы, транспорт, здоровье', 260, 0, '', '', 1, 1, 11, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_seeded')
	  AND NOT EXISTS (SELECT 1 FROM products WHERE kind = 'phase_unlock' AND ref = '3');

-- Level completion quests follow the new numbering, and Level 2 gets its own.
UPDATE quests SET title = 'Уровень 5 на 100%', description = 'Пройди все уроки пятого уровня', sort_order = 74, updated_at = '2026-10-22T00:00:00Z'
	WHERE kind = 'phase_completed' AND target = 100 AND param = '4' AND title = 'Уровень 4 на 100%' AND sort_order = 73;
UPDATE quests SET title = 'Уровень 4 на 100%', description = 'Пройди все уроки четвёртого уровня', sort_order = 73, updated_at = '2026-10-22T00:00:00Z'
	WHERE kind = 'phase_completed' AND target = 100 AND param = '3' AND title = 'Уровень 3 на 100%' AND sort_order = 72;
UPDATE quests SET title = 'Уровень 3 на 100%', description = 'Пройди все уроки третьего уровня', sort_order = 72, updated_at = '2026-10-22T00:00:00Z'
	WHERE kind = 'phase_completed' AND target = 100 AND param = '2' AND title = 'Уровень 2 на 100%' AND sort_order = 71;
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'phase_completed', 100, '1b', 'Уровень 2 на 100%', 'Пройди все уроки второго уровня', 14, 1, 71, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'phase_completed' AND target = 100 AND param = '1b');

-- Lesson rewards. The lessons that already pay keep their rows, with the reward
-- their new position in the level gives them, and the short lessons get a row each.
UPDATE quests SET reward = 4, updated_at = '2026-10-22T00:00:00Z'
	WHERE kind = 'lesson_completed' AND param = '02' AND reward = 2;
UPDATE quests SET reward = 2, updated_at = '2026-10-22T00:00:00Z'
	WHERE kind = 'lesson_completed' AND param = '03' AND reward = 4;
UPDATE quests SET reward = 4, updated_at = '2026-10-22T00:00:00Z'
	WHERE kind = 'lesson_completed' AND param = '10' AND reward = 2;
UPDATE quests SET reward = 2, updated_at = '2026-10-22T00:00:00Z'
	WHERE kind = 'lesson_completed' AND param = '15' AND reward = 4;
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '01b', 'Урок 01b', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '01b');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '02b', 'Урок 02b', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '02b');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '02c', 'Урок 02c', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '02c');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '02d', 'Урок 02d', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '02d');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '02e', 'Урок 02e', '', 4, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '02e');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '03b', 'Урок 03b', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '03b');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '04b', 'Урок 04b', '', 4, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '04b');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '04c', 'Урок 04c', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '04c');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '05b', 'Урок 05b', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '05b');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '05c', 'Урок 05c', '', 4, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '05c');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '06b', 'Урок 06b', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '06b');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '06c', 'Урок 06c', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '06c');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '07b', 'Урок 07b', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '07b');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '08b', 'Урок 08b', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '08b');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '08c', 'Урок 08c', '', 4, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '08c');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '08d', 'Урок 08d', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '08d');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '09b', 'Урок 09b', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '09b');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '10b', 'Урок 10b', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '10b');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '10c', 'Урок 10c', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '10c');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '10d', 'Урок 10d', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '10d');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '11b', 'Урок 11b', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '11b');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '12b', 'Урок 12b', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '12b');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '12c', 'Урок 12c', '', 4, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '12c');
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'lesson_completed', 1, '13b', 'Урок 13b', '', 2, 1, 100, '2026-10-22T00:00:00Z', '2026-10-22T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_lessons_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'lesson_completed' AND param = '13b');

-- Learners who have started a level that used to be free keep it.
INSERT INTO user_entitlements (user_id, kind, ref, qty, updated_at)
SELECT DISTINCT user_id, 'phase_unlock', '1b', 1, '2026-10-22T00:00:00Z' FROM lesson_progress
	WHERE lesson IN ('06', '06b', '06c', '07', '07b', '08', '08b', '08c', '08d', '09', '09b', '10', '10b', '10c', '10d', '11', '11b', '12', '12b', '12c', '13', '13b', '15')
	  AND EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_seeded')
ON CONFLICT (user_id, kind, ref) DO NOTHING;
INSERT INTO user_entitlements (user_id, kind, ref, qty, updated_at)
SELECT DISTINCT user_id, 'phase_unlock', '2', 1, '2026-10-22T00:00:00Z' FROM lesson_progress
	WHERE lesson IN ('16', '17', '18', '19', '20', '21', '22', '23', '24', '25', '26', '27', '28', '29')
	  AND EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_seeded')
ON CONFLICT (user_id, kind, ref) DO NOTHING;
INSERT INTO user_entitlements (user_id, kind, ref, qty, updated_at)
SELECT DISTINCT user_id, 'phase_unlock', '3', 1, '2026-10-22T00:00:00Z' FROM lesson_progress
	WHERE lesson IN ('30', '31', '32', '33', '34', '35', '36', '37', '38', '39', '40', '41')
	  AND EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_seeded')
ON CONFLICT (user_id, kind, ref) DO NOTHING;
