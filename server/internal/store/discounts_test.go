package store

import (
	"testing"
	"time"
)

func TestDiscountedKopecksRoundsToWholeRublesAndIgnoresNonsense(t *testing.T) {
	for _, c := range []struct {
		price   int64
		percent int
		want    int64
	}{
		{39900, 20, 31900},   // 319.20 -> 319
		{199900, 20, 159900}, // 1 599.20 -> 1 599
		{19900, 20, 15900},   // 159.20 -> 159
		{29900, 50, 15000},   // 149.50 -> 150
		{100, 90, 100},       // never below one ruble
		{39900, 0, 39900},    // out of bounds: unchanged
		{39900, 95, 39900},
		{39900, -5, 39900},
	} {
		if got := DiscountedKopecks(c.price, c.percent); got != c.want {
			t.Errorf("DiscountedKopecks(%d, %d) = %d, want %d", c.price, c.percent, got, c.want)
		}
	}
}

func TestActiveDiscountIsOfferedUntilItExpiresIsTakenBackOrIsSpent(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	other, _ := s.CreateUser("Другой")
	mk := func(id, user string, percent int, hours int) {
		t.Helper()
		if err := s.CreateDiscount(PersonalDiscount{
			ID: id, UserID: user, Percent: percent, CreatedAt: day0, ExpiresAt: day0.Add(time.Duration(hours) * time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
	}

	if _, ok, err := s.ActiveDiscount(uid, day0); err != nil || ok {
		t.Fatalf("no discount yet = (%v, %v)", ok, err)
	}

	mk("d1", uid, 20, 72)
	mk("d-other", other, 50, 72)
	d, ok, err := s.ActiveDiscount(uid, day0.Add(time.Hour))
	if err != nil || !ok || d.ID != "d1" || d.Percent != 20 {
		t.Fatalf("active = (%+v, %v, %v), want d1 at 20%%", d, ok, err)
	}
	if _, ok, _ := s.ActiveDiscount(uid, day0.Add(72*time.Hour)); ok {
		t.Error("a discount offered at the very moment it expires")
	}

	// the bigger of two wins
	mk("d2", uid, 30, 24)
	if d, _, _ := s.ActiveDiscount(uid, day0.Add(time.Hour)); d.ID != "d2" {
		t.Errorf("active = %s, want the 30%% one", d.ID)
	}
	if d, _, _ := s.ActiveDiscount(uid, day0.Add(30*time.Hour)); d.ID != "d1" {
		t.Errorf("active = %s, want d1 once the 30%% one has run out", d.ID)
	}

	// taken back by the owner
	if _, err := s.db.Exec(`UPDATE personal_discounts SET canceled_at = ? WHERE id = 'd2'`, day0.Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	if d, _, _ := s.ActiveDiscount(uid, day0.Add(time.Hour)); d.ID != "d1" {
		t.Errorf("active = %s, want d1 after d2 was taken back", d.ID)
	}

	// a pending order does not spend it; a paid one does
	id, err := s.CreatePayment(Payment{UserID: uid, TariffCode: TariffLevel, AmountKopecks: 15900, Provider: "tbank", DiscountID: "d1"}, day0)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.ActiveDiscount(uid, day0.Add(time.Hour)); !ok {
		t.Error("discount gone while its order is only pending")
	}
	if err := s.SetPaymentProviderID(id, "p-1", true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CompletePayment(id, day0.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.ActiveDiscount(uid, day0.Add(2*time.Hour)); ok {
		t.Error("discount offered again after an order was paid with it")
	}
	if p, _ := s.PaymentByID(id); p.DiscountID != "d1" {
		t.Errorf("payment discount = %q, want d1", p.DiscountID)
	}
}

func TestADiscountOutsideTheBoundsIsNeverOffered(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	for i, pc := range []int{0, 95, 100, -3} {
		if err := s.CreateDiscount(PersonalDiscount{ID: string(rune('a' + i)), UserID: uid, Percent: pc, CreatedAt: day0, ExpiresAt: day0.Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok, _ := s.ActiveDiscount(uid, day0); ok {
		t.Error("a discount outside 1–90%% was offered")
	}
}
