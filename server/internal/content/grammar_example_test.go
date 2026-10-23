package content

import (
	"regexp"
	"strings"
	"testing"
)

// A grammar card has one example for the whole card, which fits one form at most.
// A learner who is asked for "oni" and sees "Čitam knjigu. — Я читаю книгу." under
// the right answer "čitaju" reads it as the translation of that answer (a real
// support message). So an item can carry its own example, and it wins.

func TestGrammarItemExampleWinsOverTheCardExample(t *testing.T) {
	dir := t.TempDir()
	tree := baseTree()
	tree["grammar.yaml"] = `
- id: "x"
  front: "f"
  lesson: "01"
  example_sr: "Čitam knjigu."
  example_ru: "Я читаю книгу."
  items:
    - {prompt: "ja", accept: ["čitam"]}
    - {prompt: "oni", accept: ["čitaju"], example_sr: "Čitaju knjigu.", example_ru: "Они читают книгу."}
`
	writeTree(t, dir, tree)
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	g := c.Grammar[0]
	if sr, ru := g.ExampleFor(0); sr != "Čitam knjigu." || ru != "Я читаю книгу." {
		t.Errorf("item without its own example: %q / %q, want the card's", sr, ru)
	}
	if sr, ru := g.ExampleFor(1); sr != "Čitaju knjigu." || ru != "Они читают книгу." {
		t.Errorf("item with its own example: %q / %q, want its own", sr, ru)
	}
	if sr, ru := g.ExampleFor(7); sr != "" || ru != "" {
		t.Errorf("an index out of range gave %q / %q, want nothing", sr, ru)
	}
}

func word(s string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)(^|[^\p{L}])` + regexp.QuoteMeta(s) + `($|[^\p{L}])`)
}

// An example that an item carries is about that item's form, so it has to use one
// of the accepted answers: it cannot show a different form of the paradigm.
func TestEveryGrammarItemExampleUsesTheFormItIsAbout(t *testing.T) {
	c, err := Load("../../../content")
	if err != nil {
		t.Fatalf("real content: %v", err)
	}
	for _, g := range c.Grammar {
		for i, it := range g.Items {
			if it.ExampleSR == "" && it.ExampleRU == "" {
				continue
			}
			if it.ExampleSR == "" || it.ExampleRU == "" {
				t.Errorf("%s item %d (%s): the example needs both example_sr and example_ru", g.ID, i, it.Prompt)
				continue
			}
			used := false
			for _, a := range it.Accept {
				if word(a).MatchString(it.ExampleSR) {
					used = true
				}
			}
			if !used {
				t.Errorf("%s item %d (%s): example %q does not use the answer %v", g.ID, i, it.Prompt, it.ExampleSR, it.Accept)
			}
		}
	}
}

// The card's own example fits one person of a verb at most, so a card that drills
// the persons (ja, ti, on, mi, vi, oni) gives every item an example of its own.
func TestACardThatDrillsPersonsGivesEveryItemItsOwnExample(t *testing.T) {
	c, err := Load("../../../content")
	if err != nil {
		t.Fatalf("real content: %v", err)
	}
	persons := 0
	for _, g := range c.Grammar {
		if !strings.HasPrefix(g.Items[0].Prompt, "ja ") && !strings.HasPrefix(g.Items[0].Prompt, "ja,") {
			continue
		}
		persons++
		for i, it := range g.Items {
			if it.ExampleSR == "" {
				t.Errorf("%s item %d (%s): no example of its own, so it would show the card's example for another person", g.ID, i, it.Prompt)
			}
		}
	}
	if persons < 5 {
		t.Errorf("found %d person cards, want the present tenses, jesam, imati and perfekt (>= 5)", persons)
	}
}
