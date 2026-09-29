# In-app currency (backend) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the whole currency engine in `serbian-app` — ledger, quests, streak, catalogue, purchases and the JSON API — so `ucimo-content-admin` has tables to manage and the Vue app has endpoints to call.

**Architecture:** A user's balance is never stored; it is always `SUM(amount)` over an append-only `currency_ledger` whose `idempotency_key` is uniquely indexed, which is what makes double-crediting impossible and lets the admin service write into the same database directly. Quest progress is likewise never stored — it is computed on demand from `lesson_progress`, `srs_cards` and `reviews`, so a quest created today sees a user's whole history. The only materialised state is per-day activity (needed because "a day" depends on the user's timezone at write time) and the consecutive-correct-answer counter.

**Tech Stack:** Go 1.25, stdlib `net/http`, `database/sql` over pgx (Postgres) and modernc SQLite, the repo's own numbered-migration runner (`server/internal/store/migrate.go`).

**Spec:** `docs/superpowers/specs/2026-09-28-currency-design.md`

## Global Constraints

- **Two backends, one code path.** Write queries with `?` placeholders; `database.Exec/Query/QueryRow` rebind to `$N` on Postgres. Never write `$1` by hand.
- **Migrations are the only schema source.** New files go in `server/internal/store/migrations/NNN_name.sql`, use the `{{.AutoID}}` macro for autoincrement PKs, and are idempotent (`CREATE TABLE IF NOT EXISTS`).
- **No `REFERENCES` clauses** in new tables — same convention as migrations 010 and 012, because `ucimo-content-admin` writes these tables directly.
- **Timestamps** are RFC3339 UTC strings (`now.UTC().Format(time.RFC3339)`); `day` columns are `YYYY-MM-DD` (`dateFmt`).
- **Amounts are integers.** `BIGINT` in SQL, `int64` in Go. No fractional currency anywhere.
- **Comments in code are English**; user-facing strings and content are Russian/Serbian.
- **Errors are wrapped**: `fmt.Errorf("...: %w", err)`.
- **Tests live next to the code**, table-driven where it fits, and run against SQLite in-memory by default and Postgres via `make test-pg`.
- **Balance is derived, never stored.** If any task tempts you to add a `balance` column, that is a bug in the task, not a shortcut.
- Starting economy numbers (seeded by migration, editable later from the admin): daily goal 10 actions; drip ladder `[[1,1],[30,2],[100,3]]`; repair window 48h; Level 4 = 500, Level 5 = 1000, streak repair = 25, palette = 30; quest pool totals 638.

## File Structure

**New files**

| File | Responsibility |
|---|---|
| `server/internal/store/migrations/013_currency.sql` | every new table, the `users.timezone` column, the `attempts` index, and the seed rows |
| `server/internal/store/currency.go` | ledger writes/reads, balance, economy settings accessor |
| `server/internal/store/currency_test.go` | tests for the above |
| `server/internal/store/activity.go` | daily activity, streak, streak repairs, answer streak |
| `server/internal/store/activity_test.go` | tests for the above |
| `server/internal/store/quests.go` | quest CRUD, progress computation, claiming |
| `server/internal/store/quests_test.go` | tests for the above |
| `server/internal/store/shop.go` | products, promo codes, entitlements, the purchase transaction |
| `server/internal/store/shop_test.go` | tests for the above |
| `server/internal/economy/economy.go` | pure math: effective price, drip ladder parsing and lookup |
| `server/internal/economy/economy_test.go` | tests for the above |
| `server/internal/api/economy.go` | HTTP handlers: wallet, quests, claim, transactions, shop, purchase, repair |
| `server/internal/api/economy_test.go` | handler tests |

**Modified files**

| File | Change |
|---|---|
| `server/internal/store/migration_hooks.go` | register `migrate013`, the backfill hook |
| `server/internal/store/store.go` | `AddAttempt` and `GradeCard` record activity; `StreakDays` reads the new source |
| `server/internal/store/store_test.go` | `TRUNCATE` list gains the new tables |
| `server/internal/telegram/telegram.go` | add `GetChatMember` |
| `server/internal/api/api.go` | route registration for the new endpoints, phase gating in `getCourse`/`getLesson` |
| `server/internal/api/me.go` | `patchMe` accepts `timezone` |
| `server/internal/api/dto.go` | new DTOs |

---

### Task 1: Schema and seeds

**Files:**
- Create: `server/internal/store/migrations/013_currency.sql`
- Test: `server/internal/store/currency_test.go`

**Interfaces:**
- Consumes: the migration runner in `migrate.go` (`{{.AutoID}}` macro, one transaction per version).
- Produces: tables `currency_ledger`, `user_daily_activity`, `streak_repairs`, `user_answer_streak`, `quests`, `quest_claims`, `products`, `promo_codes`, `promo_code_products`, `promo_redemptions`, `user_entitlements`, `economy_settings`; column `users.timezone`; index `attempts_user_exercise`.

- [ ] **Step 1: Write the failing test**

Add to a new `server/internal/store/currency_test.go`:

```go
package store

import "testing"

func TestMigration013CreatesSchemaAndSeeds(t *testing.T) {
	s := newStore(t)

	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM economy_settings`).Scan(&n); err != nil {
		t.Fatalf("economy_settings missing: %v", err)
	}
	if n == 0 {
		t.Fatal("economy_settings not seeded")
	}

	var goal string
	if err := s.db.QueryRow(`SELECT value FROM economy_settings WHERE key = ?`, "daily_goal").Scan(&goal); err != nil {
		t.Fatalf("daily_goal: %v", err)
	}
	if goal != "10" {
		t.Fatalf("daily_goal = %q, want \"10\"", goal)
	}

	// Every new table must exist and be empty.
	for _, tbl := range []string{
		"currency_ledger", "user_daily_activity", "streak_repairs", "user_answer_streak",
		"quests", "quest_claims", "products", "promo_codes", "promo_code_products",
		"promo_redemptions", "user_entitlements",
	} {
		if _, err := s.db.Exec(`SELECT 1 FROM ` + tbl + ` WHERE 1 = 0`); err != nil {
			t.Fatalf("table %s: %v", tbl, err)
		}
	}

	// users.timezone exists with the documented default.
	id, err := s.CreateUser("Тест")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	var tz string
	if err := s.db.QueryRow(`SELECT timezone FROM users WHERE id = ?`, id).Scan(&tz); err != nil {
		t.Fatalf("timezone column: %v", err)
	}
	if tz != "Europe/Belgrade" {
		t.Fatalf("timezone = %q, want Europe/Belgrade", tz)
	}
}
```

Check `store_test.go` for the exact name of the user-creating helper before writing this — if `CreateUser` has a different signature there, match it.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./server/internal/store/ -run TestMigration013 -v`
Expected: FAIL — `no such table: economy_settings`.

- [ ] **Step 3: Write the migration**

Create `server/internal/store/migrations/013_currency.sql`:

```sql
-- Migration 013: in-app currency. See
-- docs/superpowers/specs/2026-09-28-currency-design.md.
--
-- No REFERENCES anywhere, deliberately: ucimo-content-admin writes several of
-- these tables directly through its own connection (same convention as
-- notification_recipients in 010 and bot_outbox.broadcast_id in 012).
--
-- There is no balance column and there must never be one: a balance is always
-- SUM(currency_ledger.amount), which is what makes a second writer safe.

ALTER TABLE users ADD COLUMN timezone TEXT NOT NULL DEFAULT 'Europe/Belgrade';

-- Makes "is this the first attempt at this exercise" cheap; that check gates
-- every action counted toward the daily goal.
CREATE INDEX IF NOT EXISTS attempts_user_exercise ON attempts (user_id, exercise_id);

CREATE TABLE IF NOT EXISTS currency_ledger (
	id              {{.AutoID}},
	user_id         TEXT NOT NULL,
	amount          BIGINT NOT NULL,
	kind            TEXT NOT NULL,
	ref             TEXT NOT NULL DEFAULT '',
	idempotency_key TEXT NOT NULL,
	comment         TEXT NOT NULL DEFAULT '',
	created_by      TEXT NOT NULL DEFAULT '',
	created_at      TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS currency_ledger_idem ON currency_ledger (idempotency_key);
CREATE INDEX IF NOT EXISTS currency_ledger_user ON currency_ledger (user_id, id);

CREATE TABLE IF NOT EXISTS user_daily_activity (
	user_id TEXT NOT NULL,
	day     TEXT NOT NULL,
	actions INTEGER NOT NULL DEFAULT 0,
	goal    INTEGER NOT NULL,
	PRIMARY KEY (user_id, day)
);

CREATE TABLE IF NOT EXISTS streak_repairs (
	user_id    TEXT NOT NULL,
	day        TEXT NOT NULL,
	created_at TEXT NOT NULL,
	PRIMARY KEY (user_id, day)
);

CREATE TABLE IF NOT EXISTS user_answer_streak (
	user_id    TEXT PRIMARY KEY,
	current    INTEGER NOT NULL DEFAULT 0,
	best       INTEGER NOT NULL DEFAULT 0,
	updated_at TEXT NOT NULL
);

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

CREATE TABLE IF NOT EXISTS products (
	id               {{.AutoID}},
	kind             TEXT NOT NULL,
	ref              TEXT NOT NULL,
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
	code             TEXT NOT NULL,
	discount_percent INTEGER NOT NULL,
	scope            TEXT NOT NULL DEFAULT 'all',
	valid_from       TEXT NOT NULL DEFAULT '',
	valid_to         TEXT NOT NULL DEFAULT '',
	max_redemptions  INTEGER NOT NULL DEFAULT 0,
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
CREATE INDEX IF NOT EXISTS promo_redemptions_code ON promo_redemptions (promo_code_id);

CREATE TABLE IF NOT EXISTS user_entitlements (
	user_id    TEXT NOT NULL,
	kind       TEXT NOT NULL,
	ref        TEXT NOT NULL,
	qty        BIGINT NOT NULL DEFAULT 1,
	updated_at TEXT NOT NULL,
	PRIMARY KEY (user_id, kind, ref)
);

CREATE TABLE IF NOT EXISTS economy_settings (
	key        TEXT PRIMARY KEY,
	value      TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

INSERT INTO economy_settings (key, value, updated_at) VALUES
	('currency_name_one',          'монета',              '2026-09-29T00:00:00Z'),
	('currency_name_few',          'монеты',              '2026-09-29T00:00:00Z'),
	('currency_name_many',         'монет',               '2026-09-29T00:00:00Z'),
	('daily_goal',                 '10',                  '2026-09-29T00:00:00Z'),
	('streak_drip',                '[[1,1],[30,2],[100,3]]', '2026-09-29T00:00:00Z'),
	('streak_repair_window_hours', '48',                  '2026-09-29T00:00:00Z'),
	('telegram_channel',           '',                    '2026-09-29T00:00:00Z')
ON CONFLICT (key) DO NOTHING;
```

`ALTER TABLE ... ADD COLUMN` is not wrapped in `IF NOT EXISTS` because the runner only applies a version once — this is the same shape migration 012 used for `bot_outbox.broadcast_id`.

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./server/internal/store/ -run TestMigration013 -v`
Expected: PASS.

- [ ] **Step 5: Add the new tables to the Postgres truncate list**

In `server/internal/store/store_test.go`, extend the `TRUNCATE` statement in `newStore` (currently ending `... donations, broadcasts`) to also list: `currency_ledger, user_daily_activity, streak_repairs, user_answer_streak, quests, quest_claims, products, promo_codes, promo_code_products, promo_redemptions, user_entitlements`.

Leave `economy_settings` **out** — it is seeded by the migration and every test depends on the seeds being present.

- [ ] **Step 6: Run the full store suite on both backends**

Run: `go test ./server/internal/store/`
Run: `make test-pg` (needs a local `srpski_test`; skip only if Postgres is genuinely unavailable and say so)
Expected: PASS on both.

- [ ] **Step 7: Commit**

```bash
git add server/internal/store/migrations/013_currency.sql server/internal/store/currency_test.go server/internal/store/store_test.go
git commit -m "Add migration 013: currency schema and economy settings seed"
```

---

### Task 2: Ledger and balance

**Files:**
- Create: `server/internal/store/currency.go`
- Modify: `server/internal/store/currency_test.go`

**Interfaces:**
- Consumes: `database`/`dbtx` from `store.go`, the tables from Task 1.
- Produces:
  - `type LedgerEntry struct { ID int64; UserID string; Amount int64; Kind, Ref, IdempotencyKey, Comment, CreatedBy string; CreatedAt time.Time }`
  - `func (s *Store) AddLedgerEntry(e LedgerEntry, now time.Time) (int64, error)` — returns the new row id, or `ErrDuplicateEntry`
  - `func addLedgerEntryTx(tx *dbtx, e LedgerEntry, now time.Time) (int64, error)`
  - `func (s *Store) Balance(userID string) (int64, error)`
  - `func balanceTx(tx *dbtx, userID string) (int64, error)`
  - `func (s *Store) ListLedger(userID string, limit, offset int) ([]LedgerEntry, error)`
  - `var ErrDuplicateEntry = errors.New("duplicate ledger entry")`

- [ ] **Step 1: Write the failing tests**

Append to `server/internal/store/currency_test.go`:

```go
func TestLedgerBalanceAndIdempotency(t *testing.T) {
	s := newStore(t)
	id, err := s.CreateUser("Ледж")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

	if bal, err := s.Balance(id); err != nil || bal != 0 {
		t.Fatalf("empty balance = %d, %v; want 0, nil", bal, err)
	}

	if _, err := s.AddLedgerEntry(LedgerEntry{
		UserID: id, Amount: 30, Kind: "quest_reward", Ref: "7",
		IdempotencyKey: "quest:7:" + id,
	}, now); err != nil {
		t.Fatalf("first credit: %v", err)
	}

	// Same key again must be rejected and must not change the balance.
	if _, err := s.AddLedgerEntry(LedgerEntry{
		UserID: id, Amount: 30, Kind: "quest_reward", Ref: "7",
		IdempotencyKey: "quest:7:" + id,
	}, now); !errors.Is(err, ErrDuplicateEntry) {
		t.Fatalf("second credit err = %v; want ErrDuplicateEntry", err)
	}

	if _, err := s.AddLedgerEntry(LedgerEntry{
		UserID: id, Amount: -25, Kind: "purchase", Ref: "1",
		IdempotencyKey: "purchase:" + id + ":1:1",
	}, now); err != nil {
		t.Fatalf("debit: %v", err)
	}

	bal, err := s.Balance(id)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	if bal != 5 {
		t.Fatalf("balance = %d, want 5", bal)
	}

	rows, err := s.ListLedger(id, 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0].Kind != "purchase" {
		t.Fatalf("rows[0].Kind = %q, want purchase (newest first)", rows[0].Kind)
	}
}

func TestLedgerRejectsZeroAmount(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Ноль")
	_, err := s.AddLedgerEntry(LedgerEntry{
		UserID: id, Amount: 0, Kind: "quest_reward", IdempotencyKey: "zero",
	}, time.Now())
	if err == nil {
		t.Fatal("zero-amount entry accepted; want error")
	}
}
```

Add `"errors"` and `"time"` to the file's imports.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./server/internal/store/ -run TestLedger -v`
Expected: FAIL — `undefined: LedgerEntry`.

- [ ] **Step 3: Implement**

Create `server/internal/store/currency.go`:

```go
package store

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrDuplicateEntry is returned when a ledger row with the same
// idempotency_key already exists. Callers treat it as "already done", not as
// a failure: it is exactly what stops a double-clicked claim or a retried
// request from crediting twice.
var ErrDuplicateEntry = errors.New("duplicate ledger entry")

// LedgerEntry is one row of currency_ledger. The ledger is append-only — a
// correction is a new row, never an update.
type LedgerEntry struct {
	ID             int64
	UserID         string
	Amount         int64 // >0 credit, <0 debit, never 0
	Kind           string
	Ref            string
	IdempotencyKey string
	Comment        string
	CreatedBy      string // "" = system, "admin" = manual adjustment
	CreatedAt      time.Time
}

// AddLedgerEntry appends one entry and returns its id.
func (s *Store) AddLedgerEntry(e LedgerEntry, now time.Time) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin ledger entry: %w", err)
	}
	defer tx.Rollback()
	id, err := addLedgerEntryTx(tx, e, now)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit ledger entry: %w", err)
	}
	return id, nil
}

// addLedgerEntryTx is the transaction-scoped form, used by claim and purchase
// so the credit and its side effects commit together.
func addLedgerEntryTx(tx *dbtx, e LedgerEntry, now time.Time) (int64, error) {
	if e.UserID == "" {
		return 0, errors.New("ledger entry: empty user id")
	}
	if e.Amount == 0 {
		return 0, errors.New("ledger entry: zero amount")
	}
	if e.IdempotencyKey == "" {
		return 0, errors.New("ledger entry: empty idempotency key")
	}
	var id int64
	err := tx.QueryRow(`INSERT INTO currency_ledger
		(user_id, amount, kind, ref, idempotency_key, comment, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`,
		e.UserID, e.Amount, e.Kind, e.Ref, e.IdempotencyKey, e.Comment, e.CreatedBy,
		now.UTC().Format(time.RFC3339)).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicateEntry
		}
		return 0, fmt.Errorf("insert ledger entry: %w", err)
	}
	return id, nil
}

// isUniqueViolation reports whether err is a unique-index conflict on either
// backend. Both drivers only expose this in the message text, so match on it
// rather than pulling in driver-specific error types.
func isUniqueViolation(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique index")
}

// Balance is always computed, never stored. See the design doc.
func (s *Store) Balance(userID string) (int64, error) {
	var bal int64
	err := s.db.QueryRow(`SELECT COALESCE(SUM(amount), 0) FROM currency_ledger WHERE user_id = ?`,
		userID).Scan(&bal)
	if err != nil {
		return 0, fmt.Errorf("balance: %w", err)
	}
	return bal, nil
}

// balanceTx is the transaction-scoped form; a purchase reads the balance
// inside the same transaction that debits it.
func balanceTx(tx *dbtx, userID string) (int64, error) {
	var bal int64
	err := tx.QueryRow(`SELECT COALESCE(SUM(amount), 0) FROM currency_ledger WHERE user_id = ?`,
		userID).Scan(&bal)
	if err != nil {
		return 0, fmt.Errorf("balance: %w", err)
	}
	return bal, nil
}

// ListLedger returns one user's entries, newest first.
func (s *Store) ListLedger(userID string, limit, offset int) ([]LedgerEntry, error) {
	rows, err := s.db.Query(`SELECT id, user_id, amount, kind, ref, idempotency_key,
		comment, created_by, created_at
		FROM currency_ledger WHERE user_id = ? ORDER BY id DESC LIMIT ? OFFSET ?`,
		userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list ledger: %w", err)
	}
	defer rows.Close()
	var out []LedgerEntry
	for rows.Next() {
		var e LedgerEntry
		var created string
		if err := rows.Scan(&e.ID, &e.UserID, &e.Amount, &e.Kind, &e.Ref,
			&e.IdempotencyKey, &e.Comment, &e.CreatedBy, &created); err != nil {
			return nil, fmt.Errorf("scan ledger entry: %w", err)
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, e)
	}
	return out, rows.Err()
}
```

If `RETURNING id` turns out not to work on the SQLite driver in use, fall back to `Exec` + `LastInsertId()` on SQLite and keep `RETURNING` on Postgres, branching on `tx.pg` — check `donations.go` and `bot_outbox.go` first to see whether the repo already has a helper for this.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./server/internal/store/ -run TestLedger -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add server/internal/store/currency.go server/internal/store/currency_test.go
git commit -m "Add append-only currency ledger with derived balance"
```

---

### Task 3: Economy settings accessor and pure price/drip math

**Files:**
- Create: `server/internal/economy/economy.go`, `server/internal/economy/economy_test.go`
- Modify: `server/internal/store/currency.go`, `server/internal/store/currency_test.go`

**Interfaces:**
- Consumes: `economy_settings` from Task 1.
- Produces:
  - `economy.Ladder` = `[][2]int`, `func economy.ParseLadder(s string) (Ladder, error)`, `func (l Ladder) DripFor(streakDays int) int64`
  - `func economy.EffectivePrice(base int64, salePercent, promoPercent int) (price int64, applied string)` where `applied` is `"sale"`, `"promo"` or `""`
  - `type store.Settings struct { CurrencyNameOne, CurrencyNameFew, CurrencyNameMany string; DailyGoal int; Drip economy.Ladder; RepairWindowHours int; TelegramChannel string }`
  - `func (s *Store) EconomySettings() (Settings, error)`

- [ ] **Step 1: Write the failing tests for the pure package**

Create `server/internal/economy/economy_test.go`:

```go
package economy

import "testing"

func TestParseLadderAndDrip(t *testing.T) {
	l, err := ParseLadder(`[[1,1],[30,2],[100,3]]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	cases := []struct {
		streak int
		want   int64
	}{
		{0, 0}, {1, 1}, {29, 1}, {30, 2}, {99, 2}, {100, 3}, {500, 3},
	}
	for _, c := range cases {
		if got := l.DripFor(c.streak); got != c.want {
			t.Errorf("DripFor(%d) = %d, want %d", c.streak, got, c.want)
		}
	}
}

func TestParseLadderRejectsGarbage(t *testing.T) {
	for _, in := range []string{``, `[]`, `[[0,1]]`, `[[2,1],[1,1]]`, `nope`} {
		if _, err := ParseLadder(in); err == nil {
			t.Errorf("ParseLadder(%q) succeeded; want error", in)
		}
	}
}

func TestEffectivePriceTakesTheBetterDiscount(t *testing.T) {
	cases := []struct {
		name               string
		base               int64
		sale, promo        int
		wantPrice          int64
		wantApplied        string
	}{
		{"no discounts", 500, 0, 0, 500, ""},
		{"sale only", 500, 20, 0, 400, "sale"},
		{"promo only", 500, 0, 10, 450, "promo"},
		{"sale wins", 500, 20, 10, 400, "sale"},
		{"promo wins", 500, 10, 20, 400, "promo"},
		{"tie prefers sale", 500, 20, 20, 400, "sale"},
		{"rounds up, never free by accident", 25, 99, 0, 1, "sale"},
		{"full discount still costs nothing less than zero", 25, 100, 0, 0, "sale"},
	}
	for _, c := range cases {
		price, applied := EffectivePrice(c.base, c.sale, c.promo)
		if price != c.wantPrice || applied != c.wantApplied {
			t.Errorf("%s: EffectivePrice(%d,%d,%d) = %d,%q; want %d,%q",
				c.name, c.base, c.sale, c.promo, price, applied, c.wantPrice, c.wantApplied)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./server/internal/economy/ -v`
Expected: FAIL — package does not compile, `undefined: ParseLadder`.

- [ ] **Step 3: Implement the pure package**

Create `server/internal/economy/economy.go`:

```go
// Package economy holds the currency's pure arithmetic — the streak drip
// ladder and discount maths. It has no database and no HTTP, so the rules
// that decide how much a user pays or earns are testable on their own.
package economy

import (
	"encoding/json"
	"fmt"
)

// Ladder maps a streak length to a daily payout: each entry is
// [from_day, coins_per_day], ascending, starting at day 1. Stored in
// economy_settings.streak_drip as JSON so it is editable from the admin.
type Ladder [][2]int

// ParseLadder reads the JSON form and validates it: non-empty, first entry
// starts at day 1, days strictly ascending, payouts non-negative.
func ParseLadder(s string) (Ladder, error) {
	var l Ladder
	if err := json.Unmarshal([]byte(s), &l); err != nil {
		return nil, fmt.Errorf("parse drip ladder: %w", err)
	}
	if len(l) == 0 {
		return nil, fmt.Errorf("parse drip ladder: empty")
	}
	if l[0][0] != 1 {
		return nil, fmt.Errorf("parse drip ladder: first step must start at day 1, got %d", l[0][0])
	}
	for i, step := range l {
		if step[1] < 0 {
			return nil, fmt.Errorf("parse drip ladder: step %d has negative payout", i)
		}
		if i > 0 && step[0] <= l[i-1][0] {
			return nil, fmt.Errorf("parse drip ladder: step %d does not increase (%d after %d)",
				i, step[0], l[i-1][0])
		}
	}
	return l, nil
}

// DripFor returns the payout for a day that brought the streak to streakDays.
// A zero-length streak pays nothing.
func (l Ladder) DripFor(streakDays int) int64 {
	if streakDays <= 0 {
		return 0
	}
	var out int64
	for _, step := range l {
		if streakDays >= step[0] {
			out = int64(step[1])
		}
	}
	return out
}

// EffectivePrice applies the better of a sale and a promo code — they never
// stack — and rounds up, so a discount can never make something free by
// accident. applied names the winner ("sale", "promo" or "" for neither) so
// the UI can explain why a promo code changed nothing.
func EffectivePrice(base int64, salePercent, promoPercent int) (int64, string) {
	applied := ""
	best := 0
	if salePercent > 0 {
		best, applied = salePercent, "sale"
	}
	if promoPercent > best {
		best, applied = promoPercent, "promo"
	}
	if best <= 0 {
		return base, ""
	}
	if best >= 100 {
		return 0, applied
	}
	remaining := base * int64(100-best)
	price := remaining / 100
	if remaining%100 != 0 {
		price++
	}
	return price, applied
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./server/internal/economy/ -v`
Expected: PASS.

- [ ] **Step 5: Write the failing test for the settings accessor**

Append to `server/internal/store/currency_test.go`:

```go
func TestEconomySettings(t *testing.T) {
	s := newStore(t)
	set, err := s.EconomySettings()
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if set.DailyGoal != 10 {
		t.Fatalf("DailyGoal = %d, want 10", set.DailyGoal)
	}
	if set.RepairWindowHours != 48 {
		t.Fatalf("RepairWindowHours = %d, want 48", set.RepairWindowHours)
	}
	if got := set.Drip.DripFor(30); got != 2 {
		t.Fatalf("Drip.DripFor(30) = %d, want 2", got)
	}
	if set.CurrencyNameMany != "монет" {
		t.Fatalf("CurrencyNameMany = %q, want монет", set.CurrencyNameMany)
	}
}

func TestEconomySettingsFallsBackOnGarbage(t *testing.T) {
	s := newStore(t)
	if _, err := s.db.Exec(`UPDATE economy_settings SET value = ? WHERE key = ?`,
		"not-a-number", "daily_goal"); err != nil {
		t.Fatalf("corrupt daily_goal: %v", err)
	}
	set, err := s.EconomySettings()
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if set.DailyGoal != defaultDailyGoal {
		t.Fatalf("DailyGoal = %d, want fallback %d", set.DailyGoal, defaultDailyGoal)
	}
}
```

A bad value in one setting must never take the app down — the admin can save anything into that table.

- [ ] **Step 6: Run to verify it fails**

Run: `go test ./server/internal/store/ -run TestEconomySettings -v`
Expected: FAIL — `undefined: (*Store).EconomySettings`.

- [ ] **Step 7: Implement the accessor**

Append to `server/internal/store/currency.go` (and add the `economy` and `strconv` imports):

```go
// Defaults used when a setting is missing or unparseable. The admin panel can
// write anything into economy_settings, so every read falls back rather than
// failing the request.
const (
	defaultDailyGoal         = 10
	defaultRepairWindowHours = 48
	defaultDrip              = `[[1,1],[30,2],[100,3]]`
)

// Settings is the parsed economy_settings table.
type Settings struct {
	CurrencyNameOne   string
	CurrencyNameFew   string
	CurrencyNameMany  string
	DailyGoal         int
	Drip              economy.Ladder
	RepairWindowHours int
	TelegramChannel   string
}

// EconomySettings reads and parses every setting in one query.
func (s *Store) EconomySettings() (Settings, error) {
	rows, err := s.db.Query(`SELECT key, value FROM economy_settings`)
	if err != nil {
		return Settings{}, fmt.Errorf("economy settings: %w", err)
	}
	defer rows.Close()
	raw := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return Settings{}, fmt.Errorf("scan economy setting: %w", err)
		}
		raw[k] = v
	}
	if err := rows.Err(); err != nil {
		return Settings{}, fmt.Errorf("iterate economy settings: %w", err)
	}

	atoi := func(key string, def int) int {
		n, err := strconv.Atoi(raw[key])
		if err != nil || n <= 0 {
			return def
		}
		return n
	}
	ladder, err := economy.ParseLadder(raw["streak_drip"])
	if err != nil {
		ladder, _ = economy.ParseLadder(defaultDrip)
	}
	return Settings{
		CurrencyNameOne:   raw["currency_name_one"],
		CurrencyNameFew:   raw["currency_name_few"],
		CurrencyNameMany:  raw["currency_name_many"],
		DailyGoal:         atoi("daily_goal", defaultDailyGoal),
		Drip:              ladder,
		RepairWindowHours: atoi("streak_repair_window_hours", defaultRepairWindowHours),
		TelegramChannel:   raw["telegram_channel"],
	}, nil
}
```

- [ ] **Step 8: Run to verify it passes**

Run: `go test ./server/internal/store/ -run TestEconomySettings -v`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add server/internal/economy/ server/internal/store/currency.go server/internal/store/currency_test.go
git commit -m "Add economy settings accessor and pure drip/discount maths"
```

---

### Task 4: Per-user timezone

**Files:**
- Modify: `server/internal/store/currency.go`, `server/internal/store/currency_test.go`, `server/internal/api/me.go`, `server/internal/api/me_test.go`

**Interfaces:**
- Consumes: `users.timezone` from Task 1.
- Produces:
  - `func (s *Store) UserLocation(userID string) *time.Location` — never nil, falls back to `Europe/Belgrade` then UTC
  - `func (s *Store) SetUserTimezone(userID, tz string) error` — rejects names Go cannot load
  - `PATCH /api/me` accepts an optional `timezone` field

- [ ] **Step 1: Write the failing tests**

Append to `server/internal/store/currency_test.go`:

```go
func TestUserTimezone(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Пояс")

	if loc := s.UserLocation(id); loc.String() != "Europe/Belgrade" {
		t.Fatalf("default location = %q, want Europe/Belgrade", loc)
	}

	if err := s.SetUserTimezone(id, "Europe/Moscow"); err != nil {
		t.Fatalf("set timezone: %v", err)
	}
	if loc := s.UserLocation(id); loc.String() != "Europe/Moscow" {
		t.Fatalf("location = %q, want Europe/Moscow", loc)
	}

	if err := s.SetUserTimezone(id, "Mars/Olympus"); err == nil {
		t.Fatal("bogus timezone accepted; want error")
	}
	if loc := s.UserLocation(id); loc.String() != "Europe/Moscow" {
		t.Fatalf("location changed after a rejected write: %q", loc)
	}
}

func TestUserLocationNeverNil(t *testing.T) {
	s := newStore(t)
	if loc := s.UserLocation("no-such-user"); loc == nil {
		t.Fatal("UserLocation returned nil")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./server/internal/store/ -run TestUser -v`
Expected: FAIL — `undefined: (*Store).UserLocation`.

- [ ] **Step 3: Implement the store side**

Append to `server/internal/store/currency.go`:

```go
// fallbackTZ is the course's home timezone: the audience is Serbia and
// Russia, so it is never more than a couple of hours off, and it is what the
// backfill uses for history where no real timezone is known.
const fallbackTZ = "Europe/Belgrade"

// UserLocation returns the user's timezone, never nil. An unknown user, an
// unreadable row or a name the runtime cannot load all degrade to
// Europe/Belgrade, and finally to UTC if even that is unavailable (a Go build
// without tzdata).
func (s *Store) UserLocation(userID string) *time.Location {
	var name string
	if err := s.db.QueryRow(`SELECT timezone FROM users WHERE id = ?`, userID).Scan(&name); err != nil {
		name = fallbackTZ
	}
	if loc, err := time.LoadLocation(name); err == nil {
		return loc
	}
	if loc, err := time.LoadLocation(fallbackTZ); err == nil {
		return loc
	}
	return time.UTC
}

// SetUserTimezone stores an IANA timezone name after checking the runtime can
// load it. Days already written to user_daily_activity are never recomputed,
// so changing this cannot rewrite past streaks.
func (s *Store) SetUserTimezone(userID, tz string) error {
	if _, err := time.LoadLocation(tz); err != nil {
		return fmt.Errorf("unknown timezone %q: %w", tz, err)
	}
	if _, err := s.db.Exec(`UPDATE users SET timezone = ? WHERE id = ?`, tz, userID); err != nil {
		return fmt.Errorf("set timezone: %w", err)
	}
	return nil
}
```

The server image must ship tzdata. Check the `Dockerfile`: if the final stage is `scratch` or bare `alpine`, either add the `tzdata` package or import `_ "time/tzdata"` in `server/main.go`. Verify this during this task rather than discovering it in production, where every user would silently fall back to UTC.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./server/internal/store/ -run TestUser -v`
Expected: PASS.

- [ ] **Step 5: Write the failing handler test**

Append to `server/internal/api/me_test.go`, following the request-building helpers already used in that file:

```go
func TestPatchMeAcceptsTimezone(t *testing.T) {
	env := newTestEnv(t) // match the helper name used elsewhere in this file
	user := env.newSessionUser(t, "Пояс")

	res := env.do(t, user, "PATCH", "/api/me", `{"name":"Пояс","timezone":"Europe/Moscow"}`)
	if res.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", res.Code, res.Body.String())
	}
	if loc := env.Store.UserLocation(user.ID); loc.String() != "Europe/Moscow" {
		t.Fatalf("timezone not stored: %q", loc)
	}
}

func TestPatchMeIgnoresBogusTimezone(t *testing.T) {
	env := newTestEnv(t)
	user := env.newSessionUser(t, "Пояс")

	res := env.do(t, user, "PATCH", "/api/me", `{"name":"Пояс","timezone":"Mars/Olympus"}`)
	if res.Code != 200 {
		t.Fatalf("status = %d, want 200 (a bad timezone must not fail the rename)", res.Code)
	}
	if loc := env.Store.UserLocation(user.ID); loc.String() != "Europe/Belgrade" {
		t.Fatalf("timezone = %q, want the default to survive", loc)
	}
}
```

Read the top of `me_test.go` first and use whatever setup helpers it already defines instead of inventing `newTestEnv`/`do` if the names differ.

- [ ] **Step 6: Run to verify it fails**

Run: `go test ./server/internal/api/ -run TestPatchMe -v`
Expected: FAIL — timezone not stored.

- [ ] **Step 7: Extend patchMe**

In `server/internal/api/me.go`, widen the anonymous request struct and store the timezone after the rename succeeds:

```go
	var req struct {
		Name     string `json:"name"`
		Timezone string `json:"timezone"`
	}
```

and after the existing `RenameUser` block, before `summaryFor`:

```go
	// A bad timezone is not worth failing the whole request over — the client
	// derives it from the browser and we simply keep the previous value.
	if req.Timezone != "" {
		if err := h.Store.SetUserTimezone(ac.UserID, req.Timezone); err != nil {
			log.Printf("patch me: timezone %q: %v", req.Timezone, err)
		}
	}
```

- [ ] **Step 8: Run to verify it passes**

Run: `go test ./server/internal/api/ -run TestPatchMe -v`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add server/internal/store/currency.go server/internal/store/currency_test.go server/internal/api/me.go server/internal/api/me_test.go
git commit -m "Store a per-user timezone and accept it from PATCH /api/me"
```

---

### Task 5: Daily activity, streak and the daily drip

**Files:**
- Create: `server/internal/store/activity.go`, `server/internal/store/activity_test.go`
- Modify: `server/internal/store/store.go` (delete the old `StreakDays`, wire `AddAttempt` and `GradeCard`), `server/internal/store/currency.go` (add the `querier` interface)

**Interfaces:**
- Consumes: `LedgerEntry`, `addLedgerEntryTx`, `ErrDuplicateEntry`, `Settings`, `economy.Ladder` (Tasks 2–3); `user_daily_activity`, `streak_repairs` (Task 1).
- Produces:
  - `type querier interface { Exec(string, ...any) (sql.Result, error); Query(string, ...any) (*sql.Rows, error); QueryRow(string, ...any) *sql.Row }`
  - `func economySettings(q querier) (Settings, error)` — `(*Store).EconomySettings` becomes a one-line wrapper
  - `func recordActionTx(tx *dbtx, userID string, now time.Time) error`
  - `func streakDaysTx(q querier, userID string, loc *time.Location, now time.Time) (int, error)`
  - `func (u *UserStore) StreakDays(now time.Time) (int, error)` — same signature as the function it replaces
  - `func (u *UserStore) RepairStreak(day string, now time.Time) error`

- [ ] **Step 1: Write the failing tests**

Create `server/internal/store/activity_test.go`:

```go
package store

import (
	"testing"
	"time"
)

// belgrade is the location the seeds default to; tests that don't care about
// timezones use it so day boundaries are predictable.
func belgrade(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Belgrade")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	return loc
}

// answer solves one exercise for the first time, which is the only kind of
// attempt that counts as an action.
func answer(t *testing.T, u *UserStore, exID string, correct bool, at time.Time) {
	t.Helper()
	if err := u.AddAttempt(Attempt{
		ExerciseID: exID, Lesson: "01", Block: "a", Answer: "x", Correct: correct,
	}, at); err != nil {
		t.Fatalf("attempt %s: %v", exID, err)
	}
}

func TestDayCountsOnlyFirstAttemptsAndAllReviews(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Актив")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)

	answer(t, u, "01.1", true, at)
	answer(t, u, "01.1", true, at) // same exercise again: must not count
	answer(t, u, "01.2", false, at)

	// Every SRS review counts, including repeated reviews of the same card:
	// the scheduler decides when a card is due, so they cannot be farmed.
	if err := u.EnsureCards([]CardSeed{{CardID: "vocab:x", Kind: "vocab", RefID: "x"}}); err != nil {
		t.Fatalf("seed card: %v", err)
	}
	if _, err := u.GradeCard("vocab:x", srs.Good, at); err != nil {
		t.Fatalf("grade: %v", err)
	}
	if _, err := u.GradeCard("vocab:x", srs.Good, at); err != nil {
		t.Fatalf("grade again: %v", err)
	}

	var actions, goal int
	err := s.db.QueryRow(`SELECT actions, goal FROM user_daily_activity WHERE user_id = ? AND day = ?`,
		id, "2026-09-29").Scan(&actions, &goal)
	if err != nil {
		t.Fatalf("activity row: %v", err)
	}
	if actions != 4 {
		t.Fatalf("actions = %d, want 4 (two first attempts + two reviews; the repeated attempt must not count)", actions)
	}
	if goal != 10 {
		t.Fatalf("goal = %d, want the seeded 10", goal)
	}
}

func TestDailyDripPaysOnceWhenTheGoalIsMet(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Капля")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)

	for i := 0; i < 9; i++ {
		answer(t, u, "01."+itoa(i), true, at)
	}
	if bal, _ := s.Balance(id); bal != 0 {
		t.Fatalf("balance = %d before the goal is met, want 0", bal)
	}

	answer(t, u, "01.9", true, at) // tenth action meets the goal
	if bal, _ := s.Balance(id); bal != 1 {
		t.Fatalf("balance = %d after meeting the goal, want 1", bal)
	}

	// More actions the same day must not pay again.
	answer(t, u, "01.10", true, at)
	answer(t, u, "01.11", true, at)
	if bal, _ := s.Balance(id); bal != 1 {
		t.Fatalf("balance = %d after extra actions, want 1", bal)
	}
}

func TestStreakCountsConsecutiveDaysAndRepairs(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Стрик")
	u := s.User(id)

	// Three consecutive days that met the goal, then a gap, then today.
	days := []string{"2026-09-25", "2026-09-26", "2026-09-27", "2026-09-29"}
	for _, d := range days {
		if _, err := s.db.Exec(`INSERT INTO user_daily_activity (user_id, day, actions, goal) VALUES (?, ?, ?, ?)`,
			id, d, 10, 10); err != nil {
			t.Fatalf("seed %s: %v", d, err)
		}
	}
	now := time.Date(2026, 9, 29, 20, 0, 0, 0, loc)

	got, err := u.StreakDays(now)
	if err != nil {
		t.Fatalf("streak: %v", err)
	}
	if got != 1 {
		t.Fatalf("streak = %d, want 1 (the 28th is missing)", got)
	}

	if err := u.RepairStreak("2026-09-28", now); err != nil {
		t.Fatalf("repair: %v", err)
	}
	got, err = u.StreakDays(now)
	if err != nil {
		t.Fatalf("streak after repair: %v", err)
	}
	if got != 5 {
		t.Fatalf("streak after repair = %d, want 5", got)
	}
}

func TestRepairOutsideTheWindowIsRejected(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Поздно")
	u := s.User(id)
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, loc)

	// The 25th ended four days ago; the window is 48 hours.
	if err := u.RepairStreak("2026-09-25", now); err == nil {
		t.Fatal("a repair four days late was accepted; want an error")
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM streak_repairs WHERE user_id = ?`, id).Scan(&n); err != nil {
		t.Fatalf("count repairs: %v", err)
	}
	if n != 0 {
		t.Fatalf("%d repair rows written by a rejected repair, want 0", n)
	}

	// A future day is not repairable either.
	if err := u.RepairStreak("2026-09-30", now); err == nil {
		t.Fatal("a future day was accepted; want an error")
	}
}

func TestDayUnderGoalDoesNotExtendTheStreak(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Недобор")
	u := s.User(id)
	if _, err := s.db.Exec(`INSERT INTO user_daily_activity (user_id, day, actions, goal) VALUES (?, ?, ?, ?)`,
		id, "2026-09-29", 9, 10); err != nil {
		t.Fatalf("seed: %v", err)
	}
	now := time.Date(2026, 9, 29, 20, 0, 0, 0, loc)
	if got, _ := u.StreakDays(now); got != 0 {
		t.Fatalf("streak = %d, want 0 (9 of 10 actions)", got)
	}
}

func TestDayIsStampedInTheUsersTimezone(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Владивосток")
	if err := s.SetUserTimezone(id, "Asia/Vladivostok"); err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	u := s.User(id)

	// 23:30 UTC on the 29th is already 09:30 on the 30th in Vladivostok.
	at := time.Date(2026, 9, 29, 23, 30, 0, 0, time.UTC)
	answer(t, u, "01.1", true, at)

	var day string
	if err := s.db.QueryRow(`SELECT day FROM user_daily_activity WHERE user_id = ?`, id).Scan(&day); err != nil {
		t.Fatalf("activity row: %v", err)
	}
	if day != "2026-09-30" {
		t.Fatalf("day = %q, want 2026-09-30", day)
	}
}

func TestChangingTimezoneDoesNotRewritePastDays(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Переезд")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 23, 30, 0, 0, time.UTC)
	answer(t, u, "01.1", true, at)

	if err := s.SetUserTimezone(id, "Asia/Vladivostok"); err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	var day string
	if err := s.db.QueryRow(`SELECT day FROM user_daily_activity WHERE user_id = ?`, id).Scan(&day); err != nil {
		t.Fatalf("activity row: %v", err)
	}
	if day != "2026-09-30" {
		t.Fatalf("day = %q — the already-stamped row moved when the timezone changed", day)
	}
	_ = u
}
```

Check `CardSeed`'s real field names in `store.go` (around `EnsureCards`) and the `srs` import path before writing this — the seed literal above uses plausible names, not verified ones.

`itoa` is `strconv.Itoa`; import `strconv`, `srs` and add `func itoa(n int) string { return strconv.Itoa(n) }` at the bottom of the test file, or just call `strconv.Itoa` inline.

Note the last test's expectation: 23:30 UTC on the 29th is stamped as the 30th **because the default is Europe/Belgrade** (01:30 on the 30th), and it must stay the 30th after moving to Vladivostok — the point is that the stored row does not move, not which day it is.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./server/internal/store/ -run 'TestDay|TestDaily|TestStreak|TestChanging' -v`
Expected: FAIL — `undefined: (*UserStore).RepairStreak`, and the activity table stays empty.

- [ ] **Step 3: Add the `querier` interface**

In `server/internal/store/currency.go`, add (and change `EconomySettings` to delegate):

```go
// querier is the read/write surface shared by *database and *dbtx, so helpers
// can run either standalone or inside someone else's transaction.
type querier interface {
	Exec(q string, a ...any) (sql.Result, error)
	Query(q string, a ...any) (*sql.Rows, error)
	QueryRow(q string, a ...any) *sql.Row
}

// EconomySettings reads and parses every setting in one query.
func (s *Store) EconomySettings() (Settings, error) { return economySettings(s.db) }
```

Rename the existing method body to `func economySettings(q querier) (Settings, error)` and replace its `s.db.Query` with `q.Query`. Add `"database/sql"` to the imports.

- [ ] **Step 4: Implement activity, streak and drip**

Create `server/internal/store/activity.go`:

```go
package store

import (
	"errors"
	"fmt"
	"time"
)

// recordActionTx counts one action toward the user's daily goal and, if this
// action is the one that met the goal, pays the streak drip — in the caller's
// transaction, so the action and its payout commit together.
//
// "One action" is deliberately narrow: the first attempt at a given exercise
// (see AddAttempt) and every SRS review (see GradeCard). Re-solving a finished
// lesson is worth nothing, which is what stops the daily goal, the answer
// streak and every counting quest from being farmed on lesson 00.
func recordActionTx(tx *dbtx, userID string, now time.Time) error {
	set, err := economySettings(tx)
	if err != nil {
		return err
	}
	loc, err := userLocationTx(tx, userID)
	if err != nil {
		return err
	}
	day := now.In(loc).Format(dateFmt)

	var actions, goal int
	err = tx.QueryRow(`INSERT INTO user_daily_activity (user_id, day, actions, goal)
		VALUES (?, ?, 1, ?)
		ON CONFLICT (user_id, day) DO UPDATE SET actions = user_daily_activity.actions + 1
		RETURNING actions, goal`, userID, day, set.DailyGoal).Scan(&actions, &goal)
	if err != nil {
		return fmt.Errorf("record action: %w", err)
	}

	// Pay exactly on the crossing, not on every action after it.
	if actions != goal {
		return nil
	}
	streak, err := streakDaysTx(tx, userID, loc, now)
	if err != nil {
		return err
	}
	drip := set.Drip.DripFor(streak)
	if drip <= 0 {
		return nil
	}
	_, err = addLedgerEntryTx(tx, LedgerEntry{
		UserID:         userID,
		Amount:         drip,
		Kind:           "streak_daily",
		Ref:            day,
		IdempotencyKey: "streak_daily:" + userID + ":" + day,
	}, now)
	// A lowered goal can make a later action cross again on a day already
	// paid; the unique key catches it and there is nothing to do.
	if err != nil && !errors.Is(err, ErrDuplicateEntry) {
		return err
	}
	return nil
}

// userLocationTx is UserLocation inside a transaction. Same fallbacks.
func userLocationTx(q querier, userID string) (*time.Location, error) {
	var name string
	if err := q.QueryRow(`SELECT timezone FROM users WHERE id = ?`, userID).Scan(&name); err != nil {
		name = fallbackTZ
	}
	if loc, err := time.LoadLocation(name); err == nil {
		return loc, nil
	}
	if loc, err := time.LoadLocation(fallbackTZ); err == nil {
		return loc, nil
	}
	return time.UTC, nil
}

// streakDaysTx counts consecutive active days ending today, in the user's own
// timezone. A day is active when it met that day's goal or when a purchased
// repair covers it.
func streakDaysTx(q querier, userID string, loc *time.Location, now time.Time) (int, error) {
	active := map[string]bool{}

	rows, err := q.Query(`SELECT day FROM user_daily_activity WHERE user_id = ? AND actions >= goal`, userID)
	if err != nil {
		return 0, fmt.Errorf("streak days: %w", err)
	}
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan active day: %w", err)
		}
		active[d] = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate active days: %w", err)
	}
	rows.Close()

	repairs, err := q.Query(`SELECT day FROM streak_repairs WHERE user_id = ?`, userID)
	if err != nil {
		return 0, fmt.Errorf("streak repairs: %w", err)
	}
	defer repairs.Close()
	for repairs.Next() {
		var d string
		if err := repairs.Scan(&d); err != nil {
			return 0, fmt.Errorf("scan repair day: %w", err)
		}
		active[d] = true
	}
	if err := repairs.Err(); err != nil {
		return 0, fmt.Errorf("iterate repair days: %w", err)
	}

	streak := 0
	for cur := now.In(loc); active[cur.Format(dateFmt)]; cur = cur.AddDate(0, 0, -1) {
		streak++
	}
	return streak, nil
}

// StreakDays counts consecutive days ending today that met the daily goal.
//
// This replaces the pre-currency definition ("any day with at least one SRS
// review"): lessons now count too, a day needs real work rather than one tap,
// and the boundary follows the user's timezone instead of UTC.
func (u *UserStore) StreakDays(now time.Time) (int, error) {
	loc, err := userLocationTx(u.db, u.user)
	if err != nil {
		return 0, err
	}
	return streakDaysTx(u.db, u.user, loc, now)
}

// RepairStreak marks one missed day as active. The caller is responsible for
// having consumed the entitlement that paid for it (see SpendStreakRepair);
// this function only enforces that the day is in the past, is not already
// active, and is inside the configured window.
func (u *UserStore) RepairStreak(day string, now time.Time) error {
	set, err := economySettings(u.db)
	if err != nil {
		return err
	}
	loc, err := userLocationTx(u.db, u.user)
	if err != nil {
		return err
	}
	parsed, err := time.ParseInLocation(dateFmt, day, loc)
	if err != nil {
		return fmt.Errorf("repair streak: bad day %q: %w", day, err)
	}
	// The window is measured from the end of the missed day.
	deadline := parsed.AddDate(0, 0, 1).Add(time.Duration(set.RepairWindowHours) * time.Hour)
	if now.After(deadline) {
		return fmt.Errorf("repair streak: %s is outside the %dh window", day, set.RepairWindowHours)
	}
	if !now.In(loc).After(parsed) {
		return fmt.Errorf("repair streak: %s is not in the past", day)
	}
	var actions, goal int
	err = u.db.QueryRow(`SELECT actions, goal FROM user_daily_activity WHERE user_id = ? AND day = ?`,
		u.user, day).Scan(&actions, &goal)
	if err == nil && actions >= goal {
		return fmt.Errorf("repair streak: %s is already active", day)
	}
	if _, err := u.db.Exec(`INSERT INTO streak_repairs (user_id, day, created_at) VALUES (?, ?, ?)
		ON CONFLICT (user_id, day) DO NOTHING`,
		u.user, day, now.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("repair streak: %w", err)
	}
	return nil
}
```

If the SQLite driver rejects `ON CONFLICT ... DO UPDATE ... RETURNING`, do the upsert as `INSERT ... ON CONFLICT DO UPDATE` followed by a plain `SELECT actions, goal` in the same transaction — correctness is unaffected because the transaction holds the row.

- [ ] **Step 5: Delete the old StreakDays and wire the write paths**

In `server/internal/store/store.go`:

1. Delete the whole `StreakDays` function (the one that reads `SELECT DISTINCT substr(reviewed_at,1,10) ... FROM reviews`). Its replacement lives in `activity.go` with the same signature, so `enrichProgress` and every other caller keep compiling untouched.

2. Replace `AddAttempt` with a transactional version that only counts first attempts:

```go
// AddAttempt records one exercise answer. The first attempt at a given
// exercise also counts as an action toward the daily goal and moves the
// answer streak; repeats record the answer and nothing else.
func (u *UserStore) AddAttempt(a Attempt, now time.Time) error {
	tx, err := u.db.Begin()
	if err != nil {
		return fmt.Errorf("add attempt: %w", err)
	}
	defer tx.Rollback()

	var prior int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM attempts WHERE user_id = ? AND exercise_id = ?`,
		u.user, a.ExerciseID).Scan(&prior); err != nil {
		return fmt.Errorf("add attempt: count prior: %w", err)
	}

	correct := 0
	if a.Correct {
		correct = 1
	}
	if _, err := tx.Exec(`INSERT INTO attempts (user_id, exercise_id, lesson, block, answer, correct, attempted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		u.user, a.ExerciseID, a.Lesson, a.Block, a.Answer, correct,
		now.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("add attempt: %w", err)
	}

	if prior == 0 {
		if err := recordActionTx(tx, u.user, now); err != nil {
			return err
		}
		if err := recordAnswerTx(tx, u.user, a.Correct, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}
```

3. In `GradeCard`, right after the `INSERT INTO reviews` statement and before the commit, add:

```go
	if err := recordActionTx(tx, u.user, now); err != nil {
		return srs.Card{}, err
	}
```

`recordAnswerTx` arrives in Task 6; until then, stub it in `activity.go` as `func recordAnswerTx(tx *dbtx, userID string, correct bool, now time.Time) error { return nil }` so this task compiles and its tests run, and replace the body in Task 6.

- [ ] **Step 6: Run to verify the tests pass**

Run: `go test ./server/internal/store/ -run 'TestDay|TestDaily|TestStreak|TestChanging' -v`
Expected: PASS.

- [ ] **Step 7: Run the whole suite — the streak change has blast radius**

Run: `go test ./server/...`
Expected: PASS. Any existing test that asserts the old streak definition (search for `StreakDays` and `streak_days` in `server/` and in `web/src`) must be updated to the new one, not worked around — and note in the commit message which assertions changed.

- [ ] **Step 8: Commit**

```bash
git add server/internal/store/activity.go server/internal/store/activity_test.go server/internal/store/store.go server/internal/store/currency.go
git commit -m "Count daily activity per user timezone, pay the streak drip, redefine StreakDays"
```

---

### Task 6: Answer streak

**Files:**
- Modify: `server/internal/store/activity.go`, `server/internal/store/activity_test.go`

**Interfaces:**
- Consumes: `user_answer_streak` (Task 1), the `recordAnswerTx` call site added in Task 5.
- Produces:
  - `func recordAnswerTx(tx *dbtx, userID string, correct bool, now time.Time) error` — real body
  - `func (u *UserStore) AnswerStreak() (current, best int, err error)`

- [ ] **Step 1: Write the failing tests**

Append to `server/internal/store/activity_test.go`:

```go
func TestAnswerStreakTracksCurrentAndBest(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Серия")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)

	answer(t, u, "01.1", true, at)
	answer(t, u, "01.2", true, at)
	answer(t, u, "01.3", true, at)

	cur, best, err := u.AnswerStreak()
	if err != nil {
		t.Fatalf("answer streak: %v", err)
	}
	if cur != 3 || best != 3 {
		t.Fatalf("current, best = %d, %d; want 3, 3", cur, best)
	}

	answer(t, u, "01.4", false, at) // breaks the run
	cur, best, _ = u.AnswerStreak()
	if cur != 0 || best != 3 {
		t.Fatalf("after a wrong answer: current, best = %d, %d; want 0, 3", cur, best)
	}

	answer(t, u, "01.5", true, at)
	cur, best, _ = u.AnswerStreak()
	if cur != 1 || best != 3 {
		t.Fatalf("after restarting: current, best = %d, %d; want 1, 3", cur, best)
	}
}

func TestAnswerStreakIgnoresRepeatsAndReviews(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Повтор")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)

	answer(t, u, "01.1", true, at)
	answer(t, u, "01.1", true, at) // repeat
	answer(t, u, "01.1", true, at) // repeat

	cur, best, _ := u.AnswerStreak()
	if cur != 1 || best != 1 {
		t.Fatalf("repeats moved the streak: current, best = %d, %d; want 1, 1", cur, best)
	}
}
```

The second test is the anti-farm guarantee in test form: without it, re-solving one easy exercise twenty times would complete the "20 правильных подряд" quest.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./server/internal/store/ -run TestAnswerStreak -v`
Expected: FAIL — `undefined: (*UserStore).AnswerStreak`, and current stays 0 because `recordAnswerTx` is still the stub.

- [ ] **Step 3: Replace the stub**

In `server/internal/store/activity.go`, swap the stub for:

```go
// recordAnswerTx moves the consecutive-correct-answer counter. Only called for
// first attempts (see AddAttempt): repeats and SRS reviews never touch it, so
// the "N правильных подряд" quests cannot be farmed by re-solving lesson 00.
func recordAnswerTx(tx *dbtx, userID string, correct bool, now time.Time) error {
	ts := now.UTC().Format(time.RFC3339)
	if !correct {
		if _, err := tx.Exec(`INSERT INTO user_answer_streak (user_id, current, best, updated_at)
			VALUES (?, 0, 0, ?)
			ON CONFLICT (user_id) DO UPDATE SET current = 0, updated_at = ?`,
			userID, ts, ts); err != nil {
			return fmt.Errorf("reset answer streak: %w", err)
		}
		return nil
	}
	if _, err := tx.Exec(`INSERT INTO user_answer_streak (user_id, current, best, updated_at)
		VALUES (?, 1, 1, ?)
		ON CONFLICT (user_id) DO UPDATE SET
			current = user_answer_streak.current + 1,
			best = CASE WHEN user_answer_streak.current + 1 > user_answer_streak.best
			            THEN user_answer_streak.current + 1
			            ELSE user_answer_streak.best END,
			updated_at = ?`,
		userID, ts, ts); err != nil {
		return fmt.Errorf("advance answer streak: %w", err)
	}
	return nil
}

// AnswerStreak returns the user's current and best runs of consecutive
// correct first answers. Quests check best, so a broken run never takes back
// a quest that was already earned.
func (u *UserStore) AnswerStreak() (int, int, error) {
	var cur, best int
	err := u.db.QueryRow(`SELECT current, best FROM user_answer_streak WHERE user_id = ?`,
		u.user).Scan(&cur, &best)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, fmt.Errorf("answer streak: %w", err)
	}
	return cur, best, nil
}
```

Add `"database/sql"` to `activity.go`'s imports.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./server/internal/store/ -run TestAnswerStreak -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add server/internal/store/activity.go server/internal/store/activity_test.go
git commit -m "Track the consecutive-correct-answer streak on first attempts only"
```

---

### Task 7: Backfill history

**Files:**
- Modify: `server/internal/store/migration_hooks.go`, `server/internal/store/activity_test.go`

**Interfaces:**
- Consumes: `attempts`, `reviews`, and the tables from Task 1.
- Produces: `func migrate015(tx *dbtx, pg bool) error`, registered for version **15** — a hook-only migration (the runner synthesises `015_hook` for a registered hook with no matching `.sql`, see `loadMigrations`).

**Why 15 and not a hook on 013** (controller ruling, recorded in the SDD ledger): migration 013 is applied back in Task 1, six tasks before this hook exists. Any database that boots the binary in between — including the dev `./data/app.db`, which persists across `make dev` runs — records version 13 as applied, and a hook added later would never fire for it. Version 15 always runs. The backfill only reads `attempts`, `reviews` and `economy_settings.daily_goal`, all present since 013, so running after the seed is safe.

Without this, everyone's streak resets to zero on deploy day — the worst possible first contact with a currency that is supposed to reward consistency.

- [ ] **Step 1: Write the failing test**

Append to `server/internal/store/activity_test.go`:

```go
func TestBackfillReconstructsActivityAndAnswerStreak(t *testing.T) {
	s := newStoreAtVersion(t, 14) // everything up to and including the seed, but not the backfill
	id, _ := s.CreateUser("История")

	// Two days of history, written before the backfill version runs: 10 first
	// attempts on the 27th (one wrong at the end), 10 reviews on the 28th.
	for i := 0; i < 10; i++ {
		correct := 1
		if i == 9 {
			correct = 0
		}
		if _, err := s.db.Exec(`INSERT INTO attempts (user_id, exercise_id, lesson, block, answer, correct, attempted_at)
			VALUES (?, ?, '01', 'a', 'x', ?, ?)`,
			id, "01."+strconv.Itoa(i), correct, "2026-09-27T09:00:00Z"); err != nil {
			t.Fatalf("seed attempt: %v", err)
		}
	}
	for i := 0; i < 10; i++ {
		if _, err := s.db.Exec(`INSERT INTO reviews (user_id, card_id, grade, reviewed_at) VALUES (?, ?, 3, ?)`,
			id, "vocab:"+strconv.Itoa(i), "2026-09-28T09:00:00Z"); err != nil {
			t.Fatalf("seed review: %v", err)
		}
	}

	if err := s.runMigrations(); err != nil {
		t.Fatalf("migrate to head: %v", err)
	}

	var actions int
	if err := s.db.QueryRow(`SELECT actions FROM user_daily_activity WHERE user_id = ? AND day = ?`,
		id, "2026-09-27").Scan(&actions); err != nil {
		t.Fatalf("27th: %v", err)
	}
	if actions != 10 {
		t.Fatalf("27th actions = %d, want 10", actions)
	}
	if err := s.db.QueryRow(`SELECT actions FROM user_daily_activity WHERE user_id = ? AND day = ?`,
		id, "2026-09-28").Scan(&actions); err != nil {
		t.Fatalf("28th: %v", err)
	}
	if actions != 10 {
		t.Fatalf("28th actions = %d, want 10", actions)
	}

	u := s.User(id)
	cur, best, err := u.AnswerStreak()
	if err != nil {
		t.Fatalf("answer streak: %v", err)
	}
	if best != 9 {
		t.Fatalf("best = %d, want 9 (nine correct, then one wrong)", best)
	}
	if cur != 0 {
		t.Fatalf("current = %d, want 0 (the last answer was wrong)", cur)
	}

	// The backfill must not mint currency: it reconstructs history only.
	if bal, _ := s.Balance(id); bal != 0 {
		t.Fatalf("balance = %d after backfill, want 0", bal)
	}
}
```

`newStoreAtVersion` is the existing helper pattern used by `migrate_test.go` / `migrate_test_helper_test.go` (it calls `runMigrationsUpTo`). Read those files and use the real helper name; if there is no such helper, add one there rather than inventing a parallel setup path in this file.

The "must not mint currency" assertion is deliberate: paying a year of retroactive drip to 177 accounts on deploy day is exactly the kind of irreversible mistake an append-only ledger cannot undo.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./server/internal/store/ -run TestBackfill -v`
Expected: FAIL — `user_daily_activity` has no rows.

- [ ] **Step 3: Implement the hook**

In `server/internal/store/migration_hooks.go`, add `registerHook(15, migrate015)` to `init()` and append:

```go
// migrate015 reconstructs the currency feature's derived history so nobody
// starts at zero on deploy day. A hook-only migration: it needs no schema of
// its own, and its own version number guarantees it runs even on a database
// that already applied 013.
//
// Deliberately a separate version from 013 rather than a hook on it — a
// database that booted between the two would otherwise skip the backfill
// forever.
//
// Days are reconstructed in Europe/Belgrade for everyone: no historical
// timezone exists, and the rows written here are never recomputed afterwards.
// Goals use the seeded daily_goal. Deliberately writes no ledger rows — this
// reconstructs history, it does not pay for it.
func migrate015(tx *dbtx, pg bool) error {
	loc, err := time.LoadLocation(fallbackTZ)
	if err != nil {
		loc = time.UTC
	}
	var goal int
	if err := tx.QueryRow(`SELECT value FROM economy_settings WHERE key = 'daily_goal'`).Scan(&goal); err != nil {
		goal = defaultDailyGoal
	}

	// counts[userID][day] = actions
	counts := map[string]map[string]int{}
	bump := func(userID, iso string) {
		ts, err := time.Parse(time.RFC3339, iso)
		if err != nil {
			return
		}
		day := ts.In(loc).Format(dateFmt)
		if counts[userID] == nil {
			counts[userID] = map[string]int{}
		}
		counts[userID][day]++
	}

	// First attempt per (user, exercise) only — same rule as AddAttempt.
	rows, err := tx.Query(`SELECT user_id, MIN(attempted_at) FROM attempts GROUP BY user_id, exercise_id`)
	if err != nil {
		return fmt.Errorf("backfill: read attempts: %w", err)
	}
	for rows.Next() {
		var userID, at string
		if err := rows.Scan(&userID, &at); err != nil {
			rows.Close()
			return fmt.Errorf("backfill: scan attempt: %w", err)
		}
		bump(userID, at)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("backfill: iterate attempts: %w", err)
	}
	rows.Close()

	revs, err := tx.Query(`SELECT user_id, reviewed_at FROM reviews`)
	if err != nil {
		return fmt.Errorf("backfill: read reviews: %w", err)
	}
	for revs.Next() {
		var userID, at string
		if err := revs.Scan(&userID, &at); err != nil {
			revs.Close()
			return fmt.Errorf("backfill: scan review: %w", err)
		}
		bump(userID, at)
	}
	if err := revs.Err(); err != nil {
		revs.Close()
		return fmt.Errorf("backfill: iterate reviews: %w", err)
	}
	revs.Close()

	for userID, days := range counts {
		for day, n := range days {
			if _, err := tx.Exec(`INSERT INTO user_daily_activity (user_id, day, actions, goal)
				VALUES (?, ?, ?, ?) ON CONFLICT (user_id, day) DO NOTHING`,
				userID, day, n, goal); err != nil {
				return fmt.Errorf("backfill: write activity %s/%s: %w", userID, day, err)
			}
		}
	}

	return backfillAnswerStreaks(tx)
}

// backfillAnswerStreaks replays each user's first attempts in order and stores
// the resulting current/best run.
func backfillAnswerStreaks(tx *dbtx, ) error {
	rows, err := tx.Query(`SELECT user_id, exercise_id, MIN(attempted_at) AS at,
		MIN(correct) AS correct
		FROM attempts GROUP BY user_id, exercise_id ORDER BY user_id, at`)
	if err != nil {
		return fmt.Errorf("backfill: read answer history: %w", err)
	}
	defer rows.Close()

	type run struct{ current, best int }
	runs := map[string]*run{}
	for rows.Next() {
		var userID, exID, at string
		var correct int
		if err := rows.Scan(&userID, &exID, &at, &correct); err != nil {
			return fmt.Errorf("backfill: scan answer: %w", err)
		}
		r := runs[userID]
		if r == nil {
			r = &run{}
			runs[userID] = r
		}
		if correct == 1 {
			r.current++
			if r.current > r.best {
				r.best = r.current
			}
		} else {
			r.current = 0
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("backfill: iterate answers: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for userID, r := range runs {
		if _, err := tx.Exec(`INSERT INTO user_answer_streak (user_id, current, best, updated_at)
			VALUES (?, ?, ?, ?) ON CONFLICT (user_id) DO NOTHING`,
			userID, r.current, r.best, now); err != nil {
			return fmt.Errorf("backfill: write answer streak %s: %w", userID, err)
		}
	}
	return nil
}
```

Fix the stray comma in `backfillAnswerStreaks(tx *dbtx, )` while typing it — the signature is `func backfillAnswerStreaks(tx *dbtx) error`.

`MIN(correct)` works because the group has exactly one row per `(user_id, exercise_id)` after `MIN(attempted_at)` picks the first — but on a user who answered the same exercise twice it would pick the *lowest* correct flag rather than the first attempt's. If that matters (it does: it would understate `best`), replace the whole query with a per-user ordered scan of `attempts` in Go, keeping a `seen map[string]bool` of exercise ids and processing only the first occurrence. Prefer that version — it is obviously correct and the table is small.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./server/internal/store/ -run TestBackfill -v`
Expected: PASS.

- [ ] **Step 5: Run the migration suite on both backends**

Run: `go test ./server/internal/store/`
Run: `make test-pg`
Expected: PASS on both.

- [ ] **Step 6: Commit**

```bash
git add server/internal/store/migration_hooks.go server/internal/store/activity_test.go
git commit -m "Backfill daily activity and answer streaks from existing history"
```

---

### Task 8: Quest engine

**Files:**
- Create: `server/internal/store/quests.go`, `server/internal/store/quests_test.go`
- Modify: `server/internal/economy/economy.go`, `server/internal/economy/economy_test.go`

**Interfaces:**
- Consumes: `quests`, `quest_claims` (Task 1); `addLedgerEntryTx`, `ErrDuplicateEntry` (Task 2); `AnswerStreak`, `StreakDays` (Tasks 5–6).
- Produces:
  - `type economy.Quest struct { ID int64; Kind string; Target int; Param, Title, Description string; Reward int64; Active bool; SortOrder int }`
  - `type economy.Counters struct { LessonsCompleted, VocabLearned, ReviewsDone, AnswerBest, StreakDays int; CompletedLessons map[string]bool; TelegramSubscribed bool }`
  - `func economy.QuestValue(q Quest, c Counters, phaseLessons map[string][]string) int`
  - `func economy.QuestDone(q Quest, c Counters, phaseLessons map[string][]string) bool`
  - `func (u *UserStore) QuestCounters(now time.Time) (economy.Counters, error)` — everything except `TelegramSubscribed`
  - `func (s *Store) ListQuests(activeOnly bool) ([]economy.Quest, error)`
  - `func (u *UserStore) ClaimedQuestIDs() (map[int64]bool, error)`
  - `func (u *UserStore) ClaimQuest(q economy.Quest, now time.Time) error` — returns `ErrAlreadyClaimed`
  - `var ErrAlreadyClaimed = errors.New("quest already claimed")`

The split is deliberate: the store knows nothing about `course.yaml`, so phase membership arrives as a map from the API layer, and the whole "is this done" rule stays a pure function.

- [ ] **Step 1: Write the failing tests for the pure evaluator**

Append to `server/internal/economy/economy_test.go`:

```go
func TestQuestValue(t *testing.T) {
	phases := map[string][]string{
		"1": {"00", "01", "02", "03"},
		"2": {"16", "17"},
	}
	c := Counters{
		LessonsCompleted: 12,
		VocabLearned:     140,
		ReviewsDone:      320,
		AnswerBest:       11,
		StreakDays:       9,
		CompletedLessons: map[string]bool{"00": true, "01": true, "02": true, "16": true},
	}
	cases := []struct {
		kind, param string
		want        int
	}{
		{"lessons_completed", "", 12},
		{"vocab_learned", "", 140},
		{"reviews_done", "", 320},
		{"correct_in_row", "", 11},
		{"streak_days", "", 9},
		{"phase_completed", "1", 75},  // 3 of 4
		{"phase_completed", "2", 50},  // 1 of 2
		{"phase_completed", "9", 0},   // unknown phase
		{"telegram_subscribed", "", 0},
		{"nonsense", "", 0},
	}
	for _, tc := range cases {
		q := Quest{Kind: tc.kind, Param: tc.param, Target: 100}
		if got := QuestValue(q, c, phases); got != tc.want {
			t.Errorf("QuestValue(%s,%q) = %d, want %d", tc.kind, tc.param, got, tc.want)
		}
	}
}

func TestQuestDone(t *testing.T) {
	phases := map[string][]string{"1": {"00", "01"}}
	c := Counters{LessonsCompleted: 10, CompletedLessons: map[string]bool{"00": true, "01": true}}

	if !QuestDone(Quest{Kind: "lessons_completed", Target: 10}, c, phases) {
		t.Error("exactly at target must count as done")
	}
	if QuestDone(Quest{Kind: "lessons_completed", Target: 11}, c, phases) {
		t.Error("one short must not count as done")
	}
	if !QuestDone(Quest{Kind: "phase_completed", Param: "1", Target: 100}, c, phases) {
		t.Error("a fully completed phase must count as done")
	}

	// telegram_subscribed is a flag, not a count.
	sub := c
	sub.TelegramSubscribed = true
	if !QuestDone(Quest{Kind: "telegram_subscribed", Target: 1}, sub, phases) {
		t.Error("a subscribed user must complete the telegram quest")
	}
	if QuestDone(Quest{Kind: "telegram_subscribed", Target: 1}, c, phases) {
		t.Error("an unsubscribed user must not complete the telegram quest")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./server/internal/economy/ -run TestQuest -v`
Expected: FAIL — `undefined: QuestValue`.

- [ ] **Step 3: Implement the evaluator**

Append to `server/internal/economy/economy.go`:

```go
// Quest kinds. Every kind must be computable server-side from data the user
// cannot forge — that is the whole reason the admin panel creates instances
// of a fixed list rather than free-form conditions.
const (
	QuestLessonsCompleted  = "lessons_completed"
	QuestVocabLearned      = "vocab_learned"
	QuestReviewsDone       = "reviews_done"
	QuestStreakDays        = "streak_days"
	QuestCorrectInRow      = "correct_in_row"
	QuestPhaseCompleted    = "phase_completed"
	QuestTelegramSubscribed = "telegram_subscribed"
)

// Quest is one row of the quests table.
type Quest struct {
	ID          int64
	Kind        string
	Target      int
	Param       string
	Title       string
	Description string
	Reward      int64
	Active      bool
	SortOrder   int
}

// Counters is everything a quest can be measured against, gathered once per
// request. CompletedLessons is keyed by lesson id as it appears in
// course.yaml. TelegramSubscribed is filled by the caller, since it needs a
// Bot API round trip.
type Counters struct {
	LessonsCompleted   int
	VocabLearned       int
	ReviewsDone        int
	AnswerBest         int
	StreakDays         int
	CompletedLessons   map[string]bool
	TelegramSubscribed bool
}

// QuestValue is the user's current value for q. For phase_completed the value
// is a percentage (0..100), which is why such a quest's target is always 100;
// for telegram_subscribed it is 0 or 1. An unknown kind is worth 0 rather than
// an error: an admin can save a kind this binary has not learned yet, and a
// quest nobody can finish is better than a 500 on the profile screen.
func QuestValue(q Quest, c Counters, phaseLessons map[string][]string) int {
	switch q.Kind {
	case QuestLessonsCompleted:
		return c.LessonsCompleted
	case QuestVocabLearned:
		return c.VocabLearned
	case QuestReviewsDone:
		return c.ReviewsDone
	case QuestStreakDays:
		return c.StreakDays
	case QuestCorrectInRow:
		return c.AnswerBest
	case QuestPhaseCompleted:
		lessons := phaseLessons[q.Param]
		if len(lessons) == 0 {
			return 0
		}
		done := 0
		for _, id := range lessons {
			if c.CompletedLessons[id] {
				done++
			}
		}
		return done * 100 / len(lessons)
	case QuestTelegramSubscribed:
		if c.TelegramSubscribed {
			return 1
		}
		return 0
	default:
		return 0
	}
}

// QuestDone reports whether the quest's target has been reached. An unknown
// kind is never done, because QuestValue reports 0 for it and a target of 0
// would otherwise make every unknown quest instantly claimable.
func QuestDone(q Quest, c Counters, phaseLessons map[string][]string) bool {
	if q.Target <= 0 {
		return false
	}
	return QuestValue(q, c, phaseLessons) >= q.Target
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./server/internal/economy/ -v`
Expected: PASS.

- [ ] **Step 5: Write the failing store tests**

Create `server/internal/store/quests_test.go`:

```go
package store

import (
	"errors"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/economy"
)

func seedQuest(t *testing.T, s *Store, kind string, target int, reward int64) economy.Quest {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.Exec(`INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
		VALUES (?, ?, '', ?, '', ?, 1, 0, ?, ?)`,
		kind, target, "Задание", reward, now, now); err != nil {
		t.Fatalf("seed quest: %v", err)
	}
	qs, err := s.ListQuests(true)
	if err != nil {
		t.Fatalf("list quests: %v", err)
	}
	return qs[len(qs)-1]
}

func TestQuestCountersReadRealProgress(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Счёт")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)

	answer(t, u, "01.1", true, at)
	answer(t, u, "01.2", true, at)
	if err := u.SetLessonStatus("01", "done", at); err != nil {
		t.Fatalf("lesson status: %v", err)
	}

	c, err := u.QuestCounters(at)
	if err != nil {
		t.Fatalf("counters: %v", err)
	}
	if c.AnswerBest != 2 {
		t.Fatalf("AnswerBest = %d, want 2", c.AnswerBest)
	}
	if !c.CompletedLessons["01"] {
		t.Fatal("lesson 01 not reported as completed")
	}
	if c.LessonsCompleted != 1 {
		t.Fatalf("LessonsCompleted = %d, want 1", c.LessonsCompleted)
	}
}

func TestClaimQuestCreditsOnceAndOnlyOnce(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Клейм")
	u := s.User(id)
	q := seedQuest(t, s, economy.QuestLessonsCompleted, 1, 30)
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

	if err := u.ClaimQuest(q, now); err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if bal, _ := s.Balance(id); bal != 30 {
		t.Fatalf("balance = %d, want 30", bal)
	}

	if err := u.ClaimQuest(q, now); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("second claim err = %v, want ErrAlreadyClaimed", err)
	}
	if bal, _ := s.Balance(id); bal != 30 {
		t.Fatalf("balance = %d after a repeat claim, want 30", bal)
	}

	claimed, err := u.ClaimedQuestIDs()
	if err != nil {
		t.Fatalf("claimed ids: %v", err)
	}
	if !claimed[q.ID] {
		t.Fatal("quest not reported as claimed")
	}
}

func TestListQuestsRespectsActiveFlagAndOrder(t *testing.T) {
	s := newStore(t)
	a := seedQuest(t, s, economy.QuestLessonsCompleted, 5, 5)
	b := seedQuest(t, s, economy.QuestVocabLearned, 30, 10)
	if _, err := s.db.Exec(`UPDATE quests SET active = 0 WHERE id = ?`, b.ID); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	if _, err := s.db.Exec(`UPDATE quests SET sort_order = 5 WHERE id = ?`, a.ID); err != nil {
		t.Fatalf("reorder: %v", err)
	}

	active, err := s.ListQuests(true)
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(active) != 1 || active[0].ID != a.ID {
		t.Fatalf("active quests = %+v, want only %d", active, a.ID)
	}
	all, err := s.ListQuests(false)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("all quests = %d, want 2", len(all))
	}
}
```

- [ ] **Step 6: Run to verify it fails**

Run: `go test ./server/internal/store/ -run TestQuest -v` and `-run TestClaim` and `-run TestList`
Expected: FAIL — `undefined: (*Store).ListQuests`.

- [ ] **Step 7: Implement the store side**

Create `server/internal/store/quests.go`:

```go
package store

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/grisha/serbian-app/server/internal/economy"
)

// ErrAlreadyClaimed means this user has already taken this quest's reward.
var ErrAlreadyClaimed = errors.New("quest already claimed")

// ListQuests returns quests ordered the way the profile shows them.
func (s *Store) ListQuests(activeOnly bool) ([]economy.Quest, error) {
	q := `SELECT id, kind, target, param, title, description, reward, active, sort_order
		FROM quests`
	if activeOnly {
		q += ` WHERE active = 1`
	}
	q += ` ORDER BY sort_order ASC, id ASC`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("list quests: %w", err)
	}
	defer rows.Close()
	var out []economy.Quest
	for rows.Next() {
		var qu economy.Quest
		var active int
		if err := rows.Scan(&qu.ID, &qu.Kind, &qu.Target, &qu.Param, &qu.Title,
			&qu.Description, &qu.Reward, &active, &qu.SortOrder); err != nil {
			return nil, fmt.Errorf("scan quest: %w", err)
		}
		qu.Active = active == 1
		out = append(out, qu)
	}
	return out, rows.Err()
}

// QuestCounters gathers every locally computable quest input in one go.
// TelegramSubscribed is left false — only the API layer can fill it, since it
// needs a Bot API call.
func (u *UserStore) QuestCounters(now time.Time) (economy.Counters, error) {
	var c economy.Counters

	if err := u.db.QueryRow(`SELECT COUNT(*) FROM lesson_progress WHERE user_id = ? AND status = 'done'`,
		u.user).Scan(&c.LessonsCompleted); err != nil {
		return c, fmt.Errorf("counters: lessons: %w", err)
	}
	if err := u.db.QueryRow(`SELECT COUNT(*) FROM srs_cards WHERE user_id = ? AND kind = 'vocab' AND state <> 'new'`,
		u.user).Scan(&c.VocabLearned); err != nil {
		return c, fmt.Errorf("counters: vocab: %w", err)
	}
	if err := u.db.QueryRow(`SELECT COUNT(*) FROM reviews WHERE user_id = ?`,
		u.user).Scan(&c.ReviewsDone); err != nil {
		return c, fmt.Errorf("counters: reviews: %w", err)
	}

	_, best, err := u.AnswerStreak()
	if err != nil {
		return c, err
	}
	c.AnswerBest = best

	streak, err := u.StreakDays(now)
	if err != nil {
		return c, err
	}
	c.StreakDays = streak

	rows, err := u.db.Query(`SELECT lesson FROM lesson_progress WHERE user_id = ? AND status = 'done'`, u.user)
	if err != nil {
		return c, fmt.Errorf("counters: completed lessons: %w", err)
	}
	defer rows.Close()
	c.CompletedLessons = map[string]bool{}
	for rows.Next() {
		var lesson string
		if err := rows.Scan(&lesson); err != nil {
			return c, fmt.Errorf("scan completed lesson: %w", err)
		}
		c.CompletedLessons[lesson] = true
	}
	return c, rows.Err()
}

// ClaimedQuestIDs is the set of quests this user has already taken.
func (u *UserStore) ClaimedQuestIDs() (map[int64]bool, error) {
	rows, err := u.db.Query(`SELECT quest_id FROM quest_claims WHERE user_id = ?`, u.user)
	if err != nil {
		return nil, fmt.Errorf("claimed quests: %w", err)
	}
	defer rows.Close()
	out := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan claimed quest: %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}

// ClaimQuest credits the reward and records the claim in one transaction.
//
// The caller MUST have verified completion first (economy.QuestDone) — this
// function does not re-check, because completion needs course content the
// store layer deliberately does not know about.
func (u *UserStore) ClaimQuest(q economy.Quest, now time.Time) error {
	tx, err := u.db.Begin()
	if err != nil {
		return fmt.Errorf("claim quest: %w", err)
	}
	defer tx.Rollback()

	if _, err := addLedgerEntryTx(tx, LedgerEntry{
		UserID:         u.user,
		Amount:         q.Reward,
		Kind:           "quest_reward",
		Ref:            strconv.FormatInt(q.ID, 10),
		IdempotencyKey: "quest:" + strconv.FormatInt(q.ID, 10) + ":" + u.user,
	}, now); err != nil {
		if errors.Is(err, ErrDuplicateEntry) {
			return ErrAlreadyClaimed
		}
		return err
	}
	if _, err := tx.Exec(`INSERT INTO quest_claims (quest_id, user_id, claimed_at) VALUES (?, ?, ?)`,
		q.ID, u.user, now.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("claim quest: record claim: %w", err)
	}
	return tx.Commit()
}
```

A zero-reward quest would be rejected by `addLedgerEntryTx`'s zero-amount guard. That is acceptable — a quest worth nothing is a configuration mistake — but the admin panel must validate `reward > 0` on save (Task 4 of the admin plan) so the error never reaches a user.

- [ ] **Step 8: Run to verify it passes**

Run: `go test ./server/internal/store/ -run 'TestQuest|TestClaim|TestList' -v`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add server/internal/economy/ server/internal/store/quests.go server/internal/store/quests_test.go
git commit -m "Add quest engine: pure evaluator, counters from existing progress, idempotent claim"
```

---

### Task 9: Telegram subscription check

**Files:**
- Modify: `server/internal/telegram/telegram.go`, `server/internal/telegram/telegram_test.go`

**Interfaces:**
- Consumes: the package's existing `call` helper and `apiBase` seam.
- Produces: `func GetChatMember(botToken, chat string, userID int64) (status string, err error)` and `func IsMember(status string) bool`

Production cannot reach `api.telegram.org` directly, but every call in this package goes through `apiBase`, which `main.go` repoints at the reverse proxy when `TELEGRAM_API_BASE` is set. This check inherits that for free — do not add a separate HTTP client.

- [ ] **Step 1: Write the failing test**

Append to `server/internal/telegram/telegram_test.go`, using that file's existing `withTestServer` helper:

```go
func TestGetChatMember(t *testing.T) {
	cases := []struct {
		name, body string
		want       string
		wantMember bool
		wantErr    bool
	}{
		{"member", `{"ok":true,"result":{"status":"member"}}`, "member", true, false},
		{"admin", `{"ok":true,"result":{"status":"administrator"}}`, "administrator", true, false},
		{"creator", `{"ok":true,"result":{"status":"creator"}}`, "creator", true, false},
		{"left", `{"ok":true,"result":{"status":"left"}}`, "left", false, false},
		{"kicked", `{"ok":true,"result":{"status":"kicked"}}`, "kicked", false, false},
		{"not found", `{"ok":false,"error_code":400,"description":"Bad Request: user not found"}`, "", false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			restore := withTestServer(t, c.body) // match this file's real helper signature
			defer restore()

			status, err := GetChatMember("token", "@ucimo", 12345)
			if c.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetChatMember: %v", err)
			}
			if status != c.want {
				t.Fatalf("status = %q, want %q", status, c.want)
			}
			if IsMember(status) != c.wantMember {
				t.Fatalf("IsMember(%q) = %v, want %v", status, IsMember(status), c.wantMember)
			}
		})
	}
}
```

Read the existing tests in that file first and match the helper's real name and signature rather than the placeholder used here.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./server/internal/telegram/ -run TestGetChatMember -v`
Expected: FAIL — `undefined: GetChatMember`.

- [ ] **Step 3: Implement**

Append to `server/internal/telegram/telegram.go`:

```go
// GetChatMember returns a user's membership status in a chat, e.g. "member",
// "administrator", "creator", "left", "kicked", "restricted". The bot must be
// an administrator of the channel for this to work.
//
// Like every other call here it goes through apiBase, so production's
// TELEGRAM_API_BASE proxy covers it — the Dokploy host cannot reach
// api.telegram.org directly.
func GetChatMember(botToken, chat string, userID int64) (string, error) {
	var out struct {
		Status string `json:"status"`
	}
	params := url.Values{}
	params.Set("chat_id", chat)
	params.Set("user_id", strconv.FormatInt(userID, 10))
	if err := call(botToken, "getChatMember", params, &out); err != nil {
		return "", fmt.Errorf("get chat member: %w", err)
	}
	return out.Status, nil
}

// IsMember reports whether a getChatMember status means the user is currently
// subscribed. "restricted" counts: such a user is in the channel, just limited.
func IsMember(status string) bool {
	switch status {
	case "creator", "administrator", "member", "restricted":
		return true
	default:
		return false
	}
}
```

Add `"strconv"` to the imports if it is not already there.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./server/internal/telegram/ -run TestGetChatMember -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add server/internal/telegram/telegram.go server/internal/telegram/telegram_test.go
git commit -m "Add telegram.GetChatMember for the channel-subscription quest"
```

---

### Task 10: Catalogue, promo codes and purchase

**Files:**
- Create: `server/internal/store/shop.go`, `server/internal/store/shop_test.go`
- Modify: `server/internal/economy/economy.go` (product/promo types)

**Interfaces:**
- Consumes: `products`, `promo_codes`, `promo_code_products`, `promo_redemptions`, `user_entitlements` (Task 1); `balanceTx`, `addLedgerEntryTx` (Task 2); `economy.EffectivePrice` (Task 3).
- Produces:
  - `type economy.Product struct { ID int64; Kind, Ref, Title, Description string; Price int64; DiscountPercent int; DiscountFrom, DiscountTo string; GrantQty int; Active bool; SortOrder int }`
  - `func (p economy.Product) SalePercent(now time.Time) int`
  - `const economy.ProductPhaseUnlock = "phase_unlock"`, `ProductConsumable = "consumable"`, `ProductCosmetic = "cosmetic"`
  - `func (s *Store) ListProducts(activeOnly bool) ([]economy.Product, error)`
  - `func (u *UserStore) Entitlements() (map[string]int64, error)` — keyed `kind + ":" + ref`
  - `type PurchaseResult struct { LedgerID int64; Paid int64; Applied string }`
  - `func (u *UserStore) Purchase(productID int64, promoCode string, now time.Time) (PurchaseResult, error)`
  - `func (u *UserStore) SpendStreakRepair(day string, now time.Time) error`
  - `var ErrInsufficientFunds`, `ErrAlreadyOwned`, `ErrProductUnavailable`, `ErrPromoInvalid`, `ErrNoEntitlement`

- [ ] **Step 1: Write the failing tests**

Create `server/internal/store/shop_test.go`:

```go
package store

import (
	"errors"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/economy"
)

func seedProduct(t *testing.T, s *Store, kind, ref string, price int64, grantQty int) int64 {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	var id int64
	err := s.db.QueryRow(`INSERT INTO products
		(kind, ref, title, description, price, discount_percent, discount_from, discount_to,
		 grant_qty, active, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, '', ?, 0, '', '', ?, 1, 0, ?, ?) RETURNING id`,
		kind, ref, "Товар", price, grantQty, now, now).Scan(&id)
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}
	return id
}

func credit(t *testing.T, s *Store, userID string, amount int64, key string) {
	t.Helper()
	if _, err := s.AddLedgerEntry(LedgerEntry{
		UserID: userID, Amount: amount, Kind: "admin_adjustment",
		IdempotencyKey: key, Comment: "test", CreatedBy: "admin",
	}, time.Now()); err != nil {
		t.Fatalf("credit: %v", err)
	}
}

func TestPurchaseDebitsAndGrants(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Покупка")
	u := s.User(id)
	pid := seedProduct(t, s, economy.ProductPhaseUnlock, "4", 500, 1)
	credit(t, s, id, 600, "seed:1")
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

	res, err := u.Purchase(pid, "", now)
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}
	if res.Paid != 500 || res.Applied != "" {
		t.Fatalf("paid, applied = %d, %q; want 500, \"\"", res.Paid, res.Applied)
	}
	if bal, _ := s.Balance(id); bal != 100 {
		t.Fatalf("balance = %d, want 100", bal)
	}
	ent, err := u.Entitlements()
	if err != nil {
		t.Fatalf("entitlements: %v", err)
	}
	if ent["phase_unlock:4"] != 1 {
		t.Fatalf("entitlement = %d, want 1", ent["phase_unlock:4"])
	}
}

func TestPurchaseRefusesToOverdraw(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Бедный")
	u := s.User(id)
	pid := seedProduct(t, s, economy.ProductPhaseUnlock, "4", 500, 1)
	credit(t, s, id, 499, "seed:1")

	_, err := u.Purchase(pid, "", time.Now())
	if !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("err = %v, want ErrInsufficientFunds", err)
	}
	if bal, _ := s.Balance(id); bal != 499 {
		t.Fatalf("balance = %d — a failed purchase wrote something", bal)
	}
	ent, _ := u.Entitlements()
	if len(ent) != 0 {
		t.Fatalf("entitlements = %v — a failed purchase granted something", ent)
	}
}

func TestPurchasePermanentTwiceIsRejected(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Дважды")
	u := s.User(id)
	pid := seedProduct(t, s, economy.ProductCosmetic, "palette:forest", 30, 1)
	credit(t, s, id, 100, "seed:1")

	if _, err := u.Purchase(pid, "", time.Now()); err != nil {
		t.Fatalf("first purchase: %v", err)
	}
	if _, err := u.Purchase(pid, "", time.Now()); !errors.Is(err, ErrAlreadyOwned) {
		t.Fatalf("err = %v, want ErrAlreadyOwned", err)
	}
	if bal, _ := s.Balance(id); bal != 70 {
		t.Fatalf("balance = %d, want 70 (charged once)", bal)
	}
}

func TestPurchaseConsumableStacks(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Расходник")
	u := s.User(id)
	pid := seedProduct(t, s, economy.ProductConsumable, "streak_repair", 25, 1)
	credit(t, s, id, 100, "seed:1")

	for i := 0; i < 3; i++ {
		if _, err := u.Purchase(pid, "", time.Now()); err != nil {
			t.Fatalf("purchase %d: %v", i, err)
		}
	}
	ent, _ := u.Entitlements()
	if ent["consumable:streak_repair"] != 3 {
		t.Fatalf("qty = %d, want 3", ent["consumable:streak_repair"])
	}
	if bal, _ := s.Balance(id); bal != 25 {
		t.Fatalf("balance = %d, want 25", bal)
	}
}

func TestPurchaseAppliesTheBetterDiscount(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Скидка")
	u := s.User(id)
	pid := seedProduct(t, s, economy.ProductPhaseUnlock, "4", 500, 1)
	credit(t, s, id, 1000, "seed:1")
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

	// A 20% sale running now.
	if _, err := s.db.Exec(`UPDATE products SET discount_percent = 20, discount_from = ?, discount_to = ? WHERE id = ?`,
		"2026-09-01", "2026-10-01", pid); err != nil {
		t.Fatalf("set sale: %v", err)
	}
	// A weaker 10% promo code.
	if _, err := s.db.Exec(`INSERT INTO promo_codes
		(code, discount_percent, scope, valid_from, valid_to, max_redemptions, max_per_user, active, created_at, updated_at)
		VALUES ('OSEN', 10, 'all', '', '', 0, 1, 1, ?, ?)`,
		now.Format(time.RFC3339), now.Format(time.RFC3339)); err != nil {
		t.Fatalf("seed promo: %v", err)
	}

	res, err := u.Purchase(pid, "osen", now) // lower case on purpose: codes are case-insensitive
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}
	if res.Paid != 400 || res.Applied != "sale" {
		t.Fatalf("paid, applied = %d, %q; want 400, \"sale\"", res.Paid, res.Applied)
	}
}

func TestPromoCodeLimits(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Промо")
	u := s.User(id)
	pid := seedProduct(t, s, economy.ProductConsumable, "streak_repair", 100, 1)
	credit(t, s, id, 1000, "seed:1")
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	iso := now.Format(time.RFC3339)

	if _, err := s.db.Exec(`INSERT INTO promo_codes
		(code, discount_percent, scope, valid_from, valid_to, max_redemptions, max_per_user, active, created_at, updated_at)
		VALUES ('ONCE', 50, 'all', '', '', 0, 1, 1, ?, ?)`, iso, iso); err != nil {
		t.Fatalf("seed promo: %v", err)
	}

	res, err := u.Purchase(pid, "ONCE", now)
	if err != nil || res.Paid != 50 {
		t.Fatalf("first use: paid %d, err %v; want 50, nil", res.Paid, err)
	}
	if _, err := u.Purchase(pid, "ONCE", now); !errors.Is(err, ErrPromoInvalid) {
		t.Fatalf("second use err = %v, want ErrPromoInvalid (max_per_user is 1)", err)
	}

	// An expired code is rejected too.
	if _, err := s.db.Exec(`INSERT INTO promo_codes
		(code, discount_percent, scope, valid_from, valid_to, max_redemptions, max_per_user, active, created_at, updated_at)
		VALUES ('OLD', 50, 'all', '2026-01-01', '2026-02-01', 0, 99, 1, ?, ?)`, iso, iso); err != nil {
		t.Fatalf("seed expired promo: %v", err)
	}
	if _, err := u.Purchase(pid, "OLD", now); !errors.Is(err, ErrPromoInvalid) {
		t.Fatalf("expired code err = %v, want ErrPromoInvalid", err)
	}
}

func TestSpendStreakRepairConsumesOneUnit(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Ремонт")
	u := s.User(id)
	pid := seedProduct(t, s, economy.ProductConsumable, "streak_repair", 25, 1)
	credit(t, s, id, 100, "seed:1")
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, loc)

	if _, err := u.Purchase(pid, "", now); err != nil {
		t.Fatalf("buy repair: %v", err)
	}
	if err := u.SpendStreakRepair("2026-09-28", now); err != nil {
		t.Fatalf("spend: %v", err)
	}
	ent, _ := u.Entitlements()
	if ent["consumable:streak_repair"] != 0 {
		t.Fatalf("qty = %d, want 0", ent["consumable:streak_repair"])
	}
	if err := u.SpendStreakRepair("2026-09-27", now); !errors.Is(err, ErrNoEntitlement) {
		t.Fatalf("err = %v, want ErrNoEntitlement", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./server/internal/store/ -run 'TestPurchase|TestPromo|TestSpend' -v`
Expected: FAIL — `undefined: (*UserStore).Purchase`.

- [ ] **Step 3: Add product types to the pure package**

Append to `server/internal/economy/economy.go`:

```go
// Product kinds.
const (
	ProductPhaseUnlock = "phase_unlock"
	ProductConsumable  = "consumable"
	ProductCosmetic    = "cosmetic"
)

// Product is one row of the products table. Ref identifies what is being sold
// within its kind: a phase id, a consumable key, a palette name.
type Product struct {
	ID              int64
	Kind            string
	Ref             string
	Title           string
	Description     string
	Price           int64
	DiscountPercent int
	DiscountFrom    string // YYYY-MM-DD, empty = open-ended
	DiscountTo      string
	GrantQty        int
	Active          bool
	SortOrder       int
}

// Permanent reports whether owning this product is a one-off right rather than
// a stock of uses.
func (p Product) Permanent() bool {
	return p.Kind == ProductPhaseUnlock || p.Kind == ProductCosmetic
}

// SalePercent is the product's discount if now falls inside its window, else
// 0. An empty bound is open-ended on that side.
func (p Product) SalePercent(now time.Time) int {
	if p.DiscountPercent <= 0 {
		return 0
	}
	day := now.Format("2006-01-02")
	if p.DiscountFrom != "" && day < p.DiscountFrom {
		return 0
	}
	if p.DiscountTo != "" && day > p.DiscountTo {
		return 0
	}
	return p.DiscountPercent
}
```

Add `"time"` to the package's imports. Note this compares dates as strings, which is correct for zero-padded `YYYY-MM-DD` and avoids a timezone question the admin does not want to answer.

- [ ] **Step 4: Implement the shop**

Create `server/internal/store/shop.go`:

```go
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/grisha/serbian-app/server/internal/economy"
)

var (
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrAlreadyOwned       = errors.New("already owned")
	ErrProductUnavailable = errors.New("product unavailable")
	ErrPromoInvalid       = errors.New("promo code invalid")
	ErrNoEntitlement      = errors.New("no entitlement")
)

const productCols = `id, kind, ref, title, description, price, discount_percent,
	discount_from, discount_to, grant_qty, active, sort_order`

func scanProduct(sc interface{ Scan(...any) error }) (economy.Product, error) {
	var p economy.Product
	var active int
	err := sc.Scan(&p.ID, &p.Kind, &p.Ref, &p.Title, &p.Description, &p.Price,
		&p.DiscountPercent, &p.DiscountFrom, &p.DiscountTo, &p.GrantQty, &active, &p.SortOrder)
	p.Active = active == 1
	return p, err
}

// ListProducts returns the catalogue in display order.
func (s *Store) ListProducts(activeOnly bool) ([]economy.Product, error) {
	q := `SELECT ` + productCols + ` FROM products`
	if activeOnly {
		q += ` WHERE active = 1`
	}
	q += ` ORDER BY sort_order ASC, id ASC`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()
	var out []economy.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Entitlements returns what the user owns, keyed "kind:ref".
func (u *UserStore) Entitlements() (map[string]int64, error) {
	rows, err := u.db.Query(`SELECT kind, ref, qty FROM user_entitlements WHERE user_id = ?`, u.user)
	if err != nil {
		return nil, fmt.Errorf("entitlements: %w", err)
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var kind, ref string
		var qty int64
		if err := rows.Scan(&kind, &ref, &qty); err != nil {
			return nil, fmt.Errorf("scan entitlement: %w", err)
		}
		out[kind+":"+ref] = qty
	}
	return out, rows.Err()
}

// PurchaseResult reports what was actually charged.
type PurchaseResult struct {
	LedgerID int64
	Paid     int64
	Applied  string // "sale", "promo" or ""
}

// Purchase charges the effective price and grants the product, atomically.
//
// The user's row is locked first so two concurrent purchases cannot both see
// a sufficient balance and both debit it. On SQLite writers are serialised
// anyway, so the lock is a no-op there.
func (u *UserStore) Purchase(productID int64, promoCode string, now time.Time) (PurchaseResult, error) {
	tx, err := u.db.Begin()
	if err != nil {
		return PurchaseResult{}, fmt.Errorf("purchase: %w", err)
	}
	defer tx.Rollback()

	if tx.pg {
		if _, err := tx.Exec(`SELECT id FROM users WHERE id = ? FOR UPDATE`, u.user); err != nil {
			return PurchaseResult{}, fmt.Errorf("purchase: lock user: %w", err)
		}
	}

	p, err := scanProduct(tx.QueryRow(`SELECT `+productCols+` FROM products WHERE id = ?`, productID))
	if errors.Is(err, sql.ErrNoRows) {
		return PurchaseResult{}, ErrProductUnavailable
	}
	if err != nil {
		return PurchaseResult{}, fmt.Errorf("purchase: load product: %w", err)
	}
	if !p.Active {
		return PurchaseResult{}, ErrProductUnavailable
	}

	if p.Permanent() {
		var owned int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM user_entitlements WHERE user_id = ? AND kind = ? AND ref = ?`,
			u.user, p.Kind, p.Ref).Scan(&owned); err != nil {
			return PurchaseResult{}, fmt.Errorf("purchase: check ownership: %w", err)
		}
		if owned > 0 {
			return PurchaseResult{}, ErrAlreadyOwned
		}
	}

	promoID, promoPercent, err := validatePromoTx(tx, u.user, promoCode, p.ID, now)
	if err != nil {
		return PurchaseResult{}, err
	}

	paid, applied := economy.EffectivePrice(p.Price, p.SalePercent(now), promoPercent)

	bal, err := balanceTx(tx, u.user)
	if err != nil {
		return PurchaseResult{}, err
	}
	if bal < paid {
		return PurchaseResult{}, ErrInsufficientFunds
	}

	var ledgerID int64
	if paid > 0 {
		ledgerID, err = addLedgerEntryTx(tx, LedgerEntry{
			UserID: u.user,
			Amount: -paid,
			Kind:   "purchase",
			Ref:    strconv.FormatInt(p.ID, 10),
			IdempotencyKey: "purchase:" + u.user + ":" + strconv.FormatInt(p.ID, 10) + ":" +
				strconv.FormatInt(now.UnixNano(), 10),
		}, now)
		if err != nil {
			return PurchaseResult{}, err
		}
	}

	grant := int64(p.GrantQty)
	if p.Permanent() {
		grant = 1
	}
	if _, err := tx.Exec(`INSERT INTO user_entitlements (user_id, kind, ref, qty, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (user_id, kind, ref) DO UPDATE SET
			qty = user_entitlements.qty + ?, updated_at = ?`,
		u.user, p.Kind, p.Ref, grant, now.UTC().Format(time.RFC3339),
		grant, now.UTC().Format(time.RFC3339)); err != nil {
		return PurchaseResult{}, fmt.Errorf("purchase: grant: %w", err)
	}

	if promoID != 0 {
		if _, err := tx.Exec(`INSERT INTO promo_redemptions (promo_code_id, user_id, product_id, ledger_id, created_at)
			VALUES (?, ?, ?, ?, ?)`,
			promoID, u.user, p.ID, ledgerID, now.UTC().Format(time.RFC3339)); err != nil {
			return PurchaseResult{}, fmt.Errorf("purchase: record redemption: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return PurchaseResult{}, fmt.Errorf("purchase: commit: %w", err)
	}
	return PurchaseResult{LedgerID: ledgerID, Paid: paid, Applied: applied}, nil
}

// validatePromoTx resolves a code to its id and percent, or returns
// ErrPromoInvalid. An empty code is not an error — it means no code.
func validatePromoTx(tx *dbtx, userID, code string, productID int64, now time.Time) (int64, int, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return 0, 0, nil
	}
	var (
		id                          int64
		percent, maxRedeem, maxUser int
		scope, from, to             string
		active                      int
	)
	err := tx.QueryRow(`SELECT id, discount_percent, scope, valid_from, valid_to,
		max_redemptions, max_per_user, active FROM promo_codes WHERE code = ?`, code).
		Scan(&id, &percent, &scope, &from, &to, &maxRedeem, &maxUser, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, ErrPromoInvalid
	}
	if err != nil {
		return 0, 0, fmt.Errorf("promo code: %w", err)
	}
	if active != 1 {
		return 0, 0, ErrPromoInvalid
	}
	day := now.Format("2006-01-02")
	if (from != "" && day < from) || (to != "" && day > to) {
		return 0, 0, ErrPromoInvalid
	}
	if scope == "products" {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM promo_code_products WHERE promo_code_id = ? AND product_id = ?`,
			id, productID).Scan(&n); err != nil {
			return 0, 0, fmt.Errorf("promo scope: %w", err)
		}
		if n == 0 {
			return 0, 0, ErrPromoInvalid
		}
	}
	if maxRedeem > 0 {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM promo_redemptions WHERE promo_code_id = ?`, id).Scan(&n); err != nil {
			return 0, 0, fmt.Errorf("promo redemptions: %w", err)
		}
		if n >= maxRedeem {
			return 0, 0, ErrPromoInvalid
		}
	}
	if maxUser > 0 {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM promo_redemptions WHERE promo_code_id = ? AND user_id = ?`,
			id, userID).Scan(&n); err != nil {
			return 0, 0, fmt.Errorf("promo redemptions per user: %w", err)
		}
		if n >= maxUser {
			return 0, 0, ErrPromoInvalid
		}
	}
	return id, percent, nil
}

// SpendStreakRepair consumes one streak_repair unit and repairs the day. The
// two halves commit together: a repair that fails its window check must not
// eat the consumable.
func (u *UserStore) SpendStreakRepair(day string, now time.Time) error {
	tx, err := u.db.Begin()
	if err != nil {
		return fmt.Errorf("spend repair: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(`UPDATE user_entitlements SET qty = qty - 1, updated_at = ?
		WHERE user_id = ? AND kind = ? AND ref = 'streak_repair' AND qty > 0`,
		now.UTC().Format(time.RFC3339), u.user, economy.ProductConsumable)
	if err != nil {
		return fmt.Errorf("spend repair: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("spend repair: %w", err)
	}
	if n == 0 {
		return ErrNoEntitlement
	}

	if err := u.RepairStreak(day, now); err != nil {
		return err // rollback puts the consumable back
	}
	return tx.Commit()
}
```

`RepairStreak` (Task 5) runs on `u.db`, not on this transaction, so as written the repair would commit independently of the consumable. Fix that while implementing: extract `RepairStreak`'s body into `repairStreakTx(tx *dbtx, userID, day string, now time.Time) error`, have `RepairStreak` open its own transaction and call it, and have `SpendStreakRepair` call `repairStreakTx(tx, ...)`. Add a test that a repair rejected by the window check leaves `qty` unchanged.

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./server/internal/store/ -run 'TestPurchase|TestPromo|TestSpend' -v`
Expected: PASS.

- [ ] **Step 6: Add the overdraw race test**

Append to `server/internal/store/shop_test.go`:

```go
func TestConcurrentPurchasesCannotOverdraw(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("needs Postgres: row locking is a no-op on SQLite, which serialises writers anyway")
	}
	s := newStore(t)
	id, _ := s.CreateUser("Гонка")
	u := s.User(id)
	pid := seedProduct(t, s, economy.ProductConsumable, "streak_repair", 100, 1)
	credit(t, s, id, 100, "seed:1") // enough for exactly one

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = u.Purchase(pid, "", time.Now())
		}(i)
	}
	wg.Wait()

	okCount := 0
	for _, err := range errs {
		if err == nil {
			okCount++
		}
	}
	if okCount != 1 {
		t.Fatalf("%d of 2 concurrent purchases succeeded, want exactly 1 (errs: %v)", okCount, errs)
	}
	if bal, _ := s.Balance(id); bal != 0 {
		t.Fatalf("balance = %d, want 0 — the balance went negative or nothing was charged", bal)
	}
}
```

Import `os` and `sync`. This test is the whole reason the `FOR UPDATE` line exists, so run it against Postgres before believing the lock works: `TEST_DATABASE_URL='postgres://localhost/srpski_test?sslmode=disable' go test ./server/internal/store/ -run TestConcurrent -v`.

- [ ] **Step 7: Commit**

```bash
git add server/internal/economy/ server/internal/store/shop.go server/internal/store/shop_test.go server/internal/store/activity.go
git commit -m "Add product catalogue, promo codes and the atomic purchase transaction"
```

---

### Task 11: Phase gating

**Files:**
- Modify: `server/internal/api/api.go` (`getCourse`, `getLesson`, `getExercises`, `checkExercise`), `server/internal/api/dto.go`, `server/internal/api/api_test.go`

**Interfaces:**
- Consumes: `ListProducts`, `Entitlements` (Task 10); `economy.EffectivePrice`, `Product.SalePercent` (Tasks 3, 10).
- Produces:
  - `phaseDTO` gains `Locked bool`, `Price int64`, `PriceEffective int64`, `DiscountPercent int` (all `omitempty` except `locked`)
  - `func (h handlers) phaseAccess(us *store.UserStore) (locked map[string]bool, info map[string]phaseDTO, err error)`
  - `func (h handlers) lessonLocked(us *store.UserStore, lessonID string) (bool, error)`

Hiding a locked phase in the UI is not gating. The content must be unreachable by direct request, which means every endpoint that returns lesson content checks.

- [ ] **Step 1: Write the failing tests**

Append to `server/internal/api/api_test.go`:

```go
func TestLockedPhaseIsMarkedAndItsLessonsAre403(t *testing.T) {
	env := newTestEnv(t)
	user := env.newSessionUser(t, "Гейт")

	// Phase "2" costs 500 and the user owns nothing.
	env.seedProduct(t, "phase_unlock", "2", 500)

	res := env.do(t, user, "GET", "/api/course", "")
	if res.Code != 200 {
		t.Fatalf("course status = %d", res.Code)
	}
	var course struct {
		Phases []struct {
			ID             string `json:"id"`
			Locked         bool   `json:"locked"`
			Price          int64  `json:"price"`
			PriceEffective int64  `json:"price_effective"`
		} `json:"phases"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &course); err != nil {
		t.Fatalf("decode course: %v", err)
	}
	var found bool
	for _, p := range course.Phases {
		switch p.ID {
		case "2":
			found = true
			if !p.Locked || p.Price != 500 || p.PriceEffective != 500 {
				t.Fatalf("phase 2 = %+v; want locked with price 500", p)
			}
		case "1":
			if p.Locked {
				t.Fatal("phase 1 has no product and must stay unlocked")
			}
		}
	}
	if !found {
		t.Fatal("phase 2 missing from the course response")
	}

	// A lesson inside the locked phase must not be reachable directly.
	if res := env.do(t, user, "GET", "/api/lessons/16", ""); res.Code != 403 {
		t.Fatalf("GET /api/lessons/16 = %d, want 403", res.Code)
	}
	if res := env.do(t, user, "GET", "/api/lessons/16/exercises", ""); res.Code != 403 {
		t.Fatalf("GET exercises = %d, want 403", res.Code)
	}
	if res := env.do(t, user, "POST", "/api/lessons/16/exercises/16.1/check", `{"answer":"x"}`); res.Code != 403 {
		t.Fatalf("POST check = %d, want 403", res.Code)
	}

	// A lesson in a free phase is still fine.
	if res := env.do(t, user, "GET", "/api/lessons/00", ""); res.Code != 200 {
		t.Fatalf("GET /api/lessons/00 = %d, want 200", res.Code)
	}
}

func TestOwnedPhaseUnlocks(t *testing.T) {
	env := newTestEnv(t)
	user := env.newSessionUser(t, "Куплено")
	env.seedProduct(t, "phase_unlock", "2", 500)
	env.grantEntitlement(t, user.ID, "phase_unlock", "2")

	if res := env.do(t, user, "GET", "/api/lessons/16", ""); res.Code != 200 {
		t.Fatalf("GET /api/lessons/16 = %d, want 200 after purchase", res.Code)
	}
}

func TestPhaseWithZeroPriceIsFree(t *testing.T) {
	env := newTestEnv(t)
	user := env.newSessionUser(t, "Бесплатно")
	env.seedProduct(t, "phase_unlock", "2", 0)

	if res := env.do(t, user, "GET", "/api/lessons/16", ""); res.Code != 200 {
		t.Fatalf("GET /api/lessons/16 = %d, want 200 for a zero-price phase", res.Code)
	}
}
```

Add `seedProduct` and `grantEntitlement` helpers to the test env in whatever file defines it, writing straight into `products` / `user_entitlements` with the store's `db`. Use the real lesson ids from the test course fixture rather than `16`/`00` if the fixture differs — check `server/internal/content/testdata/` first.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./server/internal/api/ -run 'TestLockedPhase|TestOwnedPhase|TestPhaseWithZero' -v`
Expected: FAIL — lessons return 200 and `locked` is absent.

- [ ] **Step 3: Extend the DTO**

In `server/internal/api/dto.go`:

```go
type phaseDTO struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Lessons         []string `json:"lessons"`
	Locked          bool     `json:"locked"`
	Price           int64    `json:"price,omitempty"`
	PriceEffective  int64    `json:"price_effective,omitempty"`
	DiscountPercent int      `json:"discount_percent,omitempty"`
}
```

- [ ] **Step 4: Implement the access helpers and wire the four endpoints**

In `server/internal/api/api.go`:

```go
// phaseAccess reports, per phase id, whether it is locked for this user and
// what it would cost. A phase with no phase_unlock product, an inactive one,
// or one priced 0 is free — that is how phases 1..3 stay open without any
// catalogue rows at all.
func (h handlers) phaseAccess(us *store.UserStore) (map[string]bool, map[string]phaseDTO, error) {
	products, err := h.Store.ListProducts(true)
	if err != nil {
		return nil, nil, err
	}
	owned, err := us.Entitlements()
	if err != nil {
		return nil, nil, err
	}
	now := h.Now()
	locked := map[string]bool{}
	info := map[string]phaseDTO{}
	for _, p := range products {
		if p.Kind != economy.ProductPhaseUnlock || p.Price <= 0 {
			continue
		}
		if owned[economy.ProductPhaseUnlock+":"+p.Ref] > 0 {
			continue
		}
		sale := p.SalePercent(now)
		effective, _ := economy.EffectivePrice(p.Price, sale, 0)
		locked[p.Ref] = true
		info[p.Ref] = phaseDTO{
			Price:           p.Price,
			PriceEffective:  effective,
			DiscountPercent: sale,
		}
	}
	return locked, info, nil
}

// lessonLocked reports whether lessonID sits inside a phase this user has not
// unlocked.
func (h handlers) lessonLocked(us *store.UserStore, lessonID string) (bool, error) {
	locked, _, err := h.phaseAccess(us)
	if err != nil {
		return false, err
	}
	if len(locked) == 0 {
		return false, nil
	}
	for _, p := range h.Course().Phases {
		for _, id := range p.Lessons {
			if id == lessonID {
				return locked[p.ID], nil
			}
		}
	}
	return false, nil
}
```

In `getCourse`, replace the phase append with:

```go
	locked, info, err := h.phaseAccess(us)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	for _, p := range c.Phases {
		dto := phaseDTO{ID: p.ID, Title: p.Title, Lessons: p.Lessons, Locked: locked[p.ID]}
		if extra, ok := info[p.ID]; ok {
			dto.Price, dto.PriceEffective, dto.DiscountPercent = extra.Price, extra.PriceEffective, extra.DiscountPercent
		}
		out.Phases = append(out.Phases, dto)
		// ...the existing lesson loop is unchanged
	}
```

In `getLesson`, `getExercises` and `checkExercise`, immediately after the lesson is resolved and found, add:

```go
	if lockedPhase, err := h.lessonLocked(us, id); err != nil {
		fail(w, 500, err.Error())
		return
	} else if lockedPhase {
		fail(w, 403, "phase locked")
		return
	}
```

Use each handler's own variable name for the lesson id (`id` in `getLesson`, check the others). Add the `economy` import to `api.go`.

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./server/internal/api/ -run 'TestLockedPhase|TestOwnedPhase|TestPhaseWithZero' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add server/internal/api/api.go server/internal/api/dto.go server/internal/api/api_test.go
git commit -m "Gate locked phases in the course response and on every lesson endpoint"
```

---

### Task 12: HTTP endpoints

**Files:**
- Create: `server/internal/api/economy.go`, `server/internal/api/economy_test.go`
- Modify: `server/internal/api/api.go` (route registration), `server/internal/api/dto.go`

**Interfaces:**
- Consumes: everything from Tasks 2–10.
- Produces the endpoints below, all behind `requireSession` (the `X-User` bridge must never reach anything that moves currency):

| method | path | handler |
|---|---|---|
| `GET` | `/api/me/wallet` | `h.getWallet` |
| `GET` | `/api/me/quests` | `h.listQuests` |
| `POST` | `/api/me/quests/{id}/claim` | `h.claimQuest` |
| `GET` | `/api/me/transactions` | `h.listTransactions` |
| `GET` | `/api/shop` | `h.getShop` |
| `POST` | `/api/me/purchases` | `h.purchase` |
| `POST` | `/api/me/streak/repair` | `h.repairStreak` |

- [ ] **Step 1: Write the failing tests**

Create `server/internal/api/economy_test.go`:

```go
package api

import (
	"encoding/json"
	"testing"
)

func TestWalletReportsBalanceAndCurrencyName(t *testing.T) {
	env := newTestEnv(t)
	user := env.newSessionUser(t, "Кошелёк")
	env.credit(t, user.ID, 42)

	res := env.do(t, user, "GET", "/api/me/wallet", "")
	if res.Code != 200 {
		t.Fatalf("status = %d: %s", res.Code, res.Body.String())
	}
	var out struct {
		Balance      int64  `json:"balance"`
		CurrencyMany string `json:"currency_many"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Balance != 42 {
		t.Fatalf("balance = %d, want 42", out.Balance)
	}
	if out.CurrencyMany != "монет" {
		t.Fatalf("currency_many = %q, want монет", out.CurrencyMany)
	}
}

func TestQuestListAndClaimFlow(t *testing.T) {
	env := newTestEnv(t)
	user := env.newSessionUser(t, "Квест")
	qid := env.seedQuest(t, "lessons_completed", 1, 30)

	// Nothing completed yet: listed, not done, not claimable.
	res := env.do(t, user, "GET", "/api/me/quests", "")
	var list struct {
		Quests []struct {
			ID      int64 `json:"id"`
			Value   int   `json:"value"`
			Target  int   `json:"target"`
			Done    bool  `json:"done"`
			Claimed bool  `json:"claimed"`
		} `json:"quests"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list.Quests) != 1 || list.Quests[0].Done {
		t.Fatalf("quests = %+v; want one, not done", list.Quests)
	}

	// Claiming an unfinished quest must be refused and must not credit.
	if res := env.do(t, user, "POST", "/api/me/quests/"+itoa64(qid)+"/claim", ""); res.Code != 409 {
		t.Fatalf("premature claim = %d, want 409", res.Code)
	}
	if bal := env.balance(t, user.ID); bal != 0 {
		t.Fatalf("balance = %d after a refused claim, want 0", bal)
	}

	env.completeLesson(t, user.ID, "00")

	if res := env.do(t, user, "POST", "/api/me/quests/"+itoa64(qid)+"/claim", ""); res.Code != 200 {
		t.Fatalf("claim = %d, want 200", res.Code)
	}
	if bal := env.balance(t, user.ID); bal != 30 {
		t.Fatalf("balance = %d, want 30", bal)
	}
	// Second claim is a no-op, not a second credit.
	if res := env.do(t, user, "POST", "/api/me/quests/"+itoa64(qid)+"/claim", ""); res.Code != 409 {
		t.Fatalf("repeat claim = %d, want 409", res.Code)
	}
	if bal := env.balance(t, user.ID); bal != 30 {
		t.Fatalf("balance = %d after a repeat claim, want 30", bal)
	}
}

func TestPurchaseEndpointReportsInsufficientFunds(t *testing.T) {
	env := newTestEnv(t)
	user := env.newSessionUser(t, "Мало")
	pid := env.seedProductID(t, "consumable", "streak_repair", 25)

	res := env.do(t, user, "POST", "/api/me/purchases", `{"product_id":`+itoa64(pid)+`}`)
	if res.Code != 409 {
		t.Fatalf("status = %d, want 409: %s", res.Code, res.Body.String())
	}
	var out struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(res.Body.Bytes(), &out)
	if out.Error != "insufficient_funds" {
		t.Fatalf("error = %q, want insufficient_funds", out.Error)
	}
}

func TestCurrencyEndpointsRejectTheLegacyHeaderBridge(t *testing.T) {
	env := newTestEnv(t)
	// A request carrying only X-User must not reach anything that moves money.
	for _, path := range []string{"/api/me/wallet", "/api/me/quests", "/api/me/transactions"} {
		if res := env.doWithLegacyHeader(t, "Гриша", "GET", path, ""); res.Code != 401 {
			t.Fatalf("%s with X-User = %d, want 401", path, res.Code)
		}
	}
}
```

Add the helpers this file uses (`credit`, `seedQuest`, `seedProductID`, `completeLesson`, `balance`, `doWithLegacyHeader`, `itoa64`) next to the existing test env. The last test matters: `requireAuth` still honours `X-User` for legacy content routes, and a currency endpoint accidentally registered there would let anyone spend anyone's balance by setting a header.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./server/internal/api/ -run 'TestWallet|TestQuest|TestPurchaseEndpoint|TestCurrencyEndpoints' -v`
Expected: FAIL — 404 on every new path.

- [ ] **Step 3: Implement the handlers**

Create `server/internal/api/economy.go`:

```go
package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/grisha/serbian-app/server/internal/economy"
	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
)

type walletDTO struct {
	Balance      int64  `json:"balance"`
	CurrencyOne  string `json:"currency_one"`
	CurrencyFew  string `json:"currency_few"`
	CurrencyMany string `json:"currency_many"`
	StreakDays   int    `json:"streak_days"`
}

type questDTO struct {
	ID          int64  `json:"id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Target      int    `json:"target"`
	Value       int    `json:"value"`
	Reward      int64  `json:"reward"`
	Done        bool   `json:"done"`
	Claimed     bool   `json:"claimed"`
}

type shopItemDTO struct {
	ID              int64  `json:"id"`
	Kind            string `json:"kind"`
	Ref             string `json:"ref"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	Price           int64  `json:"price"`
	PriceEffective  int64  `json:"price_effective"`
	DiscountPercent int    `json:"discount_percent,omitempty"`
	Owned           int64  `json:"owned"`
}

type transactionDTO struct {
	ID        int64  `json:"id"`
	Amount    int64  `json:"amount"`
	Kind      string `json:"kind"`
	Ref       string `json:"ref"`
	Comment   string `json:"comment"`
	CreatedAt string `json:"created_at"`
}

func (h handlers) getWallet(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	set, err := h.Store.EconomySettings()
	if err != nil {
		log.Printf("wallet: settings: %v", err)
		fail(w, 500, "internal error")
		return
	}
	bal, err := h.Store.Balance(ac.UserID)
	if err != nil {
		log.Printf("wallet: balance: %v", err)
		fail(w, 500, "internal error")
		return
	}
	streak, err := h.Store.User(ac.UserID).StreakDays(h.Now())
	if err != nil {
		log.Printf("wallet: streak: %v", err)
		fail(w, 500, "internal error")
		return
	}
	writeJSON(w, 200, walletDTO{
		Balance:      bal,
		CurrencyOne:  set.CurrencyNameOne,
		CurrencyFew:  set.CurrencyNameFew,
		CurrencyMany: set.CurrencyNameMany,
		StreakDays:   streak,
	})
}

// phaseLessons maps phase id to its lesson ids, for phase_completed quests.
func (h handlers) phaseLessons() map[string][]string {
	out := map[string][]string{}
	for _, p := range h.Course().Phases {
		out[p.ID] = p.Lessons
	}
	return out
}

// questCounters gathers the local counters and, only if some active quest
// needs it, the Telegram subscription flag.
func (h handlers) questCounters(userID string, quests []economy.Quest) (economy.Counters, error) {
	us := h.Store.User(userID)
	c, err := us.QuestCounters(h.Now())
	if err != nil {
		return c, err
	}
	needsTelegram := false
	for _, q := range quests {
		if q.Kind == economy.QuestTelegramSubscribed {
			needsTelegram = true
			break
		}
	}
	if !needsTelegram {
		return c, nil
	}
	c.TelegramSubscribed = h.telegramSubscribed(userID)
	return c, nil
}

// telegramSubscribed answers the channel-membership question, degrading to
// false on every failure: an unreachable Bot API must leave the profile
// screen working, just with that one quest unfinished.
func (h handlers) telegramSubscribed(userID string) bool {
	set, err := h.Store.EconomySettings()
	if err != nil || set.TelegramChannel == "" {
		return false
	}
	chatID, ok := h.Store.TelegramChatID(userID)
	if !ok {
		return false
	}
	status, err := telegram.GetChatMember(h.Config.TelegramBotToken, set.TelegramChannel, chatID)
	if err != nil {
		log.Printf("quests: getChatMember: %v", err)
		return false
	}
	return telegram.IsMember(status)
}

func (h handlers) listQuests(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	quests, err := h.Store.ListQuests(true)
	if err != nil {
		log.Printf("quests: list: %v", err)
		fail(w, 500, "internal error")
		return
	}
	// An unconfigured channel must not show a quest nobody can finish.
	set, _ := h.Store.EconomySettings()
	filtered := quests[:0]
	for _, q := range quests {
		if q.Kind == economy.QuestTelegramSubscribed && q.Param == "" && set.TelegramChannel == "" {
			continue
		}
		filtered = append(filtered, q)
	}
	quests = filtered

	counters, err := h.questCounters(ac.UserID, quests)
	if err != nil {
		log.Printf("quests: counters: %v", err)
		fail(w, 500, "internal error")
		return
	}
	claimed, err := h.Store.User(ac.UserID).ClaimedQuestIDs()
	if err != nil {
		log.Printf("quests: claimed: %v", err)
		fail(w, 500, "internal error")
		return
	}
	phases := h.phaseLessons()
	out := struct {
		Quests []questDTO `json:"quests"`
	}{Quests: []questDTO{}}
	for _, q := range quests {
		out.Quests = append(out.Quests, questDTO{
			ID: q.ID, Kind: q.Kind, Title: q.Title, Description: q.Description,
			Target: q.Target, Value: economy.QuestValue(q, counters, phases),
			Reward: q.Reward, Done: economy.QuestDone(q, counters, phases),
			Claimed: claimed[q.ID],
		})
	}
	writeJSON(w, 200, out)
}

func (h handlers) claimQuest(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, 400, "bad quest id")
		return
	}
	quests, err := h.Store.ListQuests(true)
	if err != nil {
		log.Printf("claim: list: %v", err)
		fail(w, 500, "internal error")
		return
	}
	var quest economy.Quest
	for _, q := range quests {
		if q.ID == id {
			quest = q
			break
		}
	}
	if quest.ID == 0 {
		fail(w, 404, "unknown quest")
		return
	}
	counters, err := h.questCounters(ac.UserID, []economy.Quest{quest})
	if err != nil {
		log.Printf("claim: counters: %v", err)
		fail(w, 500, "internal error")
		return
	}
	// Completion is re-checked here, on the server, every time. The client's
	// opinion that a quest is done is never trusted.
	if !economy.QuestDone(quest, counters, h.phaseLessons()) {
		fail(w, 409, "not_completed")
		return
	}
	if err := h.Store.User(ac.UserID).ClaimQuest(quest, h.Now()); err != nil {
		if errors.Is(err, store.ErrAlreadyClaimed) {
			fail(w, 409, "already_claimed")
			return
		}
		log.Printf("claim: %v", err)
		fail(w, 500, "internal error")
		return
	}
	bal, _ := h.Store.Balance(ac.UserID)
	writeJSON(w, 200, map[string]any{"reward": quest.Reward, "balance": bal})
}

func (h handlers) listTransactions(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	limit, offset := pageParams(r, 50, 200)
	rows, err := h.Store.ListLedger(ac.UserID, limit, offset)
	if err != nil {
		log.Printf("transactions: %v", err)
		fail(w, 500, "internal error")
		return
	}
	out := struct {
		Transactions []transactionDTO `json:"transactions"`
	}{Transactions: []transactionDTO{}}
	for _, e := range rows {
		out.Transactions = append(out.Transactions, transactionDTO{
			ID: e.ID, Amount: e.Amount, Kind: e.Kind, Ref: e.Ref,
			Comment: e.Comment, CreatedAt: e.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, 200, out)
}

func (h handlers) getShop(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	products, err := h.Store.ListProducts(true)
	if err != nil {
		log.Printf("shop: %v", err)
		fail(w, 500, "internal error")
		return
	}
	owned, err := h.Store.User(ac.UserID).Entitlements()
	if err != nil {
		log.Printf("shop: entitlements: %v", err)
		fail(w, 500, "internal error")
		return
	}
	now := h.Now()
	out := struct {
		Items []shopItemDTO `json:"items"`
	}{Items: []shopItemDTO{}}
	for _, p := range products {
		sale := p.SalePercent(now)
		effective, _ := economy.EffectivePrice(p.Price, sale, 0)
		out.Items = append(out.Items, shopItemDTO{
			ID: p.ID, Kind: p.Kind, Ref: p.Ref, Title: p.Title, Description: p.Description,
			Price: p.Price, PriceEffective: effective, DiscountPercent: sale,
			Owned: owned[p.Kind+":"+p.Ref],
		})
	}
	writeJSON(w, 200, out)
}

func (h handlers) purchase(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	var req struct {
		ProductID int64  `json:"product_id"`
		PromoCode string `json:"promo_code"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, 400, "bad request body")
		return
	}
	res, err := h.Store.User(ac.UserID).Purchase(req.ProductID, req.PromoCode, h.Now())
	switch {
	case err == nil:
	case errors.Is(err, store.ErrInsufficientFunds):
		fail(w, 409, "insufficient_funds")
		return
	case errors.Is(err, store.ErrAlreadyOwned):
		fail(w, 409, "already_owned")
		return
	case errors.Is(err, store.ErrProductUnavailable):
		fail(w, 404, "product_unavailable")
		return
	case errors.Is(err, store.ErrPromoInvalid):
		fail(w, 400, "promo_invalid")
		return
	default:
		log.Printf("purchase: %v", err)
		fail(w, 500, "internal error")
		return
	}
	bal, _ := h.Store.Balance(ac.UserID)
	writeJSON(w, 200, map[string]any{"paid": res.Paid, "applied": res.Applied, "balance": bal})
}

func (h handlers) repairStreak(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	var req struct {
		Day string `json:"day"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, 400, "bad request body")
		return
	}
	if err := h.Store.User(ac.UserID).SpendStreakRepair(req.Day, h.Now()); err != nil {
		if errors.Is(err, store.ErrNoEntitlement) {
			fail(w, 409, "no_repair_available")
			return
		}
		fail(w, 400, err.Error())
		return
	}
	streak, _ := h.Store.User(ac.UserID).StreakDays(h.Now())
	writeJSON(w, 200, map[string]any{"streak_days": streak})
}
```

Two things to resolve while implementing, not to guess at:

- `h.Store.TelegramChatID(userID)` may not exist. Find how the repo already resolves a user's Telegram chat id (look in `internal/store/identities.go` and at how reminders pick their recipients) and use that; add a thin accessor only if there genuinely is none.
- `pageParams` may not exist either. Reuse whatever `listNotifications` or the leaderboard pagination already does rather than adding a second convention.

- [ ] **Step 4: Register the routes**

In `server/internal/api/api.go`, add to the `requireSession` block — **not** to the `protected` mux:

```go
	root.HandleFunc("GET /api/me/wallet", h.requireSession(h.getWallet))
	root.HandleFunc("GET /api/me/quests", h.requireSession(h.listQuests))
	root.HandleFunc("POST /api/me/quests/{id}/claim", h.requireSession(h.claimQuest))
	root.HandleFunc("GET /api/me/transactions", h.requireSession(h.listTransactions))
	root.HandleFunc("GET /api/shop", h.requireSession(h.getShop))
	root.HandleFunc("POST /api/me/purchases", h.requireSession(h.purchase))
	root.HandleFunc("POST /api/me/streak/repair", h.requireSession(h.repairStreak))
```

`/api/shop` is account-scoped (it reports what you own), so it belongs here despite the path not starting with `/api/me`.

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./server/internal/api/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add server/internal/api/economy.go server/internal/api/economy_test.go server/internal/api/api.go server/internal/api/dto.go
git commit -m "Add wallet, quest, shop, purchase and streak-repair endpoints"
```

---

### Task 13: Seed the starting economy and verify end to end

**Files:**
- Create: `server/internal/store/seed.go`
- Modify: `server/internal/store/currency_test.go`, `server/main.go`

**Interfaces:**
- Consumes: `quests`, `products` and `economy_settings` from Task 1.
- Produces: `func (s *Store) SeedEconomyDefaults(now time.Time) error` — idempotent, called once from `server/main.go` at startup, so the feature is live on deploy without anyone opening the admin panel.

**Why a function and not a migration** (controller ruling, recorded in the SDD ledger): seeding through a migration puts 19 quests and 3 products into *every* store test. On SQLite each test opens a fresh in-memory database, migrations run, and the seeds are there — which breaks Task 8's `TestListQuestsRespectsActiveFlagAndOrder` (it asserts exactly 2 quests) and Task 10's product tests. On Postgres `newStore`'s TRUNCATE would wipe the seeds instead, so the same tests would behave differently on the two backends. An explicit call keeps tests on empty tables, keeps production seeded on first boot, and behaves identically on both.

- [ ] **Step 1: Write the failing test**

Append to `server/internal/store/currency_test.go`:

```go
func TestSeededEconomyMatchesTheDesign(t *testing.T) {
	s := newStore(t)
	now := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	if err := s.SeedEconomyDefaults(now); err != nil {
		t.Fatalf("seed: %v", err)
	}

	quests, err := s.ListQuests(true)
	if err != nil {
		t.Fatalf("list quests: %v", err)
	}
	var pool int64
	for _, q := range quests {
		pool += q.Reward
	}
	if pool != 638 {
		t.Fatalf("quest pool = %d, want 638 (see the design doc's economy table)", pool)
	}

	products, err := s.ListProducts(true)
	if err != nil {
		t.Fatalf("list products: %v", err)
	}
	want := map[string]int64{
		"phase_unlock:4":            500,
		"phase_unlock:5":            1000,
		"consumable:streak_repair":  25,
	}
	got := map[string]int64{}
	for _, p := range products {
		got[p.Kind+":"+p.Ref] = p.Price
	}
	for key, price := range want {
		if got[key] != price {
			t.Errorf("%s price = %d, want %d", key, got[key], price)
		}
	}

	// Level 4 must be reachable on quests alone — that is the whole promise
	// of "100% + все задания".
	if got["phase_unlock:4"] > pool {
		t.Fatalf("Level 4 costs %d but the entire quest pool is %d", got["phase_unlock:4"], pool)
	}
	// Level 5 must NOT be, otherwise the streak has no purpose.
	if got["phase_unlock:5"] <= pool {
		t.Fatalf("Level 5 costs %d, which the quest pool (%d) already covers", got["phase_unlock:5"], pool)
	}
}

func TestSeedIsIdempotent(t *testing.T) {
	s := newStore(t)
	now := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		if err := s.SeedEconomyDefaults(now); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	quests, _ := s.ListQuests(false)
	if len(quests) != 19 {
		t.Fatalf("%d quests after three seed runs, want 19", len(quests))
	}
}

func TestSeedDoesNotResurrectDeletedQuests(t *testing.T) {
	s := newStore(t)
	now := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	if err := s.SeedEconomyDefaults(now); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.db.Exec(`DELETE FROM quests`); err != nil {
		t.Fatalf("delete quests: %v", err)
	}
	if err := s.SeedEconomyDefaults(now); err != nil {
		t.Fatalf("re-seed: %v", err)
	}
	quests, _ := s.ListQuests(false)
	if len(quests) != 0 {
		t.Fatalf("%d quests came back after being deleted in the admin; the seed guard did not hold", len(quests))
	}
}
```

The third test is the one that matters operationally: once the owner curates
the quest list in the admin panel, a restart must not undo their work.

These last two assertions are the economy's shape expressed as a test: if someone later edits the seeds and breaks the relationship between the quest pool and the two prices, this fails loudly.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./server/internal/store/ -run TestSeededEconomy -v`
Expected: FAIL — quest pool is 0.

- [ ] **Step 3: Write the seeding function**

Create `server/internal/store/seed.go`. The whole thing runs in one
transaction and writes the guard row **last**, so a boot that dies mid-seed
leaves nothing marked done and the next boot retries cleanly:

```go
package store

import (
	"errors"
	"database/sql"
	"fmt"
	"time"
)

// seededKey marks the economy defaults as already written. Checked and set
// inside SeedEconomyDefaults' own transaction, so deleting every quest in the
// admin panel does not resurrect them on the next restart.
const seededKey = "economy_seeded"

// SeedEconomyDefaults writes the starting quest list and product catalogue
// from docs/superpowers/specs/2026-09-28-currency-design.md, once. Called
// from main.go at startup; tests call it explicitly, which is why it is not
// a migration (see the plan's Task 13 for the reasoning).
//
// Quest rewards total 638, which covers Level 4 (500) and deliberately does
// not cover Level 5 (1000): Level 5 is what the daily streak drip is for.
// Every number here is editable from the admin panel afterwards.
func (s *Store) SeedEconomyDefaults(now time.Time) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("seed economy: %w", err)
	}
	defer tx.Rollback()

	var marker string
	err = tx.QueryRow(`SELECT value FROM economy_settings WHERE key = ?`, seededKey).Scan(&marker)
	if err == nil {
		return nil // already seeded
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("seed economy: check marker: %w", err)
	}

	ts := now.UTC().Format(time.RFC3339)
	type quest struct {
		kind        string
		target      int
		param       string
		title       string
		description string
		reward      int64
		sortOrder   int
	}
	quests := []quest{
		{"telegram_subscribed", 1, "", "Подписаться на канал", "Подпишись на наш Telegram-канал", 15, 10},
		{"lessons_completed", 5, "", "Пройти 5 уроков", "", 5, 20},
		{"lessons_completed", 10, "", "Пройти 10 уроков", "", 10, 21},
		{"lessons_completed", 20, "", "Пройти 20 уроков", "", 20, 22},
		{"lessons_completed", 30, "", "Пройти 30 уроков", "", 30, 23},
		{"vocab_learned", 30, "", "Выучить 30 слов", "", 10, 30},
		{"vocab_learned", 100, "", "Выучить 100 слов", "", 25, 31},
		{"vocab_learned", 300, "", "Выучить 300 слов", "", 50, 32},
		{"reviews_done", 100, "", "Сделать 100 повторений", "", 10, 40},
		{"reviews_done", 500, "", "Сделать 500 повторений", "", 30, 41},
		{"correct_in_row", 5, "", "5 правильных подряд", "", 3, 50},
		{"correct_in_row", 10, "", "10 правильных подряд", "", 5, 51},
		{"correct_in_row", 20, "", "20 правильных подряд", "", 15, 52},
		{"streak_days", 7, "", "Стрик 7 дней", "Занимайся 7 дней подряд", 10, 60},
		{"streak_days", 30, "", "Стрик 30 дней", "Занимайся 30 дней подряд", 40, 61},
		{"streak_days", 100, "", "Стрик 100 дней", "Занимайся 100 дней подряд", 150, 62},
		{"phase_completed", 100, "1", "Уровень 1 на 100%", "Пройди все уроки первого уровня", 50, 70},
		{"phase_completed", 100, "2", "Уровень 2 на 100%", "Пройди все уроки второго уровня", 70, 71},
		{"phase_completed", 100, "3", "Уровень 3 на 100%", "Пройди все уроки третьего уровня", 90, 72},
	}
	for _, q := range quests {
		if _, err := tx.Exec(`INSERT INTO quests
			(kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?, ?)`,
			q.kind, q.target, q.param, q.title, q.description, q.reward, q.sortOrder, ts, ts); err != nil {
			return fmt.Errorf("seed quest %q: %w", q.title, err)
		}
	}

	type product struct {
		kind        string
		ref         string
		title       string
		description string
		price       int64
		sortOrder   int
	}
	products := []product{
		{"phase_unlock", "4", "Уровень 4 — Мнения и жизнь", "Открывает уровень A2.2", 500, 10},
		{"phase_unlock", "5", "Уровень 5 — Уверенно", "Открывает уровень B1.1", 1000, 11},
		{"consumable", "streak_repair", "Восстановить стрик", "Вернёт сгоревший стрик в течение 48 часов", 25, 20},
	}
	for _, p := range products {
		if _, err := tx.Exec(`INSERT INTO products
			(kind, ref, title, description, price, discount_percent, discount_from, discount_to,
			 grant_qty, active, sort_order, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, 0, '', '', 1, 1, ?, ?, ?)
			ON CONFLICT (kind, ref) DO NOTHING`,
			p.kind, p.ref, p.title, p.description, p.price, p.sortOrder, ts, ts); err != nil {
			return fmt.Errorf("seed product %s:%s: %w", p.kind, p.ref, err)
		}
	}

	// Written last: a boot that dies mid-seed marks nothing done.
	if _, err := tx.Exec(`INSERT INTO economy_settings (key, value, updated_at) VALUES (?, '1', ?)`,
		seededKey, ts); err != nil {
		return fmt.Errorf("seed economy: mark done: %w", err)
	}
	return tx.Commit()
}
```

Then call it once from `server/main.go`, right after the store opens and
before the HTTP server starts, logging and continuing on error rather than
refusing to boot — a failed seed must not take the site down:

```go
	if err := st.SeedEconomyDefaults(time.Now()); err != nil {
		log.Printf("seed economy defaults: %v", err)
	}
```

Palettes are left out on purpose: the app has no shop screen yet, and a
cosmetic nobody can see is a row that will drift out of date before it is
ever used. Add them in the iteration that builds the shop UI.

<details>
<summary>The same data as SQL, for reference if you prefer a different shape</summary>

```sql
INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at) VALUES
	('telegram_subscribed', 1,   '',  'Подписаться на канал',        'Подпишись на наш Telegram-канал',        15,  1,  10, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('lessons_completed',   5,   '',  'Пройти 5 уроков',             '',                                       5,   1,  20, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('lessons_completed',   10,  '',  'Пройти 10 уроков',            '',                                       10,  1,  21, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('lessons_completed',   20,  '',  'Пройти 20 уроков',            '',                                       20,  1,  22, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('lessons_completed',   30,  '',  'Пройти 30 уроков',            '',                                       30,  1,  23, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('vocab_learned',       30,  '',  'Выучить 30 слов',             '',                                       10,  1,  30, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('vocab_learned',       100, '',  'Выучить 100 слов',            '',                                       25,  1,  31, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('vocab_learned',       300, '',  'Выучить 300 слов',            '',                                       50,  1,  32, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('reviews_done',        100, '',  'Сделать 100 повторений',      '',                                       10,  1,  40, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('reviews_done',        500, '',  'Сделать 500 повторений',      '',                                       30,  1,  41, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('correct_in_row',      5,   '',  '5 правильных подряд',         '',                                       3,   1,  50, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('correct_in_row',      10,  '',  '10 правильных подряд',        '',                                       5,   1,  51, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('correct_in_row',      20,  '',  '20 правильных подряд',        '',                                       15,  1,  52, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('streak_days',         7,   '',  'Стрик 7 дней',                'Занимайся 7 дней подряд',                10,  1,  60, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('streak_days',         30,  '',  'Стрик 30 дней',               'Занимайся 30 дней подряд',               40,  1,  61, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('streak_days',         100, '',  'Стрик 100 дней',              'Занимайся 100 дней подряд',              150, 1,  62, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('phase_completed',     100, '1', 'Уровень 1 на 100%',           'Пройди все уроки первого уровня',        50,  1,  70, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('phase_completed',     100, '2', 'Уровень 2 на 100%',           'Пройди все уроки второго уровня',        70,  1,  71, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('phase_completed',     100, '3', 'Уровень 3 на 100%',           'Пройди все уроки третьего уровня',       90,  1,  72, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z');

INSERT INTO products (kind, ref, title, description, price, discount_percent, discount_from, discount_to, grant_qty, active, sort_order, created_at, updated_at) VALUES
	('phase_unlock', '4',             'Уровень 4 — Мнения и жизнь',  'Открывает уровень A2.2',                 500,  0, '', '', 1, 1, 10, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('phase_unlock', '5',             'Уровень 5 — Уверенно',        'Открывает уровень B1.1',                 1000, 0, '', '', 1, 1, 11, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z'),
	('consumable',   'streak_repair', 'Восстановить стрик',          'Вернёт сгоревший стрик в течение 48 часов', 25, 0, '', '', 1, 1, 20, '2026-09-29T00:00:00Z', '2026-09-29T00:00:00Z')
ON CONFLICT (kind, ref) DO NOTHING;
```

</details>

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./server/internal/store/ -run TestSeededEconomy -v`
Expected: PASS.

Note the seed makes phases 4 and 5 **locked** from the first boot after this deploy. That is correct and currently invisible: neither has any lesson content. Verify in the same run that `GET /api/course` still returns phases 1–3 unlocked.

- [ ] **Step 5: Run everything, both backends, both languages**

Run: `go test ./server/...`
Run: `make test-pg`
Run: `cd web && npx vitest run && npx vue-tsc -b`
Expected: PASS. The frontend still compiles because `phaseDTO` only gained fields; if a Vue test asserted the old streak semantics, fix the assertion to the new definition.

- [ ] **Step 6: Build the binary**

Run: `make build`
Expected: succeeds. Confirm tzdata is available in the built image (Task 4, Step 3) — `SetUserTimezone` silently rejecting every timezone would leave every user on `Europe/Belgrade` with no visible error.

- [ ] **Step 7: Commit**

```bash
git add server/internal/store/seed.go server/internal/store/currency_test.go server/main.go
git commit -m "Seed the starting quest list and product catalogue on first boot"
```

---

## Done criteria

- `go test ./server/...` and `make test-pg` pass.
- `make build` produces a binary, and a fresh SQLite database migrates from empty to head without error.
- An existing database migrates without losing streaks: run the binary against a copy of a real dump and spot-check a few `user_daily_activity` rows against that user's `attempts`/`reviews`.
- No balance column exists anywhere: `grep -rn "balance" server/internal/store/migrations/` returns nothing.
- Currency endpoints reject the `X-User` bridge (Task 12's last test).

## Deploy

Out of scope for this plan — see the design doc's "Deploy order". In short: this service first, then the `GRANT` block run by the account owner, then `ucimo-content-admin`. Do not deploy the admin service before the grants exist.
