package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestADetailAndTheCheckoutStepsAreReported(t *testing.T) {
	h, st := tariffsAPI(t, nil)
	_, c := listedUser(t, h, st)
	for _, body := range []string{
		`{"kind":"view","source":"unlock_modal","detail":"phase:2"}`,
		`{"kind":"earn","source":"unlock_modal","detail":"phase:2"}`,
		`{"kind":"open","source":"checkout","detail":"from:profile"}`,
		`{"kind":"press","source":"checkout","detail":"year"}`,
		`{"kind":"receipt_view","source":"checkout","detail":"year"}`,
	} {
		if rr := doCookie(h, c, "POST", "/api/me/tariff-events", body); rr.Code != http.StatusNoContent {
			t.Fatalf("%s = %d %s, want 204", body, rr.Code, rr.Body)
		}
	}
	got, _ := st.DebugQuery("SELECT detail FROM tariff_events WHERE kind = 'press'")
	if len(got) != 1 || got[0] != "year" {
		t.Fatalf("press detail = %v, want year", got)
	}
	// a step of paying is not a showing
	if rr := doCookie(h, c, "POST", "/api/me/tariff-events", `{"kind":"view","source":"checkout"}`); rr.Code != http.StatusBadRequest {
		t.Errorf("view/checkout = %d, want 400", rr.Code)
	}
}

// An order that is turned away makes no payment row, so the refusal is written down as an event.
func TestARefusedOrderIsWrittenDownWithItsReason(t *testing.T) {
	tb := newFakeTBank(t)
	h, st, advance, _ := cooldownAPI(t, tb, 3)
	_, c := listedUser(t, h, st)
	o := order(t, h, c, "level")
	tb.set("9001", "CONFIRMED")
	doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	if rr := chooseLevel(h, c, o.ID, "B"); rr.Code != 200 {
		t.Fatalf("choose B = %d %s", rr.Code, rr.Body)
	}
	advance(time.Minute)
	if rr := doCookie(h, c, "POST", "/api/me/orders", `{"tariff":"level"}`); rr.Code != http.StatusConflict {
		t.Fatalf("second order = %d, want 409", rr.Code)
	}
	got, _ := st.DebugQuery("SELECT detail FROM tariff_events WHERE kind = 'error' AND source = 'checkout'")
	if len(got) != 1 || got[0] != "level_cooldown:level" {
		t.Fatalf("refusal events = %v, want level_cooldown:level", got)
	}
}
