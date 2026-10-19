package api

import (
	"strings"
	"testing"
)

// A lesson about the letters themselves says so, and the client then ignores the
// learner's script choice inside it.
func TestLessonCarriesTheKeepLatinFlag(t *testing.T) {
	h, _ := newTestAPI(t)

	keep := decodeBody[lessonDTO](t, do(h, "GET", "/api/lessons/91", ""))
	if !keep.KeepLatin {
		t.Errorf("lesson 91 (`script: latin`) keep_latin = false, want true")
	}

	rr := do(h, "GET", "/api/lessons/90", "")
	if strings.Contains(rr.Body.String(), "keep_latin") {
		t.Errorf("a lesson without the flag must not carry it: %s", rr.Body)
	}
}
