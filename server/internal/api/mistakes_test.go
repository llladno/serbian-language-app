package api

import (
	"strings"
	"testing"

	"github.com/grisha/serbian-app/server/internal/store"
)

// mistakesBody mirrors the wire shape of GET /api/mistakes on purpose (not the
// production DTO): the test pins the contract the client depends on.
type mistakesBody struct {
	Total int `json:"total"`
	Items []struct {
		Lesson   string `json:"lesson"`
		Exercise struct {
			ID      string   `json:"id"`
			Type    string   `json:"type"`
			Prompt  string   `json:"prompt"`
			Audio   string   `json:"audio"`
			Text    string   `json:"text"`
			Options []string `json:"options"`
		} `json:"exercise"`
	} `json:"items"`
}

func mistakesIDs(m mistakesBody) []string {
	ids := []string{}
	for _, it := range m.Items {
		ids = append(ids, it.Exercise.ID)
	}
	return ids
}

func TestMistakesListsLatestWrongExercisesWithoutAnswers(t *testing.T) {
	h, _ := newTestAPI(t)
	do(h, "POST", "/api/lessons/01/exercises/01-A-1/check", `{"answer":"Zdravo! Kako si?"}`) // right
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"x"}`)                // wrong

	rr := do(h, "GET", "/api/mistakes", "")
	if rr.Code != 200 {
		t.Fatalf("GET /api/mistakes = %d %s", rr.Code, rr.Body)
	}
	for _, leak := range []string{`"answer"`, `"accept"`, `"pairs"`, `"say"`, "Ćao"} {
		if strings.Contains(rr.Body.String(), leak) {
			t.Errorf("mistakes leaked %s: %s", leak, rr.Body)
		}
	}
	got := decodeBody[mistakesBody](t, rr)
	if got.Total != 1 || len(got.Items) != 1 {
		t.Fatalf("mistakes = %+v, want exactly the one wrong exercise", got)
	}
	it := got.Items[0]
	if it.Lesson != "01" || it.Exercise.ID != "01-A-2" || it.Exercise.Type != "translate" || it.Exercise.Prompt != "Привет!" {
		t.Errorf("item = %+v", it)
	}
}

func TestMistakesDropsAnExerciseOnceAnsweredRight(t *testing.T) {
	h, _ := newTestAPI(t)
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"x"}`)
	if got := decodeBody[mistakesBody](t, do(h, "GET", "/api/mistakes", "")); got.Total != 1 {
		t.Fatalf("before fixing: total = %d, want 1", got.Total)
	}

	// The same check endpoint the lesson player uses records the fix.
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"Zdravo!"}`)

	rr := do(h, "GET", "/api/mistakes", "")
	raw := rr.Body.String() // decodeBody drains the recorder
	if !strings.Contains(raw, `"items":[]`) {
		t.Errorf("empty list must be [] not null: %s", raw)
	}
	got := decodeBody[mistakesBody](t, do(h, "GET", "/api/mistakes", ""))
	if got.Total != 0 || len(got.Items) != 0 {
		t.Fatalf("after fixing: %+v", got)
	}
}

func TestMistakesLimitCapsTheBatchButTotalCountsAll(t *testing.T) {
	h, _ := newTestAPI(t)
	do(h, "POST", "/api/lessons/01/exercises/01-A-1/check", `{"answer":"x"}`)
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"x"}`)
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"y"}`) // 01-A-2 is the worse one

	got := decodeBody[mistakesBody](t, do(h, "GET", "/api/mistakes?limit=1", ""))
	if got.Total != 2 {
		t.Errorf("total = %d, want 2", got.Total)
	}
	if ids := mistakesIDs(got); len(ids) != 1 || ids[0] != "01-A-2" {
		t.Errorf("limit=1 gave %v, want the most-missed exercise first", ids)
	}

	// Nonsense limits fall back to the default batch instead of failing.
	for _, q := range []string{"?limit=abc", "?limit=0", "?limit=-3"} {
		rr := do(h, "GET", "/api/mistakes"+q, "")
		if rr.Code != 200 {
			t.Fatalf("GET /api/mistakes%s = %d", q, rr.Code)
		}
		if got := decodeBody[mistakesBody](t, rr); len(got.Items) != 2 {
			t.Errorf("%s: %d items, want both mistakes", q, len(got.Items))
		}
	}
}

// Content is edited after learners have answered it, so attempts can point at
// exercises that no longer exist. They must not break the screen or inflate the
// count.
func TestMistakesSkipsAttemptsAtRemovedExercises(t *testing.T) {
	h, st := newTestAPI(t)
	u := st.User(testUserID(t, st))
	if err := u.AddAttempt(store.Attempt{ExerciseID: "gone-1", Lesson: "01", Block: "A", Answer: "x"}, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := u.AddAttempt(store.Attempt{ExerciseID: "01-A-2", Lesson: "01", Block: "A", Answer: "x"}, fixedNow); err != nil {
		t.Fatal(err)
	}

	rr := do(h, "GET", "/api/mistakes", "")
	if rr.Code != 200 {
		t.Fatalf("GET /api/mistakes = %d %s", rr.Code, rr.Body)
	}
	got := decodeBody[mistakesBody](t, rr)
	if got.Total != 1 || len(got.Items) != 1 || got.Items[0].Exercise.ID != "01-A-2" {
		t.Fatalf("mistakes = %+v, want only the exercise that still exists", got)
	}
}

func TestMistakesSkipsLockedPhases(t *testing.T) {
	h, st := gatedAPI(t)
	// Lesson 01 is open until the phase becomes a product, so the learner can
	// get one wrong first.
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"x"}`)
	seedPhaseProduct(t, st, "B", 500)

	rr := do(h, "GET", "/api/mistakes", "")
	if rr.Code != 200 {
		t.Fatalf("GET /api/mistakes = %d %s", rr.Code, rr.Body)
	}
	got := decodeBody[mistakesBody](t, rr)
	if got.Total != 0 || len(got.Items) != 0 {
		t.Fatalf("a locked phase's exercise was served: %+v", got)
	}
}

// A learner who cannot listen turns dictation into "translate this sentence";
// the mistakes screen must honour the same switch as the lesson player.
func TestMistakesAudioOffTurnsDictationIntoTranslation(t *testing.T) {
	h, _ := newTestAPI(t)
	do(h, "POST", "/api/lessons/01/exercises/01-A-4/check", `{"answer":"x"}`)

	on := decodeBody[mistakesBody](t, do(h, "GET", "/api/mistakes", ""))
	if len(on.Items) != 1 || on.Items[0].Exercise.Type != "listen" {
		t.Fatalf("audio on: %+v", on)
	}
	if on.Items[0].Exercise.Text != "" {
		t.Errorf("audio on leaked the sentence: %+v", on.Items[0].Exercise)
	}

	off := decodeBody[mistakesBody](t, do(h, "GET", "/api/mistakes?audio=off", ""))
	if len(off.Items) != 1 {
		t.Fatalf("audio off: %+v", off)
	}
	ex := off.Items[0].Exercise
	if ex.Text != "Zdravo, kako si?" || ex.Prompt != "Переведи на русский." || ex.Audio != "" {
		t.Errorf("audio off exercise = %+v", ex)
	}
}

// "Стоит повторить" on the profile and the mistakes screen are one list: a row on
// the profile that the screen then refuses to open would send the learner to
// "Ошибок нет".
func TestProfileWeakListSkipsWhatTheMistakesScreenSkips(t *testing.T) {
	h, st := gatedAPI(t)
	u := st.User(testUserID(t, st))
	if err := u.AddAttempt(store.Attempt{ExerciseID: "gone-1", Lesson: "01", Block: "A", Answer: "x"}, fixedNow); err != nil {
		t.Fatal(err)
	}
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"x"}`)

	ids := func() []string {
		p := decodeBody[progressDTO](t, do(h, "GET", "/api/progress", ""))
		out := []string{}
		for _, w := range p.WeakExercises {
			out = append(out, w.ExerciseID)
		}
		return out
	}
	// An attempt at an exercise that no longer exists is not a mistake you can fix.
	if got := ids(); len(got) != 1 || got[0] != "01-A-2" {
		t.Fatalf("profile weak list = %v, want only the exercise that still exists", got)
	}

	// Lesson 01 sits in a phase that is now for sale and not owned.
	seedPhaseProduct(t, st, "B", 500)
	if got := ids(); len(got) != 0 {
		t.Fatalf("profile weak list = %v, want it empty while the phase is locked", got)
	}
}

func TestProfileWeakRowShowsThePromptOfItsOwnLesson(t *testing.T) {
	h, _ := newTestAPI(t)
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"x"}`)
	p := decodeBody[progressDTO](t, do(h, "GET", "/api/progress", ""))
	if len(p.WeakExercises) != 1 || p.WeakExercises[0].Prompt != "Привет!" || p.WeakExercises[0].Lesson != "01" {
		t.Fatalf("weak rows = %+v", p.WeakExercises)
	}
}
