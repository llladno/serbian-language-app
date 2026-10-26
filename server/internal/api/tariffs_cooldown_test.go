package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/store"
)

// One level a month: after a level is paid for in rubles, the next one cannot be
// bought for rubles until the cooldown ends. The year and levels bought for
// зёрнышки are not held back.

// cooldownAPI is the tariffs app with a movable clock and a three-day cooldown
// (short, so that moving the clock stays inside the test session's lifetime).
func cooldownAPI(t *testing.T, tb *fakeTBank, days int) (http.Handler, *store.Store, func(time.Duration), func() time.Time) {
	t.Helper()
	now := fixedNow
	h, st := tariffsAPIWith(t, tb, func(d *Deps) {
		d.Now = func() time.Time { return now }
		d.Config.TariffLevelCooldownDays = days
	})
	return h, st, func(by time.Duration) { now = now.Add(by) }, func() time.Time { return now }
}

func TestALevelBoughtForRublesHoldsTheNextOneBackUntilTheCooldownEnds(t *testing.T) {
	tb := newFakeTBank(t)
	h, st, advance, clock := cooldownAPI(t, tb, 3)
	_, c := listedUser(t, h, st)

	// before any purchase nothing is held back
	if got := getTariffsOf(t, h, c); got.Level.AvailableFrom != "" {
		t.Fatalf("available_from before any purchase = %q, want empty", got.Level.AvailableFrom)
	}

	o := order(t, h, c, "level")
	tb.set("9001", "CONFIRMED")
	doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	// the paid level is spent on level B (the learner's choice), which frees the next order
	// from "choose first" and leaves only the monthly limit
	if rr := chooseLevel(h, c, o.ID, "B"); rr.Code != 200 {
		t.Fatalf("choose B = %d %s", rr.Code, rr.Body)
	}

	got := getTariffsOf(t, h, c)
	want := clock().Add(72 * time.Hour).UTC().Format(time.RFC3339)
	if got.Level.AvailableFrom != want {
		t.Fatalf("available_from = %q, want %q (paid now + 3 days)", got.Level.AvailableFrom, want)
	}
	if got.Level.Next == nil || got.Level.Next.ID != "C" {
		t.Errorf("next = %+v: the next level is still named, it just cannot be bought yet", got.Level.Next)
	}

	// the order is refused before anything is created or charged
	rr := doCookie(h, c, "POST", "/api/me/orders", `{"tariff":"level"}`)
	if rr.Code != http.StatusConflict || !strings.Contains(rr.Body.String(), "level_cooldown") {
		t.Fatalf("second level order = %d %s, want 409 level_cooldown", rr.Code, rr.Body)
	}
	if tb.count() != 1 {
		t.Errorf("the bank saw %d payments, want 1: a refused order must not reach the payment service", tb.count())
	}

	// the year is not held back
	if y := order(t, h, c, "year"); y.Status != "pending" {
		t.Fatalf("year order during the cooldown = %+v", y)
	}

	// and once the cooldown is over the next level can be bought
	advance(72 * time.Hour)
	if got := getTariffsOf(t, h, c); got.Level.AvailableFrom != "" {
		t.Fatalf("available_from after the cooldown = %q, want empty", got.Level.AvailableFrom)
	}
	if o2 := order(t, h, c, "level"); o2.Status != "pending" {
		t.Fatalf("level order after the cooldown = %+v", o2)
	}
}

func TestOnlyAPaidLevelStartsTheCooldown(t *testing.T) {
	tb := newFakeTBank(t)
	h, st, _, _ := cooldownAPI(t, tb, 3)
	_, c := listedUser(t, h, st)

	// an order that was never paid, and one that was canceled, hold nothing back
	order(t, h, c, "level")
	if got := getTariffsOf(t, h, c); got.Level.AvailableFrom != "" {
		t.Fatalf("a pending order started the cooldown: %q", got.Level.AvailableFrom)
	}
	tb.set("9001", "REJECTED")
	getTariffsOf(t, h, c) // opening the screen settles the canceled one
	if o := order(t, h, c, "level"); o.Status != "pending" {
		t.Fatalf("order after a canceled one = %+v", o)
	}
}

func TestTheYearDoesNotStartTheCooldown(t *testing.T) {
	tb := newFakeTBank(t)
	h, st, _, _ := cooldownAPI(t, tb, 3)
	_, c := listedUser(t, h, st)

	order(t, h, c, "year")
	tb.set("9001", "CONFIRMED")
	got := getTariffsOf(t, h, c) // settles the year
	if got.Year.Until == "" {
		t.Fatalf("no running year after paying for it: %+v", got)
	}
	if got.Level.AvailableFrom != "" {
		t.Errorf("the year started the level cooldown: %q", got.Level.AvailableFrom)
	}
}

func TestAZeroCooldownTurnsTheLimitOff(t *testing.T) {
	tb := newFakeTBank(t)
	h, st, _, _ := cooldownAPI(t, tb, 0)
	_, c := listedUser(t, h, st)

	o := order(t, h, c, "level")
	tb.set("9001", "CONFIRMED")
	doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	if rr := chooseLevel(h, c, o.ID, "B"); rr.Code != 200 {
		t.Fatalf("choose B = %d %s", rr.Code, rr.Body)
	}

	if got := getTariffsOf(t, h, c); got.Level.AvailableFrom != "" {
		t.Fatalf("available_from with the limit off = %q", got.Level.AvailableFrom)
	}
	if o2 := order(t, h, c, "level"); o2.Status != "pending" {
		t.Fatalf("second level order with the limit off = %+v", o2)
	}
}

// The limit follows the account, whichever payment service took the money.
func TestTheLimitAppliesToTBankPaymentsToo(t *testing.T) {
	tb := newFakeTBank(t)
	now := fixedNow
	h, st := tariffsAPIWith(t, tb, func(d *Deps) {
		d.Now = func() time.Time { return now }
		d.Config.TariffLevelCooldownDays = 3
	})
	_, c := tbankAccount(t, h, st)

	o := order(t, h, c, "level")
	tb.set("9001", "CONFIRMED")
	doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	if rr := chooseLevel(h, c, o.ID, "B"); rr.Code != 200 {
		t.Fatalf("choose B = %d %s", rr.Code, rr.Body)
	}

	rr := doCookie(h, c, "POST", "/api/me/orders", `{"tariff":"level"}`)
	if rr.Code != http.StatusConflict || !strings.Contains(rr.Body.String(), "level_cooldown") {
		t.Fatalf("second level order = %d %s, want 409 level_cooldown", rr.Code, rr.Body)
	}
	if tb.count() != 1 {
		t.Errorf("the bank saw %d payments, want 1", tb.count())
	}
}
