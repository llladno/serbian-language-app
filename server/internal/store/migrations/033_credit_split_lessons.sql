-- On 2026-10-26 at about 19:20 UTC the old lessons 01-13 went live cut into short
-- ones: old "02" became "02", "02b", "02c", "02d", "02e" (00, 14 and 15 were not
-- cut). A learner who had finished an old lesson saw the reused id still done and
-- every new piece undone, though they had learned what the pieces teach. The owner
-- asked (2026-10-27) to count those pieces as done.
--
-- Only an old lesson finished before the cut counts: the last answer to an old
-- exercise came at 19:18:40 and the first to a new one at 19:28:35, with nothing
-- finished in between, so a lesson finished after 19:20 is the new short one and
-- gives nothing extra. A piece the learner has already finished is left alone, one
-- they have started becomes done and keeps its start time. The dates are the old
-- lesson's, since that is when the learner did the work.
--
-- Nothing else is touched: no attempts, steps, coins or unlocks (everyone with a
-- Level 2 piece to credit already has Level 2 open). Quests follow on their own,
-- as for any finished lesson: "Пройди N уроков" and the level bonuses become
-- claimable, and the learner collects them with «Забрать» (the owner's choice).
--
-- Every credited row is logged first, with what was there before,
-- and scripts/rollback_033_lesson_credit.sql undoes the whole thing from the log.

CREATE TABLE IF NOT EXISTS lesson_credit_log (
	user_id           TEXT NOT NULL,
	lesson            TEXT NOT NULL,
	from_lesson       TEXT NOT NULL,
	prev_status       TEXT,
	prev_started_at   TEXT,
	prev_completed_at TEXT,
	credited_at       TEXT NOT NULL,
	PRIMARY KEY (user_id, lesson)
);

WITH split(old_lesson, new_lesson) AS (VALUES
	('01', '01b'),
	('02', '02b'), ('02', '02c'), ('02', '02d'), ('02', '02e'),
	('03', '03b'),
	('04', '04b'), ('04', '04c'),
	('05', '05b'), ('05', '05c'),
	('06', '06b'), ('06', '06c'),
	('07', '07b'),
	('08', '08b'), ('08', '08c'), ('08', '08d'),
	('09', '09b'),
	('10', '10b'), ('10', '10c'), ('10', '10d'),
	('11', '11b'),
	('12', '12b'), ('12', '12c'),
	('13', '13b')
)
INSERT INTO lesson_credit_log (user_id, lesson, from_lesson, prev_status, prev_started_at, prev_completed_at, credited_at)
SELECT p.user_id, s.new_lesson, s.old_lesson, cur.status, cur.started_at, cur.completed_at, '2026-10-27T00:00:00Z'
FROM lesson_progress p
JOIN split s ON s.old_lesson = p.lesson
LEFT JOIN lesson_progress cur ON cur.user_id = p.user_id AND cur.lesson = s.new_lesson
WHERE p.status = 'done'
	AND p.completed_at < '2026-10-26T19:20:00Z'
	AND (cur.status IS NULL OR cur.status <> 'done')
	AND NOT EXISTS (SELECT 1 FROM lesson_credit_log l WHERE l.user_id = p.user_id AND l.lesson = s.new_lesson);

INSERT INTO lesson_progress (user_id, lesson, status, started_at, completed_at)
SELECT l.user_id, l.lesson, 'done', COALESCE(l.prev_started_at, p.started_at), p.completed_at
FROM lesson_credit_log l
JOIN lesson_progress p ON p.user_id = l.user_id AND p.lesson = l.from_lesson
WHERE l.credited_at = '2026-10-27T00:00:00Z'
ON CONFLICT (user_id, lesson) DO UPDATE SET
	status = 'done',
	completed_at = excluded.completed_at;
