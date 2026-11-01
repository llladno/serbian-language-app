package content

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

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
	t, ok := fillTemplate(prompt)
	if !ok {
		return nil
	}
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

// fillTemplate cuts the Serbian sentence with the single gap out of a fill-in-the-blank prompt.
func fillTemplate(prompt string) (string, bool) {
	if len(blankRE.FindAllString(prompt, -1)) != 1 {
		return "", false
	}
	t := prompt
	if m := firstQuotedWithBlank(prompt); m != "" {
		t = m
	} else if i := strings.LastIndex(prompt, "→"); i >= 0 {
		t = prompt[i+len("→"):]
	} else if i := strings.LastIndex(prompt, " — "); i >= 0 && blankRE.MatchString(prompt[i:]) {
		t = prompt[i+len(" — "):]
	}
	return parenRE.ReplaceAllString(t, ""), true
}

// FillAnswers returns every spelling of a fill-in-the-blank answer the check should accept: the
// accepted word itself and the word with any unbroken run of the prompt's own Serbian words around
// it («___ mama» takes "tvoja", "tvoja mama", and the whole sentence). Copying a neighbour from the
// task is not a mistake, only a longer answer. A gap glued to a letter («ne___») has no such
// neighbours, so only the whole sentence applies there.
func FillAnswers(prompt string, accept []string) []string {
	t, ok := fillTemplate(prompt)
	if !ok {
		return nil
	}
	loc := blankRE.FindStringIndex(t)
	before, after := t[:loc[0]], t[loc[1]:]
	if lastIsLetter(before) || firstIsLetter(after) {
		return FillSentences(prompt, accept)
	}
	pre, post := strings.Fields(before), strings.Fields(after)
	var out []string
	add := func(words []string) {
		s := strings.Join(words, " ")
		if s != "" && !cyrillicR.MatchString(s) {
			out = append(out, s)
		}
	}
	for _, w := range accept {
		if strings.TrimSpace(w) == "" {
			continue
		}
		for i := 0; i <= len(pre); i++ {
			for j := 0; j <= len(post); j++ {
				words := append(append(append([]string{}, pre[len(pre)-i:]...), w), post[:j]...)
				add(words)
			}
		}
	}
	return out
}

func lastIsLetter(s string) bool {
	r, _ := utf8.DecodeLastRuneInString(s)
	return unicode.IsLetter(r)
}

func firstIsLetter(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsLetter(r)
}

func firstQuotedWithBlank(prompt string) string {
	for _, m := range quotedRE.FindAllStringSubmatch(prompt, -1) {
		if blankRE.MatchString(m[1]) {
			return m[1]
		}
	}
	return ""
}
