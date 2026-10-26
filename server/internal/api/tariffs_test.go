package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/content"
	"github.com/grisha/serbian-app/server/internal/store"
)

// listedEmail is in TARIFFS_EMAILS (tbankEmail, in the other test file, is in TBANK_EMAILS): both lists open the
// tariffs, and the bank is the only service.
const listedEmail = "gregorymansurov@gmail.com"

// tariffsAPI is the app with three levels (A free, B and C for sale), payments
// pointed at the fake bank, and listedEmail and tbankEmail (one in each of the two
// lists, which have the same effect) allowed to use the tariffs. A nil bank leaves
// the terminal without credentials.
func tariffsAPI(t *testing.T, tb *fakeTBank) (http.Handler, *store.Store) {
	t.Helper()
	return tariffsAPIWith(t, tb)
}

// tariffsAPIWith is tariffsAPI with further changes to the dependencies.
func tariffsAPIWith(t *testing.T, tb *fakeTBank, opts ...func(*Deps)) (http.Handler, *store.Store) {
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
		if tb != nil {
			d.Config.TBankTerminalKey = tb.terminal
			d.Config.TBankPassword = tb.password
			d.Config.TBankAPIBase = tb.srv.URL
			d.Config.TBankTaxation = "usn_income"
		}
		for _, o := range opts {
			o(d)
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
		// AvailableFrom: set while the monthly limit holds the next level back.
		AvailableFrom string `json:"available_from"`
		Bonus         int64  `json:"bonus"`
		Next          *struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"next"`
		// Choices are the levels a paid level can open; Credits the paid orders not yet spent.
		Choices []struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Lessons int    `json:"lessons"`
			After   string `json:"after"`
		} `json:"choices"`
		Credits []int64 `json:"credits"`
	} `json:"level"`
	Year struct {
		Price    int64  `json:"price"`
		OldPrice int64  `json:"old_price"`
		Bonus    int64  `json:"bonus"`
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
	tb := newFakeTBank(t)
	h, st := tariffsAPI(t, tb)

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
	h, st := tariffsAPI(t, newFakeTBank(t))
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
	tb := newFakeTBank(t)
	h, st := newTestAPIWith(t, func(d *Deps) {
		d.Async = func(f func()) { f() }
		d.Config.TariffsEmails = []string{"gregorymansurov@gmail.com"}
		d.Config.TBankTerminalKey, d.Config.TBankPassword, d.Config.TBankAPIBase = tb.terminal, tb.password, tb.srv.URL
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
	h, st := tariffsAPI(t, newFakeTBank(t))
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
	h, st := tariffsAPI(t, newFakeTBank(t))
	_, c := listedUser(t, h, st)
	got := getTariffsOf(t, h, c)

	if !got.PaymentsReady || got.HasPaidLevel {
		t.Errorf("flags = %+v", got)
	}
	if got.Level.Price != 199 || got.Level.RegularPrice != 299 {
		t.Errorf("level price = %d (regular %d), want 199 for the first purchase and 299 after", got.Level.Price, got.Level.RegularPrice)
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
	if hook := anon(h, "POST", "/api/tbank/webhook", `{"Status":"CONFIRMED"}`); hook.Code != 200 || hook.Body.String() != "OK" {
		t.Errorf("webhook without a terminal = %d %q, want a quiet 200 OK", hook.Code, hook.Body)
	}
}

// ---- ordering and paying ----

func TestReturningFromTheBankGivesACreditThatOpensTheChosenLevelAndTheNextOneCostsFullPrice(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPI(t, tb)
	_, c := listedUser(t, h, st)

	o := order(t, h, c, "level")
	// back on the site before paying: still pending
	rr := doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	if got := decodeBody[orderDTO](t, rr).Status; got != "pending" {
		t.Fatalf("status before paying = %s", got)
	}
	tb.set("9001", "CONFIRMED")
	rr = doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	if got := decodeBody[orderDTO](t, rr).Status; got != "succeeded" {
		t.Fatalf("status after paying = %s", got)
	}
	for _, id := range []string{"B", "C"} {
		if !phaseLocked(t, h, c, id) {
			t.Errorf("level %s opened by the payment itself: the learner chooses the level", id)
		}
	}
	if rr := chooseLevel(h, c, o.ID, "B"); rr.Code != 200 {
		t.Fatalf("choose B = %d %s", rr.Code, rr.Body)
	}
	if phaseLocked(t, h, c, "B") {
		t.Error("level B is still locked after it was chosen")
	}
	if !phaseLocked(t, h, c, "C") {
		t.Error("level C opened too: a level payment opens one level")
	}
	got := getTariffsOf(t, h, c)
	if !got.HasPaidLevel || got.Level.Price != 299 || got.Level.Next == nil || got.Level.Next.ID != "C" {
		t.Errorf("after the first level: %+v, want 299 and C next", got)
	}
	// the second level order is at the regular price, and again names no level
	o2 := order(t, h, c, "level")
	if tb.get("9002").Amount != 29900 {
		t.Errorf("second order amount = %d, want 29900 kopecks (299 ₽)", tb.get("9002").Amount)
	}
	if p, _ := st.PaymentByID(o2.ID); p.GrantedPhase != "" {
		t.Errorf("second order already opens %q, want it to wait for a choice", p.GrantedPhase)
	}
}

// Opening the tariffs screen finds a payment the notification has not reported.
func TestOpeningTheScreenSettlesAPaymentThatWasMadeInTheMeantime(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPI(t, tb)
	_, c := listedUser(t, h, st)
	order(t, h, c, "level")
	tb.set("9001", "CONFIRMED")

	got := getTariffsOf(t, h, c)
	if len(got.Level.Credits) != 1 || !got.HasPaidLevel {
		t.Fatalf("level = %+v, want the payment settled into a credit waiting to be spent", got)
	}
}

func TestAnUnknownTariffIsRefused(t *testing.T) {
	h, st := tariffsAPI(t, newFakeTBank(t))
	_, c := listedUser(t, h, st)
	for _, body := range []string{`{"tariff":"month"}`, `{"tariff":""}`, `{}`} {
		if rr := doCookie(h, c, "POST", "/api/me/orders", body); rr.Code != http.StatusNotFound {
			t.Errorf("order %s = %d, want 404", body, rr.Code)
		}
	}
}

func TestNoLevelCanBeBoughtOnceEveryLevelIsOpen(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPI(t, tb)
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
	if tb.count() != 0 {
		t.Error("the bank was asked to charge for a level that does not exist")
	}
	if got := getTariffsOf(t, h, c); got.Level.Next != nil {
		t.Errorf("next = %+v, want none", got.Level.Next)
	}
}

func TestAnOrderIsPrivateToItsBuyer(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPI(t, tb)
	_, c := listedUser(t, h, st)
	o := order(t, h, c, "level")

	// a second listed account is not possible (one address), so use the store
	// to put a payment under somebody else and ask for it as the listed user
	other, _ := st.CreateUser("Other")
	pid, err := st.CreatePayment(store.Payment{UserID: other, TariffCode: "level", GrantedPhase: "B", AmountKopecks: 24900, Provider: "tbank"}, fixedNow)
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
	tb := newFakeTBank(t)
	h, st := tariffsAPI(t, tb)
	_, c := listedUser(t, h, st)

	o := order(t, h, c, "year")
	if tb.get("9001").Amount != 100000 {
		t.Fatalf("year amount = %d, want 100000 kopecks (1 000 ₽)", tb.get("9001").Amount)
	}
	tb.set("9001", "CONFIRMED")
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
	tb.set("9002", "CONFIRMED")
	if rr := doCookie(h, c, "GET", "/api/me/tariffs", ""); rr.Code != 200 {
		t.Fatal(rr.Body)
	}
	got = getTariffsOf(t, h, c)
	until, _ = time.Parse(time.RFC3339, got.Year.Until)
	if want := fixedNow.AddDate(0, 0, 365).AddDate(0, 0, 365); !until.Equal(want) {
		t.Errorf("after extending, until = %v, want %v", until, want)
	}
}
