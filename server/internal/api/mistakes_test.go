package api

import (
	"testing"

	"github.com/grisha/serbian-app/server/internal/store"
)

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
