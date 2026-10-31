package content

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

// splitLatinWords returns the lower-case Latin-script words of s. Anything that
// is not a Latin letter (Cyrillic transcriptions, digits, punctuation, "/")
// separates words.
func splitLatinWords(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.Is(unicode.Latin, r)
	})
}

func foldSerbian(w string) string {
	return strings.NewReplacer("č", "c", "ć", "c", "š", "s", "ž", "z", "đ", "dj").Replace(w)
}

// TestDictationAndConjugationSeenInText is the strict sibling of
// TestLexiconGuardRealContent. The lexicon guard trusts vocab.yaml's `lesson`
// field and a broad allow-list, so it let through a dictation of "Drago mi je"
// and a "be" conjugation over mi/vi/oni in lessons that never showed those
// words. Here a dictated phrase (`listen`) or a conjugation table cell
// (`conjugate`) may only use words that appeared in the TEXT of an earlier
// step or lesson: teach/reading markdown or a dialogue line. Only proper names
// and a few prepositions/particles are exempt, and a conjugation form of a long
// verb may be new if its stem was shown (the table pattern was taught).
func TestDictationAndConjugationSeenInText(t *testing.T) {
	c, err := Load("../../../content")
	if err != nil {
		t.Fatal(err)
	}

	exempt := map[string]bool{}
	for _, w := range strings.Fields("i a ali ili ne da li se u na o za iz do od s sa po bez") {
		exempt[w] = true
	}
	var allow allowWordsFile
	if err := readYAML(filepath.Join("../../../content", "allow-words.yaml"), &allow); err != nil {
		t.Fatal(err)
	}
	for _, name := range allow["names"] {
		for _, w := range splitLatinWords(name) {
			exempt[w] = true
		}
	}

	seen := map[string]bool{}
	see := func(text string) {
		for _, w := range splitLatinWords(text) {
			seen[w] = true
			seen[foldSerbian(w)] = true
		}
	}
	// sameStem reports whether a long word shares its first four letters with a
	// word already shown: that is a verb form built by a taught pattern
	// (čitam shown, čitate asked), not a new word.
	sameStem := func(w string) bool {
		r := []rune(foldSerbian(w))
		if len(r) < 5 {
			return false
		}
		prefix := string(r[:4])
		for k := range seen {
			if strings.HasPrefix(k, prefix) {
				return true
			}
		}
		return false
	}
	check := func(lesson, step, ex, kind, text string, alsoOK map[string]bool) {
		for _, w := range splitLatinWords(text) {
			if exempt[w] || alsoOK[w] || seen[w] || seen[foldSerbian(w)] {
				continue
			}
			if strings.HasPrefix(kind, "conjugation") && sameStem(w) {
				continue
			}
			t.Errorf("lesson %s step %s ex %s (%s): %q is not shown in any earlier "+
				"theory/reading/dialogue text (in %q); add it to the lesson's teach "+
				"text or use known words", lesson, step, ex, kind, w, text)
		}
	}

	for _, ph := range c.Phases {
		for _, id := range ph.Lessons {
			l := c.Lessons[id]
			if l == nil || !l.Manifest {
				continue
			}
			for _, s := range l.Steps {
				alsoOK := map[string]bool{}
				for _, a := range s.AlsoOK {
					for _, w := range splitLatinWords(a) {
						alsoOK[w] = true
					}
				}
				see(s.Markdown)
				for _, e := range s.Exercises {
					switch e.Type {
					case "listen":
						check(id, s.ID, e.ID, "dictation", e.Say, alsoOK)
					case "choice":
						// «Послушай и выбери»: the clip is a sentence the learner has to
						// recognise, so it too may only use words already shown in text.
						if e.Say != "" {
							check(id, s.ID, e.ID, "listened phrase", e.Say, alsoOK)
						}
					case "conjugate":
						for _, f := range e.Forms {
							check(id, s.ID, e.ID, "conjugation label", f, alsoOK)
						}
						for _, row := range e.AcceptForms {
							for _, a := range row {
								check(id, s.ID, e.ID, "conjugation form", a, alsoOK)
							}
						}
					}
				}
				for _, turn := range s.Turns {
					see(turn.SR)
				}
			}
		}
	}
}

// russianWords returns the lower-case Cyrillic words of s with ё folded to е.
func russianWords(s string) []string {
	return strings.FieldsFunc(strings.ToLower(strings.ReplaceAll(s, "ё", "е")), func(r rune) bool {
		return !unicode.Is(unicode.Cyrillic, r)
	})
}

// russianStem is the part of a Russian word compared across inflections:
// «аппетита» and «аппетит» share a stem, «спасибо» and «спасибки» do not.
func russianStem(w string) string {
	r := []rune(w)
	if len(r) > 5 {
		r = r[:5]
	}
	return string(r)
}

// TestDictationRussianAnswerSeenInText guards the Russian side of a dictation.
// In «Не могу прослушать» mode (and in the repeat session) a `listen` exercise
// asks "translate into Russian" and shows `ru[0]` as «Правильно: …». That
// translation must not teach a meaning the course has not shown: «Hvala,
// prijatno!» was answered «Спасибо, приятного аппетита» while the lesson that
// introduces `prijatno` only said «всего доброго». Every word of the displayed
// translation has to occur in the Russian text of an earlier step or lesson
// (theory, reading translation, dialogue line, lesson title/subtitle). The
// other `ru` variants are only accepted, never shown, so they are free.
func TestDictationRussianAnswerSeenInText(t *testing.T) {
	c, err := Load("../../../content")
	if err != nil {
		t.Fatal(err)
	}
	exempt := map[string]bool{}
	for _, w := range strings.Fields("и а но или не ни да нет в на у к о об от до за из по с со без для при как что это то ты вы я мы он она они мой моя мое твой ваш вам вас мне тебе нас его ее их там тут здесь очень еще уже тоже вот") {
		exempt[w] = true
	}
	// A Russian numeral is taught by its digits («15 petnaest»), so the word itself
	// never has to appear in the text.
	for _, w := range strings.Fields("ноль нуль один одна одно два две три четыре пять шесть семь восемь девять десять " +
		"одиннадцать двенадцать тринадцать четырнадцать пятнадцать шестнадцать семнадцать восемнадцать девятнадцать " +
		"двадцать тридцать сорок пятьдесят шестьдесят семьдесят восемьдесят девяносто сто двести триста четыреста " +
		"пятьсот шестьсот семьсот восемьсот девятьсот тысяча тысячи тысяч") {
		exempt[w] = true
	}
	seen := map[string]bool{}
	see := func(text string) {
		for _, w := range russianWords(text) {
			seen[w] = true
			seen[russianStem(w)] = true
		}
	}
	known := func(w string) bool {
		return exempt[w] || seen[w] || (len([]rune(w)) > 3 && seen[russianStem(w)])
	}
	for _, ph := range c.Phases {
		for _, id := range ph.Lessons {
			l := c.Lessons[id]
			if l == nil || !l.Manifest {
				continue
			}
			see(l.Title)
			see(l.Subtitle)
			for _, s := range l.Steps {
				see(s.Markdown)
				see(s.MarkdownRU)
				see(s.Scene)
				for _, e := range s.Exercises {
					if e.Type != "listen" || len(e.RU) == 0 {
						continue
					}
					for _, w := range russianWords(e.RU[0]) {
						if !known(w) {
							t.Errorf("lesson %s step %s ex %s: the shown Russian answer %q uses %q, "+
								"which no earlier theory/reading/dialogue text has shown; teach that meaning "+
								"in the lesson or put a taught translation first in `ru`", id, s.ID, e.ID, e.RU[0], w)
						}
					}
				}
				for _, turn := range s.Turns {
					see(turn.RU)
				}
			}
		}
	}
}
