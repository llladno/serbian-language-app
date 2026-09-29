package store

import (
	"errors"
	"testing"
	"time"
)

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

	ledgerID, err := s.AddLedgerEntry(LedgerEntry{
		UserID: id, Amount: 30, Kind: "quest_reward", Ref: "7",
		IdempotencyKey: "quest:7:" + id,
	}, now)
	if err != nil {
		t.Fatalf("first credit: %v", err)
	}
	// promo_redemptions.ledger_id (a later task) stores this id, so it must
	// be a real row id, not a zero value silently returned on error.
	if ledgerID == 0 {
		t.Fatal("AddLedgerEntry returned id = 0, want a real row id")
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

// TestLedgerDuplicateLeavesTransactionUsable is the regression test for I2:
// on Postgres, a driver-level unique-violation error poisons the whole
// enclosing transaction, so any statement after it (including COMMIT) would
// fail with "current transaction is aborted" unless the duplicate is
// reported without failing the INSERT itself (ON CONFLICT DO NOTHING). This
// is exactly the pattern a quest-claim/purchase handler relies on: catch
// ErrDuplicateEntry, keep working on the same tx, commit successfully.
func TestLedgerDuplicateLeavesTransactionUsable(t *testing.T) {
	s := newStore(t)
	id, err := s.CreateUser("Транза")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

	tx, err := s.db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}

	if _, err := addLedgerEntryTx(tx, LedgerEntry{
		UserID: id, Amount: 30, Kind: "quest_reward", Ref: "7",
		IdempotencyKey: "quest:7:" + id,
	}, now); err != nil {
		tx.Rollback()
		t.Fatalf("first credit: %v", err)
	}

	if _, err := addLedgerEntryTx(tx, LedgerEntry{
		UserID: id, Amount: 30, Kind: "quest_reward", Ref: "7",
		IdempotencyKey: "quest:7:" + id,
	}, now); !errors.Is(err, ErrDuplicateEntry) {
		tx.Rollback()
		t.Fatalf("second credit err = %v; want ErrDuplicateEntry", err)
	}

	// More work on the same tx after the duplicate — this is the part that
	// fails on Postgres if the duplicate poisoned the transaction.
	if _, err := addLedgerEntryTx(tx, LedgerEntry{
		UserID: id, Amount: 7, Kind: "bonus", Ref: "1",
		IdempotencyKey: "bonus:" + id + ":1",
	}, now); err != nil {
		tx.Rollback()
		t.Fatalf("follow-up entry on same tx: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit after duplicate: %v", err)
	}

	bal, err := s.Balance(id)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	if bal != 37 {
		t.Fatalf("balance = %d, want 37 (30 + 7, duplicate not double-counted)", bal)
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
