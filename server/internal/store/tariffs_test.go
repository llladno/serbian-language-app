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
	// 299 ₽, 199 ₽ for the first purchase, never expires, 1000 зёрнышек with it.
	if level.PriceKopecks != 29900 || level.FirstPriceKopecks != 19900 || level.PeriodDays != 0 || !level.Active ||
		level.BonusCurrency != 1000 {
		t.Fatalf("level = %+v", level)
	}
	year, _ := s.Tariff(TariffYear)
	if year.PriceKopecks != 100000 || year.OldPriceKopecks != 200000 || year.PeriodDays != 365 ||
		year.BonusCurrency != 15000 {
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

func TestLastLevelPaymentAtIsTheLatestPaidLevelOnly(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	if _, ok, err := s.LastLevelPaymentAt(uid); err != nil || ok {
		t.Fatalf("new learner: ok=%t err=%v, want none", ok, err)
	}

	// neither an unpaid order, a canceled one nor a paid year is a paid level
	pending := newPayment(t, s, uid, TariffLevel, "4", 24900)
	canceled := newPayment(t, s, uid, TariffLevel, "4", 24900)
	s.CancelPayment(canceled)
	s.CompletePayment(newPayment(t, s, uid, TariffYear, "", 100000), day0)
	if _, ok, _ := s.LastLevelPaymentAt(uid); ok {
		t.Fatal("something that is not a paid level counted")
	}

	first, second := day0.Add(time.Hour), day0.Add(48*time.Hour)
	s.CompletePayment(pending, first)
	if got, ok, _ := s.LastLevelPaymentAt(uid); !ok || !got.Equal(first) {
		t.Fatalf("after one level: %v ok=%t, want %v", got, ok, first)
	}
	s.CompletePayment(newPayment(t, s, uid, TariffLevel, "5", 29900), second)
	if got, _, _ := s.LastLevelPaymentAt(uid); !got.Equal(second) {
		t.Fatalf("after two levels: %v, want the later one %v", got, second)
	}

	// someone else's payment is not mine
	other, _ := s.CreateUser("Оля")
	if _, ok, _ := s.LastLevelPaymentAt(other); ok {
		t.Fatal("another learner's level counted")
	}
}

func TestPayingForATariffCreditsItsBonusCurrencyOnce(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")

	level := newPayment(t, s, uid, TariffLevel, "4", 24900)
	if bal, _ := s.Balance(uid); bal != 0 {
		t.Fatalf("balance before paying = %d", bal)
	}
	if granted, err := s.CompletePayment(level, day0); err != nil || !granted {
		t.Fatalf("CompletePayment: %t %v", granted, err)
	}
	if bal, _ := s.Balance(uid); bal != 1000 {
		t.Fatalf("balance after a level = %d, want its 1000 bonus", bal)
	}

	// a second run of the same payment (a notification and the return page
	// arriving together) credits nothing more
	if granted, _ := s.CompletePayment(level, day0); granted {
		t.Fatal("the same payment granted twice")
	}
	if bal, _ := s.Balance(uid); bal != 1000 {
		t.Fatalf("balance after the second run = %d, want still 1000", bal)
	}

	s.CompletePayment(newPayment(t, s, uid, TariffYear, "", 100000), day0)
	if bal, _ := s.Balance(uid); bal != 16000 {
		t.Fatalf("balance after the year = %d, want 1000 + 15000", bal)
	}

	// each credit is one `topup` entry that says which payment it came from
	entries, err := s.ListLedger(uid, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("%d ledger entries, want 2: %+v", len(entries), entries)
	}
	for _, e := range entries {
		if e.Kind != LedgerTopup || e.Ref == "" || e.IdempotencyKey != "payment:"+e.Ref || e.CreatedBy != "" {
			t.Errorf("entry = %+v", e)
		}
	}
}

func TestAPaymentThatDoesNotSucceedCreditsNothing(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	pending := newPayment(t, s, uid, TariffLevel, "4", 24900)
	canceled := newPayment(t, s, uid, TariffLevel, "4", 24900)
	s.CancelPayment(canceled)
	s.CompletePayment(canceled, day0) // a canceled payment cannot be completed
	if bal, _ := s.Balance(uid); bal != 0 {
		t.Fatalf("balance = %d, want 0 while nothing is paid (pending %d)", bal, pending)
	}
}

func TestATariffWithoutABonusCreditsNothing(t *testing.T) {
	s := newStore(t)
	if _, err := s.db.Exec(`UPDATE tariffs SET bonus_currency = 0 WHERE code = ?`, TariffLevel); err != nil {
		t.Fatal(err)
	}
	uid, _ := s.CreateUser("Гриша")
	s.CompletePayment(newPayment(t, s, uid, TariffLevel, "4", 24900), day0)
	if bal, _ := s.Balance(uid); bal != 0 {
		t.Fatalf("balance = %d, want 0 for a tariff with no bonus", bal)
	}
	if ok, _ := s.HasPaidLevel(uid); !ok {
		t.Fatal("the level was not granted either")
	}
}
