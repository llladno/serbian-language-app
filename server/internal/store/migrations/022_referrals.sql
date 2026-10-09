-- Invite a friend: every account gets a short public code, and an account
-- created through somebody's link remembers who that was (first touch, never
-- overwritten). Counting is done on demand from these two columns, like every
-- other quest input, so there is no counter to drift.
ALTER TABLE users ADD COLUMN referral_code TEXT;
ALTER TABLE users ADD COLUMN referred_by TEXT;
CREATE UNIQUE INDEX users_referral_code_idx ON users (referral_code) WHERE referral_code IS NOT NULL;
CREATE INDEX users_referred_by_idx ON users (referred_by) WHERE referred_by IS NOT NULL;

-- The quest itself. Only on a database the seed has already run on (a fresh
-- database gets it from the seed, and inserting here as well would duplicate
-- it), and only if it is missing — same rule as 020.
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
SELECT 'friends_invited', 1, '', 'Пригласить друга', 'Друг пришёл по твоей ссылке', 30, 1, 12,
	'2026-10-09T00:00:00Z', '2026-10-09T00:00:00Z'
	WHERE EXISTS (SELECT 1 FROM economy_settings WHERE key = 'economy_seeded')
	  AND NOT EXISTS (SELECT 1 FROM quests WHERE kind = 'friends_invited');
