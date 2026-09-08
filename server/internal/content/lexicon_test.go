package content

import "testing"

func TestUnknownTokens(t *testing.T) {
	known := map[string]bool{"grad": true, "zdravo": true}
	got := unknownTokens("Zdravo, idem u grad danas", known, []string{"idem"})

	want := map[string]bool{"u": true, "danas": true}
	for _, g := range got {
		if !want[g] {
			t.Errorf("unexpected unknown %q (all: %v)", g, got)
		}
		delete(want, g)
	}
	for k := range want {
		t.Errorf("missed unknown %q", k)
	}
}

func TestPrefixCoverage(t *testing.T) {
	known := map[string]bool{"grad": true}
	if u := unknownTokens("u gradu", known, nil); len(u) != 1 || u[0] != "u" {
		t.Errorf("gradu should be covered by grad prefix; got %v", u)
	}
}

// TestLexiconGuardRealContent checks that every strict string in a manifest
// lesson uses only words taught by that point. Legacy lessons are advisory.
func TestLexiconGuardRealContent(t *testing.T) {
	c, err := Load("../../../content")
	if err != nil {
		t.Fatal(err)
	}
	for _, ph := range c.Phases {
		for _, id := range ph.Lessons {
			l := c.Lessons[id]
			if l == nil || len(l.Steps) == 0 {
				continue
			}
			known := knownWords(c, id)
			for _, s := range l.Steps {
				for _, e := range s.Exercises {
					for _, txt := range strictStrings(e) {
						for _, u := range unknownTokens(txt, known, s.AlsoOK) {
							msg := "lesson %s step %s ex %s: unknown word %q " +
								"(add it to the step's also_ok or the lesson's teaches)"
							if l.Manifest {
								t.Errorf(msg, id, s.ID, e.ID, u)
							} else {
								t.Logf("[legacy] "+msg, id, s.ID, e.ID, u)
							}
						}
					}
				}
			}
		}
	}
}
