package store

import (
	"errors"
	"os"
	"sync"
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

	res, err := u.Purchase(pid, "", "", now)
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

	_, err := u.Purchase(pid, "", "", time.Now())
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

	if _, err := u.Purchase(pid, "", "", time.Now()); err != nil {
		t.Fatalf("first purchase: %v", err)
	}
	if _, err := u.Purchase(pid, "", "", time.Now()); !errors.Is(err, ErrAlreadyOwned) {
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
		if _, err := u.Purchase(pid, "", "", time.Now()); err != nil {
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

// A retried request must not charge twice. Buying a consumable three times is
// three purchases; the same purchase arriving three times because the client
// lost the response is one. The caller's key is what tells them apart.
func TestPurchaseWithTheSameKeyChargesOnce(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Повтор")
	u := s.User(id)
	pid := seedProduct(t, s, economy.ProductConsumable, "streak_repair", 25, 1)
	credit(t, s, id, 100, "seed:1")
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

	first, err := u.Purchase(pid, "", "req-1", now)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if first.Replayed {
		t.Fatal("the first purchase reported itself as a replay")
	}
	again, err := u.Purchase(pid, "", "req-1", now)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if !again.Replayed {
		t.Fatal("the retry was not reported as a replay")
	}
	if again.Paid != first.Paid || again.LedgerID != first.LedgerID {
		t.Fatalf("retry = %+v, want the first result %+v", again, first)
	}
	if bal, _ := s.Balance(id); bal != 75 {
		t.Fatalf("balance = %d, want 75 (charged once)", bal)
	}
	ent, _ := u.Entitlements()
	if ent["consumable:streak_repair"] != 1 {
		t.Fatalf("qty = %d, want 1 — the retry granted a second unit", ent["consumable:streak_repair"])
	}

	// A different key is a genuinely new purchase.
	if _, err := u.Purchase(pid, "", "req-2", now); err != nil {
		t.Fatalf("second purchase: %v", err)
	}
	if bal, _ := s.Balance(id); bal != 50 {
		t.Fatalf("balance = %d, want 50", bal)
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

	res, err := u.Purchase(pid, "osen", "", now) // lower case on purpose: codes are case-insensitive
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}
	if res.Paid != 400 || res.Applied != "sale" {
		t.Fatalf("paid, applied = %d, %q; want 400, \"sale\"", res.Paid, res.Applied)
	}

	// The code lost to the sale, so it must not have been spent: a one-per-user
	// code the user never benefited from has to stay usable.
	var redemptions int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM promo_redemptions WHERE user_id = ?`, id).Scan(&redemptions); err != nil {
		t.Fatalf("count redemptions: %v", err)
	}
	if redemptions != 0 {
		t.Fatalf("%d redemptions recorded for a code that was not applied, want 0", redemptions)
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

	res, err := u.Purchase(pid, "ONCE", "", now)
	if err != nil || res.Paid != 50 {
		t.Fatalf("first use: paid %d, err %v; want 50, nil", res.Paid, err)
	}
	if _, err := u.Purchase(pid, "ONCE", "", now); !errors.Is(err, ErrPromoInvalid) {
		t.Fatalf("second use err = %v, want ErrPromoInvalid (max_per_user is 1)", err)
	}

	// An expired code is rejected too.
	if _, err := s.db.Exec(`INSERT INTO promo_codes
		(code, discount_percent, scope, valid_from, valid_to, max_redemptions, max_per_user, active, created_at, updated_at)
		VALUES ('OLD', 50, 'all', '2026-01-01', '2026-02-01', 0, 99, 1, ?, ?)`, iso, iso); err != nil {
		t.Fatalf("seed expired promo: %v", err)
	}
	if _, err := u.Purchase(pid, "OLD", "", now); !errors.Is(err, ErrPromoInvalid) {
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

	if _, err := u.Purchase(pid, "", "", now); err != nil {
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

// A repair the window rules reject must not eat the consumable. This is why
// SpendStreakRepair and the repair itself share one transaction.
func TestSpendStreakRepairKeepsTheUnitWhenTheRepairIsRefused(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Отказ")
	u := s.User(id)
	pid := seedProduct(t, s, economy.ProductConsumable, "streak_repair", 25, 1)
	credit(t, s, id, 100, "seed:1")
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, loc)
	if _, err := u.Purchase(pid, "", "", now); err != nil {
		t.Fatalf("buy repair: %v", err)
	}

	// Today is not a missed day, so the repair is refused.
	if err := u.SpendStreakRepair("2026-09-29", now); err == nil {
		t.Fatal("repairing today was accepted")
	}
	ent, _ := u.Entitlements()
	if ent["consumable:streak_repair"] != 1 {
		t.Fatalf("qty = %d after a refused repair, want 1", ent["consumable:streak_repair"])
	}

	// Far outside the 48h window, same guarantee.
	if err := u.SpendStreakRepair("2026-09-01", now); err == nil {
		t.Fatal("repairing a day outside the window was accepted")
	}
	ent, _ = u.Entitlements()
	if ent["consumable:streak_repair"] != 1 {
		t.Fatalf("qty = %d after a refused repair, want 1", ent["consumable:streak_repair"])
	}
}

func TestConcurrentPurchasesCannotOverdraw(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("needs Postgres: row locking is a no-op on SQLite, which serialises writers anyway")
	}
	s := newStore(t)
	id, _ := s.CreateUser("Гонка")
	u := s.User(id)
	pid := seedProduct(t, s, economy.ProductConsumable, "streak_repair", 100, 1)
	credit(t, s, id, 100, "seed:1") // enough for exactly one

	// Park both transactions in the window between reading the balance and
	// writing the debit. Whoever arrives waits for the other; the timeout is
	// what lets this pass when the lock works and the second transaction is
	// still blocked and never arrives at all. Without this the goroutines
	// finish too fast to interleave and the test passes even with no lock.
	var mu sync.Mutex
	arrived := 0
	both := make(chan struct{})
	purchaseBalanceHook = func() {
		mu.Lock()
		arrived++
		if arrived == 2 {
			close(both)
		}
		mu.Unlock()
		select {
		case <-both:
		case <-time.After(500 * time.Millisecond):
		}
	}
	t.Cleanup(func() { purchaseBalanceHook = nil })

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = u.Purchase(pid, "", "", time.Now())
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
