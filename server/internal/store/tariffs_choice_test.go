package store

import (
	"sync"
	"testing"
	"time"
)

// A level payment made without a level (the learner chooses after paying) is a
// credit: it pays the bonus and counts as a paid level, but opens nothing yet.
func TestAPaymentWithoutALevelPaysTheBonusAndOpensNothing(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	id := newPayment(t, s, uid, TariffLevel, "", 19900)

	granted, err := s.CompletePayment(id, day0)
	if err != nil || !granted {
		t.Fatalf("CompletePayment = (%v, %v), want granted", granted, err)
	}
	owned, _ := s.User(uid).Entitlements()
	for k := range owned {
		if len(k) > 12 && k[:12] == "phase_unlock" {
			t.Fatalf("entitlements = %v, a credit must not open a level by itself", owned)
		}
	}
	if paid, _ := s.HasPaidLevel(uid); !paid {
		t.Error("a paid credit must count as a paid level (first-purchase price gone, repeats unlimited)")
	}
	if at, ok, _ := s.LastLevelPaymentAt(uid); !ok || at.IsZero() {
		t.Error("a paid credit must start the one-level-a-month limit")
	}
	if bonus, _ := s.PaymentBonus(id); bonus != 1000 {
		t.Errorf("bonus = %d, want the level's 1000", bonus)
	}
}

func TestOnlyPaidLevelPaymentsWithoutALevelAreUnspentCredits(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	other, _ := s.CreateUser("Другой")

	credit := newPayment(t, s, uid, TariffLevel, "", 19900)
	pending := newPayment(t, s, uid, TariffLevel, "", 29900) // not paid yet
	old := newPayment(t, s, uid, TariffLevel, "4", 24900)    // an old order that fixed its level
	year := newPayment(t, s, uid, TariffYear, "", 100000)
	theirs := newPayment(t, s, other, TariffLevel, "", 19900)
	for _, id := range []int64{credit, old, year, theirs} {
		if _, err := s.CompletePayment(id, day0); err != nil {
			t.Fatal(err)
		}
	}
	_ = pending

	got, err := s.UnclaimedLevelPayments(uid)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != credit {
		t.Fatalf("credits = %+v, want exactly payment %d", got, credit)
	}
	if err := s.ClaimLevel(uid, credit, "4", day0); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.UnclaimedLevelPayments(uid); len(got) != 0 {
		t.Fatalf("credits after spending = %+v, want none", got)
	}
}

func TestSpendingACreditOpensTheChosenLevelOnce(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	id := newPayment(t, s, uid, TariffLevel, "", 19900)
	s.CompletePayment(id, day0)

	if err := s.ClaimLevel(uid, id, "4", day0.Add(time.Hour)); err != nil {
		t.Fatalf("ClaimLevel: %v", err)
	}
	owned, _ := s.User(uid).Entitlements()
	if owned["phase_unlock:4"] != 1 {
		t.Fatalf("entitlements = %v, want phase_unlock:4", owned)
	}
	if p, _ := s.PaymentByID(id); p.GrantedPhase != "4" {
		t.Fatalf("payment opens %q, want 4", p.GrantedPhase)
	}
	// a second click on the same level changes nothing and is not an error
	if err := s.ClaimLevel(uid, id, "4", day0.Add(2*time.Hour)); err != nil {
		t.Fatalf("same level again = %v, want nil", err)
	}
	// another level on a spent credit is refused and opens nothing
	if err := s.ClaimLevel(uid, id, "5", day0.Add(3*time.Hour)); err != ErrCreditSpent {
		t.Fatalf("other level = %v, want ErrCreditSpent", err)
	}
	if owned, _ := s.User(uid).Entitlements(); owned["phase_unlock:5"] != 0 {
		t.Fatalf("level 5 opened on a spent credit: %v", owned)
	}
}

func TestACreditCanOnlyBeSpentByItsOwnerWhenItIsPaid(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	other, _ := s.CreateUser("Другой")
	paid := newPayment(t, s, uid, TariffLevel, "", 19900)
	s.CompletePayment(paid, day0)
	unpaid := newPayment(t, s, uid, TariffLevel, "", 29900)
	year := newPayment(t, s, uid, TariffYear, "", 100000)
	s.CompletePayment(year, day0)

	cases := []struct {
		name string
		uid  string
		id   int64
		ph   string
	}{
		{"somebody else's credit", other, paid, "4"},
		{"a payment that is not paid", uid, unpaid, "4"},
		{"a year", uid, year, "4"},
		{"a payment that does not exist", uid, 9999, "4"},
		{"no level named", uid, paid, ""},
	}
	for _, c := range cases {
		if err := s.ClaimLevel(c.uid, c.id, c.ph, day0); err != ErrNotClaimable {
			t.Errorf("%s: err = %v, want ErrNotClaimable", c.name, err)
		}
	}
	if owned, _ := s.User(uid).Entitlements(); owned["phase_unlock:4"] != 0 {
		t.Fatalf("a refused claim opened a level: %v", owned)
	}
}

// Two devices (or two quick taps) choosing different levels: one wins, the other
// is refused, and exactly one level is open afterwards.
func TestTwoChoicesAtOnceOpenExactlyOneLevel(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	id := newPayment(t, s, uid, TariffLevel, "", 19900)
	s.CompletePayment(id, day0)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, ph := range []string{"4", "5"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = s.ClaimLevel(uid, id, ph, day0)
		}()
	}
	wg.Wait()

	won := 0
	for _, e := range errs {
		if e == nil {
			won++
		} else if e != ErrCreditSpent {
			t.Errorf("unexpected error: %v", e)
		}
	}
	if won != 1 {
		t.Fatalf("%d choices won, want exactly one (errors %v)", won, errs)
	}
	owned, _ := s.User(uid).Entitlements()
	if owned["phase_unlock:4"]+owned["phase_unlock:5"] != 1 {
		t.Fatalf("entitlements = %v, want exactly one of the two levels", owned)
	}
}
