package content

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/grisha/serbian-app/server/internal/checker"
)

// lexTokens splits a string into normalized word tokens (lower-case, no
// punctuation) using the same normalization as answer checking.
func lexTokens(s string) []string {
	return strings.Fields(checker.Normalize(s))
}

func isCyrillic(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Cyrillic, r) {
			return true
		}
	}
	return false
}

// allowWordsFile is the shape of content/allow-words.yaml: a few named lists
// that are flattened into one allow-set.
type allowWordsFile map[string][]string

func loadAllowWords(dir string) []string {
	var f allowWordsFile
	if err := readYAML(filepath.Join(dir, "allow-words.yaml"), &f); err != nil {
		return nil
	}
	var out []string
	for _, list := range f {
		for _, w := range list {
			out = append(out, lexTokens(w)...)
		}
	}
	return out
}

// knownWords is the cumulative set of word tokens a learner has met by the
// time they reach lesson upTo: every vocab entry from an earlier-or-equal
// lesson (in course order), the lesson's own `teaches`, and the global
// allow-list.
func knownWords(c *Course, upTo string) map[string]bool {
	order := map[string]int{}
	n := 0
	for _, ph := range c.Phases {
		for _, id := range ph.Lessons {
			order[id] = n
			n++
		}
	}
	limit, ok := order[upTo]
	if !ok {
		limit = 1 << 30
	}

	known := map[string]bool{}
	byID := map[string]*Vocab{}
	for i := range c.Vocab {
		v := &c.Vocab[i]
		byID[v.ID] = v
		if o, ok := order[v.Lesson]; ok && o <= limit {
			for _, tok := range lexTokens(v.Latin) {
				known[tok] = true
			}
		}
	}
	if l := c.Lessons[upTo]; l != nil {
		for _, id := range l.Teaches {
			if v := byID[id]; v != nil {
				for _, tok := range lexTokens(v.Latin) {
					known[tok] = true
				}
			}
		}
	}
	for _, w := range c.allowWords {
		known[w] = true
	}
	return known
}

// unknownTokens returns the tokens of text that are neither known, covered by
// a known prefix (e.g. "grad" covers "gradu"), nor listed in the step's
// also_ok.
func unknownTokens(text string, known map[string]bool, alsoOK []string) []string {
	ok := map[string]bool{}
	for _, a := range alsoOK {
		for _, tok := range lexTokens(a) {
			ok[tok] = true
		}
	}
	var out []string
	for _, tok := range lexTokens(text) {
		if known[tok] || ok[tok] || coveredByPrefix(tok, known) {
			continue
		}
		out = append(out, tok)
	}
	return out
}

func coveredByPrefix(tok string, known map[string]bool) bool {
	for k := range known {
		if len(k) >= 3 && strings.HasPrefix(tok, k) {
			return true
		}
	}
	return false
}

// strictStrings collects the pure-Serbian strings of an exercise that the
// lexicon guard checks strictly (prompt is checked leniently elsewhere).
func strictStrings(e Exercise) []string {
	var out []string
	out = append(out, e.Accept...)
	out = append(out, e.Options...)
	out = append(out, e.Bank...)
	if e.Say != "" {
		out = append(out, e.Say)
	}
	for _, p := range e.Pairs {
		if isCyrillic(p[1]) {
			out = append(out, p[0]) // left Serbian, right Russian
		} else {
			out = append(out, p[0], p[1])
		}
	}
	return out
}
