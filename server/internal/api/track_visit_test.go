package api

import (
	"net/http"
	"testing"

	"github.com/grisha/serbian-app/server/internal/ratelimit"
)

func TestTrackVisitRecordsAttribution(t *testing.T) {
	h, st := newTestAPI(t)

	rr := anon(h, "POST", "/api/track-visit",
		`{"utm_source":"vk","utm_medium":"social","utm_campaign":"launch"}`)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("track-visit = %d %s, want 204", rr.Code, rr.Body)
	}

	rows, err := st.DebugQuery(`SELECT utm_source FROM link_visits`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 1 || rows[0] != "vk" {
		t.Fatalf("link_visits utm_source rows = %v, want [vk]", rows)
	}
}

func TestTrackVisitIgnoresEmptyBody(t *testing.T) {
	h, st := newTestAPI(t)

	rr := anon(h, "POST", "/api/track-visit", `{}`)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("track-visit = %d %s, want 204", rr.Code, rr.Body)
	}
	rows, err := st.DebugQuery(`SELECT utm_source FROM link_visits`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("link_visits has %d rows, want 0", len(rows))
	}
}

func TestTrackVisitThrottled(t *testing.T) {
	h, _ := newTestAPIWith(t, func(d *Deps) {
		d.Visits = ratelimit.NewLimiter(1, 0) // burst 0: never admits
	})

	rr := anon(h, "POST", "/api/track-visit", `{"utm_source":"vk","utm_medium":"m","utm_campaign":"c"}`)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("throttled track-visit = %d %s, want 204 (silent drop)", rr.Code, rr.Body)
	}
}
