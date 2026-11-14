package api

import (
	"net/http"
	"testing"
)

func TestClientInfoEndpointRequiresASession(t *testing.T) {
	h, _, _ := newTelegramBotAPI(t)
	if rr := anon(h, "POST", "/api/me/client", `{"version":"10.1","platform":"ios"}`); rr.Code != http.StatusUnauthorized {
		t.Fatalf("POST without a session = %d, want 401", rr.Code)
	}
}

func TestClientInfoEndpointStoresTheReportedApp(t *testing.T) {
	h, st, _ := newTelegramBotAPI(t)
	uid := registerAndVerify(t, h, st, "client@example.com", "secret1234", "Client")
	cookie := authed(t, st, uid)

	rr := doCookie(h, cookie, "POST", "/api/me/client", `{"version":"10.1","platform":"ios"}`)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("POST = %d %s, want 204", rr.Code, rr.Body)
	}
	got, err := st.ClientInfoOf(uid)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "10.1" || got.Platform != "ios" {
		t.Fatalf("stored = %+v, want 10.1 / ios", got)
	}

	// Junk is answered like success and changes nothing: the call is a courtesy and must never block the person.
	rr = doCookie(h, cookie, "POST", "/api/me/client", `{"version":"<script>","platform":"ios"}`)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("junk POST = %d, want 204", rr.Code)
	}
	if again, _ := st.ClientInfoOf(uid); again.Version != "10.1" {
		t.Fatalf("junk overwrote the version: %+v", again)
	}
}
