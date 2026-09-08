package content

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestLessonLoadsSteps(t *testing.T) {
	c, err := Load("testdata/content")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	l := c.Lessons["90"]
	if l == nil {
		t.Fatal("lesson 90 missing")
	}
	if !l.Manifest {
		t.Error("lesson 90: Manifest = false, want true")
	}
	if len(l.Steps) != 4 {
		t.Fatalf("lesson 90: %d steps, want 4", len(l.Steps))
	}
	if l.Steps[0].Kind != "teach" || l.Steps[0].Markdown == "" {
		t.Errorf("step 0: kind=%q md-empty=%v", l.Steps[0].Kind, l.Steps[0].Markdown == "")
	}
	if l.Steps[1].Kind != "practice" || len(l.Steps[1].Exercises) != 2 {
		t.Errorf("step 1: kind=%q exercises=%d", l.Steps[1].Kind, len(l.Steps[1].Exercises))
	}
	r := l.Steps[2]
	if r.Kind != "reading" || r.Markdown != "Zdravo. Kako si?" || r.MarkdownRU != "Привет. Как дела?" {
		t.Errorf("step 2 reading: md=%q ru=%q", r.Markdown, r.MarkdownRU)
	}
	ch := l.Steps[1].Exercises[0]
	if ch.Type != "choice" || ch.Answer != "Da" || len(ch.Options) != 2 {
		t.Errorf("choice exercise not parsed: %+v", ch)
	}
	if l.Reading != "Zdravo. Kako si?" {
		t.Errorf("legacy-compat Lesson.Reading = %q", l.Reading)
	}
}

func TestManifestExercisesCollected(t *testing.T) {
	c, err := Load("testdata/content")
	if err != nil {
		t.Fatal(err)
	}
	blocks := c.Exercises["90"]
	if len(blocks) != 2 { // practice 90.2 + checkpoint 90.4
		t.Fatalf("lesson 90: %d exercise blocks, want 2", len(blocks))
	}
	if blocks[0].ID != "90.2" || blocks[1].ID != "90.4" {
		t.Errorf("block ids = %q, %q; want 90.2, 90.4", blocks[0].ID, blocks[1].ID)
	}
	if _, _, ok := findExerciseIn(c, "90", "90.2.1"); !ok {
		t.Error("findExerciseIn cannot locate manifest exercise 90.2.1")
	}
}

// findExerciseIn mirrors api.findExercise for test purposes.
func findExerciseIn(c *Course, lesson, exID string) (Exercise, string, bool) {
	for _, b := range c.Exercises[lesson] {
		for _, e := range b.Exercises {
			if e.ID == exID {
				return e, b.ID, true
			}
		}
	}
	return Exercise{}, "", false
}

func TestManifestRejectsBadSteps(t *testing.T) {
	cases := map[string]string{
		"teach with exercises": `lesson: "90"
title: t
steps:
  - id: "90.1"
    kind: teach
    md: "x.md"
    exercises:
      - {id: "90.1.1", type: translate, prompt: p, accept: ["a"]}
`,
		"practice without exercises": `lesson: "90"
title: t
steps:
  - id: "90.1"
    kind: practice
    title: p
`,
		"step id missing lesson prefix": `lesson: "90"
title: t
steps:
  - id: "1.1"
    kind: teach
    md: "x.md"
`,
		"unknown kind": `lesson: "90"
title: t
steps:
  - id: "90.1"
    kind: quiz
    md: "x.md"
`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "lessons"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "lessons", "90.yaml"), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			_ = os.WriteFile(filepath.Join(dir, "lessons", "x.md"), []byte("# x\n"), 0o644)
			if _, err := parseManifest(dir, "lessons/90.yaml"); err == nil {
				t.Fatalf("%s: expected error", name)
			}
		})
	}
}
