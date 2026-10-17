package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/grisha/serbian-app/server/internal/store"
)

// With TARIFFS_FOR_ALL on, the tariffs are open to every account with a verified
// email, paying through Т-Касса.

func openAPI(t *testing.T, yk *fakeYooKassa, tb *fakeTBank) (http.Handler, *store.Store) {
	t.Helper()
	return tariffsAPIWith(t, yk, tb, func(d *Deps) { d.Config.TariffsForAll = true })
}

func TestWhenOpenToAllAnOrdinaryVerifiedAccountSeesTheTariffsAndPaysThroughTBank(t *testing.T) {
	yk, tb := newFakeYooKassa(t), newFakeTBank(t)
	h, st := openAPI(t, yk, tb)
	uid := registerAndVerify(t, h, st, "someone@example.com", "secret1234", "Anna")
	c := authed(t, st, uid)

	if !decodeBody[sessionUserDTO](t, doCookie(h, c, "GET", "/api/auth/session", "")).Features.Tariffs {
		t.Fatal("features.tariffs is off for an ordinary verified account")
	}
	if got := getTariffsOf(t, h, c); !got.PaymentsReady || got.Level.Price != 199 {
		t.Fatalf("tariffs = %+v, want payments ready and the first-purchase price", got)
	}
	o := order(t, h, c, "level")
	if o.ConfirmationURL != "https://pay.test/tb/9001" || tb.count() != 1 || yk.count() != 0 {
		t.Fatalf("order = %+v (Т-Касса %d, ЮKassa %d payments), want it paid through Т-Касса", o, tb.count(), yk.count())
	}
	if p, _ := st.PaymentByID(o.ID); p.Provider != "tbank" || p.ReceiptContact != "someone@example.com" {
		t.Errorf("payment row = %+v", p)
	}
}

func TestWhenOpenToAllTheListedAddressesKeepTheirService(t *testing.T) {
	yk, tb := newFakeYooKassa(t), newFakeTBank(t)
	h, st := openAPI(t, yk, tb)
	_, gc := listedUser(t, h, st) // listed for ЮKassa
	if o := order(t, h, gc, "level"); !strings.HasPrefix(o.ConfirmationURL, "https://pay.test/confirm/") || yk.count() != 1 || tb.count() != 0 {
		t.Fatalf("the ЮKassa-listed account's order = %+v (ЮKassa %d, Т-Касса %d)", o, yk.count(), tb.count())
	}
}

// Open to all does not mean open to anyone who types an address: the receipt goes
// to a verified email.
func TestWhenOpenToAllAnAccountWithoutAVerifiedEmailStillSeesNothing(t *testing.T) {
	h, st := openAPI(t, newFakeYooKassa(t), newFakeTBank(t))

	if rr := anon(h, "POST", "/api/auth/register", registerBody("unverified@example.com", "secret1234", "U")); rr.Code != 200 {
		t.Fatalf("register = %d", rr.Code)
	}
	id, err := st.IdentityByProviderUID("password", "unverified@example.com")
	if err != nil {
		t.Fatal(err)
	}
	tgOnly, err := st.CreateUser("Tg")
	if err != nil {
		t.Fatal(err)
	}
	for name, uid := range map[string]string{"unverified email": id.UserID, "telegram only": tgOnly} {
		c := authed(t, st, uid)
		if rr := doCookie(h, c, "GET", "/api/me/tariffs", ""); rr.Code != http.StatusNotFound {
			t.Errorf("%s: GET tariffs = %d, want 404", name, rr.Code)
		}
		if rr := doCookie(h, c, "POST", "/api/me/orders", `{"tariff":"year"}`); rr.Code != http.StatusNotFound {
			t.Errorf("%s: POST order = %d, want 404", name, rr.Code)
		}
	}
}

func TestWhenOpenToAllButTBankIsNotConnectedNothingCanBeBought(t *testing.T) {
	h, st := openAPI(t, newFakeYooKassa(t), nil) // ЮKassa only: Т-Касса has no terminal
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
