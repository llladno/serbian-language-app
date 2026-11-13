-- Return after each text of the «давно не заглядывали» nudge: share of sends followed by a review or an exercise
-- attempt within 24 h / 48 h. Variants n1..n5 carry a name or @nick, p1..p5 do not.
WITH act AS (
  SELECT user_id, reviewed_at::timestamptz AS t FROM reviews
  UNION ALL SELECT user_id, attempted_at::timestamptz FROM attempts
), s AS (
  SELECT id, user_id, variant, sent_at::timestamptz AS at FROM reminder_sends WHERE kind = 'inactive'
)
SELECT s.variant, count(*) AS sent,
  count(*) FILTER (WHERE EXISTS (SELECT 1 FROM act WHERE act.user_id = s.user_id AND act.t > s.at AND act.t <= s.at + interval '24 hours')) AS back_24h,
  count(*) FILTER (WHERE EXISTS (SELECT 1 FROM act WHERE act.user_id = s.user_id AND act.t > s.at AND act.t <= s.at + interval '48 hours')) AS back_48h
FROM s GROUP BY s.variant ORDER BY s.variant;
