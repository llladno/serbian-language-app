package api

import (
	"strings"
	"testing"
)

func TestLookupEndpoint(t *testing.T) {
	h, _ := newTestAPI(t)

	tests := []struct {
		name        string
		q           string
		wantIDs     []string // matches, in order
		wantPartial bool
	}{
		{name: "exact latin", q: "zdravo", wantIDs: []string{"zdravo"}},
		{name: "case and punctuation stripped", q: "Zdravo!", wantIDs: []string{"zdravo"}},
		{name: "exact cyrillic", q: "здраво", wantIDs: []string{"zdravo"}},
		{name: "word inside a phrase entry", q: "dan", wantIDs: []string{"dobar-dan"}},
		{name: "inflected form matches lemma by prefix", q: "radiš", wantIDs: []string{"raditi"}, wantPartial: true},
		{name: "nothing found", q: "квмокчь", wantIDs: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := do(h, "GET", "/api/lookup?q="+tt.q, "")
			if rr.Code != 200 {
				t.Fatalf("%d %s", rr.Code, rr.Body)
			}
			got := decodeBody[lookupResultDTO](t, rr)
			var ids []string
			for _, m := range got.Matches {
				ids = append(ids, m.ID)
			}
			if strings.Join(ids, ",") != strings.Join(tt.wantIDs, ",") {
				t.Errorf("matches = %v, want %v", ids, tt.wantIDs)
			}
			if got.Partial != tt.wantPartial {
				t.Errorf("partial = %v, want %v", got.Partial, tt.wantPartial)
			}
		})
	}
}

func TestLookupEmptyQuery(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "GET", "/api/lookup?q=", "")
	if rr.Code != 200 {
		t.Fatalf("%d", rr.Code)
	}
	got := decodeBody[lookupResultDTO](t, rr)
	if len(got.Matches) != 0 {
		t.Errorf("matches = %+v", got.Matches)
	}
}
