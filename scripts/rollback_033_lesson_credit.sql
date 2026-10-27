-- Undoes migration 033 (lessons credited for the old lessons that were cut into
-- short ones): every row it wrote goes back to what lesson_credit_log says was
-- there before. Run in one transaction with a role that can write. The log stays,
-- so the run can be checked afterwards. A credited lesson the learner has since
-- gone through again goes back too: the log cannot tell the two apart.
-- Kept in step with TestMigration033CreditsSplitLessons, which runs this file.

DELETE FROM lesson_progress
WHERE EXISTS (SELECT 1 FROM lesson_credit_log l
	WHERE l.user_id = lesson_progress.user_id AND l.lesson = lesson_progress.lesson AND l.prev_status IS NULL);

UPDATE lesson_progress SET
	status = (SELECT l.prev_status FROM lesson_credit_log l
		WHERE l.user_id = lesson_progress.user_id AND l.lesson = lesson_progress.lesson),
	started_at = (SELECT l.prev_started_at FROM lesson_credit_log l
		WHERE l.user_id = lesson_progress.user_id AND l.lesson = lesson_progress.lesson),
	completed_at = (SELECT l.prev_completed_at FROM lesson_credit_log l
		WHERE l.user_id = lesson_progress.user_id AND l.lesson = lesson_progress.lesson)
WHERE EXISTS (SELECT 1 FROM lesson_credit_log l
	WHERE l.user_id = lesson_progress.user_id AND l.lesson = lesson_progress.lesson AND l.prev_status IS NOT NULL);
