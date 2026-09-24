package api

import (
	"reflect"
	"testing"

	"github.com/grisha/serbian-app/server/internal/content"
)

func TestNextNewGrammarCardIDs(t *testing.T) {
	cards := []content.GrammarCard{
		{ID: "late", Lesson: "05"},
		{ID: "early", Lesson: "01"},
		{ID: "mid", Lesson: "03"},
	}

	got := nextNewGrammarCardIDs(cards, map[string]bool{}, 2)
	want := []string{"gram:early", "gram:mid"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v (lesson order, not file order)", got, want)
	}

	// A passed point is skipped, but the daily limit still caps how many
	// unpassed ones come back.
	passed := map[string]bool{"gram:early": true}
	got = nextNewGrammarCardIDs(cards, passed, 1)
	want = []string{"gram:mid"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestReviewQueueIncludesGrammarCard checks that a grammar point from
// grammar.yaml rides along in /api/review/queue next to vocab/ff cards, and
// that — unlike a brand-new vocab card — it never carries a multiple-choice
// quiz (there's no fair way to build wrong-answer options for a whole
// conjugation paradigm).
func TestReviewQueueIncludesGrammarCard(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "GET", "/api/review/queue", "")
	if rr.Code != 200 {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
	q := decodeBody[[]reviewCardDTO](t, rr)

	var gram *reviewCardDTO
	for i := range q {
		if q[i].Kind == "gram" {
			gram = &q[i]
			break
		}
	}
	if gram == nil {
		t.Fatalf("no gram card in queue: %+v", q)
	}
	if gram.CardID != "gram:prezent-am" {
		t.Errorf("card_id = %q", gram.CardID)
	}
	if gram.Front != "презент, тип -am: raditi" || gram.Back != "radim · radiš · radi · radimo · radite · rade" {
		t.Errorf("front/back = %q / %q", gram.Front, gram.Back)
	}
	if len(gram.Options) != 0 {
		t.Errorf("gram card should have no quiz options, got %v", gram.Options)
	}
}

// TestReviewGradeGrammarCard checks a grammar card grades exactly like a
// vocab/ff card — GradeCard doesn't special-case kind, but this pins that
// the "gram:" prefix round-trips through EnsureCards/DueQueue/GradeCard.
func TestReviewGradeGrammarCard(t *testing.T) {
	h, _ := newTestAPI(t)
	do(h, "GET", "/api/review/queue", "") // seed cards

	rr := do(h, "POST", "/api/review/grade", `{"card_id":"gram:prezent-am","grade":2}`)
	if rr.Code != 200 {
		t.Fatalf("grade: %d %s", rr.Code, rr.Body)
	}
	g := decodeBody[gradeResultDTO](t, rr)
	if g.State != "review" || g.IntervalDays != 1 {
		t.Errorf("grade result = %+v", g)
	}
}
