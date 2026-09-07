package api

import (
	"net/http"
	"strings"

	"github.com/grisha/serbian-app/server/internal/checker"
	"github.com/grisha/serbian-app/server/internal/content"
)

// lookupResultDTO is the response of GET /api/lookup.
type lookupResultDTO struct {
	Query   string     `json:"query"`
	Matches []vocabDTO `json:"matches"`
	Partial bool       `json:"partial"` // matches are prefix guesses, not exact
}

func (h handlers) lookup(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	found, partial := lookupVocab(h.Course().Vocab, q)
	out := lookupResultDTO{Query: q, Matches: []vocabDTO{}, Partial: partial}
	for _, v := range found {
		out.Matches = append(out.Matches, vocabDTO{
			ID: v.ID, Latin: v.Latin, Cyrillic: v.Cyrillic, RU: v.RU, Note: v.Note,
			Lesson: v.Lesson, POS: v.POS, Gender: v.Gender, Aspect: v.Aspect, Tags: v.Tags,
			Emoji: v.Emoji, Image: v.Image, Audio: v.Audio,
		})
	}
	writeJSON(w, 200, out)
}

// lookupVocab finds dictionary entries for a word taken from a reading text.
// The word is usually inflected, so after an exact (normalized) match on the
// latin or cyrillic headword — or the word appearing whole inside a phrase
// entry — it falls back to entries sharing a 4+ character prefix, returned with
// partial=true.
func lookupVocab(vocab []content.Vocab, q string) (matches []content.Vocab, partial bool) {
	nq := checker.Normalize(q)
	if nq == "" {
		return nil, false
	}
	var exact, prefix []content.Vocab
	for _, v := range vocab {
		nl, nc := checker.Normalize(v.Latin), checker.Normalize(v.Cyrillic)
		switch {
		case nl == nq || nc == nq || wordIn(nl, nq) || wordIn(nc, nq):
			exact = append(exact, v)
		case commonPrefixLen(nl, nq) >= 4 || commonPrefixLen(nc, nq) >= 4:
			prefix = append(prefix, v)
		}
	}
	if len(exact) > 0 {
		return exact, false
	}
	return prefix, len(prefix) > 0
}

// wordIn reports whether word is one of the space-separated tokens of s.
// Single-token entries are handled by the equality check in lookupVocab, so a
// bare word only matches here when s is a multi-word phrase.
func wordIn(s, word string) bool {
	fields := strings.Fields(s)
	if len(fields) < 2 {
		return false
	}
	for _, f := range fields {
		if f == word {
			return true
		}
	}
	return false
}

func commonPrefixLen(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	n := 0
	for n < len(ra) && n < len(rb) && ra[n] == rb[n] {
		n++
	}
	return n
}
