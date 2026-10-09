# In-app currency (v1) — design

Owner doc for the whole feature. The admin-side screens are specified in
`ucimo-content-admin`'s `docs/superpowers/specs/2026-09-28-currency-admin-design.md`,
which reads and writes the tables defined here.

## Goal

Introduce an in-app currency that users earn by studying and spend on locked
course levels and a small catalogue of extras. Everything that has a number —
prices, rewards, discounts, streak thresholds — is editable from the admin
panel without a deploy.

Two income sources with deliberately different shapes:

- **Quests** (finite pool) — one-off achievements shown in the profile. Sized
  to fully cover Level 4.
- **Daily streak drip** (renewable) — small per-day payout for meeting a daily
  activity goal. The only long-horizon source, sized so Level 5 takes months of
  consistency (or a future top-up).

## Terminology

The word "задание" is overloaded in Russian. In this doc and in code:

- **quest** — an achievement in the profile ("пройти 10 уроков"), rewarded in currency.
- **exercise** — an item solved inside a lesson. Never rewarded directly.
- **level / phase** — an entry in `course.yaml`'s `phases[]`. This is the unit
  that gets unlocked and priced. The product word "курс" maps onto groups of
  phases and exists only in copy, never in code.

## Access map

| Product wording | `course.yaml` phases | Access |
|---|---|---|
| first course (free) | 1 (Первый контакт), 2 (Быт), 3 (Связная речь) | free |
| second course | 4 — Падежи в жизни | currency; costs what the free levels pay plus the Telegram quest (230) |
| third course | 5 — Жизнь на сербском | currency; costs Level 4's price plus what Level 4 pays and the "100 words" and "100 reviews" quests (375); needs nearly every quest or a long streak |

There is no "pay with money" unlock path. Money, when it arrives, buys currency
(see Out of scope), never a level directly.

**Content status (2026-10-08):** lessons `00`–`59` are authored, i.e. phases 1–4.
Phase 5 (`60`–`77`) is declared in `course.yaml` but has no content yet, so its
product is for sale before it can be read — the shop copy says "скоро".

## Data model

New tables, added as numbered migrations starting at `013`, following the
existing `{{.AutoID}}` macro and `?`-placeholder conventions (`migrate.go`,
`store.go` `rebind`). Timestamps are RFC3339 UTC strings like everywhere else in
this store; `day` columns are `YYYY-MM-DD` strings.

### Balance is never stored

The single most important invariant: **there is no balance column anywhere.**
A user's balance is always

```sql
SELECT COALESCE(SUM(amount), 0) FROM currency_ledger WHERE user_id = ?
```

This is what makes it safe for `ucimo-content-admin` to write directly into this
database (the pattern already used by `notifications.service.ts`): a manual
adjustment from the admin is a single `INSERT`, and there is no second copy of
the truth to drift.

### `currency_ledger`

```sql
CREATE TABLE IF NOT EXISTS currency_ledger (
	id              {{.AutoID}},
	user_id         TEXT NOT NULL,
	amount          BIGINT NOT NULL,          -- >0 credit, <0 debit, never 0
	kind            TEXT NOT NULL,            -- see below
	ref             TEXT NOT NULL DEFAULT '', -- quest id / product id / day
	idempotency_key TEXT NOT NULL,
	comment         TEXT NOT NULL DEFAULT '',
	created_by      TEXT NOT NULL DEFAULT '', -- '' = system, otherwise 'admin'
	created_at      TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS currency_ledger_idem ON currency_ledger (idempotency_key);
CREATE INDEX IF NOT EXISTS currency_ledger_user ON currency_ledger (user_id, id);
```

Append-only: rows are never updated or deleted. A correction is a new row.

`kind` is one of `quest_reward`, `streak_daily`, `purchase`, `admin_adjustment`,
`topup` (reserved, unused in v1).

`idempotency_key` is the concurrency defence and is built deterministically:

| kind | key |
|---|---|
| `quest_reward` | `quest:<quest_id>:<user_id>` |
| `streak_daily` | `streak_daily:<user_id>:<day>` |
| `purchase` | `purchase:<user_id>:<product_id>:<unix_nano>` |
| `admin_adjustment` | `adj:<user_id>:<unix_nano>` |

The unique index means a double claim or a double daily payout cannot happen
even under a race — the second `INSERT` fails and the transaction rolls back.
Purchases of consumables are intentionally repeatable, hence the nonce.

### `users.timezone`

```sql
ALTER TABLE users ADD COLUMN timezone TEXT NOT NULL DEFAULT 'Europe/Belgrade';
```

An IANA name. The client sends it via `PATCH /api/me`; unknown or unparseable
values fall back to `Europe/Belgrade`. Days already written to
`user_daily_activity` are never recomputed, so changing the timezone cannot
rewrite past streaks.

### `user_daily_activity`

```sql
CREATE TABLE IF NOT EXISTS user_daily_activity (
	user_id TEXT NOT NULL,
	day     TEXT NOT NULL,            -- YYYY-MM-DD in the user's timezone at write time
	actions INTEGER NOT NULL DEFAULT 0,
	goal    INTEGER NOT NULL,         -- threshold in force that day
	PRIMARY KEY (user_id, day)
);
```

Storing `goal` per row means raising the daily threshold later does not
retroactively break streaks people already earned.

### `streak_repairs`

```sql
CREATE TABLE IF NOT EXISTS streak_repairs (
	user_id    TEXT NOT NULL,
	day        TEXT NOT NULL,
	created_at TEXT NOT NULL,
	PRIMARY KEY (user_id, day)
);
```

A purchased repair marks one missed day as active without any activity.

### `user_answer_streak`

```sql
CREATE TABLE IF NOT EXISTS user_answer_streak (
	user_id    TEXT PRIMARY KEY,
	current    INTEGER NOT NULL DEFAULT 0,
	best       INTEGER NOT NULL DEFAULT 0,
	updated_at TEXT NOT NULL
);
```

The one materialised counter. Consecutive correct **first** answers; `best` is
what quests check, so a broken run never takes back an already-earned quest.

### `quests` and `quest_claims`

```sql
CREATE TABLE IF NOT EXISTS quests (
	id          {{.AutoID}},
	kind        TEXT NOT NULL,
	target      INTEGER NOT NULL DEFAULT 0,
	param       TEXT NOT NULL DEFAULT '',
	title       TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	reward      BIGINT NOT NULL,
	active      INTEGER NOT NULL DEFAULT 1,
	sort_order  INTEGER NOT NULL DEFAULT 0,
	created_at  TEXT NOT NULL,
	updated_at  TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS quest_claims (
	quest_id   BIGINT NOT NULL,
	user_id    TEXT NOT NULL,
	claimed_at TEXT NOT NULL,
	PRIMARY KEY (quest_id, user_id)
);
```

Note what is *not* here: per-user quest progress. Progress is computed on demand
from tables that already exist, so a quest created today immediately sees a
user's entire history — which is the agreed behaviour — and there is no
progress state to drift.

### `products`, promo codes, entitlements

```sql
CREATE TABLE IF NOT EXISTS products (
	id               {{.AutoID}},
	kind             TEXT NOT NULL,            -- phase_unlock | consumable | cosmetic
	ref              TEXT NOT NULL,            -- "4" | "streak_repair" | "palette:forest"
	title            TEXT NOT NULL,
	description      TEXT NOT NULL DEFAULT '',
	price            BIGINT NOT NULL,
	discount_percent INTEGER NOT NULL DEFAULT 0,
	discount_from    TEXT NOT NULL DEFAULT '',
	discount_to      TEXT NOT NULL DEFAULT '',
	grant_qty        INTEGER NOT NULL DEFAULT 1,
	active           INTEGER NOT NULL DEFAULT 1,
	sort_order       INTEGER NOT NULL DEFAULT 0,
	created_at       TEXT NOT NULL,
	updated_at       TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS products_kind_ref ON products (kind, ref);

CREATE TABLE IF NOT EXISTS promo_codes (
	id               {{.AutoID}},
	code             TEXT NOT NULL,            -- stored upper-case
	discount_percent INTEGER NOT NULL,
	scope            TEXT NOT NULL DEFAULT 'all', -- all | products
	valid_from       TEXT NOT NULL DEFAULT '',
	valid_to         TEXT NOT NULL DEFAULT '',
	max_redemptions  INTEGER NOT NULL DEFAULT 0,  -- 0 = unlimited
	max_per_user     INTEGER NOT NULL DEFAULT 1,
	active           INTEGER NOT NULL DEFAULT 1,
	created_at       TEXT NOT NULL,
	updated_at       TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS promo_codes_code ON promo_codes (code);

CREATE TABLE IF NOT EXISTS promo_code_products (
	promo_code_id BIGINT NOT NULL,
	product_id    BIGINT NOT NULL,
	PRIMARY KEY (promo_code_id, product_id)
);

CREATE TABLE IF NOT EXISTS promo_redemptions (
	id            {{.AutoID}},
	promo_code_id BIGINT NOT NULL,
	user_id       TEXT NOT NULL,
	product_id    BIGINT NOT NULL,
	ledger_id     BIGINT NOT NULL,
	created_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_entitlements (
	user_id    TEXT NOT NULL,
	kind       TEXT NOT NULL,
	ref        TEXT NOT NULL,
	qty        BIGINT NOT NULL DEFAULT 1,   -- 1 for permanent, remaining count for consumables
	updated_at TEXT NOT NULL,
	PRIMARY KEY (user_id, kind, ref)
);
```

### `economy_settings`

```sql
CREATE TABLE IF NOT EXISTS economy_settings (
	key        TEXT PRIMARY KEY,
	value      TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
```

Seeded by the migration with the starting values from "Economy" below:

| key | seed | meaning |
|---|---|---|
| `currency_name_one` / `_few` / `_many` | `монета` / `монеты` / `монет` | display name, editable |
| `daily_goal` | `10` | actions needed for a day to count |
| `streak_drip` | `[[1,1],[30,2],[100,3]]` | JSON ladder `[from_day, coins_per_day]` |
| `streak_repair_window_hours` | `48` | how long after a break a repair is allowed |
| `telegram_channel` | `""` | `@channel` for the subscription quest |

No `REFERENCES` clauses anywhere — same convention as
`notification_recipients.notification_id` and `bot_outbox.broadcast_id`
(migrations 010 and 012): the admin writes these tables directly and the
existing store deliberately avoids FK coupling across features.

## Earning

### Quest kinds

Eight kinds. Each is a pure "user → current value" function; a quest is complete
when value ≥ `target`.

| `kind` | value computed from | `param` |
|---|---|---|
| `lessons_completed` | `lesson_progress` where `status='completed'` | — |
| `vocab_learned` | `srs_cards` where `kind='vocab'` and `state <> 'new'` | — |
| `reviews_done` | `COUNT(*)` of `reviews` | — |
| `streak_days` | current streak (below) | — |
| `correct_in_row` | `user_answer_streak.best` | — |
| `phase_completed` | percent of that phase's lessons completed, 0..100; `target` is therefore always 100 | phase id, e.g. `2` |
| `telegram_subscribed` | Telegram `getChatMember` | channel, defaults to `economy_settings.telegram_channel` |
| `friends_invited` | `COUNT(*)` of users whose `referred_by` is this user and who have a Telegram identity or a confirmed email | — |

Adding a new kind is a code change plus a deploy, by design: every kind must be
verifiable server-side, otherwise currency is mintable from the client. The
admin panel creates *instances* of these kinds, never free-form conditions.

`friends_invited` ("Пригласить друга", 30 зёрнышек, one time) rests on two
`users` columns from migration 022. `referral_code` is minted lazily the first
time the learner opens the invite modal (8 characters from an alphabet without
look-alikes, unique index). `referred_by` is written once, first touch wins,
by `ApplyReferral`, which refuses an unknown code, the account's own code, a
code of someone the account itself invited, and any signup older than 7 days.
The link is `<APP_BASE_URL>/register?ref=CODE`, the site rather than the bot, so
it works from every messenger and does not depend on the Telegram API being
reachable. The browser keeps the code in `localStorage` (`ucimo_referral`) and
sends it with the registration and with a Telegram login; after any sign-in it
is also posted to `POST /api/me/referral/claim`, which covers an email confirmed
on another device and the bot login, and answers 200 for a stale code. A friend
only counts once confirmed, so throwaway addresses do not pay out.

`telegram_subscribed` needs a new `telegram.GetChatMember(botToken, chat, userID)`
in `internal/telegram`. Prod cannot reach `api.telegram.org` directly, but every
call in that package already goes through `apiBase`, which production overrides
via `TELEGRAM_API_BASE` (see `SetAPIBase`), so the proxy covers this too. The
quest is only completable by users with a linked Telegram identity; for everyone
else it reports progress 0 and the UI should say why. If both the quest's `param`
and `economy_settings.telegram_channel` are empty the quest is treated as
inactive and never listed — an unconfigured channel must not show up as a quest
nobody can finish. The check runs on demand
during `GET /api/me/quests` and is cached for 10 minutes per user to avoid
hammering the Bot API.

### Claiming

`POST /api/me/quests/{id}/claim` verifies completion server-side, then inserts
one `quest_reward` row and one `quest_claims` row in a single transaction. The
unique `idempotency_key` makes a double click a no-op.

Rewards are never credited automatically. This is what makes it safe to create a
quest that everyone has already satisfied: nothing is minted until each user
opens the app and claims.

### What counts as an action

- the **first** attempt at a given `exercise_id` (correct or not) — re-solving a
  finished lesson earns nothing;
- **every** SRS review — the scheduler decides when those are due, so they can't
  be farmed.

Requires an index on `attempts (user_id, exercise_id)` to make the "is this the
first attempt" check cheap.

### Daily activity and streak

On each counted action, inside the same transaction that records the attempt or
review:

1. resolve the user's `day` from `users.timezone`;
2. `UPSERT user_daily_activity` incrementing `actions`, storing the current
   `daily_goal` as `goal` on insert;
3. if `actions` has just reached `goal`, compute the streak length and insert a
   `streak_daily` row for `streak_drip`'s value at that length.

No cron job, no nightly batch. The payout happens in the request that earned it,
and `streak_daily:<user>:<day>` guarantees exactly one per day.

Streak length = walk back from today in the user's timezone while the day is
active, where active means `actions >= goal` or a row exists in
`streak_repairs`.

This **replaces** the existing `StreakDays()` (currently "days with ≥1 review"),
which is used by `/api/me` and the leaderboard rows. Historical
`user_daily_activity` is backfilled by a migration hook from `attempts` (first
attempt per exercise) and `reviews`, using `Europe/Belgrade` for the
reconstruction since no historical timezone is known. Some users' displayed
streak will move in either direction; the leaderboard's ordering is unaffected
(`ORDER BY lessons_done, cards_known, created_at`).

### Answer streak

`user_answer_streak.current` increments on a correct first attempt, resets to 0
on an incorrect first attempt, and is untouched by repeats and by SRS reviews.
`best` is `MAX(best, current)`. Backfilled by the same migration hook.

### Streak repair

`POST /api/me/streak/repair {day}` — allowed only when the day is within
`streak_repair_window_hours` of now, is currently inactive, and the user holds a
`consumable:streak_repair` entitlement. Consumes one unit and inserts into
`streak_repairs`.

Repair is deliberately two steps — buy the consumable, then spend it on a day —
so that stocking up in advance and repairing on the spot are the same mechanism.
The "стрик сгорел, вернуть за 25" button in the app is simply both calls back to
back.

## Spending

### Effective price

```
base      = product.price
promo     = applicable promo code's percent, or 0
sale      = product.discount_percent if now is inside [discount_from, discount_to], else 0
effective = round_up(base * (100 - max(promo, sale)) / 100)
```

Discounts do **not** stack — the better of the two wins, and the response says
which one applied so the UI can explain "промокод не выгоднее текущей акции".
Rounding is up, so a discount can never make something free by accident.

### Purchase

`POST /api/me/purchases {product_id, promo_code?}`, one transaction:

1. `SELECT id FROM users WHERE id = ? FOR UPDATE` — serialises concurrent
   purchases by the same user on Postgres; SQLite serialises writers anyway;
2. load the product, reject if inactive;
3. for `phase_unlock` and `cosmetic`, reject with 409 if already owned;
4. validate the promo code if given (active, in window, under both limits, in
   scope for this product);
5. compute the effective price; compare against `SUM(ledger)`; reject with 409
   `insufficient_funds` if short;
6. insert the `purchase` ledger row for `-effective`;
7. `UPSERT user_entitlements`: `qty = 1` for permanent kinds, `qty = qty + grant_qty`
   for consumables;
8. insert `promo_redemptions` if a code was used.

No refunds in v1. A mistaken purchase is undone by an `admin_adjustment` plus a
manual entitlement fix.

### Gating

`GET /api/course` gains, per phase: `locked`, `price`, `price_effective`,
`discount_percent`. `GET /api/lessons/{id}` returns **403** for a lesson inside a
locked phase — hiding it in the UI is not enough, the content must not be
reachable by direct request. Phases with no product row, or with a product
priced 0, are free.

## API surface

App endpoints (all behind `requireSession`; the Vue screens land in a later
iteration, this branch ships the backend):

| method | path | purpose |
|---|---|---|
| `GET` | `/api/me/wallet` | balance + currency display name |
| `GET` | `/api/me/quests` | active quests with progress, done/claimed flags |
| `POST` | `/api/me/quests/{id}/claim` | claim one quest |
| `GET` | `/api/me/transactions` | the user's own ledger, paginated |
| `GET` | `/api/shop` | active products with effective prices and ownership |
| `POST` | `/api/me/purchases` | buy |
| `POST` | `/api/me/streak/repair` | spend a repair on a given day |
| `PATCH` | `/api/me` | extended with `timezone` |

## Economy — starting numbers

All editable from the admin panel; these are seeds, not constants.

**Quests (one-off pool)**

| Quest | Reward |
|---|---|
| Подписка на Telegram-канал | 20 |
| Пригласить друга (разово, друг подтвердил аккаунт) | 30 |
| Пройти 5 / 10 / 20 / 30 уроков | 5 / 10 / 20 / 30 |
| Выучить 30 / 100 / 300 слов | 10 / 25 / 50 |
| Сделать 100 / 500 повторений | 10 / 30 |
| Серия 5 / 10 / 20 правильных подряд | 3 / 5 / 15 |
| Стрик 3 / 7 / 30 / 100 дней (разово) | 20 / 60 / 40 / 150 |
| Уровни 1 / 2 / 3 / 4: уроки + бонус за завершение | 50 / 70 / 90 / 110 |
| **Total** | **853** |

**Daily drip:** days 1–29 → 1, days 30–99 → 2, day 100+ → 3. 169 over 100
consecutive days, ~960 over a year.

**Prices**

| Product | Price | Rationale |
|---|---|---|
| Уровень 4 | 230 | 50 + 70 + 90 (levels 1–3) + 20 (Telegram) — everything the free course pays |
| Уровень 5 | 375 | 230 + 110 (Level 4 pays) + 25 ("100 слов") + 10 ("100 повторений"): total 605 earned out of 853 in the pool, so it takes nearly every quest, or months of streak drip |
| Восстановление стрика | 25 | 48-hour window |
| Палитра | 30 | per palette |

At launch the only live sinks are repairs and cosmetics, so balances will
accumulate for a while. That is expected and fine — users arrive at Level 4 with
savings already banked.

## Admin integration

`ucimo-content-admin` reads and writes these tables directly through
`ProdDbService`, exactly as `notifications.service.ts` already does. The screens
are specified in that repo's companion design doc.

**Grants are manual and must run as part of this deploy, not after something
breaks.** The admin's role is `ucimo_admin_ro`:

```sql
GRANT SELECT ON currency_ledger, quests, quest_claims, products, promo_codes,
                promo_code_products, promo_redemptions, user_entitlements,
                economy_settings, user_daily_activity, streak_repairs,
                user_answer_streak TO ucimo_admin_ro;

GRANT INSERT, UPDATE, DELETE ON quests, products, promo_codes,
                promo_code_products, economy_settings TO ucimo_admin_ro;

GRANT INSERT ON currency_ledger TO ucimo_admin_ro;

GRANT USAGE, SELECT ON SEQUENCE quests_id_seq, products_id_seq,
                promo_codes_id_seq, currency_ledger_id_seq TO ucimo_admin_ro;
```

These must be executed by the account owner — running DDL with the owner-role
password is blocked for Claude by the auto-mode classifier.

## Testing

Go store tests run on SQLite in-memory and against Postgres via `make test-pg`;
the `TRUNCATE` list in `testutil_test.go` needs every new table added, the same
maintenance the broadcast feature needed.

Cases that must exist:

- claiming twice credits once;
- two concurrent purchases cannot overdraw the balance;
- purchase with insufficient funds writes nothing at all;
- re-solving a finished lesson moves no counter;
- an SRS review always counts;
- an incorrect first answer resets `current` but not `best`;
- changing `users.timezone` does not change already-stamped days;
- a repair outside the 48-hour window is rejected;
- promo and sale together apply the better one, never both;
- the migration backfill reproduces a known streak from fixture data.

## Deploy order

1. deploy `serbian-app` — migrations apply on binary startup;
2. run the `GRANT` block above as the DB owner;
3. deploy `ucimo-content-admin`;
4. verify by hitting the live `admin.ucimo.ru` endpoint and reading the
   response — a green Dokploy deploy proves nothing about grants. A 503 from the
   admin means reading the container logs for the real Postgres error, since
   `ProdDbService` wraps every failure in the same "туннель не поднят" message.

## Out of scope for this branch

- Buying currency with real money. The ledger reserves `kind = 'topup'` and
  `created_by` so it can be added without reshaping anything.
- The Vue screens in `serbian-app/web` — profile wallet, quest list, shop,
  promo-code input. The backend ships first; the UI is a separate iteration.
- AI message packs. They fit the existing `consumable` product kind when they
  arrive; nothing about the catalogue needs to change.
- Refunds, and per-user personal discounts.
