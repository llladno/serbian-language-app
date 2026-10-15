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

	"github.com/grisha/serbian-app/server/internal/content"
	"github.com/grisha/serbian-app/server/internal/store"
)

const listedEmail = "gregorymansurov@gmail.com"

// fakeYooKassa stands in for api.yookassa.ru: it records what it was asked and
// lets a test decide how each payment ended, the way ЮKassa's own page would.
type fakeYooKassa struct {
	mu       sync.Mutex
	srv      *httptest.Server
	payments map[string]*fakePayment
	order    []string
	failNext int // answer this many creates with 400
	getFails bool
}

type fakePayment struct {
	ID       string
	Status   string
	Amount   string
	Metadata map[string]string
	Request  map[string]any
	User     string
	Pass     string
	IdemKey  string
}

func newFakeYooKassa(t *testing.T) *fakeYooKassa {
	t.Helper()
	f := &fakeYooKassa{payments: map[string]*fakePayment{}}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeYooKassa) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == "POST" && r.URL.Path == "/payments":
		if f.failNext > 0 {
			f.failNext--
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"type":"error","code":"invalid_request","description":"nope"}`)
			return
		}
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		user, pass, _ := r.BasicAuth()
		id := fmt.Sprintf("yk-%d", len(f.payments)+1)
		meta := map[string]string{}
		for k, v := range req["metadata"].(map[string]any) {
			meta[k] = v.(string)
		}
		p := &fakePayment{ID: id, Status: "pending", Amount: req["amount"].(map[string]any)["value"].(string),
			Metadata: meta, Request: req, User: user, Pass: pass, IdemKey: r.Header.Get("Idempotence-Key")}
		f.payments[id] = p
		f.order = append(f.order, id)
		io.WriteString(w, f.json(p))
	case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/payments/"):
		if f.getFails {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		p := f.payments[strings.TrimPrefix(r.URL.Path, "/payments/")]
		if p == nil {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, `{"type":"error","code":"not_found"}`)
			return
		}
		io.WriteString(w, f.json(p))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeYooKassa) json(p *fakePayment) string {
	meta, _ := json.Marshal(p.Metadata)
	conf := ""
	if p.Status == "pending" {
		conf = fmt.Sprintf(`,"confirmation":{"type":"redirect","confirmation_url":"https://pay.test/confirm/%s"}`, p.ID)
	}
	return fmt.Sprintf(`{"id":%q,"status":%q,"paid":%t,"test":true,"amount":{"value":%q,"currency":"RUB"},"metadata":%s%s}`,
		p.ID, p.Status, p.Status == "succeeded", p.Amount, meta, conf)
}

func (f *fakeYooKassa) set(id, status string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.payments[id].Status = status
}

func (f *fakeYooKassa) get(id string) fakePayment {
	f.mu.Lock()
	defer f.mu.Unlock()
	return *f.payments[id]
}

func (f *fakeYooKassa) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.payments)
}

// tariffsAPI is the app with three levels (A free, B and C for sale), payments
// pointed at the fake, and the listed email allowed to use the tariffs.
func tariffsAPI(t *testing.T, yk *fakeYooKassa) (http.Handler, *store.Store) {
	t.Helper()
	return tariffsAPIWith(t, yk, nil)
}

// tariffsAPIWith is tariffsAPI with Т-Касса as well: tbankEmail pays through the
// fake bank, listedEmail through the fake ЮKassa. Either fake may be nil, which
// leaves that service without credentials.
func tariffsAPIWith(t *testing.T, yk *fakeYooKassa, tb *fakeTBank) (http.Handler, *store.Store) {
	t.Helper()
	h, st := newTestAPIWith(t, func(d *Deps) {
		d.Async = func(f func()) { f() }
		base := d.Course()
		three := *base
		three.Phases = []content.Phase{
			{ID: "A", Title: "Уровень 3", Lessons: []string{"02"}},
			{ID: "B", Title: "Уровень 4 — Падежи", Lessons: []string{"01"}},
			{ID: "C", Title: "Уровень 5 — Жизнь", Lessons: []string{"90", "91"}},
		}
		d.Course = func() *content.Course { return &three }
		d.Config.TariffsEmails = []string{listedEmail}
		d.Config.TBankEmails = []string{tbankEmail}
		if yk != nil {
			d.Config.YooKassaShopID = "shop-1"
			d.Config.YooKassaSecretKey = "secret-1"
			d.Config.YooKassaAPIBase = yk.srv.URL
		}
		if tb != nil {
			d.Config.TBankTerminalKey = tb.terminal
			d.Config.TBankPassword = tb.password
			d.Config.TBankAPIBase = tb.srv.URL
			d.Config.TBankTaxation = "usn_income"
		}
	})
	seedPhaseProduct(t, st, "B", 230)
	seedPhaseProduct(t, st, "C", 375)
	return h, st
}

func listedUser(t *testing.T, h http.Handler, st *store.Store) (string, *http.Cookie) {
	t.Helper()
	uid := registerAndVerify(t, h, st, listedEmail, "secret1234", "Greg")
	return uid, authed(t, st, uid)
}

type tariffsResp struct {
	PaymentsReady bool `json:"payments_ready"`
	HasPaidLevel  bool `json:"has_paid_level"`
	Level         struct {
		Price        int64 `json:"price"`
		RegularPrice int64 `json:"regular_price"`
		Next         *struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"next"`
	} `json:"level"`
	Year struct {
		Price    int64  `json:"price"`
		OldPrice int64  `json:"old_price"`
		Until    string `json:"until"`
	} `json:"year"`
}

func getTariffsOf(t *testing.T, h http.Handler, c *http.Cookie) tariffsResp {
	t.Helper()
	rr := doCookie(h, c, "GET", "/api/me/tariffs", "")
	if rr.Code != 200 {
		t.Fatalf("GET tariffs = %d %s", rr.Code, rr.Body)
	}
	return decodeBody[tariffsResp](t, rr)
}

func order(t *testing.T, h http.Handler, c *http.Cookie, tariff string) orderDTO {
	t.Helper()
	rr := doCookie(h, c, "POST", "/api/me/orders", fmt.Sprintf(`{"tariff":%q}`, tariff))
	if rr.Code != 200 {
		t.Fatalf("POST order %s = %d %s", tariff, rr.Code, rr.Body)
	}
	return decodeBody[orderDTO](t, rr)
}

func phaseLocked(t *testing.T, h http.Handler, c *http.Cookie, id string) bool {
	t.Helper()
	rr := doCookie(h, c, "GET", "/api/course", "")
	var course struct {
		Phases []struct {
			ID     string `json:"id"`
			Locked bool   `json:"locked"`
		} `json:"phases"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &course); err != nil {
		t.Fatal(err)
	}
	for _, p := range course.Phases {
		if p.ID == id {
			return p.Locked
		}
	}
	t.Fatalf("no phase %s", id)
	return false
}

// ---- who sees it ----

func TestTariffsAreClosedToEveryoneButTheListedVerifiedAccount(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := tariffsAPI(t, yk)

	other := registerAndVerify(t, h, st, "someone@example.com", "secret1234", "Other")
	oc := authed(t, st, other)
	for _, c := range []struct{ method, path, body string }{
		{"GET", "/api/me/tariffs", ""},
		{"POST", "/api/me/orders", `{"tariff":"year"}`},
		{"GET", "/api/me/orders/1", ""},
	} {
		if rr := doCookie(h, oc, c.method, c.path, c.body); rr.Code != http.StatusNotFound {
			t.Errorf("%s %s as an unlisted account = %d, want 404 (as if it did not exist)", c.method, c.path, rr.Code)
		}
	}
	if rr := anon(h, "GET", "/api/me/tariffs", ""); rr.Code != http.StatusUnauthorized {
		t.Errorf("GET tariffs without a session = %d, want 401", rr.Code)
	}
	if f := decodeBody[sessionUserDTO](t, doCookie(h, oc, "GET", "/api/auth/session", "")).Features; f.Tariffs {
		t.Error("features.tariffs is on for an unlisted account")
	}

	_, lc := listedUser(t, h, st)
	if rr := doCookie(h, lc, "GET", "/api/me/tariffs", ""); rr.Code != 200 {
		t.Fatalf("GET tariffs as the listed account = %d %s", rr.Code, rr.Body)
	}
	if f := decodeBody[sessionUserDTO](t, doCookie(h, lc, "GET", "/api/auth/session", "")).Features; !f.Tariffs {
		t.Error("features.tariffs is off for the listed account")
	}
}

// Typing someone's address into the sign-up form must not unlock anything: only
// a verified address counts.
func TestAnUnverifiedAccountWithTheListedAddressSeesNothing(t *testing.T) {
	h, st := tariffsAPI(t, newFakeYooKassa(t))
	rr := anon(h, "POST", "/api/auth/register", registerBody(listedEmail, "secret1234", "Impostor"))
	if rr.Code != 200 {
		t.Fatalf("register = %d", rr.Code)
	}
	id, err := st.IdentityByProviderUID("password", listedEmail)
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	c := authed(t, st, id.UserID) // a session without the email ever being verified
	if rr := doCookie(h, c, "GET", "/api/me/tariffs", ""); rr.Code != http.StatusNotFound {
		t.Fatalf("GET tariffs with an unverified listed address = %d, want 404", rr.Code)
	}
}

func TestTheListMatchesCaseInsensitivelyAndAnEmptyListMatchesNobody(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := newTestAPIWith(t, func(d *Deps) {
		d.Async = func(f func()) { f() }
		d.Config.TariffsEmails = []string{"gregorymansurov@gmail.com"}
		d.Config.YooKassaShopID, d.Config.YooKassaSecretKey, d.Config.YooKassaAPIBase = "s", "k", yk.srv.URL
	})
	uid := registerAndVerify(t, h, st, "GregoryMansurov@Gmail.com", "secret1234", "Greg")
	if rr := doCookie(h, authed(t, st, uid), "GET", "/api/me/tariffs", ""); rr.Code != 200 {
		t.Fatalf("mixed-case address = %d, want 200", rr.Code)
	}

	h2, st2 := newTestAPIWith(t, func(d *Deps) { d.Async = func(f func()) { f() } }) // nobody listed
	uid2 := registerAndVerify(t, h2, st2, listedEmail, "secret1234", "Greg")
	if rr := doCookie(h2, authed(t, st2, uid2), "GET", "/api/me/tariffs", ""); rr.Code != http.StatusNotFound {
		t.Fatalf("with no list configured = %d, want 404 for everybody", rr.Code)
	}
}

func TestATelegramOnlyAccountHasNoEmailSoSeesNothing(t *testing.T) {
	h, st := tariffsAPI(t, newFakeYooKassa(t))
	uid, err := st.CreateUser("Tg")
	if err != nil {
		t.Fatal(err)
	}
	if rr := doCookie(h, authed(t, st, uid), "GET", "/api/me/tariffs", ""); rr.Code != http.StatusNotFound {
		t.Fatalf("GET tariffs for an account with no email = %d, want 404", rr.Code)
	}
}

// ---- what they see ----

func TestTariffsShowTheAgreedPricesAndTheNextLevel(t *testing.T) {
	h, st := tariffsAPI(t, newFakeYooKassa(t))
	_, c := listedUser(t, h, st)
	got := getTariffsOf(t, h, c)

	if !got.PaymentsReady || got.HasPaidLevel {
		t.Errorf("flags = %+v", got)
	}
	if got.Level.Price != 249 || got.Level.RegularPrice != 299 {
		t.Errorf("level price = %d (regular %d), want 249 for the first purchase and 299 after", got.Level.Price, got.Level.RegularPrice)
	}
	if got.Level.Next == nil || got.Level.Next.ID != "B" || !strings.Contains(got.Level.Next.Title, "Падежи") {
		t.Errorf("next level = %+v, want B", got.Level.Next)
	}
	if got.Year.Price != 1000 || got.Year.OldPrice != 2000 || got.Year.Until != "" {
		t.Errorf("year = %+v, want 1000 (2000 struck) and no running year", got.Year)
	}
}

func TestWithoutShopCredentialsThePageShowsButNothingCanBeBought(t *testing.T) {
	h, st := tariffsAPI(t, nil)
	_, c := listedUser(t, h, st)
	if got := getTariffsOf(t, h, c); got.PaymentsReady {
		t.Error("payments_ready with no credentials")
	}
	rr := doCookie(h, c, "POST", "/api/me/orders", `{"tariff":"level"}`)
	if rr.Code != http.StatusServiceUnavailable || !strings.Contains(rr.Body.String(), "payments_unavailable") {
		t.Fatalf("order without credentials = %d %s, want 503 payments_unavailable", rr.Code, rr.Body)
	}
	if hook := anon(h, "POST", "/api/yookassa/webhook", `{"event":"payment.succeeded","object":{"id":"x"}}`); hook.Code != 200 {
		t.Errorf("webhook without credentials = %d, want a quiet 200", hook.Code)
	}
}

// ---- ordering and paying ----

func TestOrderingALevelAsksYooKassaForExactlyThatPayment(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := tariffsAPI(t, yk)
	uid, c := listedUser(t, h, st)

	o := order(t, h, c, "level")
	if o.Status != "pending" || !strings.HasPrefix(o.ConfirmationURL, "https://pay.test/confirm/") {
		t.Fatalf("order = %+v, want pending with ЮKassa's page", o)
	}
	p := yk.get("yk-1")
	if p.Amount != "249.00" {
		t.Errorf("amount = %s, want 249.00 (first purchase)", p.Amount)
	}
	if p.User != "shop-1" || p.Pass != "secret-1" {
		t.Errorf("auth = %s:%s", p.User, p.Pass)
	}
	if p.IdemKey != fmt.Sprintf("order-%d", o.ID) {
		t.Errorf("Idempotence-Key = %q: a retried request must not charge twice", p.IdemKey)
	}
	conf := p.Request["confirmation"].(map[string]any)
	if conf["return_url"] != fmt.Sprintf("%s/tariffs?order=%d", testBaseURL, o.ID) {
		t.Errorf("return_url = %v", conf["return_url"])
	}
	rec := p.Request["receipt"].(map[string]any)
	if rec["customer"].(map[string]any)["email"] != listedEmail {
		t.Errorf("receipt customer = %v, want the account's own address", rec["customer"])
	}
	if p.Metadata["order_id"] != fmt.Sprint(o.ID) {
		t.Errorf("metadata = %v", p.Metadata)
	}

	pay, err := st.PaymentByID(o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if pay.UserID != uid || pay.GrantedPhase != "B" || pay.AmountKopecks != 24900 || pay.Status != "pending" || !pay.Test {
		t.Errorf("payment row = %+v, want pending, level B fixed at order time, test", pay)
	}
	if !phaseLocked(t, h, c, "B") {
		t.Error("level B opened before anything was paid")
	}
}

func TestReturningFromTheBankOpensTheLevelAndTheNextOneCostsFullPrice(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := tariffsAPI(t, yk)
	_, c := listedUser(t, h, st)

	o := order(t, h, c, "level")
	// back on the site before paying: still pending
	rr := doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	if got := decodeBody[orderDTO](t, rr).Status; got != "pending" {
		t.Fatalf("status before paying = %s", got)
	}
	yk.set("yk-1", "succeeded")
	rr = doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	if got := decodeBody[orderDTO](t, rr).Status; got != "succeeded" {
		t.Fatalf("status after paying = %s", got)
	}
	if phaseLocked(t, h, c, "B") {
		t.Error("level B is still locked after the payment succeeded")
	}
	if !phaseLocked(t, h, c, "C") {
		t.Error("level C opened too: a level payment opens one level")
	}
	got := getTariffsOf(t, h, c)
	if !got.HasPaidLevel || got.Level.Price != 299 || got.Level.Next == nil || got.Level.Next.ID != "C" {
		t.Errorf("after the first level: %+v, want 299 and C next", got)
	}
	// the second level order is for C, at the regular price
	o2 := order(t, h, c, "level")
	if yk.get("yk-2").Amount != "299.00" {
		t.Errorf("second order amount = %s, want 299.00", yk.get("yk-2").Amount)
	}
	if p, _ := st.PaymentByID(o2.ID); p.GrantedPhase != "C" {
		t.Errorf("second order opens %s, want C", p.GrantedPhase)
	}
}

// Opening the tariffs screen finds a payment the notification has not reported.
func TestOpeningTheScreenSettlesAPaymentThatWasMadeInTheMeantime(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := tariffsAPI(t, yk)
	_, c := listedUser(t, h, st)
	order(t, h, c, "level")
	yk.set("yk-1", "succeeded")

	got := getTariffsOf(t, h, c)
	if got.Level.Next == nil || got.Level.Next.ID != "C" {
		t.Fatalf("next = %+v, want the paid level to be gone and C to be next", got.Level.Next)
	}
}

func TestTheNotificationOpensTheLevelWithoutAnySession(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := tariffsAPI(t, yk)
	_, c := listedUser(t, h, st)
	order(t, h, c, "level")
	yk.set("yk-1", "succeeded")

	body := `{"type":"notification","event":"payment.succeeded","object":{"id":"yk-1","status":"succeeded"}}`
	// no cookie and no Origin header: ЮKassa sends neither
	if rr := anon(h, "POST", "/api/yookassa/webhook", body); rr.Code != 200 {
		t.Fatalf("webhook = %d %s", rr.Code, rr.Body)
	}
	if phaseLocked(t, h, c, "B") {
		t.Error("level B still locked after the notification")
	}
	// ЮKassa retries; the second delivery must change nothing
	if rr := anon(h, "POST", "/api/yookassa/webhook", body); rr.Code != 200 {
		t.Fatalf("second delivery = %d", rr.Code)
	}
	p, _ := st.PaymentByProviderID("yookassa", "yk-1")
	if p.Status != "succeeded" {
		t.Errorf("status = %s", p.Status)
	}
}

// The body of a notification is not signed, so a forged one must open nothing.
func TestAForgedNotificationForAnUnpaidPaymentOpensNothing(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := tariffsAPI(t, yk)
	_, c := listedUser(t, h, st)
	order(t, h, c, "level") // still pending at ЮKassa

	rr := anon(h, "POST", "/api/yookassa/webhook", `{"event":"payment.succeeded","object":{"id":"yk-1","status":"succeeded","paid":true}}`)
	if rr.Code != 200 {
		t.Fatalf("webhook = %d", rr.Code)
	}
	if !phaseLocked(t, h, c, "B") {
		t.Fatal("a notification that claimed success opened a level ЮKassa never marked as paid")
	}
	if p, _ := st.PaymentByProviderID("yookassa", "yk-1"); p.Status != "pending" {
		t.Fatalf("status = %s, want it to stay pending", p.Status)
	}
}

func TestNotificationsWeHaveNothingToDoWithGetAQuiet200(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, _ := tariffsAPI(t, yk)
	for _, body := range []string{
		`{"event":"payment.succeeded","object":{"id":"not-ours"}}`,
		`{"event":"refund.succeeded","object":{"id":"r-1"}}`,
		`{"event":"payment.waiting_for_capture","object":{"id":"yk-1"}}`,
		`not json`,
		`{}`,
	} {
		if rr := anon(h, "POST", "/api/yookassa/webhook", body); rr.Code != 200 {
			t.Errorf("webhook %q = %d, want 200 so ЮKassa does not retry for a day", body, rr.Code)
		}
	}
}

// When ЮKassa cannot be asked, the notification is answered with an error so
// that ЮKassa delivers it again.
func TestTheNotificationIsRetriedWhenWeCannotAskYooKassa(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := tariffsAPI(t, yk)
	_, c := listedUser(t, h, st)
	order(t, h, c, "level")
	yk.mu.Lock()
	yk.getFails = true
	yk.mu.Unlock()

	rr := anon(h, "POST", "/api/yookassa/webhook", `{"event":"payment.succeeded","object":{"id":"yk-1"}}`)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("webhook with ЮKassa down = %d, want 500 so it is retried", rr.Code)
	}
}

func TestACanceledPaymentOpensNothingAndCanBeRetried(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := tariffsAPI(t, yk)
	_, c := listedUser(t, h, st)
	o := order(t, h, c, "level")
	yk.set("yk-1", "canceled")

	rr := doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	if got := decodeBody[orderDTO](t, rr).Status; got != "canceled" {
		t.Fatalf("status = %s, want canceled", got)
	}
	if !phaseLocked(t, h, c, "B") {
		t.Fatal("a canceled payment opened a level")
	}
	again := order(t, h, c, "level")
	if again.ID == o.ID || yk.count() != 2 {
		t.Fatalf("retry = %+v after %d payments at ЮKassa, want a fresh order", again, yk.count())
	}
}

func TestAPaymentWhoseAmountDiffersFromTheOrderIsNeverGranted(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := tariffsAPI(t, yk)
	_, c := listedUser(t, h, st)
	o := order(t, h, c, "level")
	yk.mu.Lock()
	yk.payments["yk-1"].Amount = "1.00"
	yk.payments["yk-1"].Status = "succeeded"
	yk.mu.Unlock()

	doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	if !phaseLocked(t, h, c, "B") {
		t.Fatal("a payment of 1 ₽ opened a level that costs 249 ₽")
	}
}

func TestAPaymentOfAnotherOrderIsNeverGranted(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := tariffsAPI(t, yk)
	_, c := listedUser(t, h, st)
	o := order(t, h, c, "level")
	yk.mu.Lock()
	yk.payments["yk-1"].Metadata["order_id"] = "9999"
	yk.payments["yk-1"].Status = "succeeded"
	yk.mu.Unlock()

	doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	if !phaseLocked(t, h, c, "B") {
		t.Fatal("a payment tagged with a different order opened this one")
	}
}

func TestWhenYooKassaRefusesTheOrderIsClosedAndNothingIsLeftPending(t *testing.T) {
	yk := newFakeYooKassa(t)
	yk.failNext = 1
	h, st := tariffsAPI(t, yk)
	uid, c := listedUser(t, h, st)

	rr := doCookie(h, c, "POST", "/api/me/orders", `{"tariff":"level"}`)
	if rr.Code != http.StatusBadGateway || !strings.Contains(rr.Body.String(), "payment_failed") {
		t.Fatalf("order = %d %s, want 502 payment_failed", rr.Code, rr.Body)
	}
	pending, _ := st.PendingPayments(uid, fixedNow.Add(-time.Hour))
	if len(pending) != 0 {
		t.Fatalf("pending = %+v, want none: a refused order must not linger", pending)
	}
	// and the learner can simply try again
	if o := order(t, h, c, "level"); o.Status != "pending" {
		t.Fatalf("retry = %+v", o)
	}
}

func TestAnUnknownTariffIsRefused(t *testing.T) {
	h, st := tariffsAPI(t, newFakeYooKassa(t))
	_, c := listedUser(t, h, st)
	for _, body := range []string{`{"tariff":"month"}`, `{"tariff":""}`, `{}`} {
		if rr := doCookie(h, c, "POST", "/api/me/orders", body); rr.Code != http.StatusNotFound {
			t.Errorf("order %s = %d, want 404", body, rr.Code)
		}
	}
}

func TestNoLevelCanBeBoughtOnceEveryLevelIsOpen(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := tariffsAPI(t, yk)
	uid, c := listedUser(t, h, st)
	for _, ref := range []string{"B", "C"} {
		if err := st.User(uid).GrantEntitlement("phase_unlock", ref, 1, fixedNow); err != nil {
			t.Fatal(err)
		}
	}
	rr := doCookie(h, c, "POST", "/api/me/orders", `{"tariff":"level"}`)
	if rr.Code != http.StatusConflict || !strings.Contains(rr.Body.String(), "nothing_to_buy") {
		t.Fatalf("order = %d %s, want 409 nothing_to_buy", rr.Code, rr.Body)
	}
	if yk.count() != 0 {
		t.Error("ЮKassa was asked to charge for a level that does not exist")
	}
	if got := getTariffsOf(t, h, c); got.Level.Next != nil {
		t.Errorf("next = %+v, want none", got.Level.Next)
	}
}

func TestAnOrderIsPrivateToItsBuyer(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := tariffsAPI(t, yk)
	_, c := listedUser(t, h, st)
	o := order(t, h, c, "level")

	// a second listed account is not possible (one address), so use the store
	// to put a payment under somebody else and ask for it as the listed user
	other, _ := st.CreateUser("Other")
	pid, err := st.CreatePayment(store.Payment{UserID: other, TariffCode: "level", GrantedPhase: "B", AmountKopecks: 24900, Provider: "yookassa"}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if rr := doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", pid), ""); rr.Code != http.StatusNotFound {
		t.Errorf("someone else's order = %d, want 404", rr.Code)
	}
	if rr := doCookie(h, c, "GET", "/api/me/orders/abc", ""); rr.Code != http.StatusNotFound {
		t.Errorf("order abc = %d, want 404", rr.Code)
	}
	if rr := doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), ""); rr.Code != 200 {
		t.Errorf("own order = %d", rr.Code)
	}
}

// ---- the year ----

func TestAYearOpensEveryLevelUntilItEndsAndLevelsAreNotSoldDuringIt(t *testing.T) {
	yk := newFakeYooKassa(t)
	h, st := tariffsAPI(t, yk)
	_, c := listedUser(t, h, st)

	o := order(t, h, c, "year")
	if yk.get("yk-1").Amount != "1000.00" {
		t.Fatalf("year amount = %s, want 1000.00", yk.get("yk-1").Amount)
	}
	yk.set("yk-1", "succeeded")
	doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")

	for _, id := range []string{"B", "C"} {
		if phaseLocked(t, h, c, id) {
			t.Errorf("level %s is locked during a running year", id)
		}
	}
	got := getTariffsOf(t, h, c)
	until, err := time.Parse(time.RFC3339, got.Year.Until)
	if err != nil || !until.Equal(fixedNow.AddDate(0, 0, 365)) {
		t.Errorf("year.until = %q (%v), want a year from now", got.Year.Until, err)
	}
	if got.Level.Next != nil {
		t.Errorf("next level = %+v during a year, want none", got.Level.Next)
	}
	if rr := doCookie(h, c, "POST", "/api/me/orders", `{"tariff":"level"}`); rr.Code != http.StatusConflict {
		t.Errorf("level order during a year = %d, want 409", rr.Code)
	}
	// extending is allowed, and starts where the running year ends
	order(t, h, c, "year")
	yk.set("yk-2", "succeeded")
	if rr := doCookie(h, c, "GET", "/api/me/tariffs", ""); rr.Code != 200 {
		t.Fatal(rr.Body)
	}
	got = getTariffsOf(t, h, c)
	until, _ = time.Parse(time.RFC3339, got.Year.Until)
	if want := fixedNow.AddDate(0, 0, 365).AddDate(0, 0, 365); !until.Equal(want) {
		t.Errorf("after extending, until = %v, want %v", until, want)
	}
}
