package content

import (
	"strings"
	"testing"
)

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
	if len(c.Exercises["01"]) < 4 {
		t.Errorf("lesson 01 blocks = %d, want >= 4", len(c.Exercises["01"]))
	}
	if len(c.Exercises["02"]) < 5 {
		t.Errorf("lesson 02 blocks = %d, want >= 5", len(c.Exercises["02"]))
	}
	for _, id := range []string{"03", "04", "05"} {
		if c.Lessons[id].Planned {
			t.Errorf("lesson %s should have content", id)
		}
		if len(c.Exercises[id]) < 5 {
			t.Errorf("lesson %s blocks = %d, want >= 5", id, len(c.Exercises[id]))
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
	if !c.Lessons["06"].Planned {
		t.Error("lesson 06 should be planned")
	}
	if c.Lessons["01"].Planned {
		t.Error("lesson 01 should have content")
	}
	if len(c.Phases) != 3 {
		t.Errorf("phases = %d, want 3", len(c.Phases))
	}
}
