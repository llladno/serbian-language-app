package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/store"
)

// The stand-in cash desk (PAYMENTS_DEV): the same orders, settlement and grants as with a bank, but
// the "bank" is a page with two buttons, so a developer can try the whole flow without one.

func devAPI(t *testing.T) (http.Handler, *store.Store) {
	t.Helper()
	return tariffsAPIWith(t, nil, func(d *Deps) { d.Config.PaymentsDev = true })
}

func devPay(h http.Handler, c *http.Cookie, orderID int64, outcome string) *httptest.ResponseRecorder {
	return doCookie(h, c, "POST", fmt.Sprintf("/api/dev/pay/%d", orderID), fmt.Sprintf(`{"outcome":%q}`, outcome))
}

type devOrderInfo struct {
	ID     int64  `json:"id"`
	Tariff string `json:"tariff"`
	Title  string `json:"title"`
	Amount int64  `json:"amount"`
	Status string `json:"status"`
}

func devInfo(t *testing.T, h http.Handler, c *http.Cookie, orderID int64) devOrderInfo {
	t.Helper()
	rr := doCookie(h, c, "GET", fmt.Sprintf("/api/dev/pay/%d", orderID), "")
	if rr.Code != 200 {
		t.Fatalf("GET dev pay = %d %s", rr.Code, rr.Body)
	}
	return decodeBody[devOrderInfo](t, rr)
}

func TestTheStandInDeskTakesAnOrderWithoutAnyBank(t *testing.T) {
	h, st := devAPI(t)
	_, c := listedUser(t, h, st)

	if !getTariffsOf(t, h, c).PaymentsReady {
		t.Fatal("payments_ready = false: the stand-in desk needs no credentials")
	}
	o := order(t, h, c, "level")
	if o.Status != store.PaymentPending || o.ConfirmationURL != fmt.Sprintf("/dev/pay?order=%d", o.ID) {
		t.Fatalf("order = %+v, want pending with the stand-in page as the place to pay", o)
	}
	info := devInfo(t, h, c, o.ID)
	if info.Tariff != "level" || info.Amount != 199 || info.Status != store.PaymentPending || info.Title == "" {
		t.Fatalf("page info = %+v, want a pending level order for 199 ₽", info)
	}

	rr := devPay(h, c, o.ID, "paid")
	if rr.Code != 200 {
		t.Fatalf("pay = %d %s", rr.Code, rr.Body)
	}
	got := decodeBody[orderDTO](t, rr)
	if got.Status != store.PaymentSucceeded || !got.Choose || got.Bonus != 1000 {
		t.Fatalf("settled order = %+v, want paid, to be chosen, with the bonus", got)
	}
	// a paid level is a credit; nothing is open until the learner chooses
	if !phaseLocked(t, h, c, "B") {
		t.Fatal("level B opened by the payment itself")
	}
	if rr := chooseLevel(h, c, o.ID, "B"); rr.Code != 200 {
		t.Fatalf("choose = %d %s", rr.Code, rr.Body)
	}
	if phaseLocked(t, h, c, "B") {
		t.Fatal("level B still locked after the choice")
	}
	// and the stand-in payment is marked as a test one, like the bank's test terminal
	if p, _ := st.PaymentByID(o.ID); !p.Test || p.Provider != "dev" {
		t.Fatalf("payment = provider %q test %v, want dev and test", p.Provider, p.Test)
	}
}

func TestDecliningAtTheStandInDeskOpensNothingAndTheOrderCanBeRetried(t *testing.T) {
	h, st := devAPI(t)
	_, c := listedUser(t, h, st)
	o := order(t, h, c, "level")

	rr := devPay(h, c, o.ID, "failed")
	if rr.Code != 200 {
		t.Fatalf("decline = %d %s", rr.Code, rr.Body)
	}
	if got := decodeBody[orderDTO](t, rr); got.Status != store.PaymentCanceled || got.Choose {
		t.Fatalf("declined order = %+v, want canceled", got)
	}
	if got := getTariffsOf(t, h, c); len(got.Level.Credits) != 0 || got.HasPaidLevel {
		t.Fatalf("a declined payment left %+v", got)
	}
	if again := order(t, h, c, "level"); again.ID == o.ID || again.Status != store.PaymentPending {
		t.Fatalf("second order = %+v, want a fresh pending one", again)
	}
}

func TestPayingTwiceAtTheStandInDeskGrantsOnce(t *testing.T) {
	h, st := devAPI(t)
	uid, c := listedUser(t, h, st)
	o := order(t, h, c, "level")
	devPay(h, c, o.ID, "paid")
	before, _ := st.Balance(uid)
	if rr := devPay(h, c, o.ID, "paid"); rr.Code != 200 {
		t.Fatalf("second pay = %d %s", rr.Code, rr.Body)
	}
	after, _ := st.Balance(uid)
	if before != after {
		t.Fatalf("balance %d -> %d, the bonus was paid twice", before, after)
	}
	if got := getTariffsOf(t, h, c); len(got.Level.Credits) != 1 {
		t.Fatalf("credits = %v, want exactly one", got.Level.Credits)
	}
}

func TestAPaidOrderCannotBeDeclinedAfterwards(t *testing.T) {
	h, st := devAPI(t)
	_, c := listedUser(t, h, st)
	o := order(t, h, c, "level")
	devPay(h, c, o.ID, "paid")
	if rr := devPay(h, c, o.ID, "failed"); rr.Code != http.StatusConflict {
		t.Fatalf("decline after paying = %d, want 409", rr.Code)
	}
	if p, _ := st.PaymentByID(o.ID); p.Status != store.PaymentSucceeded {
		t.Fatalf("status = %q, want it to stay paid", p.Status)
	}
}

func TestAYearPaidAtTheStandInDeskOpensEveryLevel(t *testing.T) {
	h, st := devAPI(t)
	_, c := listedUser(t, h, st)
	o := order(t, h, c, "year")
	if info := devInfo(t, h, c, o.ID); info.Amount != 1000 {
		t.Fatalf("year amount = %d, want 1000", info.Amount)
	}
	if rr := devPay(h, c, o.ID, "paid"); rr.Code != 200 {
		t.Fatalf("pay = %d %s", rr.Code, rr.Body)
	}
	if phaseLocked(t, h, c, "B") || phaseLocked(t, h, c, "C") {
		t.Fatal("a paid year left a level locked")
	}
}

func TestTheStandInDeskIsPrivateAndStrict(t *testing.T) {
	h, st := devAPI(t)
	_, c := listedUser(t, h, st)
	o := order(t, h, c, "level")

	// somebody else's order, an order that does not exist, a malformed id
	other := authed(t, st, registerAndVerify(t, h, st, tbankEmail, "secret1234", "Man"))
	for _, rr := range []*httptest.ResponseRecorder{
		devPay(h, other, o.ID, "paid"),
		devPay(h, c, 9999, "paid"),
		doCookie(h, other, "GET", fmt.Sprintf("/api/dev/pay/%d", o.ID), ""),
		doCookie(h, c, "POST", "/api/dev/pay/abc", `{"outcome":"paid"}`),
	} {
		if rr.Code != http.StatusNotFound {
			t.Errorf("got %d %s, want 404", rr.Code, rr.Body)
		}
	}
	// only the two outcomes a bank can give
	if rr := devPay(h, c, o.ID, "refunded"); rr.Code != http.StatusBadRequest {
		t.Errorf("unknown outcome = %d, want 400", rr.Code)
	}
	if rr := doCookie(h, c, "POST", fmt.Sprintf("/api/dev/pay/%d", o.ID), `nope`); rr.Code != http.StatusBadRequest {
		t.Errorf("bad body = %d, want 400", rr.Code)
	}
	// nobody without a session
	rr := httptest.NewRecorder()
	anon := httptest.NewRequest("POST", fmt.Sprintf("/api/dev/pay/%d", o.ID), strings.NewReader(`{"outcome":"paid"}`))
	setTestOrigin(anon)
	h.ServeHTTP(rr, anon)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("no session = %d, want 401", rr.Code)
	}
	if p, _ := st.PaymentByID(o.ID); p.Status != store.PaymentPending {
		t.Fatalf("status = %q after refused attempts, want pending", p.Status)
	}
}

// With the flag off the endpoint does not exist, whatever the payment is: a real order of a real
// service cannot be settled by pressing a button.
func TestTheStandInDeskDoesNothingWhenItIsSwitchedOff(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPI(t, tb)
	uid, c := listedUser(t, h, st)
	o := order(t, h, c, "level")
	if strings.HasPrefix(o.ConfirmationURL, "/dev/") {
		t.Fatalf("confirmation url %q: the stand-in desk was used without the flag", o.ConfirmationURL)
	}
	if rr := devPay(h, c, o.ID, "paid"); rr.Code != http.StatusNotFound {
		t.Fatalf("pay with the flag off = %d, want 404", rr.Code)
	}
	if rr := doCookie(h, c, "GET", fmt.Sprintf("/api/dev/pay/%d", o.ID), ""); rr.Code != http.StatusNotFound {
		t.Fatalf("info with the flag off = %d, want 404", rr.Code)
	}
	if p, _ := st.PaymentByID(o.ID); p.Status != store.PaymentPending {
		t.Fatalf("status = %q, want pending", p.Status)
	}
	// even an order that did come from the stand-in desk (a database copied from a developer's
	// machine, say) cannot be settled by a button where the desk is off
	pid, err := st.CreatePayment(store.Payment{UserID: uid, TariffCode: store.TariffLevel, AmountKopecks: 19900, Provider: "dev"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_ = st.SetPaymentProviderID(pid, fmt.Sprintf("dev-%d-19900", pid), true)
	if rr := devPay(h, c, pid, "paid"); rr.Code != http.StatusNotFound {
		t.Fatalf("pay a stand-in order with the flag off = %d, want 404", rr.Code)
	}
	if p, _ := st.PaymentByID(pid); p.Status != store.PaymentPending {
		t.Fatalf("stand-in order status = %q, want pending", p.Status)
	}
}

// A payment the bank took is not the stand-in's to settle, even with the flag on.
func TestTheStandInDeskCannotSettleAPaymentOfARealService(t *testing.T) {
	h, st := devAPI(t)
	uid, c := listedUser(t, h, st)
	pid, err := st.CreatePayment(store.Payment{
		UserID: uid, TariffCode: store.TariffLevel, AmountKopecks: 19900, Provider: "tbank",
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetPaymentProviderID(pid, "9001", false); err != nil {
		t.Fatal(err)
	}
	if rr := devPay(h, c, pid, "paid"); rr.Code != http.StatusNotFound {
		t.Fatalf("pay = %d %s, want 404", rr.Code, rr.Body)
	}
	if p, _ := st.PaymentByID(pid); p.Status != store.PaymentPending {
		t.Fatalf("status = %q, want it untouched", p.Status)
	}
}
