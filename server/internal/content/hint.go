package content

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/grisha/serbian-app/server/internal/checker"
)

// validateHint rejects a hint that spells out the answer: a hint is a nudge
// (a rule, a first letter, "look at the table"), not the solution. It looks for
// every accepted answer as a whole run of words inside the hint.
func validateHint(ex Exercise) error {
	if strings.TrimSpace(ex.Hint) == "" {
		return nil
	}
	hint := " " + checker.Normalize(ex.Hint) + " "
	var answers []string
	switch ex.Type {
	case "choice":
		answers = []string{ex.Answer}
	case "conjugate":
		for _, row := range ex.AcceptForms {
			answers = append(answers, row...)
		}
	case "match":
		return nil
	default:
		answers = ex.Accept
	}
	for _, a := range answers {
		n := checker.Normalize(a)
		if n == "" {
			continue
		}
		if strings.Contains(hint, " "+n+" ") {
			return fmt.Errorf("hint contains the answer %q — a hint must nudge, not tell", a)
		}
	}
	return nil
}

var (
	blankRE   = regexp.MustCompile(`_{2,}`)
	quotedRE  = regexp.MustCompile(`«([^»]*)»`)
	parenRE   = regexp.MustCompile(`\([^)]*\)`)
	cyrillicR = regexp.MustCompile(`[\x{0400}-\x{04FF}]`)
)

// FillSentences returns the full Serbian sentence of a fill-in-the-blank prompt
// with the gap filled by each accepted word, e.g. «Я в центре» → Ja sam u ___.
// with "centru" gives "Ja sam u centru.". Learners often type the whole
// sentence into the blank; Check accepts those too. Mirrors fill_sentence in
// scripts/tts.py and fillSentence in web/src/lib/phraseAudio.ts — keep in sync.
func FillSentences(prompt string, accept []string) []string {
	if len(blankRE.FindAllString(prompt, -1)) != 1 {
		return nil
	}
	t := prompt
	if m := firstQuotedWithBlank(prompt); m != "" {
		t = m
	} else if i := strings.LastIndex(prompt, "→"); i >= 0 {
		t = prompt[i+len("→"):]
	} else if i := strings.LastIndex(prompt, " — "); i >= 0 && blankRE.MatchString(prompt[i:]) {
		t = prompt[i+len(" — "):]
	}
	t = parenRE.ReplaceAllString(t, "")
	var out []string
	for _, w := range accept {
		if strings.TrimSpace(w) == "" {
			continue
		}
		s := strings.Join(strings.Fields(blankRE.ReplaceAllString(t, w)), " ")
		if s != "" && !cyrillicR.MatchString(s) {
			out = append(out, s)
		}
	}
	return out
}

func firstQuotedWithBlank(prompt string) string {
	for _, m := range quotedRE.FindAllStringSubmatch(prompt, -1) {
		if blankRE.MatchString(m[1]) {
			return m[1]
		}
	}
	return ""
}
