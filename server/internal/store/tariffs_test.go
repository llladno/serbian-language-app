package store

import (
	"testing"
	"time"
)

func newPayment(t *testing.T, s *Store, uid, code, phase string, amount int64) int64 {
	t.Helper()
	id, err := s.CreatePayment(Payment{
		UserID: uid, TariffCode: code, GrantedPhase: phase, AmountKopecks: amount,
		Provider: "yookassa", ReceiptContact: "a@example.com",
	}, day0)
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if err := s.SetPaymentProviderID(id, "pay-"+code+"-"+phase+time.Now().Format("150405.000000000"), true); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestSeededTariffs(t *testing.T) {
	s := newStore(t)
	level, err := s.Tariff(TariffLevel)
	if err != nil {
		t.Fatal(err)
	}
	// 299 ₽, 249 ₽ for the first purchase, never expires.
	if level.PriceKopecks != 29900 || level.FirstPriceKopecks != 24900 || level.PeriodDays != 0 || !level.Active {
		t.Fatalf("level = %+v", level)
	}
	year, _ := s.Tariff(TariffYear)
	if year.PriceKopecks != 100000 || year.OldPriceKopecks != 200000 || year.PeriodDays != 365 {
		t.Fatalf("year = %+v", year)
	}
	if _, err := s.Tariff("nope"); err != ErrTariffUnavailable {
		t.Fatalf("unknown tariff err = %v, want ErrTariffUnavailable", err)
	}
}

func TestCompletingALevelPaymentOpensThatPhaseForGood(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	id := newPayment(t, s, uid, TariffLevel, "4", 24900)

	granted, err := s.CompletePayment(id, day0)
	if err != nil || !granted {
		t.Fatalf("CompletePayment = (%v, %v), want granted", granted, err)
	}
	owned, _ := s.User(uid).Entitlements()
	if owned["phase_unlock:4"] != 1 {
		t.Fatalf("entitlements = %v, want phase_unlock:4", owned)
	}
	if owned["phase_unlock:5"] != 0 {
		t.Fatalf("level 5 must stay closed: %v", owned)
	}
	p, _ := s.PaymentByID(id)
	if p.Status != PaymentSucceeded || p.PaidAt.IsZero() {
		t.Fatalf("payment = %+v, want succeeded with a paid_at", p)
	}
}

// A notification and the return page both report success; the level opens once.
func TestCompletingTheSamePaymentTwiceGrantsOnce(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	id := newPayment(t, s, uid, TariffLevel, "4", 24900)

	first, _ := s.CompletePayment(id, day0)
	second, err := s.CompletePayment(id, day0.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if !first || second {
		t.Fatalf("granted = (%v, %v), want (true, false)", first, second)
	}
}

func TestACanceledPaymentNeverGrantsAnything(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	id := newPayment(t, s, uid, TariffLevel, "4", 24900)
	if err := s.CancelPayment(id); err != nil {
		t.Fatal(err)
	}
	granted, _ := s.CompletePayment(id, day0)
	if granted {
		t.Fatal("a canceled payment granted a level")
	}
	if owned, _ := s.User(uid).Entitlements(); owned["phase_unlock:4"] != 0 {
		t.Fatalf("entitlements = %v", owned)
	}
}

func TestCancelNeverUndoesASucceededPayment(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	id := newPayment(t, s, uid, TariffLevel, "4", 24900)
	s.CompletePayment(id, day0)
	if err := s.CancelPayment(id); err != nil {
		t.Fatal(err)
	}
	if p, _ := s.PaymentByID(id); p.Status != PaymentSucceeded {
		t.Fatalf("status = %s, want it to stay succeeded", p.Status)
	}
}

func TestAYearRunsForPeriodDaysAndOpensNothingBefore(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	us := s.User(uid)

	if _, ok, _ := us.AllAccessUntil(day0); ok {
		t.Fatal("a learner who never paid has a year")
	}
	id := newPayment(t, s, uid, TariffYear, "", 100000)
	if granted, err := s.CompletePayment(id, day0); err != nil || !granted {
		t.Fatalf("CompletePayment = (%v, %v)", granted, err)
	}
	until, ok, err := us.AllAccessUntil(day0)
	if err != nil || !ok {
		t.Fatalf("AllAccessUntil = (%v, %v, %v)", until, ok, err)
	}
	if want := day0.AddDate(0, 0, 365); !until.Equal(want) {
		t.Fatalf("until = %v, want %v", until, want)
	}
	// and it ends: a day after the end there is no year
	if _, ok, _ := us.AllAccessUntil(day0.AddDate(0, 0, 366)); ok {
		t.Fatal("an ended year still counts")
	}
}

// Paying while a year is running adds the new year after the old one ends, so
// paying early never costs the days already bought.
func TestPayingForAYearEarlyExtendsFromTheCurrentEnd(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	first := newPayment(t, s, uid, TariffYear, "", 100000)
	s.CompletePayment(first, day0)

	later := day0.AddDate(0, 6, 0)
	second := newPayment(t, s, uid, TariffYear, "", 100000)
	if granted, _ := s.CompletePayment(second, later); !granted {
		t.Fatal("second year not granted")
	}
	until, _, _ := s.User(uid).AllAccessUntil(later)
	if want := day0.AddDate(0, 0, 365).AddDate(0, 0, 365); !until.Equal(want) {
		t.Fatalf("until = %v, want %v (the old end plus a year, not later+365)", until, want)
	}
}

// A year that already ended does not push the new one into the past.
func TestPayingAfterAYearEndedStartsFromNow(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	s.CompletePayment(newPayment(t, s, uid, TariffYear, "", 100000), day0)

	later := day0.AddDate(2, 0, 0)
	s.CompletePayment(newPayment(t, s, uid, TariffYear, "", 100000), later)
	until, ok, _ := s.User(uid).AllAccessUntil(later)
	if !ok || !until.Equal(later.AddDate(0, 0, 365)) {
		t.Fatalf("until = %v (ok=%v), want a year from the second payment", until, ok)
	}
}

func TestHasPaidLevelOnlyCountsSucceededLevels(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	if ok, _ := s.HasPaidLevel(uid); ok {
		t.Fatal("new learner has paid")
	}
	pending := newPayment(t, s, uid, TariffLevel, "4", 24900)
	if ok, _ := s.HasPaidLevel(uid); ok {
		t.Fatal("a pending payment counts as paid")
	}
	s.CompletePayment(pending, day0)
	if ok, _ := s.HasPaidLevel(uid); !ok {
		t.Fatal("a paid level does not count")
	}
	// a year is not a level: it does not take the first-purchase price away
	other, _ := s.CreateUser("Оля")
	s.CompletePayment(newPayment(t, s, other, TariffYear, "", 100000), day0)
	if ok, _ := s.HasPaidLevel(other); ok {
		t.Fatal("a year counted as a paid level")
	}
}

func TestPendingPaymentsAreOnlyTheCallersAndOnlyRecent(t *testing.T) {
	s := newStore(t)
	a, _ := s.CreateUser("А")
	b, _ := s.CreateUser("Б")
	mine := newPayment(t, s, a, TariffLevel, "4", 24900)
	newPayment(t, s, b, TariffLevel, "4", 24900)
	done := newPayment(t, s, a, TariffYear, "", 100000)
	s.CompletePayment(done, day0)

	got, err := s.PendingPayments(a, day0.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != mine {
		t.Fatalf("pending = %+v, want only a's unpaid level order", got)
	}
	if old, _ := s.PendingPayments(a, day0.Add(time.Hour)); len(old) != 0 {
		t.Fatalf("pending since the future = %+v, want none", old)
	}
}
