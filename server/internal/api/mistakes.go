package api

import (
	"github.com/grisha/serbian-app/server/internal/content"
	"github.com/grisha/serbian-app/server/internal/store"
)

// weakScanLimit bounds how many of a learner's weakest exercises are read before the ones
// whose content is gone or locked are dropped.
const weakScanLimit = 500

type mistakeDTO struct {
	Lesson   string      `json:"lesson"`
	Exercise exerciseDTO `json:"exercise"`
	// Context is what the question is about, when it is not self-contained: the reading text.
	Context *contextDTO `json:"context,omitempty"`
	// KeepLatin: the lesson teaches the letters, so the exercise stays in Latin whatever script the
	// learner reads in.
	KeepLatin bool `json:"keep_latin,omitempty"`
}

// contextDTO is a reading text with its translation.
type contextDTO struct {
	Serbian string `json:"serbian"`
	Russian string `json:"russian,omitempty"`
}

// openMistake is a current mistake that can actually be opened: its exercise
// still exists and its lesson is not locked.
type openMistake struct {
	store.WeakExercise
	Exercise content.Exercise
}

// inDialogue reports whether an exercise is a turn of a dialogue. Out of its chat it has no line to
// answer, so neither the repeat batch nor the weak list takes it.
func (h handlers) inDialogue(lesson, exerciseID string) bool {
	_, ok := h.findTurn(lesson, exerciseID)
	return ok
}

// keepsLatin reports whether a lesson teaches the letters and so stays in Latin.
func (h handlers) keepsLatin(lesson string) bool {
	l := h.Course().Lessons[lesson]
	return l != nil && l.KeepLatin
}

// openMistakes is the profile's "Стоит повторить" list, worst first. An attempt can outlive its
// exercise (content is edited after learners have answered it), and a locked phase's exercises
// are not reachable at all, so neither is listed; the repeat batch (repeat.go) filters the same way.
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
		if locked[wk.Lesson] || h.inDialogue(wk.Lesson, wk.ExerciseID) {
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
