package api

import (
	"testing"

	"github.com/grisha/serbian-app/server/internal/content"
)

// The client tells the learner what to type (one word, a phrase, the missing word, the whole sentence)
// from the shape of the answer; the answer itself never travels.
func TestExerciseShapeSaysWhatToType(t *testing.T) {
	for _, c := range []struct {
		name string
		ex   content.Exercise
		want string
	}{
		{"translate to one word", content.Exercise{Type: "translate", Accept: []string{"džem"}}, "word"},
		{"translate to a phrase", content.Exercise{Type: "translate", Accept: []string{"Hvala, prijatno!"}}, "phrase"},
		{"one-word and phrase variants are a phrase", content.Exercise{Type: "translate", Accept: []string{"Zdravo", "Dobar dan"}}, "phrase"},
		{"a blank in a sentence", content.Exercise{Type: "fill_blank", Prompt: "Ovo je ___.", Accept: []string{"brat"}}, "blank"},
		{"fix the sentence", content.Exercise{Type: "fix_error", Prompt: "Imam brat.", Accept: []string{"Imam brata."}}, "sentence"},
		{"dictation of a word", content.Exercise{Type: "listen", Accept: []string{"konj"}}, "word"},
		{"dictation of a sentence", content.Exercise{Type: "listen", Accept: []string{"Dobar dan, kako ste?"}}, "phrase"},
		{"a choice has no shape", content.Exercise{Type: "choice", Options: []string{"a", "b"}}, ""},
	} {
		if got := toExerciseDTO(c.ex, false).Shape; got != c.want {
			t.Errorf("%s: shape = %q, want %q", c.name, got, c.want)
		}
	}
}
