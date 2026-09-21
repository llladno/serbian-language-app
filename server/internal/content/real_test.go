package content

import (
	"fmt"
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
//
// Levels 1-2 (lessons "00".."29") are authored as short manifests: each
// original topic was split into 2-3 lessons (see
// docs/superpowers/specs/2026-09-21-split-lessons-00-12-design.md), so a
// single lesson no longer necessarily carries its own reading/checkpoint —
// only the closing lesson of each original topic does. The checks below
// verify structure in aggregate across the whole authored range instead of
// per-lesson.
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
	if len(c.Phases) != 5 {
		t.Errorf("phases = %d, want 5", len(c.Phases))
	}
	if len(c.Lessons) != 59 {
		t.Errorf("lessons = %d, want 59", len(c.Lessons))
	}

	// "00".."29" are the fully authored lessons of levels 1-2.
	var authored []string
	for i := 0; i <= 29; i++ {
		authored = append(authored, fmt.Sprintf("%02d", i))
	}
	// Lessons whose first step is a reading (the opening half of a
	// "Провера" review lesson), so they don't have to start with "teach".
	readingFirst := map[string]bool{"14": true, "28": true}

	kindTotals := map[string]int{}
	listen := 0
	for _, id := range authored {
		l := c.Lessons[id]
		if l == nil {
			t.Errorf("lesson %s missing", id)
			continue
		}
		if l.Planned || !l.Manifest {
			t.Errorf("lesson %s should be a non-planned manifest", id)
		}
		if len(l.Steps) < 2 {
			t.Errorf("lesson %s: %d steps, want >= 2", id, len(l.Steps))
		}
		if len(l.Steps) > 0 {
			first := l.Steps[0].Kind
			if !readingFirst[id] && first != "teach" && first != "practice" {
				t.Errorf("lesson %s: first step kind = %q, want teach", id, first)
			}
		}
		for _, s := range l.Steps {
			kindTotals[s.Kind]++
		}
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
	}
	if kindTotals["checkpoint"] != 15 {
		t.Errorf("checkpoint steps across 00-29 = %d, want 15", kindTotals["checkpoint"])
	}
	if kindTotals["reading"] != 12 {
		t.Errorf("reading steps across 00-29 = %d, want 12", kindTotals["reading"])
	}
	if kindTotals["dialogue"] != 6 {
		t.Errorf("dialogue steps across 00-29 = %d, want 6", kindTotals["dialogue"])
	}
	if listen != 34 {
		t.Errorf("listen exercises across 00-29 = %d, want 34", listen)
	}

	if c.Lessons["00"].Planned {
		t.Error("lesson 00 should have content")
	}
	for _, id := range []string{"39", "49", "58"} {
		if !c.Lessons[id].Planned {
			t.Errorf("checkpoint lesson %s should be planned", id)
		}
	}
}
