-- Sample economy data for LOCAL development only.
--
-- Never run this against production: it invents users and writes ledger rows,
-- and the ledger is append-only — there is no undo. The guard below refuses to
-- run on anything but a database whose name ends in _local.
--
-- Amounts follow the real rules: a balance is only ever the sum of its ledger
-- rows, debits are negative, and every row carries a unique idempotency key.
DO $$
BEGIN
  IF current_database() NOT LIKE '%\_local' THEN
    RAISE EXCEPTION 'refusing to seed %: this script is for *_local databases only',
      current_database();
  END IF;
END $$;

BEGIN;

INSERT INTO users (id, name, created_at, timezone) VALUES
  ('usr_local_anya',  'Аня',    '2026-07-01T09:00:00Z', 'Europe/Belgrade'),
  ('usr_local_boris', 'Борис',  '2026-08-12T09:00:00Z', 'Europe/Moscow'),
  ('usr_local_vera',  'Вера',   '2026-09-20T09:00:00Z', 'Europe/Belgrade'),
  ('usr_local_grisha','Гриша',  '2026-09-25T09:00:00Z', 'Europe/Belgrade')
ON CONFLICT (id) DO NOTHING;

-- An email identity for one of them and a Telegram one for another, so the
-- wallet modal has both shapes to render.
INSERT INTO identities (id, user_id, provider, provider_uid, email, created_at) VALUES
  ('idn_local_anya', 'usr_local_anya', 'password', 'anya@example.com', 'anya@example.com', '2026-07-01T09:00:00Z')
ON CONFLICT (id) DO NOTHING;
INSERT INTO identities (id, user_id, provider, provider_uid, tg_username, created_at) VALUES
  ('idn_local_boris', 'usr_local_boris', 'telegram', '100500', 'boris_tg', '2026-08-12T09:00:00Z')
ON CONFLICT (id) DO NOTHING;

-- Daily activity: Аня has a long run of goal-meeting days, Борис a short one,
-- Вера only below-goal days, Гриша none at all.
INSERT INTO user_daily_activity (user_id, day, actions, goal)
SELECT 'usr_local_anya', to_char('2026-09-15'::date + i, 'YYYY-MM-DD'), 12 + i, 10
FROM generate_series(0, 13) AS i
ON CONFLICT (user_id, day) DO NOTHING;

INSERT INTO user_daily_activity (user_id, day, actions, goal)
SELECT 'usr_local_boris', to_char('2026-09-27'::date + i, 'YYYY-MM-DD'), 10, 10
FROM generate_series(0, 2) AS i
ON CONFLICT (user_id, day) DO NOTHING;

INSERT INTO user_daily_activity (user_id, day, actions, goal) VALUES
  ('usr_local_vera', '2026-09-28', 3, 10),
  ('usr_local_vera', '2026-09-29', 1, 10)
ON CONFLICT (user_id, day) DO NOTHING;

INSERT INTO user_answer_streak (user_id, current, best, updated_at) VALUES
  ('usr_local_anya',  7, 21, '2026-09-29T09:00:00Z'),
  ('usr_local_boris', 0,  9, '2026-09-29T09:00:00Z')
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO lesson_progress (user_id, lesson, status, started_at, completed_at)
SELECT 'usr_local_anya', lpad(i::text, 2, '0'), 'done',
       '2026-09-01T09:00:00Z', '2026-09-02T09:00:00Z'
FROM generate_series(0, 17) AS i
ON CONFLICT DO NOTHING;

INSERT INTO lesson_progress (user_id, lesson, status, started_at, completed_at)
SELECT 'usr_local_boris', lpad(i::text, 2, '0'), 'done',
       '2026-09-01T09:00:00Z', '2026-09-02T09:00:00Z'
FROM generate_series(0, 5) AS i
ON CONFLICT DO NOTHING;

-- Quest rewards, taken from the seeded catalogue so the amounts are real.
INSERT INTO quest_claims (quest_id, user_id, claimed_at)
SELECT q.id, 'usr_local_anya', '2026-09-20T10:00:00Z'
FROM quests q
WHERE (q.kind, q.target) IN (('lessons_completed', 5), ('lessons_completed', 10), ('vocab_learned', 30))
ON CONFLICT (quest_id, user_id) DO NOTHING;

INSERT INTO currency_ledger (user_id, amount, kind, ref, idempotency_key, comment, created_by, created_at)
SELECT 'usr_local_anya', q.reward, 'quest_reward', q.id::text,
       'quest:usr_local_anya:' || q.id, '', '', '2026-09-20T10:00:00Z'
FROM quests q
WHERE (q.kind, q.target) IN (('lessons_completed', 5), ('lessons_completed', 10), ('vocab_learned', 30))
ON CONFLICT (idempotency_key) DO NOTHING;

INSERT INTO quest_claims (quest_id, user_id, claimed_at)
SELECT q.id, 'usr_local_boris', '2026-09-28T10:00:00Z'
FROM quests q WHERE q.kind = 'lessons_completed' AND q.target = 5
ON CONFLICT (quest_id, user_id) DO NOTHING;

INSERT INTO currency_ledger (user_id, amount, kind, ref, idempotency_key, comment, created_by, created_at)
SELECT 'usr_local_boris', q.reward, 'quest_reward', q.id::text,
       'quest:usr_local_boris:' || q.id, '', '', '2026-09-28T10:00:00Z'
FROM quests q WHERE q.kind = 'lessons_completed' AND q.target = 5
ON CONFLICT (idempotency_key) DO NOTHING;

-- Fourteen days of streak drip for Аня, one coin a day at the seeded ladder's
-- first step.
INSERT INTO currency_ledger (user_id, amount, kind, ref, idempotency_key, comment, created_by, created_at)
SELECT 'usr_local_anya', 1, 'streak_daily',
       to_char('2026-09-15'::date + i, 'YYYY-MM-DD'),
       'streak_daily:usr_local_anya:' || to_char('2026-09-15'::date + i, 'YYYY-MM-DD'),
       '', '', to_char('2026-09-15'::date + i, 'YYYY-MM-DD') || 'T21:00:00Z'
FROM generate_series(0, 13) AS i
ON CONFLICT (idempotency_key) DO NOTHING;

-- One purchase: Аня buys a streak repair, so there is a debit to look at and
-- an entitlement the wallet can show.
INSERT INTO currency_ledger (user_id, amount, kind, ref, idempotency_key, comment, created_by, created_at)
SELECT 'usr_local_anya', -p.price, 'purchase', p.id::text,
       'purchase:usr_local_anya:' || p.id || ':local-1', '', '', '2026-09-29T12:00:00Z'
FROM products p WHERE p.kind = 'consumable' AND p.ref = 'streak_repair'
ON CONFLICT (idempotency_key) DO NOTHING;

INSERT INTO user_entitlements (user_id, kind, ref, qty, updated_at) VALUES
  ('usr_local_anya', 'consumable', 'streak_repair', 1, '2026-09-29T12:00:00Z')
ON CONFLICT (user_id, kind, ref) DO NOTHING;

-- One manual adjustment, so the Транзакции screen has an admin-attributed row
-- with a comment to display.
INSERT INTO currency_ledger (user_id, amount, kind, ref, idempotency_key, comment, created_by, created_at) VALUES
  ('usr_local_vera', 25, 'admin_adjustment', '', 'adj:usr_local_vera:local-1',
   'компенсация за баг с повторениями', 'admin', '2026-09-30T08:00:00Z')
ON CONFLICT (idempotency_key) DO NOTHING;

-- A promo code, one scoped to a product, plus a redemption to count.
INSERT INTO promo_codes (code, discount_percent, scope, valid_from, valid_to,
                         max_redemptions, max_per_user, active, created_at, updated_at) VALUES
  ('OSEN2026', 10, 'all', '', '', 0, 1, 1, '2026-09-01T00:00:00Z', '2026-09-01T00:00:00Z'),
  ('LVL4',     25, 'products', '2026-09-01', '2026-12-31', 50, 1, 1, '2026-09-01T00:00:00Z', '2026-09-01T00:00:00Z')
ON CONFLICT (code) DO NOTHING;

INSERT INTO promo_code_products (promo_code_id, product_id)
SELECT c.id, p.id FROM promo_codes c, products p
WHERE c.code = 'LVL4' AND p.kind = 'phase_unlock' AND p.ref = '4'
ON CONFLICT (promo_code_id, product_id) DO NOTHING;

COMMIT;

\echo '--- balances (balance must equal earned - spent) ---'
SELECT u.name,
       COALESCE(SUM(l.amount), 0) AS balance,
       COALESCE(SUM(CASE WHEN l.amount > 0 THEN l.amount ELSE 0 END), 0) AS earned,
       COALESCE(-SUM(CASE WHEN l.amount < 0 THEN l.amount ELSE 0 END), 0) AS spent
FROM users u LEFT JOIN currency_ledger l ON l.user_id = u.id
WHERE u.id LIKE 'usr_local_%'
GROUP BY u.name ORDER BY balance DESC;
