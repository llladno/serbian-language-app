package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/store"
)

// repeatBody mirrors the wire shape of GET /api/repeat on purpose (not the production DTO):
// the test pins the contract the client depends on.
type repeatBody struct {
	Total     int  `json:"total"`
	Unlimited bool `json:"unlimited"`
	Session   struct {
		Size      int `json:"size"`
		Remaining int `json:"remaining"`
	} `json:"session"`
	Items []struct {
		Lesson   string `json:"lesson"`
		Exercise struct {
			ID     string `json:"id"`
			Type   string `json:"type"`
			Prompt string `json:"prompt"`
			Audio  string `json:"audio"`
			Text   string `json:"text"`
		} `json:"exercise"`
	} `json:"items"`
}

type limitBody struct {
	Error         string `json:"error"`
	NextAvailable string `json:"next_available"`
}

func repeatIDs(b repeatBody) []string {
	ids := []string{}
	for _, it := range b.Items {
		ids = append(ids, it.Exercise.ID)
	}
	return ids
}

// movableClock builds the API on a clock the test can move, to cross midnight.
func movableAPI(t *testing.T) (http.Handler, *store.Store, *time.Time) {
	t.Helper()
	now := fixedNow
	h, st := newTestAPIWith(t, func(d *Deps) { d.Now = func() time.Time { return now } })
	return h, st, &now
}

func TestRepeatServesMistakesFirstThenTheRest(t *testing.T) {
	h, _, now := movableAPI(t)
	do(h, "POST", "/api/lessons/01/exercises/01-A-1/check", `{"answer":"Zdravo! Kako si?"}`) // right
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"x"}`)                // wrong
	*now = now.Add(48 * time.Hour)                                                           // two days later the right one is filler

	rr := do(h, "GET", "/api/repeat", "")
	if rr.Code != 200 {
		t.Fatalf("GET /api/repeat = %d %s", rr.Code, rr.Body)
	}
	for _, leak := range []string{`"answer"`, `"accept"`, `"pairs"`, `"say"`} {
		if strings.Contains(rr.Body.String(), leak) {
			t.Errorf("repeat leaked %s: %s", leak, rr.Body)
		}
	}
	got := decodeBody[repeatBody](t, rr)
	if ids := repeatIDs(got); len(ids) != 2 || ids[0] != "01-A-2" || ids[1] != "01-A-1" {
		t.Fatalf("batch = %v, want the mistake first and the solved exercise after it", ids)
	}
	if got.Total != 2 || got.Session.Size != 2 || got.Session.Remaining != 2 || got.Unlimited {
		t.Errorf("counts = %+v", got)
	}
	if got.Items[0].Lesson != "01" || got.Items[0].Exercise.Type != "translate" || got.Items[0].Exercise.Prompt != "Привет!" {
		t.Errorf("item = %+v", got.Items[0])
	}
}

func TestRepeatPutsAlmostRightAnswersAfterMistakes(t *testing.T) {
	h, _, _ := movableAPI(t)
	do(h, "POST", "/api/lessons/01/exercises/01-A-1/check", `{"answer":"Zdravp! Kako si?"}`) // one letter off: forgiven
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"x"}`)                // wrong

	got := decodeBody[repeatBody](t, do(h, "GET", "/api/repeat", ""))
	if ids := repeatIDs(got); len(ids) != 2 || ids[0] != "01-A-2" || ids[1] != "01-A-1" {
		t.Fatalf("batch = %v, want the mistake, then the almost-right answer", ids)
	}
}

func TestRepeatTypoReturnsUntilWrittenExactly(t *testing.T) {
	h, _, now := movableAPI(t)
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"Zdravp!"}`) // forgiven, stored as a typo
	*now = now.Add(24 * time.Hour)
	if got := decodeBody[repeatBody](t, do(h, "GET", "/api/repeat", "")); len(got.Items) != 1 {
		t.Fatalf("the typo must come back: %+v", got)
	}
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"Zdravo!"}`) // exact now
	*now = now.Add(24 * time.Hour)
	// It is an ordinary solved exercise again: back only as filler, never in the first tiers.
	got := decodeBody[repeatBody](t, do(h, "GET", "/api/repeat", ""))
	if len(got.Items) != 1 {
		t.Fatalf("filler pool should still hold it: %+v", got)
	}
}

func TestRepeatIsOncePerDayButNeverBlocksTheButton(t *testing.T) {
	h, _, now := movableAPI(t)
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"x"}`)

	first := do(h, "GET", "/api/repeat", "")
	if first.Code != 200 {
		t.Fatalf("first batch of the day = %d %s", first.Code, first.Body)
	}
	// Opened again before it is done: the same batch, not a block.
	if again := decodeBody[repeatBody](t, do(h, "GET", "/api/repeat", "")); len(again.Items) != 1 {
		t.Fatalf("an unfinished batch must come back: %+v", again)
	}

	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"Zdravo!"}`) // finishes it
	rr := do(h, "GET", "/api/repeat", "")
	if rr.Code != 402 {
		t.Fatalf("second batch of the day = %d %s, want 402", rr.Code, rr.Body)
	}
	lim := decodeBody[limitBody](t, rr)
	if lim.Error != "repeat_limit" {
		t.Errorf("error = %q, want repeat_limit", lim.Error)
	}
	next, err := time.Parse(time.RFC3339, lim.NextAvailable)
	if err != nil || !next.After(*now) || next.Sub(*now) > 24*time.Hour+time.Minute {
		t.Errorf("next_available = %q (%v), want the coming midnight", lim.NextAvailable, err)
	}

	*now = now.Add(24 * time.Hour) // tomorrow
	if rr := do(h, "GET", "/api/repeat", ""); rr.Code != 200 {
		t.Fatalf("the next day = %d %s, want a new batch", rr.Code, rr.Body)
	}
}

func TestRepeatResumeServesOnlyWhatIsLeft(t *testing.T) {
	h, _, _ := movableAPI(t)
	do(h, "POST", "/api/lessons/01/exercises/01-A-1/check", `{"answer":"x"}`)
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"x"}`)
	if got := decodeBody[repeatBody](t, do(h, "GET", "/api/repeat", "")); got.Session.Size != 2 {
		t.Fatalf("batch = %+v", got)
	}
	do(h, "POST", "/api/lessons/01/exercises/01-A-1/check", `{"answer":"Zdravo! Kako si?"}`)

	got := decodeBody[repeatBody](t, do(h, "GET", "/api/repeat", ""))
	if ids := repeatIDs(got); len(ids) != 1 || ids[0] != "01-A-2" || got.Session.Size != 2 || got.Session.Remaining != 1 {
		t.Fatalf("resume = %+v, want only the exercise still open (size 2, remaining 1)", got)
	}
}

func TestRepeatIsUnlimitedWithAYear(t *testing.T) {
	h, st, _ := movableAPI(t)
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"x"}`)
	id, err := st.CreatePayment(store.Payment{
		UserID: testUserID(t, st), TariffCode: store.TariffYear, AmountKopecks: 100000, Provider: "test", ProviderPaymentID: "y1",
	}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CompletePayment(id, fixedNow); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		rr := do(h, "GET", "/api/repeat", "")
		if rr.Code != 200 {
			t.Fatalf("request %d = %d %s, a year has no daily limit", i, rr.Code, rr.Body)
		}
		if got := decodeBody[repeatBody](t, rr); !got.Unlimited {
			t.Fatalf("request %d: unlimited = false", i)
		}
		do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"Zdravo!"}`)
	}
}

func TestRepeatEmptyDoesNotSpendTheDay(t *testing.T) {
	h, _, _ := movableAPI(t)
	rr := do(h, "GET", "/api/repeat", "")
	if rr.Code != 200 {
		t.Fatalf("a learner with nothing to repeat = %d %s", rr.Code, rr.Body)
	}
	raw := rr.Body.String() // decodeBody drains the recorder
	if !strings.Contains(raw, `"items":[]`) {
		t.Errorf("empty list must be [] not null: %s", raw)
	}
	if got := decodeBody[repeatBody](t, do(h, "GET", "/api/repeat", "")); got.Total != 0 {
		t.Fatalf("empty = %+v", got)
	}

	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"x"}`)
	if rr := do(h, "GET", "/api/repeat", ""); rr.Code != 200 {
		t.Fatalf("after the first answer = %d, an empty pool must not have used the day", rr.Code)
	}
}

// Content is edited after learners have answered it, so attempts can point at exercises that no
// longer exist. They must not break the screen or inflate the count.
func TestRepeatSkipsAttemptsAtRemovedExercises(t *testing.T) {
	h, st, _ := movableAPI(t)
	u := st.User(testUserID(t, st))
	if err := u.AddAttempt(store.Attempt{ExerciseID: "gone-1", Lesson: "01", Block: "A", Answer: "x"}, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := u.AddAttempt(store.Attempt{ExerciseID: "01-A-2", Lesson: "01", Block: "A", Answer: "x"}, fixedNow); err != nil {
		t.Fatal(err)
	}
	got := decodeBody[repeatBody](t, do(h, "GET", "/api/repeat", ""))
	if ids := repeatIDs(got); len(ids) != 1 || ids[0] != "01-A-2" {
		t.Fatalf("batch = %v, want only the exercise that still exists", ids)
	}
}

func TestRepeatSkipsLockedPhases(t *testing.T) {
	h, st := gatedAPI(t)
	// Lesson 01 is open until the phase becomes a product, so the learner can get one wrong first.
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"x"}`)
	seedPhaseProduct(t, st, "B", 500)

	rr := do(h, "GET", "/api/repeat", "")
	if rr.Code != 200 {
		t.Fatalf("GET /api/repeat = %d %s", rr.Code, rr.Body)
	}
	if got := decodeBody[repeatBody](t, rr); got.Total != 0 {
		t.Fatalf("a locked phase's exercise was served: %+v", got)
	}
}

// A learner who cannot listen turns dictation into "translate this sentence"; the repeat screen
// must honour the same switch as the lesson player.
func TestRepeatAudioOffTurnsDictationIntoTranslation(t *testing.T) {
	h, _, _ := movableAPI(t)
	do(h, "POST", "/api/lessons/01/exercises/01-A-4/check", `{"answer":"x"}`)

	on := decodeBody[repeatBody](t, do(h, "GET", "/api/repeat", ""))
	if len(on.Items) != 1 || on.Items[0].Exercise.Type != "listen" || on.Items[0].Exercise.Text != "" {
		t.Fatalf("audio on: %+v", on)
	}
	// Same day, same stored batch: the switch only changes how the exercise is shown.
	off := decodeBody[repeatBody](t, do(h, "GET", "/api/repeat?audio=off", ""))
	if len(off.Items) != 1 {
		t.Fatalf("audio off: %+v", off)
	}
	ex := off.Items[0].Exercise
	if ex.Text != "Zdravo, kako si?" || ex.Prompt != "Переведи на русский." || ex.Audio != "" {
		t.Errorf("audio off exercise = %+v", ex)
	}
}

func TestCheckStoresTheTypoFlag(t *testing.T) {
	h, st, _ := movableAPI(t)
	rr := do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"Zdravp!"}`)
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), `"typo":true`) {
		t.Fatalf("check = %d %s, want the typo flag in the answer", rr.Code, rr.Body)
	}
	cands, err := st.User(testUserID(t, st)).RepeatCandidates()
	if err != nil || len(cands) != 1 || !cands[0].LastCorrect || !cands[0].LastTypo {
		t.Fatalf("stored = %+v (%v), want a correct attempt flagged as a typo", cands, err)
	}
}

// Someone who has just finished a lesson has nothing older to repeat, and still has to see something:
// an empty «Повторить» right after the lesson looks broken (it did, for the first tester).
func TestRepeatRightAfterALessonStillServesWhatWasAnsweredToday(t *testing.T) {
	h, _, _ := movableAPI(t)
	do(h, "POST", "/api/lessons/01/exercises/01-A-1/check", `{"answer":"Zdravo! Kako si?"}`)
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"Zdravo!"}`)

	rr := do(h, "GET", "/api/repeat", "")
	if rr.Code != 200 {
		t.Fatalf("GET /api/repeat = %d %s", rr.Code, rr.Body)
	}
	got := decodeBody[repeatBody](t, rr)
	if got.Total != 2 || got.Session.Size != 2 {
		t.Fatalf("batch = %+v, want both exercises answered today", got)
	}
}

// An answer that is only a piece of the right one is a slip of format: the learner is asked to write
// the whole thing, and neither the mistake nor the attempt is recorded.
func TestCheckAnswerThatIsOnlyAPieceAsksForTheRestAndRecordsNothing(t *testing.T) {
	h, st, _ := movableAPI(t)
	for _, piece := range []string{"Zdravo", "Zdravo! Kako", "Ćao"} {
		rr := do(h, "POST", "/api/lessons/01/exercises/01-A-1/check", `{"answer":"`+piece+`"}`)
		if rr.Code != 200 {
			t.Fatalf("%q: check = %d %s", piece, rr.Code, rr.Body)
		}
		raw := rr.Body.String()
		if !strings.Contains(raw, `"format":"too_short"`) || strings.Contains(raw, `"ok":true`) {
			t.Errorf("%q: %s, want a too_short answer", piece, raw)
		}
		for _, leak := range []string{"expected", "diff", "Kako si"} {
			if strings.Contains(raw, leak) {
				t.Errorf("%q: the free retry leaked %q: %s", piece, leak, raw)
			}
		}
	}
	cands, err := st.User(testUserID(t, st)).RepeatCandidates()
	if err != nil || len(cands) != 0 {
		t.Fatalf("attempts after three too-short answers = %+v (%v), want none", cands, err)
	}

	// A wrong answer that is not a piece is still a mistake, and is recorded.
	rr := do(h, "POST", "/api/lessons/01/exercises/01-A-1/check", `{"answer":"Dobar dan"}`)
	if strings.Contains(rr.Body.String(), "too_short") || !strings.Contains(rr.Body.String(), `"ok":false`) {
		t.Errorf("a wrong answer = %s", rr.Body)
	}
	if cands, _ := st.User(testUserID(t, st)).RepeatCandidates(); len(cands) != 1 {
		t.Errorf("the wrong answer must be recorded: %+v", cands)
	}
}
