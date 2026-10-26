package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/grisha/serbian-app/server/internal/store"
)

// With TARIFFS_FOR_ALL on, the tariffs are open to every account with a verified
// email, paying through Т-Касса.

func openAPI(t *testing.T, tb *fakeTBank) (http.Handler, *store.Store) {
	t.Helper()
	return tariffsAPIWith(t, tb, func(d *Deps) { d.Config.TariffsForAll = true })
}

func TestWhenOpenToAllAnOrdinaryVerifiedAccountSeesTheTariffsAndPaysThroughTBank(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := openAPI(t, tb)
	uid := registerAndVerify(t, h, st, "someone@example.com", "secret1234", "Anna")
	c := authed(t, st, uid)

	if !decodeBody[sessionUserDTO](t, doCookie(h, c, "GET", "/api/auth/session", "")).Features.Tariffs {
		t.Fatal("features.tariffs is off for an ordinary verified account")
	}
	if got := getTariffsOf(t, h, c); !got.PaymentsReady || got.Level.Price != 199 {
		t.Fatalf("tariffs = %+v, want payments ready and the first-purchase price", got)
	}
	o := order(t, h, c, "level")
	if o.ConfirmationURL != "https://pay.test/tb/9001" || tb.count() != 1 {
		t.Fatalf("order = %+v (%d bank payments), want it paid through Т-Касса", o, tb.count())
	}
	if p, _ := st.PaymentByID(o.ID); p.Provider != "tbank" || p.ReceiptContact != "someone@example.com" {
		t.Errorf("payment row = %+v", p)
	}
}

// Open to all does not mean open to anyone who types an address: an account
// whose email was typed at sign-up but never verified still sees nothing.
func TestWhenOpenToAllAnAccountWithAnUnverifiedEmailStillSeesNothing(t *testing.T) {
	h, st := openAPI(t, newFakeTBank(t))

	if rr := anon(h, "POST", "/api/auth/register", registerBody("unverified@example.com", "secret1234", "U")); rr.Code != 200 {
		t.Fatalf("register = %d", rr.Code)
	}
	id, err := st.IdentityByProviderUID("password", "unverified@example.com")
	if err != nil {
		t.Fatal(err)
	}
	c := authed(t, st, id.UserID)
	if rr := doCookie(h, c, "GET", "/api/me/tariffs", ""); rr.Code != http.StatusNotFound {
		t.Errorf("GET tariffs = %d, want 404", rr.Code)
	}
	if rr := doCookie(h, c, "POST", "/api/me/orders", `{"tariff":"year"}`); rr.Code != http.StatusNotFound {
		t.Errorf("POST order = %d, want 404", rr.Code)
	}
}

// A Telegram-only account (the Mini App's usual visitor) has no email at all.
// Once the tariffs are open to all it sees them and pays through Т-Касса, naming
// the email or phone its receipt goes to.
func TestWhenOpenToAllATelegramOnlyAccountPaysAndNamesItsReceiptContact(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := openAPI(t, tb)
	tgOnly, err := st.CreateUser("Tg")
	if err != nil {
		t.Fatal(err)
	}
	c := authed(t, st, tgOnly)

	if !decodeBody[sessionUserDTO](t, doCookie(h, c, "GET", "/api/auth/session", "")).Features.Tariffs {
		t.Fatal("features.tariffs is off for a Telegram-only account although the tariffs are open to all")
	}
	if got := getTariffsOf(t, h, c); !got.PaymentsReady {
		t.Fatalf("tariffs = %+v, want payments ready", got)
	}

	// Nothing, or something that is neither an email nor a phone: nothing is
	// created, nothing charged.
	for _, body := range []string{
		`{"tariff":"year"}`, `{"tariff":"year","contact":"  "}`, `{"tariff":"year","contact":"not a contact"}`,
		`{"tariff":"year","contact":"Anna <a@example.com>"}`, `{"tariff":"year","contact":"12345"}`,
		`{"tariff":"year","contact":"+0123456789"}`, `{"tariff":"year","contact":"903 123-45-6x"}`,
	} {
		rr := doCookie(h, c, "POST", "/api/me/orders", body)
		if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "receipt_contact") {
			t.Errorf("order %s = %d %s, want 400 receipt_contact_invalid", body, rr.Code, rr.Body)
		}
	}
	if tb.count() != 0 {
		t.Fatalf("%d payments were started without a receipt contact", tb.count())
	}

	// An email goes to the bank as the receipt's Email, lower-cased.
	rr := doCookie(h, c, "POST", "/api/me/orders", `{"tariff":"year","contact":"  Buyer@Example.com "}`)
	if rr.Code != 200 {
		t.Fatalf("order = %d %s", rr.Code, rr.Body)
	}
	o := decodeBody[orderDTO](t, rr)
	if o.ConfirmationURL != "https://pay.test/tb/9001" || tb.count() != 1 {
		t.Fatalf("order = %+v (%d bank payments), want it paid through Т-Касса", o, tb.count())
	}
	if p, _ := st.PaymentByID(o.ID); p.Provider != "tbank" || p.ReceiptContact != "buyer@example.com" {
		t.Errorf("payment row = %+v, want the receipt email kept, lower-cased", p)
	}
	if r, _ := tb.get("9001").Request["Receipt"].(map[string]any); r["Email"] != "buyer@example.com" || r["Phone"] != nil {
		t.Errorf("the receipt sent to the bank = %v, want the email and no phone", r)
	}

	// A phone goes as the receipt's Phone in the bank's "+7…" form, with no Email.
	rr = doCookie(h, c, "POST", "/api/me/orders", `{"tariff":"year","contact":"8 (903) 123-45-67"}`)
	if rr.Code != 200 {
		t.Fatalf("order with a phone = %d %s", rr.Code, rr.Body)
	}
	o = decodeBody[orderDTO](t, rr)
	if p, _ := st.PaymentByID(o.ID); p.ReceiptContact != "+79031234567" {
		t.Errorf("payment row = %+v, want the normalized phone kept as the receipt contact", p)
	}
	if r, _ := tb.get("9002").Request["Receipt"].(map[string]any); r["Phone"] != "+79031234567" || r["Email"] != nil {
		t.Errorf("the receipt sent to the bank = %v, want the phone and no email", r)
	}
}

func TestNormalizePhone(t *testing.T) {
	for in, want := range map[string]string{
		"8 (903) 123-45-67":  "+79031234567",
		"89031234567":        "+79031234567",
		"+7 903 123 45 67":   "+79031234567",
		"79031234567":        "+79031234567",
		"9031234567":         "+79031234567",
		"903.123.45.67":      "+79031234567",
		"+381 64 123 4567":   "+381641234567", // abroad: any international number
		" +995 555 12 34 56": "+995555123456",
	} {
		if got, ok := normalizePhone(in); !ok || got != want {
			t.Errorf("normalizePhone(%q) = %q, %v, want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "+", "12345", "1234567890", "+0123456789", "+1234567", "+1234567890123456", "abc", "903-123-45-6x", "7(903)123", "8903123456789"} {
		if got, ok := normalizePhone(in); ok {
			t.Errorf("normalizePhone(%q) = %q, want it refused", in, got)
		}
	}
}

// Closed to all, a Telegram-only account sees nothing, as before.
func TestATelegramOnlyAccountSeesNothingWhileTheTariffsAreNotOpenToAll(t *testing.T) {
	h, st := tariffsAPIWith(t, newFakeTBank(t))
	tgOnly, err := st.CreateUser("Tg")
	if err != nil {
		t.Fatal(err)
	}
	c := authed(t, st, tgOnly)
	if rr := doCookie(h, c, "GET", "/api/me/tariffs", ""); rr.Code != http.StatusNotFound {
		t.Errorf("GET tariffs = %d, want 404", rr.Code)
	}
	if rr := doCookie(h, c, "POST", "/api/me/orders", `{"tariff":"year","contact":"a@example.com"}`); rr.Code != http.StatusNotFound {
		t.Errorf("POST order = %d, want 404", rr.Code)
	}
}

func TestWhenOpenToAllButTBankIsNotConnectedNothingCanBeBought(t *testing.T) {
	h, st := openAPI(t, nil) // Т-Касса has no terminal
	uid := registerAndVerify(t, h, st, "someone@example.com", "secret1234", "Anna")
	c := authed(t, st, uid)
	if got := getTariffsOf(t, h, c); got.PaymentsReady {
		t.Error("payments_ready although the service the account pays through is not connected")
	}
	rr := doCookie(h, c, "POST", "/api/me/orders", `{"tariff":"level"}`)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("order = %d, want 503 payments_unavailable", rr.Code)
	}
}
