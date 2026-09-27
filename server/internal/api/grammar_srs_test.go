package api

import (
	"fmt"
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

// fixtureGramItemAnswer maps the two testdata/content/grammar.yaml items (by
// their prompt) to their accepted answer, so tests can answer whichever one
// reviewQueue happens to have picked without hard-coding the random index.
var fixtureGramItemAnswer = map[string]string{
	"ja (я)":  "radim",
	"ti (ты)": "radiš",
}

// TestReviewQueueIncludesGrammarCard checks that a grammar point from
// grammar.yaml rides along in /api/review/queue next to vocab/ff cards, that
// it carries one randomly-picked item's prompt/index rather than a full
// paradigm, and — unlike a brand-new vocab card — never a multiple-choice
// quiz (there's no fair way to build wrong-answer options for a single
// conjugated form either).
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
	if gram.Front != "презент, тип -am: raditi" {
		t.Errorf("front = %q", gram.Front)
	}
	if gram.Back != "" {
		t.Errorf("back = %q, want empty — the accepted answer must never reach the client unchecked", gram.Back)
	}
	if _, ok := fixtureGramItemAnswer[gram.ItemPrompt]; !ok {
		t.Errorf("item_prompt = %q, want one of the fixture's items", gram.ItemPrompt)
	}
	if gram.ItemIndex < 0 || gram.ItemIndex > 1 {
		t.Errorf("item_index = %d, want 0 or 1", gram.ItemIndex)
	}
	if len(gram.Options) != 0 {
		t.Errorf("gram card should have no quiz options, got %v", gram.Options)
	}
}

// TestReviewGradeRejectsGramCard checks the old self-report endpoint refuses
// a grammar card — grading one now has to go through an actual correctness
// check (grade-gram), not a bare client-supplied grade.
func TestReviewGradeRejectsGramCard(t *testing.T) {
	h, _ := newTestAPI(t)
	do(h, "GET", "/api/review/queue", "") // seed cards

	rr := do(h, "POST", "/api/review/grade", `{"card_id":"gram:prezent-am","grade":2}`)
	if rr.Code != 400 {
		t.Fatalf("code = %d, want 400: %s", rr.Code, rr.Body)
	}
}

// TestReviewGradeGramCorrect answers whichever item reviewQueue picked
// correctly and checks it grades Good (state -> review, 1-day interval —
// same as a self-reported Good would).
func TestReviewGradeGramCorrect(t *testing.T) {
	h, _ := newTestAPI(t)
	q := decodeBody[[]reviewCardDTO](t, do(h, "GET", "/api/review/queue", ""))
	var gram reviewCardDTO
	for _, c := range q {
		if c.Kind == "gram" {
			gram = c
		}
	}
	if gram.CardID == "" {
		t.Fatal("no gram card in queue")
	}
	answer := fixtureGramItemAnswer[gram.ItemPrompt]

	rr := do(h, "POST", "/api/review/grade-gram", fmt.Sprintf(`{"card_id":%q,"item_index":%d,"answer":%q}`, gram.CardID, gram.ItemIndex, answer))
	if rr.Code != 200 {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
	res := decodeBody[gramCheckResultDTO](t, rr)
	if !res.OK {
		t.Errorf("ok = false, want true for the correct answer %q", answer)
	}
	if res.State != "review" || res.IntervalDays != 1 {
		t.Errorf("grade result = %+v, want state=review interval=1 (Good)", res)
	}
}

// TestReviewGradeGramWrong answers wrong and checks it grades Again (stays
// in learning) and reveals the expected form.
func TestReviewGradeGramWrong(t *testing.T) {
	h, _ := newTestAPI(t)
	q := decodeBody[[]reviewCardDTO](t, do(h, "GET", "/api/review/queue", ""))
	var gram reviewCardDTO
	for _, c := range q {
		if c.Kind == "gram" {
			gram = c
		}
	}
	if gram.CardID == "" {
		t.Fatal("no gram card in queue")
	}

	rr := do(h, "POST", "/api/review/grade-gram", fmt.Sprintf(`{"card_id":%q,"item_index":%d,"answer":"definitelywrong"}`, gram.CardID, gram.ItemIndex))
	if rr.Code != 200 {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
	res := decodeBody[gramCheckResultDTO](t, rr)
	if res.OK {
		t.Error("ok = true for a wrong answer")
	}
	if res.Expected != fixtureGramItemAnswer[gram.ItemPrompt] {
		t.Errorf("expected = %q, want %q", res.Expected, fixtureGramItemAnswer[gram.ItemPrompt])
	}
	if res.State != "learning" {
		t.Errorf("state = %q, want learning (Again)", res.State)
	}
}

func TestReviewGradeGramRejectsNonGramCard(t *testing.T) {
	h, _ := newTestAPI(t)
	do(h, "GET", "/api/review/queue", "")

	rr := do(h, "POST", "/api/review/grade-gram", `{"card_id":"vocab:zdravo","item_index":0,"answer":"x"}`)
	if rr.Code != 400 {
		t.Fatalf("code = %d, want 400: %s", rr.Code, rr.Body)
	}
}

func TestReviewGradeGramRejectsBadItemIndex(t *testing.T) {
	h, _ := newTestAPI(t)
	do(h, "GET", "/api/review/queue", "")

	rr := do(h, "POST", "/api/review/grade-gram", `{"card_id":"gram:prezent-am","item_index":99,"answer":"x"}`)
	if rr.Code != 404 {
		t.Fatalf("code = %d, want 404: %s", rr.Code, rr.Body)
	}
}
