-- Должно вернуть 0 строк: каждому, кто начинал платный уровень, он остался открытым.
WITH started AS (
  SELECT user_id, lesson FROM lesson_progress
  UNION SELECT user_id, lesson FROM lesson_step_progress
  UNION SELECT user_id, lesson FROM attempts
), phased AS (
  SELECT DISTINCT user_id,
    CASE
      WHEN n BETWEEN 6 AND 13 OR n = 15 THEN '1b'
      WHEN n BETWEEN 16 AND 29 THEN '2'
      WHEN n BETWEEN 30 AND 41 THEN '3'
      WHEN n BETWEEN 42 AND 59 THEN '4'
      WHEN n BETWEEN 60 AND 77 THEN '5'
    END AS phase
  FROM (SELECT user_id, substring(lesson from '^[0-9]+')::int AS n FROM started WHERE lesson ~ '^[0-9]+') s
)
SELECT p.user_id, p.phase
FROM phased p
WHERE p.phase IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM user_entitlements e WHERE e.user_id = p.user_id AND e.kind = 'phase_unlock' AND e.ref = p.phase)
  AND NOT EXISTS (SELECT 1 FROM user_entitlements e WHERE e.user_id = p.user_id AND e.kind = 'all_access' AND e.expires_at > to_char(now() AT TIME ZONE 'utc', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'));
