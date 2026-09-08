package content

import (
	"strings"
	"testing"
)

// TestLegacyLessonSynthesizesSteps checks that a legacy .md lesson is turned
// into a playable step flow (teach card -> practice blocks -> reading). The
// real content/ tree has no legacy lessons left, so this uses the fixture.
func TestLegacyLessonSynthesizesSteps(t *testing.T) {
	c, err := Load("testdata/content")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	l := c.Lessons["01"]
	if l.Manifest {
		t.Fatal("fixture lesson 01 should be legacy")
	}
	if len(l.Steps) < 3 {
		t.Fatalf("lesson 01: %d synthetic steps, want >= 3", len(l.Steps))
	}
	if l.Steps[0].Kind != "teach" || l.Steps[0].Markdown == "" {
		t.Errorf("step 0 not a teach card: %+v", l.Steps[0])
	}
	last := l.Steps[len(l.Steps)-1]
	if last.Kind != "reading" || last.MarkdownRU == "" {
		t.Errorf("last step not a reading step: %+v", last)
	}
	hasPractice := false
	for _, s := range l.Steps {
		if s.Kind == "practice" && len(s.Exercises) > 0 {
			hasPractice = true
		}
	}
	if !hasPractice {
		t.Error("no practice step carries exercises")
	}
}

// TestRealContentLoads guards the migrated content/ tree at the repo root.
func TestRealContentLoads(t *testing.T) {
	c, err := Load("../../../content")
	if err != nil {
		t.Fatalf("real content: %v", err)
	}
	if len(c.Vocab) < 60 {
		t.Errorf("vocab = %d, want >= 60", len(c.Vocab))
	}
	withAudio := 0
	for _, v := range c.Vocab {
		if v.Audio != "" {
			withAudio++
			if v.Audio != v.ID+".mp3" {
				t.Errorf("vocab %q: audio = %q, want %q", v.ID, v.Audio, v.ID+".mp3")
			}
		}
	}
	if withAudio < 60 {
		t.Errorf("vocab with audio = %d, want >= 60 (run scripts/tts.py)", withAudio)
	}
	if len(c.FalseFriends) < 50 {
		t.Errorf("false friends = %d, want >= 50", len(c.FalseFriends))
	}
	{
		l := c.Lessons["01"]
		if !l.Manifest {
			t.Fatal("lesson 01 should be a manifest")
		}
		if len(l.Steps) < 6 {
			t.Errorf("lesson 01: %d steps, want >= 6", len(l.Steps))
		}
		kinds := map[string]int{}
		for _, s := range l.Steps {
			kinds[s.Kind]++
		}
		if kinds["teach"] < 2 || kinds["practice"] < 2 || kinds["checkpoint"] < 1 {
			t.Errorf("lesson 01 step kinds: %v", kinds)
		}
	}
	{
		l := c.Lessons["02"]
		if !l.Manifest {
			t.Fatal("lesson 02 should be a manifest")
		}
		if len(l.Steps) < 6 {
			t.Errorf("lesson 02: %d steps, want >= 6", len(l.Steps))
		}
		kinds := map[string]int{}
		for _, s := range l.Steps {
			kinds[s.Kind]++
		}
		if kinds["teach"] < 2 || kinds["practice"] < 2 || kinds["checkpoint"] < 1 {
			t.Errorf("lesson 02 step kinds: %v", kinds)
		}
	}
	{
		l := c.Lessons["03"]
		if !l.Manifest || l.Title == "Pitanja" {
			t.Fatalf("lesson 03 should be the 'Ljudi oko mene' manifest, got title=%q manifest=%v", l.Title, l.Manifest)
		}
		if len(l.Steps) < 6 {
			t.Errorf("lesson 03: %d steps, want >= 6", len(l.Steps))
		}
		kinds := map[string]int{}
		for _, s := range l.Steps {
			kinds[s.Kind]++
		}
		if kinds["reading"] < 1 || kinds["checkpoint"] < 1 {
			t.Errorf("lesson 03 step kinds: %v", kinds)
		}
	}
	for _, id := range []string{"04", "05", "06"} {
		l := c.Lessons[id]
		if l.Planned || !l.Manifest {
			t.Errorf("lesson %s should be a non-planned manifest", id)
		}
		if len(c.Exercises[id]) < 5 {
			t.Errorf("lesson %s blocks = %d, want >= 5", id, len(c.Exercises[id]))
		}
		if len(l.Steps) < 6 {
			t.Errorf("lesson %s: %d steps, want >= 6", id, len(l.Steps))
		}
		kinds := map[string]int{}
		for _, s := range l.Steps {
			kinds[s.Kind]++
		}
		if kinds["checkpoint"] < 1 {
			t.Errorf("lesson %s: no checkpoint step (%v)", id, kinds)
		}
	}
	for _, id := range []string{"04", "05"} {
		if len(c.Lessons[id].Steps) < 9 {
			t.Errorf("lesson %s: %d steps, want >= 9 (broken into small pieces)", id, len(c.Lessons[id].Steps))
		}
	}
	for _, id := range []string{"01", "02", "03", "04", "05"} {
		l := c.Lessons[id]
		if l.Reading == "" || l.ReadingRU == "" {
			t.Errorf("lesson %s: want a reading block with translation", id)
		}
		if strings.Contains(l.Markdown, "<!-- reading") {
			t.Errorf("lesson %s: reading markers left in markdown", id)
		}
		if len(l.Steps) < 3 {
			t.Errorf("lesson %s: %d steps, want >= 3", id, len(l.Steps))
		}
		if len(l.Steps) > 0 && l.Steps[0].Kind != "teach" {
			t.Errorf("lesson %s: first step kind = %q, want teach", id, l.Steps[0].Kind)
		}

		listen := 0
		for _, b := range c.Exercises[id] {
			for _, e := range b.Exercises {
				if e.Type != "listen" {
					continue
				}
				listen++
				if e.Say == "" || len(e.Accept) == 0 {
					t.Errorf("lesson %s: listen %s missing say/accept", id, e.ID)
				}
				if e.Audio != e.ID+".mp3" {
					t.Errorf("lesson %s: listen %s has no audio (run scripts/tts.py)", id, e.ID)
				}
			}
		}
		if listen < 3 {
			t.Errorf("lesson %s: listen exercises = %d, want >= 3", id, listen)
		}
	}
	if c.Lessons["01"].Planned {
		t.Error("lesson 01 should have content")
	}
	if len(c.Phases) != 5 {
		t.Errorf("phases = %d, want 5", len(c.Phases))
	}
	if len(c.Lessons) < 28 {
		t.Errorf("lessons = %d, want >= 28", len(c.Lessons))
	}
	for _, id := range []string{"18", "24", "30"} {
		if !c.Lessons[id].Planned {
			t.Errorf("checkpoint lesson %s should be planned", id)
		}
	}
}
