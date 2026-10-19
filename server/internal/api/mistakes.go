package api

import (
	"net/http"
	"strconv"

	"github.com/grisha/serbian-app/server/internal/content"
	"github.com/grisha/serbian-app/server/internal/store"
)

const (
	// A session is a short batch: ten exercises is a few minutes, and the rest
	// wait for next time rather than greeting a learner with a wall of work.
	mistakesDefaultBatch = 10
	mistakesMaxBatch     = 30
	// weakScanLimit bounds how many of a learner's weakest exercises are read
	// before the ones whose content is gone or locked are dropped.
	weakScanLimit = 500
)

type mistakeDTO struct {
	Lesson   string      `json:"lesson"`
	Exercise exerciseDTO `json:"exercise"`
}

type mistakesDTO struct {
	// Total is every exercise currently marked as a mistake, not just the batch.
	Total int          `json:"total"`
	Items []mistakeDTO `json:"items"`
}

// openMistake is a current mistake that can actually be opened: its exercise
// still exists and its lesson is not locked.
type openMistake struct {
	store.WeakExercise
	Exercise content.Exercise
}

// openMistakes is the one list of mistakes, worst first. Both the profile's "Стоит
// повторить" and the mistakes screen are built from it, so a row on the profile is
// never one the screen refuses to open. An attempt can outlive its exercise
// (content is edited after learners have answered it), and a locked phase's
// exercises are not reachable at all.
func (h handlers) openMistakes(us *store.UserStore) ([]openMistake, error) {
	weak, err := us.WeakExercises(weakScanLimit)
	if err != nil {
		return nil, err
	}
	locked, err := h.lockedLessonIDs(us)
	if err != nil {
		return nil, err
	}
	var out []openMistake
	for _, wk := range weak {
		if locked[wk.Lesson] {
			continue
		}
		ex, _, found := h.findExercise(wk.Lesson, wk.ExerciseID)
		if !found {
			continue
		}
		out = append(out, openMistake{WeakExercise: wk, Exercise: ex})
	}
	return out, nil
}

// getMistakes serves the exercises the learner last got wrong, so they can be
// answered again without going back into the lesson. Answering goes through the
// ordinary check endpoint, which is what takes a fixed exercise off this list.
func (h handlers) getMistakes(w http.ResponseWriter, r *http.Request) {
	us, ok := h.user(w, r)
	if !ok {
		return
	}
	limit := mistakesDefaultBatch
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 {
		limit = min(n, mistakesMaxBatch)
	}
	audioOff := r.URL.Query().Get("audio") == "off"

	open, err := h.openMistakes(us)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}

	out := mistakesDTO{Total: len(open), Items: []mistakeDTO{}}
	for _, m := range open {
		if len(out.Items) < limit {
			out.Items = append(out.Items, mistakeDTO{Lesson: m.Lesson, Exercise: toExerciseDTO(m.Exercise, audioOff)})
		}
	}
	writeJSON(w, 200, out)
}
