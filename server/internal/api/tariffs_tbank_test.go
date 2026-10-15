package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/tbank"
)

// The account that pays through Т-Касса; listedEmail pays through ЮKassa.
const tbankEmail = "man30968@gmail.com"

// fakeTBank stands in for securepay.tinkoff.ru. Like the bank it refuses a
// request whose Token does not verify, and it lets a test decide how each payment
// ended.
type fakeTBank struct {
	mu       sync.Mutex
	srv      *httptest.Server
	terminal string
	password string
	payments map[string]*tbPayment
	getFails bool
	failInit bool
}

type tbPayment struct {
	ID      string
	OrderID string
	Status  string
	Amount  int64
	Request map[string]any
}

func newFakeTBank(t *testing.T) *fakeTBank {
	t.Helper()
	f := &fakeTBank{terminal: "TERM1DEMO", password: "pw-1", payments: map[string]*tbPayment{}}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeTBank) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	body, _ := io.ReadAll(r.Body)
	var m map[string]any
	d := json.NewDecoder(strings.NewReader(string(body)))
	d.UseNumber()
	_ = d.Decode(&m)
	if r.Method != "POST" || m == nil || m["Token"] != tbank.Sign(m, f.password) || m["TerminalKey"] != f.terminal {
		io.WriteString(w, `{"Success":false,"ErrorCode":"4","Message":"Неверный токен. Группа реквизитов не прошла проверку подлинности"}`)
		return
	}
	switch r.URL.Path {
	case "/v2/Init":
		if f.failInit {
			io.WriteString(w, `{"Success":false,"ErrorCode":"501","Message":"Неверные параметры","Details":"nope"}`)
			return
		}
		amount, _ := m["Amount"].(json.Number).Int64()
		id := fmt.Sprintf("%d", 9000+len(f.payments)+1)
		f.payments[id] = &tbPayment{ID: id, OrderID: m["OrderId"].(string), Status: "NEW", Amount: amount, Request: m}
		fmt.Fprintf(w, `{"Success":true,"ErrorCode":"0","TerminalKey":%q,"Status":"NEW","PaymentId":%q,"OrderId":%q,"Amount":%d,"PaymentURL":"https://pay.test/tb/%s"}`,
			f.terminal, id, m["OrderId"], amount, id)
	case "/v2/GetState":
		if f.getFails {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		p := f.payments[m["PaymentId"].(string)]
		if p == nil {
			io.WriteString(w, `{"Success":false,"ErrorCode":"3","Message":"Платеж не найден"}`)
			return
		}
		fmt.Fprintf(w, `{"Success":true,"ErrorCode":"0","Message":"OK","TerminalKey":%q,"Status":%q,"PaymentId":%q,"OrderId":%q,"Amount":%d}`,
			f.terminal, p.Status, p.ID, p.OrderID, p.Amount)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeTBank) set(id, status string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.payments[id].Status = status
}

func (f *fakeTBank) get(id string) tbPayment {
	f.mu.Lock()
	defer f.mu.Unlock()
	return *f.payments[id]
}

func (f *fakeTBank) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.payments)
}

// notification is the bank's POST body for a payment, signed with the terminal
// password as the bank signs it.
func (f *fakeTBank) notification(id, status string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := f.payments[id]
	m := map[string]any{
		"TerminalKey": f.terminal, "OrderId": p.OrderID, "Success": true, "Status": status,
		"PaymentId": json.Number(id), "ErrorCode": "0", "Amount": json.Number(fmt.Sprint(p.Amount)),
		"Pan": "430000******0777", "ExpDate": "1230", "CardId": json.Number("7"),
		"DATA": map[string]any{"OperationInitiatorType": "0"},
	}
	m["Token"] = tbank.Sign(m, f.password)
	b, _ := json.Marshal(m)
	return string(b)
}

func tbankAccount(t *testing.T, h http.Handler, st *store.Store) (string, *http.Cookie) {
	t.Helper()
	uid := registerAndVerify(t, h, st, tbankEmail, "secret1234", "Man")
	return uid, authed(t, st, uid)
}

// ---- who pays through whom ----

func TestEachListedAccountPaysThroughItsOwnService(t *testing.T) {
	yk, tb := newFakeYooKassa(t), newFakeTBank(t)
	h, st := tariffsAPIWith(t, yk, tb)
	_, gc := listedUser(t, h, st)
	_, mc := tbankAccount(t, h, st)

	og := order(t, h, gc, "level")
	if !strings.HasPrefix(og.ConfirmationURL, "https://pay.test/confirm/") || yk.count() != 1 || tb.count() != 0 {
		t.Fatalf("the ЮKassa account's order = %+v (ЮKassa %d, Т-Касса %d payments)", og, yk.count(), tb.count())
	}
	om := order(t, h, mc, "level")
	if !strings.HasPrefix(om.ConfirmationURL, "https://pay.test/tb/") || tb.count() != 1 || yk.count() != 1 {
		t.Fatalf("the Т-Касса account's order = %+v (ЮKassa %d, Т-Касса %d payments)", om, yk.count(), tb.count())
	}
	if p, _ := st.PaymentByID(og.ID); p.Provider != "yookassa" {
		t.Errorf("first payment provider = %q", p.Provider)
	}
	p, _ := st.PaymentByID(om.ID)
	if p.Provider != "tbank" || p.ProviderPaymentID != "9001" || !p.Test {
		t.Errorf("second payment = %+v, want tbank / 9001 / test (a DEMO terminal)", p)
	}
	for _, c := range []*http.Cookie{gc, mc} {
		if rr := doCookie(h, c, "GET", "/api/auth/session", ""); !decodeBody[sessionUserDTO](t, rr).Features.Tariffs {
			t.Error("features.tariffs is off for a listed account")
		}
	}
}

func TestEveryoneElseSeesNoPaymentsEvenWithBothServicesConnected(t *testing.T) {
	h, st := tariffsAPIWith(t, newFakeYooKassa(t), newFakeTBank(t))
	other := registerAndVerify(t, h, st, "someone@example.com", "secret1234", "Other")
	oc := authed(t, st, other)
	for _, c := range []struct{ method, path, body string }{
		{"GET", "/api/me/tariffs", ""},
		{"POST", "/api/me/orders", `{"tariff":"year"}`},
		{"GET", "/api/me/orders/1", ""},
	} {
		if rr := doCookie(h, oc, c.method, c.path, c.body); rr.Code != http.StatusNotFound {
			t.Errorf("%s %s = %d, want 404", c.method, c.path, rr.Code)
		}
	}
	if decodeBody[sessionUserDTO](t, doCookie(h, oc, "GET", "/api/auth/session", "")).Features.Tariffs {
		t.Error("features.tariffs is on for an unlisted account")
	}
}

// Only a verified address counts for Т-Касса too.
func TestAnUnverifiedAccountWithTheTBankAddressSeesNothing(t *testing.T) {
	h, st := tariffsAPIWith(t, nil, newFakeTBank(t))
	if rr := anon(h, "POST", "/api/auth/register", registerBody(tbankEmail, "secret1234", "Impostor")); rr.Code != 200 {
		t.Fatalf("register = %d", rr.Code)
	}
	id, err := st.IdentityByProviderUID("password", tbankEmail)
	if err != nil {
		t.Fatal(err)
	}
	if rr := doCookie(h, authed(t, st, id.UserID), "GET", "/api/me/tariffs", ""); rr.Code != http.StatusNotFound {
		t.Fatalf("GET tariffs = %d, want 404", rr.Code)
	}
}

// Each account needs ITS service connected: ЮKassa's keys do not make Т-Касса work.
func TestAnAccountWhoseServiceIsNotConnectedCannotBuy(t *testing.T) {
	h, st := tariffsAPIWith(t, newFakeYooKassa(t), nil) // ЮKassa only
	_, mc := tbankAccount(t, h, st)
	_, gc := listedUser(t, h, st)

	if got := getTariffsOf(t, h, mc); got.PaymentsReady {
		t.Error("payments_ready for the Т-Касса account with no terminal")
	}
	rr := doCookie(h, mc, "POST", "/api/me/orders", `{"tariff":"level"}`)
	if rr.Code != http.StatusServiceUnavailable || !strings.Contains(rr.Body.String(), "payments_unavailable") {
		t.Fatalf("order = %d %s, want 503 payments_unavailable", rr.Code, rr.Body)
	}
	if got := getTariffsOf(t, h, gc); !got.PaymentsReady {
		t.Error("the ЮKassa account lost its payments because Т-Касса is missing")
	}
	if hook := anon(h, "POST", "/api/tbank/webhook", `{"Status":"CONFIRMED"}`); hook.Code != 200 || hook.Body.String() != "OK" {
		t.Errorf("webhook without a terminal = %d %q, want a quiet 200 OK", hook.Code, hook.Body)
	}
}

// ---- ordering and paying through Т-Касса ----

func TestOrderingALevelAsksTheBankForExactlyThatPayment(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPIWith(t, nil, tb)
	uid, c := tbankAccount(t, h, st)

	o := order(t, h, c, "level")
	if o.Status != "pending" || o.ConfirmationURL != "https://pay.test/tb/9001" {
		t.Fatalf("order = %+v, want pending with the bank's page", o)
	}
	req := tb.get("9001").Request
	num := func(k string) string { return req[k].(json.Number).String() }
	if num("Amount") != "24900" {
		t.Errorf("Amount = %s, want 24900 kopecks (first purchase)", num("Amount"))
	}
	if oid := req["OrderId"].(string); !strings.HasPrefix(oid, fmt.Sprintf("%d-", o.ID)) || len(oid) > 50 {
		t.Errorf("OrderId = %q, want our order id then a time, within the bank's 50 characters", oid)
	}
	ret := fmt.Sprintf("%s/tariffs?order=%d", testBaseURL, o.ID)
	if req["SuccessURL"] != ret || req["FailURL"] != ret {
		t.Errorf("SuccessURL/FailURL = %v / %v, want the tariffs page for this order", req["SuccessURL"], req["FailURL"])
	}
	if req["NotificationURL"] != testBaseURL+"/api/tbank/webhook" {
		t.Errorf("NotificationURL = %v", req["NotificationURL"])
	}
	if req["PayType"] != "O" || !strings.Contains(req["Description"].(string), "Падежи") {
		t.Errorf("PayType/Description = %v / %v", req["PayType"], req["Description"])
	}
	rec := req["Receipt"].(map[string]any)
	item := rec["Items"].([]any)[0].(map[string]any)
	if rec["Email"] != tbankEmail || rec["Taxation"] != "usn_income" ||
		item["Amount"].(json.Number).String() != "24900" || item["Tax"] != "none" {
		t.Errorf("receipt = %v", rec)
	}

	p, err := st.PaymentByID(o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.UserID != uid || p.GrantedPhase != "B" || p.AmountKopecks != 24900 || p.Status != "pending" {
		t.Errorf("payment row = %+v", p)
	}
	if !phaseLocked(t, h, c, "B") {
		t.Error("level B opened before anything was paid")
	}
}

func TestReturningFromTheBankOpensTheLevelThroughTBank(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPIWith(t, nil, tb)
	_, c := tbankAccount(t, h, st)
	o := order(t, h, c, "level")
	path := fmt.Sprintf("/api/me/orders/%d", o.ID)

	// still on the bank's form, then the card is accepted but the money not yet taken
	for _, status := range []string{"FORM_SHOWED", "AUTHORIZED"} {
		tb.set("9001", status)
		if got := decodeBody[orderDTO](t, doCookie(h, c, "GET", path, "")).Status; got != "pending" {
			t.Fatalf("%s: status = %s, want pending", status, got)
		}
	}
	tb.set("9001", "CONFIRMED")
	if got := decodeBody[orderDTO](t, doCookie(h, c, "GET", path, "")).Status; got != "succeeded" {
		t.Fatalf("status after CONFIRMED = %s", got)
	}
	if phaseLocked(t, h, c, "B") {
		t.Error("level B still locked after the payment was confirmed")
	}
	if !phaseLocked(t, h, c, "C") {
		t.Error("level C opened too")
	}
	if got := getTariffsOf(t, h, c); !got.HasPaidLevel || got.Level.Price != 299 || got.Level.Next == nil || got.Level.Next.ID != "C" {
		t.Errorf("after the first level: %+v", got)
	}
}

func TestAYearBoughtThroughTBankOpensEveryLevel(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPIWith(t, nil, tb)
	_, c := tbankAccount(t, h, st)
	order(t, h, c, "year")
	if tb.get("9001").Request["Amount"].(json.Number).String() != "100000" {
		t.Errorf("year amount = %v, want 100000 kopecks", tb.get("9001").Request["Amount"])
	}
	tb.set("9001", "CONFIRMED")
	got := getTariffsOf(t, h, c) // opening the screen settles
	if got.Year.Until == "" {
		t.Fatalf("no running year after the payment: %+v", got)
	}
	if phaseLocked(t, h, c, "B") || phaseLocked(t, h, c, "C") {
		t.Error("a level is still locked during a paid year")
	}
}

func TestARejectedBankPaymentOpensNothingAndCanBeRetried(t *testing.T) {
	for _, status := range []string{"REJECTED", "AUTH_FAIL", "CANCELED", "DEADLINE_EXPIRED"} {
		tb := newFakeTBank(t)
		h, st := tariffsAPIWith(t, nil, tb)
		_, c := tbankAccount(t, h, st)
		o := order(t, h, c, "level")
		tb.set("9001", status)
		if got := decodeBody[orderDTO](t, doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")).Status; got != "canceled" {
			t.Errorf("%s: status = %s, want canceled", status, got)
		}
		if !phaseLocked(t, h, c, "B") {
			t.Errorf("%s: level B opened", status)
		}
		if again := order(t, h, c, "level"); again.ID == o.ID || again.Status != "pending" {
			t.Errorf("%s: a retry gave %+v", status, again)
		}
	}
}

func TestWhenTheBankRefusesTheOrderIsClosed(t *testing.T) {
	tb := newFakeTBank(t)
	tb.failInit = true
	h, st := tariffsAPIWith(t, nil, tb)
	uid, c := tbankAccount(t, h, st)
	rr := doCookie(h, c, "POST", "/api/me/orders", `{"tariff":"level"}`)
	if rr.Code != http.StatusBadGateway || !strings.Contains(rr.Body.String(), "payment_failed") {
		t.Fatalf("order = %d %s, want 502 payment_failed", rr.Code, rr.Body)
	}
	pending, _ := st.PendingPayments(uid, fixedNow.Add(-time.Hour))
	if len(pending) != 0 {
		t.Errorf("%d payments left pending after the bank refused", len(pending))
	}
}

// A payment whose amount or order is not ours is never granted, whatever its status.
func TestABankPaymentOfTheWrongAmountOrOrderIsNeverGranted(t *testing.T) {
	for name, tamper := range map[string]func(p *tbPayment){
		"amount":      func(p *tbPayment) { p.Amount = 100 },
		"other order": func(p *tbPayment) { p.OrderID = "999-1" },
	} {
		tb := newFakeTBank(t)
		h, st := tariffsAPIWith(t, nil, tb)
		_, c := tbankAccount(t, h, st)
		o := order(t, h, c, "level")
		tb.mu.Lock()
		tamper(tb.payments["9001"])
		tb.payments["9001"].Status = "CONFIRMED"
		tb.mu.Unlock()
		doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
		if p, _ := st.PaymentByID(o.ID); p.Status != "pending" {
			t.Errorf("%s: status = %s, want it left pending", name, p.Status)
		}
		if !phaseLocked(t, h, c, "B") {
			t.Errorf("%s: level B opened", name)
		}
	}
}

// ---- the bank's notification ----

func TestASignedNotificationOpensTheLevelWithoutAnySession(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPIWith(t, nil, tb)
	_, c := tbankAccount(t, h, st)
	order(t, h, c, "level")
	tb.set("9001", "CONFIRMED")

	// no cookie and no Origin header: the bank sends neither
	rr := anon(h, "POST", "/api/tbank/webhook", tb.notification("9001", "CONFIRMED"))
	if rr.Code != 200 || rr.Body.String() != "OK" {
		t.Fatalf("webhook = %d %q, want 200 with the body OK the bank waits for", rr.Code, rr.Body)
	}
	if phaseLocked(t, h, c, "B") {
		t.Error("level B still locked after the notification")
	}
	// the bank sends AUTHORIZED and CONFIRMED together, and retries; nothing may change
	for _, status := range []string{"AUTHORIZED", "CONFIRMED"} {
		if rr := anon(h, "POST", "/api/tbank/webhook", tb.notification("9001", status)); rr.Code != 200 || rr.Body.String() != "OK" {
			t.Fatalf("%s delivery = %d %q", status, rr.Code, rr.Body)
		}
	}
	if p, _ := st.PaymentByProviderID("tbank", "9001"); p.Status != "succeeded" {
		t.Errorf("status = %s", p.Status)
	}
}

func TestANotificationThatIsNotSignedByTheBankIsRefusedAndOpensNothing(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPIWith(t, nil, tb)
	_, c := tbankAccount(t, h, st)
	order(t, h, c, "level")
	tb.set("9001", "CONFIRMED") // even if the bank did mark it paid

	good := tb.notification("9001", "CONFIRMED")
	for name, body := range map[string]string{
		"altered after signing": strings.Replace(good, `"CONFIRMED"`, `"CONFIRMED "`, 1),
		"no token":              strings.Replace(good, `"Token":`, `"Tok":`, 1),
		"wrong password":        mustJSON(map[string]any{"TerminalKey": tb.terminal, "PaymentId": 9001, "Status": "CONFIRMED", "Token": tbank.Sign(map[string]any{"TerminalKey": tb.terminal, "PaymentId": 9001, "Status": "CONFIRMED"}, "guess")}),
		"not json":              "CONFIRMED",
	} {
		if rr := anon(h, "POST", "/api/tbank/webhook", body); rr.Code != http.StatusBadRequest {
			t.Errorf("%s: webhook = %d, want 400", name, rr.Code)
		}
	}
	if !phaseLocked(t, h, c, "B") {
		t.Fatal("an unsigned notification opened a level")
	}
}

// A correctly signed notification is only a nudge: if the bank's own answer says
// the payment is not paid, nothing opens.
func TestASignedNotificationForAnUnpaidPaymentOpensNothing(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPIWith(t, nil, tb)
	_, c := tbankAccount(t, h, st)
	order(t, h, c, "level") // NEW at the bank

	if rr := anon(h, "POST", "/api/tbank/webhook", tb.notification("9001", "CONFIRMED")); rr.Code != 200 {
		t.Fatalf("webhook = %d", rr.Code)
	}
	if !phaseLocked(t, h, c, "B") {
		t.Fatal("a notification claiming success opened a level the bank has as NEW")
	}
	if p, _ := st.PaymentByProviderID("tbank", "9001"); p.Status != "pending" {
		t.Fatalf("status = %s, want pending", p.Status)
	}
}

func TestBankNotificationsWeHaveNothingToDoWithGetAQuietOK(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPIWith(t, nil, tb)
	_, c := tbankAccount(t, h, st)
	order(t, h, c, "level")

	signed := func(m map[string]any) string {
		m["TerminalKey"] = tb.terminal
		m["Token"] = tbank.Sign(m, tb.password)
		return mustJSON(m)
	}
	for name, body := range map[string]string{
		"a payment that is not ours": signed(map[string]any{"PaymentId": 424242, "Status": "CONFIRMED", "Success": true}),
		"a receipt notification":     signed(map[string]any{"PaymentId": 9001, "Type": "IncomeReceipt", "Success": true}),
	} {
		rr := anon(h, "POST", "/api/tbank/webhook", body)
		if rr.Code != 200 || rr.Body.String() != "OK" {
			t.Errorf("%s: %d %q, want 200 OK", name, rr.Code, rr.Body)
		}
	}
	if !phaseLocked(t, h, c, "B") {
		t.Error("something opened level B")
	}
}

func TestTheBankNotificationIsRetriedWhenWeCannotAskTheBank(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPIWith(t, nil, tb)
	_, c := tbankAccount(t, h, st)
	order(t, h, c, "level")
	tb.set("9001", "CONFIRMED")
	body := tb.notification("9001", "CONFIRMED")

	tb.getFails = true
	if rr := anon(h, "POST", "/api/tbank/webhook", body); rr.Code != http.StatusInternalServerError {
		t.Fatalf("webhook while the bank is down = %d, want 500 so that the bank retries", rr.Code)
	}
	tb.getFails = false
	if rr := anon(h, "POST", "/api/tbank/webhook", body); rr.Code != 200 {
		t.Fatalf("webhook after = %d", rr.Code)
	}
	if phaseLocked(t, h, c, "B") {
		t.Error("level B still locked after the retry")
	}
}

// A ЮKassa payment is still settled by ЮKassa when Т-Касса is connected too, and
// the other way round: the payment row, not the account, says whose it is.
func TestEachPaymentIsSettledWithTheServiceThatMadeIt(t *testing.T) {
	yk, tb := newFakeYooKassa(t), newFakeTBank(t)
	h, st := tariffsAPIWith(t, yk, tb)
	_, gc := listedUser(t, h, st)
	_, mc := tbankAccount(t, h, st)
	og, om := order(t, h, gc, "level"), order(t, h, mc, "level")

	yk.set("yk-1", "succeeded") // only the ЮKassa payment was paid
	doCookie(h, gc, "GET", fmt.Sprintf("/api/me/orders/%d", og.ID), "")
	if phaseLocked(t, h, gc, "B") {
		t.Error("the ЮKassa account's level did not open")
	}
	if !phaseLocked(t, h, mc, "B") {
		t.Error("the Т-Касса account's level opened though only ЮKassa's payment was paid")
	}
	tb.set("9001", "CONFIRMED")
	doCookie(h, mc, "GET", fmt.Sprintf("/api/me/orders/%d", om.ID), "")
	if phaseLocked(t, h, mc, "B") {
		t.Error("the Т-Касса account's level did not open once the bank confirmed")
	}
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
