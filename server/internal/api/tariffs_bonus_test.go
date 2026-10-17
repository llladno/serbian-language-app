package api

import (
	"fmt"
	"net/http"
	"testing"
)

// Paying for a tariff brings зёрнышки with it: 1000 for a level, 15000 for the year.

func TestTheTariffsScreenQuotesTheBonusCurrency(t *testing.T) {
	h, st := tariffsAPI(t, newFakeYooKassa(t))
	_, c := listedUser(t, h, st)
	got := getTariffsOf(t, h, c)
	if got.Level.Bonus != 1000 || got.Year.Bonus != 15000 {
		t.Fatalf("bonus = level %d, year %d, want 1000 and 15000", got.Level.Bonus, got.Year.Bonus)
	}
}

func TestPayingForALevelCreditsItsBonus(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := tariffsAPI(t, yk)
	uid, c := listedUser(t, h, st)

	o := order(t, h, c, "level")
	if bal, _ := st.Balance(uid); bal != 0 {
		t.Fatalf("balance with the order only created = %d", bal)
	}
	yk.set("yk-1", "succeeded")
	doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	if bal, _ := st.Balance(uid); bal != 1000 {
		t.Fatalf("balance after paying for a level = %d, want 1000", bal)
	}

	// a notification for the same payment, and another visit to the return page,
	// must not credit again
	anon(h, "POST", "/api/yookassa/webhook", `{"event":"payment.succeeded","object":{"id":"yk-1"}}`)
	doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	if bal, _ := st.Balance(uid); bal != 1000 {
		t.Fatalf("balance after repeats = %d, want still 1000", bal)
	}
}

func TestPayingForTheYearCreditsItsBonusThroughTBankToo(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPIWith(t, nil, tb)
	uid, c := tbankAccount(t, h, st)

	order(t, h, c, "year")
	tb.set("9001", "CONFIRMED")
	rr := anon(h, "POST", "/api/tbank/webhook", tb.notification("9001", "CONFIRMED"))
	if rr.Code != http.StatusOK {
		t.Fatalf("webhook = %d", rr.Code)
	}
	if bal, _ := st.Balance(uid); bal != 15000 {
		t.Fatalf("balance after the year = %d, want 15000", bal)
	}
}

func TestACanceledPaymentCreditsNothing(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := tariffsAPI(t, yk)
	uid, c := listedUser(t, h, st)
	o := order(t, h, c, "year")
	yk.set("yk-1", "canceled")
	doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	if bal, _ := st.Balance(uid); bal != 0 {
		t.Fatalf("balance after a canceled payment = %d", bal)
	}
}

// The thank-you screen asks the order what it brought.
func TestAPaidOrderSaysWhichLevelItOpenedAndWhatBonusCameWithIt(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := tariffsAPI(t, yk)
	_, c := listedUser(t, h, st)
	path := func(id int64) string { return fmt.Sprintf("/api/me/orders/%d", id) }

	level := order(t, h, c, "level")
	pending := decodeBody[orderDTO](t, doCookie(h, c, "GET", path(level.ID), ""))
	if pending.PhaseTitle != "" || pending.Bonus != 0 {
		t.Fatalf("an unpaid order already claims %+v", pending)
	}

	yk.set("yk-1", "succeeded")
	paid := decodeBody[orderDTO](t, doCookie(h, c, "GET", path(level.ID), ""))
	if paid.Status != "succeeded" || paid.PhaseTitle != "Уровень 4 — Падежи" || paid.Bonus != 1000 {
		t.Fatalf("paid level order = %+v, want the level's title and 1000", paid)
	}

	year := order(t, h, c, "year")
	yk.set("yk-2", "succeeded")
	paidYear := decodeBody[orderDTO](t, doCookie(h, c, "GET", path(year.ID), ""))
	if paidYear.PhaseTitle != "" || paidYear.Bonus != 15000 {
		t.Fatalf("paid year order = %+v, want no level title and 15000", paidYear)
	}
}
