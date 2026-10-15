package content

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
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
	if len(c.Lessons) != 78 {
		t.Errorf("lessons = %d, want 78", len(c.Lessons))
	}

	// "00".."59" are the fully authored lessons of levels 1-4.
	var authored []string
	for i := 0; i <= 59; i++ {
		authored = append(authored, fmt.Sprintf("%02d", i))
	}
	// Lessons whose first step is a reading (the opening half of a
	// "Провера" review lesson), so they don't have to start with "teach".
	readingFirst := map[string]bool{"14": true, "28": true, "32": true, "36": true, "41": true, "48": true, "58": true, "59": true}

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
	// 17 before the self-graded "free" exercises were removed: four checkpoints
	// (15.3, 29.4, 36.4, 41.5) held nothing else, so they went with them; lessons 48, 58 and 59 add three more.
	if kindTotals["checkpoint"] != 16 {
		t.Errorf("checkpoint steps across 00-59 = %d, want 16", kindTotals["checkpoint"])
	}
	if kindTotals["reading"] != 40 {
		t.Errorf("reading steps across 00-59 = %d, want 40", kindTotals["reading"])
	}
	if kindTotals["dialogue"] != 37 {
		t.Errorf("dialogue steps across 00-59 = %d, want 37", kindTotals["dialogue"])
	}
	if listen != 119 {
		t.Errorf("listen exercises across 00-59 = %d, want 119", listen)
	}

	if c.Lessons["00"].Planned {
		t.Error("lesson 00 should have content")
	}
	// A lesson is "planned" exactly when course.yaml gives it no file: once a
	// lesson of level 5 gets content it stops being planned, and one without
	// content must not pretend to have steps.
	for id, l := range c.Lessons {
		if l.Planned && len(l.Steps) > 0 {
			t.Errorf("lesson %s is planned but has %d steps", id, len(l.Steps))
		}
		if !l.Planned && len(l.Steps) == 0 {
			t.Errorf("lesson %s has a file but no steps", id)
		}
	}
}

// TestNoEmptyAudioClips: a zero-byte clip is what a synthesis request leaves
// behind when it fails half-way. The speaker button would play silence, and
// scripts/tts.py used to take the empty file for finished audio.
func TestNoEmptyAudioClips(t *testing.T) {
	root := "../../../content/audio"
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".mp3") {
			return err
		}
		if st, serr := d.Info(); serr != nil {
			return serr
		} else if st.Size() == 0 {
			t.Errorf("%s is empty (run scripts/tts.py --force --only <id>)", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
}

func TestClipPresentIgnoresEmptyFiles(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.mp3")
	full := filepath.Join(dir, "full.mp3")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if clipPresent(empty) {
		t.Error("an empty file must not count as audio")
	}
	if !clipPresent(full) {
		t.Error("a non-empty file must count as audio")
	}
	if clipPresent(filepath.Join(dir, "missing.mp3")) {
		t.Error("a missing file must not count as audio")
	}
}

// TestLevel5IsFullyAuthored: lessons 60-77 are real manifests with audio for
// every vocabulary word, dictation and dialogue line. Level 5 was a list of
// planned stubs until 2026-10-14; this keeps it from quietly turning back into
// one (or shipping silent buttons).
func TestLevel5IsFullyAuthored(t *testing.T) {
	c, err := Load("../../../content")
	if err != nil {
		t.Fatalf("real content: %v", err)
	}
	// Review lessons open with a reading, the rest with a teach step.
	readingFirst := map[string]bool{"67": true, "77": true}
	for i := 60; i <= 77; i++ {
		id := fmt.Sprintf("%02d", i)
		l := c.Lessons[id]
		if l == nil {
			t.Errorf("lesson %s missing", id)
			continue
		}
		if l.Planned || !l.Manifest {
			t.Errorf("lesson %s should be a non-planned manifest", id)
			continue
		}
		if len(l.Steps) < 5 {
			t.Errorf("lesson %s: %d steps, want >= 5", id, len(l.Steps))
			continue
		}
		first := l.Steps[0].Kind
		if readingFirst[id] && first != "reading" || !readingFirst[id] && first != "teach" {
			t.Errorf("lesson %s: first step kind = %q", id, first)
		}
		hasDialogue, hasListen := false, false
		for _, s := range l.Steps {
			if s.Kind == "dialogue" {
				hasDialogue = true
				for n, turn := range s.Turns {
					if turn.Audio == "" {
						t.Errorf("lesson %s step %s turn %d has no audio (run scripts/tts.py)", id, s.ID, n+1)
					}
				}
			}
		}
		for _, b := range c.Exercises[id] {
			for _, e := range b.Exercises {
				if e.Type == "listen" {
					hasListen = true
					if e.Audio != e.ID+".mp3" {
						t.Errorf("lesson %s: listen %s has no audio (run scripts/tts.py)", id, e.ID)
					}
				}
			}
		}
		if !hasDialogue || !hasListen {
			t.Errorf("lesson %s: dialogue=%v listen=%v, want both", id, hasDialogue, hasListen)
		}
	}
	for _, v := range c.Vocab {
		if n, err := strconv.Atoi(v.Lesson); err == nil && n >= 60 && n <= 77 && v.Audio == "" {
			t.Errorf("vocab %q (lesson %s) has no audio (run scripts/tts.py)", v.ID, v.Lesson)
		}
	}
}
