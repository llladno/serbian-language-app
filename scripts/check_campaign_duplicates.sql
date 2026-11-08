-- Run on the prod database after each wave of the «return» campaign (read-only). Every query must return 0 rows.
-- Usage: psql "$PROD_DATABASE_URL" -f scripts/check_campaign_duplicates.sql   (never print the URL)

-- 1. A person written to more than once by the campaign (the table allows one row per person, so this only fires if that broke).
SELECT user_id, COUNT(*) AS messages
FROM campaign_sends
WHERE campaign = 'return' AND kind = 'sent' AND sent_at <> ''
GROUP BY user_id HAVING COUNT(*) > 1;

-- 2. A Telegram chat that got more than one campaign picture.
SELECT chat_id, COUNT(*) AS pictures
FROM bot_outbox
WHERE photo <> '' AND status IN ('pending', 'sending', 'sent')
GROUP BY chat_id HAVING COUNT(*) > 1;

-- 3. A person who got both a Telegram picture and a campaign e-mail (a person is in one channel only).
SELECT cs.user_id
FROM campaign_sends cs
WHERE cs.campaign = 'return' AND cs.kind = 'sent' AND cs.sent_at <> ''
GROUP BY cs.user_id HAVING COUNT(DISTINCT cs.channel) > 1;

-- 4. A person written to by the campaign and by another study message (reminder, «all done», «Повторить» nudge) within 24 hours.
--    Expected 0: the sweeps keep a day's gap between them. Rows here are for a look, not for panic.
SELECT cs.user_id, cs.sent_at AS campaign_at, o.created_at AS other_at, LEFT(o.text, 40) AS other_text
FROM campaign_sends cs
JOIN identities i ON i.user_id = cs.user_id AND i.provider = 'telegram'
JOIN bot_outbox o ON o.chat_id = i.provider_uid AND o.photo = '' AND o.status IN ('sent', 'sending')
WHERE cs.campaign = 'return' AND cs.kind = 'sent' AND cs.sent_at <> '' AND cs.channel = 'tg'
  AND o.created_at >= to_char((cs.sent_at::timestamptz - interval '24 hours') AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
  AND o.created_at <= to_char((cs.sent_at::timestamptz + interval '24 hours') AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
  AND (o.text LIKE 'Все %' OR o.text LIKE 'Слова заждались%' OR o.text LIKE 'День без практики%' OR o.text LIKE 'Напоминаем%'
       OR o.text LIKE 'Сербский не выучится%' OR o.text LIKE 'Вы давно%' OR o.text LIKE 'Пройденные уроки%' OR o.text LIKE '%без повторения%');
