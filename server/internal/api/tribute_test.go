package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grisha/serbian-app/server/internal/store"
)

const testTributeKey = "test-tribute-key"

func newTributeAPI(t *testing.T) (http.Handler, *store.Store) {
	return newTestAPIWith(t, func(d *Deps) { d.Config.TributeAPIKey = testTributeKey })
}

func signTribute(body string) string {
	mac := hmac.New(sha256.New, []byte(testTributeKey))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

// postTribute posts body to the webhook with the given signature header
// (empty = header omitted). The webhook is exempt from checkOrigin.
func postTribute(h http.Handler, sig, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/api/tribute/webhook", strings.NewReader(body))
	if sig != "" {
		r.Header.Set("trbt-signature", sig)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	return rr
}

// donationBody mirrors a real webhook delivery observed 2026-09-23: event
// name is snake_case ("new_donation", not "newDonation"), and amount is
// already in minor units (10000 = 100 RUB), not major units needing ×100.
const donationBody = `{"name":"new_donation","payload":{"id":"evt_abc","telegram_user_id":"555","telegram_username":"alice","amount":10000,"currency":"RUB"}}`

func TestTributeWebhookWrongSignatureIs401(t *testing.T) {
	h, _ := newTributeAPI(t)
	rr := postTribute(h, "wrong", donationBody)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("webhook wrong signature = %d, want 401", rr.Code)
	}
}

func TestTributeWebhookMissingSignatureIs401(t *testing.T) {
	h, _ := newTributeAPI(t)
	rr := postTribute(h, "", donationBody)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("webhook missing signature = %d, want 401", rr.Code)
	}
}

func TestTributeWebhookStoresDonation(t *testing.T) {
	h, st := newTributeAPI(t)
	rr := postTribute(h, signTribute(donationBody), donationBody)
	if rr.Code != http.StatusOK {
		t.Fatalf("webhook = %d %s, want 200", rr.Code, rr.Body)
	}
	got, err := st.ListDonations()
	if err != nil {
		t.Fatalf("ListDonations: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListDonations = %d rows, want 1", len(got))
	}
	d := got[0]
	if d.TelegramUserID != "555" || d.TelegramUsername != "alice" || d.Currency != "RUB" ||
		d.EventType != "new_donation" || d.TributeEventID != "evt_abc" {
		t.Errorf("stored donation = %+v, want fields parsed from the webhook body", d)
	}
	if d.AmountMinorUnits != 10000 { // amount is already minor units, passed through as-is
		t.Errorf("AmountMinorUnits = %d, want 10000", d.AmountMinorUnits)
	}
}

func TestTributeWebhookLinksKnownDonor(t *testing.T) {
	h, st := newTributeAPI(t)
	uid, err := st.CreateUser("Alice")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if err := st.CreateIdentity(store.Identity{
		ID: "id_1", UserID: uid, Provider: "telegram", ProviderUID: "555", TgUsername: "alice",
	}); err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}

	rr := postTribute(h, signTribute(donationBody), donationBody)
	if rr.Code != http.StatusOK {
		t.Fatalf("webhook = %d %s, want 200", rr.Code, rr.Body)
	}
	got, err := st.ListDonations()
	if err != nil {
		t.Fatalf("ListDonations: %v", err)
	}
	if len(got) != 1 || got[0].UserID != uid {
		t.Fatalf("ListDonations = %+v, want one donation linked to %s", got, uid)
	}
}

func TestTributeWebhookRetryIsIdempotent(t *testing.T) {
	h, st := newTributeAPI(t)
	sig := signTribute(donationBody)
	postTribute(h, sig, donationBody)
	rr := postTribute(h, sig, donationBody) // Tribute retries on a slow/failed response
	if rr.Code != http.StatusOK {
		t.Fatalf("webhook retry = %d, want 200", rr.Code)
	}
	got, err := st.ListDonations()
	if err != nil {
		t.Fatalf("ListDonations: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListDonations = %d rows after a retried delivery, want 1", len(got))
	}
}
