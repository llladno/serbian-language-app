package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
	"github.com/grisha/serbian-app/server/internal/config"
	"github.com/grisha/serbian-app/server/internal/store"
)

// An announced price change (migration 041): from the announcement the quote carries the countdown and the prices
// to come, the live prices stay until the moment, and at the moment the server itself switches them.

// quoteOf is the part of the tariffs quote that the price change touches.
type quoteOf struct {
	Level struct {
		Price        int64 `json:"price"`
		RegularPrice int64 `json:"regular_price"`
	} `json:"level"`
	Year struct {
		Price    int64 `json:"price"`
		OldPrice int64 `json:"old_price"`
	} `json:"year"`
	PriceChange *struct {
		At         string `json:"at"`
		Now        string `json:"now"`
		LevelPrice int64  `json:"level_price"`
		YearPrice  int64  `json:"year_price"`
		Preview    bool   `json:"preview"`
	} `json:"price_change"`
}

func quote(t *testing.T, h http.Handler, c *http.Cookie) quoteOf {
	t.Helper()
	rr := doCookie(h, c, "GET", "/api/me/tariffs", "")
	if rr.Code != 200 {
		t.Fatalf("GET tariffs = %d %s", rr.Code, rr.Body)
	}
	var q quoteOf
	if err := json.Unmarshal(rr.Body.Bytes(), &q); err != nil {
		t.Fatal(err)
	}
	return q
}

var (
	beforeAnnounce = time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	inTheWindow    = time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	afterTheChange = time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
)

// sessionAt is a session cookie made at `at` and good for 30 days: the ones the helpers make start at fixedNow in
// September and would be expired by the time these tests set the clock to late November.
func sessionAt(t *testing.T, st *store.Store, userID string, at time.Time) *http.Cookie {
	t.Helper()
	raw, hash, err := auth.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSession(hash, userID, "test-agent", at, at.Add(720*time.Hour)); err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: "session", Value: raw}
}

func priceChangeAPI(t *testing.T, tb *fakeTBank, at time.Time) (http.Handler, *store.Store, func(time.Time)) {
	t.Helper()
	now := at
	h, st := tariffsAPIWith(t, tb, func(d *Deps) { d.Now = func() time.Time { return now } })
	return h, st, func(to time.Time) { now = to }
}

func TestThePriceChangeComesWithItsCountdownAndLivePricesWaitForTheMoment(t *testing.T) {
	h, st, setNow := priceChangeAPI(t, newFakeTBank(t), beforeAnnounce)
	uid, _ := listedUser(t, h, st)
	c := sessionAt(t, st, uid, beforeAnnounce)

	// listedEmail is one of the owner's own accounts: it is shown the change before the announcement date, marked as
	// a preview (that others are not is checked on priceChange itself, below)
	q := quote(t, h, c)
	if q.PriceChange == nil || !q.PriceChange.Preview {
		t.Fatalf("price_change before the announcement for the owner's account = %+v, want a preview", q.PriceChange)
	}

	setNow(inTheWindow)
	q = quote(t, h, c)
	if q.PriceChange == nil {
		t.Fatal("no price_change inside the window")
	}
	if q.PriceChange.At != "2026-11-15T23:00:00Z" || q.PriceChange.Now != inTheWindow.Format(time.RFC3339) {
		t.Errorf("moments = %+v, want the change at 2026-11-15T23:00:00Z and the server's clock", q.PriceChange)
	}
	// a learner with no paid level: the first-purchase price to come, not the regular one
	if q.PriceChange.LevelPrice != 399 || q.PriceChange.YearPrice != 1999 {
		t.Errorf("prices to come = %+v, want 399 and 1999", q.PriceChange)
	}
	// nothing is charged differently until the moment
	if q.Level.Price != 199 || q.Level.RegularPrice != 299 || q.Year.Price != 1000 || q.Year.OldPrice != 2000 {
		t.Errorf("live prices in the window = %+v, want the old ones", q)
	}
}

func TestAtTheMomentTheServerSwitchesThePricesAndTheOrderChargesTheNewOnes(t *testing.T) {
	tb := newFakeTBank(t)
	h, st, setNow := priceChangeAPI(t, tb, inTheWindow)
	uid, _ := listedUser(t, h, st)
	c := sessionAt(t, st, uid, inTheWindow)

	// an order made before the moment is for the old price, and stays so when it is paid after it
	early := order(t, h, c, "level")
	if tb.get("9001").Amount != 19900 {
		t.Fatalf("order before the change = %d kopecks, want 19900", tb.get("9001").Amount)
	}

	setNow(afterTheChange)
	q := quote(t, h, c)
	if q.PriceChange != nil {
		t.Errorf("price_change after the change = %+v, want none", q.PriceChange)
	}
	if q.Level.Price != 399 || q.Level.RegularPrice != 399 || q.Year.Price != 1999 || q.Year.OldPrice != 0 {
		t.Errorf("live prices after the change = %+v, want 399 for everybody and a year at 1999 with nothing struck", q)
	}

	tb.set("9001", "CONFIRMED")
	if rr := doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", early.ID), ""); rr.Code != 200 {
		t.Fatalf("settle the early order = %d %s", rr.Code, rr.Body)
	}
	if p, err := st.PaymentByID(early.ID); err != nil || p.Status != store.PaymentSucceeded || p.AmountKopecks != 19900 {
		t.Fatalf("early order after the change = %+v, %v, want paid at the price it was made at", p, err)
	}

	order(t, h, c, "year")
	if tb.get("9002").Amount != 199900 {
		t.Errorf("year order after the change = %d kopecks, want 199900", tb.get("9002").Amount)
	}
}

func TestThePriceToComeFollowsTheFirstPurchaseRule(t *testing.T) {
	level := store.Tariff{PriceKopecks: 29900, FirstPriceKopecks: 19900, NextPriceKopecks: 59900, NextFirstPriceKopecks: 39900,
		AnnounceAt: beforeAnnounce, ChangeAt: afterTheChange}
	year := store.Tariff{PriceKopecks: 100000, NextPriceKopecks: 199900}

	if got := priceChange(level, year, true, inTheWindow, false); got == nil || got.LevelPrice != 599 {
		t.Errorf("for a learner who has paid a level: %+v, want 599", got)
	}
	// nothing scheduled, and a year with no next price of its own, quote what they have
	if got := priceChange(store.Tariff{PriceKopecks: 29900}, year, false, inTheWindow, false); got != nil {
		t.Errorf("nothing scheduled: %+v, want none", got)
	}
	if got := priceChange(level, store.Tariff{PriceKopecks: 100000}, false, inTheWindow, false); got == nil || got.YearPrice != 1000 {
		t.Errorf("year with no change of its own: %+v, want its live price", got)
	}
	// the moment itself is already the new price
	if got := priceChange(level, year, false, afterTheChange, false); got != nil {
		t.Errorf("at the moment: %+v, want none", got)
	}
}

func TestTheOwnersAccountSeesTheChangeBeforeItsAnnouncementAndOthersDoNot(t *testing.T) {
	level := store.Tariff{PriceKopecks: 29900, FirstPriceKopecks: 19900, NextPriceKopecks: 59900, NextFirstPriceKopecks: 39900,
		AnnounceAt: inTheWindow, ChangeAt: afterTheChange}
	year := store.Tariff{PriceKopecks: 100000, NextPriceKopecks: 199900}

	if got := priceChange(level, year, false, beforeAnnounce, false); got != nil {
		t.Errorf("other account before the announcement: %+v, want none", got)
	}
	got := priceChange(level, year, false, beforeAnnounce, true)
	if got == nil || !got.Preview || got.LevelPrice != 399 {
		t.Errorf("owner's account before the announcement: %+v, want the change, marked as a preview", got)
	}
	// even the owner's account has nothing to see once the change is made
	if got := priceChange(level, year, false, afterTheChange, true); got != nil {
		t.Errorf("owner's account after the change: %+v, want none", got)
	}
	// and one that was never scheduled
	if got := priceChange(store.Tariff{PriceKopecks: 29900}, year, false, beforeAnnounce, true); got != nil {
		t.Errorf("nothing scheduled: %+v, want none", got)
	}
}

func TestOnlyTheListedEmailsCountAsTheOwnersAccount(t *testing.T) {
	cfg := config.Config{TariffsEmails: []string{listedEmail}, TBankEmails: []string{tbankEmail}}
	for email, want := range map[string]bool{listedEmail: true, tbankEmail: true, " " + strings.ToUpper(listedEmail) + " ": true, "someone@example.com": false, "": false} {
		if got := cfg.SeesPriceChangeEarly(email); got != want {
			t.Errorf("SeesPriceChangeEarly(%q) = %v, want %v", email, got, want)
		}
	}
}

func TestThePublicPricesFollowTheChangeWithoutASession(t *testing.T) {
	h, _, setNow := priceChangeAPI(t, newFakeTBank(t), beforeAnnounce)
	get := func() (int, map[string]any, string) {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest("GET", "/api/prices", nil))
		var out map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode %q: %v", rr.Body, err)
		}
		return rr.Code, out, rr.Header().Get("Cache-Control")
	}
	level := func(o map[string]any) map[string]any { return o["level"].(map[string]any) }

	// before the announcement: the live prices only
	code, out, cache := get()
	if code != 200 || level(out)["first"] != 199.0 || level(out)["regular"] != 299.0 || out["year"] != 1000.0 || out["change"] != nil {
		t.Fatalf("before the announcement = %d %v", code, out)
	}
	if cache != "public, max-age=30" {
		t.Errorf("Cache-Control = %q, want half a minute at most", cache)
	}

	// announced: the live prices and the change
	setNow(inTheWindow)
	_, out, _ = get()
	ch, _ := out["change"].(map[string]any)
	if ch == nil || ch["at"] != "2026-11-15T23:00:00Z" || ch["level_first"] != 399.0 || ch["level_regular"] != 399.0 || ch["year"] != 1999.0 {
		t.Fatalf("announced = %v", out)
	}
	if level(out)["first"] != 199.0 || out["year"] != 1000.0 {
		t.Errorf("live prices in the window = %v, want the old ones", out)
	}

	// made: the new prices are the live ones, nothing is announced
	setNow(afterTheChange)
	_, out, _ = get()
	if level(out)["first"] != 399.0 || level(out)["regular"] != 399.0 || out["year"] != 1999.0 || out["change"] != nil {
		t.Fatalf("after the change = %v", out)
	}
}

func TestOneNextPriceForEverybodyWhenThereIsNoFirstPurchasePrice(t *testing.T) {
	// the schedule of 2026-11: «Уровень» is 399 ₽ for everybody after the change, with no separate first-purchase price
	level := store.Tariff{PriceKopecks: 29900, FirstPriceKopecks: 19900, NextPriceKopecks: 39900, NextFirstPriceKopecks: 0,
		AnnounceAt: beforeAnnounce, ChangeAt: afterTheChange}
	year := store.Tariff{PriceKopecks: 100000, NextPriceKopecks: 199900}
	for _, hasPaid := range []bool{false, true} {
		if got := priceChange(level, year, hasPaid, inTheWindow, false); got == nil || got.LevelPrice != 399 || got.YearPrice != 1999 {
			t.Errorf("hasPaid=%v: %+v, want 399 and 1999", hasPaid, got)
		}
	}
}
