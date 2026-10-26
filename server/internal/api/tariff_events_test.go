package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The places that sell the tariffs report each showing and each press on the way to them
// (POST /api/me/tariff-events); the admin's «Воронка тарифов» counts them.

func TestAListedAccountReportsAShowingAndAPress(t *testing.T) {
	h, st := tariffsAPI(t, nil)
	_, c := listedUser(t, h, st)

	for _, body := range []string{
		`{"kind":"view","source":"profile_banner"}`,
		`{"kind":"click","source":"profile_banner"}`,
		`{"kind":"view","source":"repeat_limit"}`,
		`{"kind":"view","source":"unlock_modal"}`,
	} {
		if rr := doCookie(h, c, "POST", "/api/me/tariff-events", body); rr.Code != http.StatusNoContent {
			t.Fatalf("%s = %d %s, want 204", body, rr.Code, rr.Body)
		}
	}
	got, _ := st.DebugQuery("SELECT COUNT(*) FROM tariff_events")
	if got[0] != "4" {
		t.Fatalf("rows = %s, want 4", got[0])
	}
	who, _ := st.DebugQuery("SELECT DISTINCT user_id FROM tariff_events")
	if len(who) != 1 {
		t.Fatalf("users = %v, want the one who reported", who)
	}
}

func TestTheSameEventTwiceInARowIsOneRow(t *testing.T) {
	h, st := tariffsAPI(t, nil)
	_, c := listedUser(t, h, st)
	for i := 0; i < 3; i++ {
		if rr := doCookie(h, c, "POST", "/api/me/tariff-events", `{"kind":"view","source":"profile_banner"}`); rr.Code != http.StatusNoContent {
			t.Fatalf("report %d = %d", i, rr.Code)
		}
	}
	if got, _ := st.DebugQuery("SELECT COUNT(*) FROM tariff_events"); got[0] != "1" {
		t.Fatalf("rows = %s, want 1 (a reload is not a second showing)", got[0])
	}
}

func TestAnAccountWithoutTariffsReportsNothingAndGetsA404(t *testing.T) {
	h, st := tariffsAPI(t, nil)
	other := registerAndVerify(t, h, st, "someone@example.com", "secret1234", "Other")
	oc := authed(t, st, other)
	rr := doCookie(h, oc, "POST", "/api/me/tariff-events", `{"kind":"view","source":"profile_banner"}`)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("= %d, want 404 for an account the tariffs are closed to", rr.Code)
	}
	if got, _ := st.DebugQuery("SELECT COUNT(*) FROM tariff_events"); got[0] != "0" {
		t.Fatalf("rows = %s, want none", got[0])
	}
}

func TestAnUnknownKindOrSourceOrBodyIsRefused(t *testing.T) {
	h, st := tariffsAPI(t, nil)
	_, c := listedUser(t, h, st)
	for _, body := range []string{
		`{"kind":"hover","source":"profile_banner"}`,
		`{"kind":"view","source":"nowhere"}`,
		`{"kind":"view"}`,
		`{}`,
		`not json`,
	} {
		if rr := doCookie(h, c, "POST", "/api/me/tariff-events", body); rr.Code != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", body, rr.Code)
		}
	}
	if got, _ := st.DebugQuery("SELECT COUNT(*) FROM tariff_events"); got[0] != "0" {
		t.Fatalf("rows = %s after refused reports, want none", got[0])
	}
}

func TestReportingNeedsASession(t *testing.T) {
	h, _ := tariffsAPI(t, nil)
	r := httptest.NewRequest("POST", "/api/me/tariff-events", strings.NewReader(`{"kind":"view","source":"profile_banner"}`))
	setTestOrigin(r)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("= %d, want 401", rr.Code)
	}
}
