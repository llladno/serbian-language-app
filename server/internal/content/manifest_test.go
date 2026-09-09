package content

import (
	"os"
	"path/filepath"
	"strings"
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

func writeDialogueFixture(t *testing.T, dir string) {
	t.Helper()
	body := `lesson: "05"
title: "Куповина"
steps:
  - id: "05.9"
    kind: dialogue
    title: "У пекари"
    scene: "Ты зашёл в пекару."
    turns:
      - who: npc
        sr: "Izvolite?"
        ru: "Слушаю вас?"
      - who: me
        sr: "Jedan hleb, molim."
        ru: "Один хлеб, пожалуйста."
        exercise:
          id: "05.9.1"
          type: choice
          prompt: "Попроси хлеб"
          options: ["Jedan hleb, molim.", "Jedan hleb, hvala."]
          answer: "Jedan hleb, molim."
      - who: npc
        sr: "Hvala."
        ru: "Спасибо."
`
	writeLessonYAML(t, dir, body)
}

// writeLessonYAML drops a lessons/05.yaml manifest into an empty content dir.
func writeLessonYAML(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "lessons"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lessons", "05.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseDialogueStep(t *testing.T) {
	dir := t.TempDir()
	writeDialogueFixture(t, dir)

	l, err := parseManifest(dir, "lessons/05.yaml")
	if err != nil {
		t.Fatal(err)
	}
	s := l.Steps[0]
	if s.Kind != "dialogue" || s.Scene == "" || s.Voice != "f" {
		t.Fatalf("bad step: kind=%q scene=%q voice=%q", s.Kind, s.Scene, s.Voice)
	}
	if len(s.Turns) != 3 {
		t.Fatalf("want 3 turns, got %d", len(s.Turns))
	}
	if s.Turns[0].Who != "npc" || s.Turns[0].Exercise != nil {
		t.Error("turn 1 should be a plain npc line")
	}
	if s.Turns[1].Exercise == nil || s.Turns[1].Exercise.ID != "05.9.1" {
		t.Fatal("turn 2 should carry exercise 05.9.1")
	}
	if len(s.Exercises) != 1 || s.Exercises[0].ID != "05.9.1" {
		t.Fatalf("want exercise 05.9.1 in Step.Exercises, got %+v", s.Exercises)
	}
	if s.Turns[1].Exercise != &s.Exercises[0] {
		t.Error("Turn.Exercise must point into Step.Exercises")
	}
}

func TestDialogueValidation(t *testing.T) {
	cases := []struct{ name, turns, want string }{
		{"npc with exercise", `
      - who: npc
        sr: "Izvolite?"
        exercise:
          id: "05.9.1"
          type: choice
          prompt: "x"
          options: ["a", "b"]
          answer: "a"`, "cannot have an exercise"},
		{"me without exercise", `
      - who: me
        sr: "Hvala."`, "needs an exercise"},
		{"unknown who", `
      - who: waiter
        sr: "Izvolite?"`, "who must be npc or me"},
		{"forbidden type", `
      - who: me
        sr: "Hvala."
        exercise:
          id: "05.9.1"
          type: free
          prompt: "x"`, "not allowed in a dialogue"},
		{"line differs from answer", `
      - who: me
        sr: "Dobar dan."
        exercise:
          id: "05.9.1"
          type: choice
          prompt: "x"
          options: ["Hvala.", "Molim."]
          answer: "Hvala."`, "must equal the choice answer"},
		{"line outside accept", `
      - who: me
        sr: "Dobar dan."
        exercise:
          id: "05.9.1"
          type: translate
          prompt: "x"
          accept: ["Hvala."]`, "is not among accept"},
		{"blank not in line", `
      - who: me
        sr: "Ja sam iz Rusije."
        exercise:
          id: "05.9.1"
          type: fill_blank
          prompt: "Ja ___ iz Rusije."
          accept: ["jesam"]`, "is not part of sr"},
		{"no me turn", `
      - who: npc
        sr: "Izvolite?"`, "at least one me turn"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeLessonYAML(t, dir, `lesson: "05"
title: "Куповина"
steps:
  - id: "05.9"
    kind: dialogue
    title: "У пекари"
    scene: "Ты зашёл в пекару."
    turns:`+tc.turns+"\n")
			_, err := parseManifest(dir, "lessons/05.yaml")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestDialogueRequiresScene(t *testing.T) {
	dir := t.TempDir()
	writeLessonYAML(t, dir, `lesson: "05"
title: "Куповина"
steps:
  - id: "05.9"
    kind: dialogue
    title: "У пекари"
    turns:
      - who: npc
        sr: "Izvolite?"
`)
	if _, err := parseManifest(dir, "lessons/05.yaml"); err == nil || !strings.Contains(err.Error(), "needs a scene") {
		t.Fatalf("want scene error, got %v", err)
	}
}

func TestDialogueTurnAudio(t *testing.T) {
	dir := t.TempDir()
	writeDialogueFixture(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, "audio"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "audio", "05.9-t1.mp3"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	l, err := parseManifest(dir, "lessons/05.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got := l.Steps[0].Turns[0].Audio; got != "05.9-t1.mp3" {
		t.Errorf("turn 1 audio = %q, want 05.9-t1.mp3", got)
	}
	if got := l.Steps[0].Turns[1].Audio; got != "" {
		t.Errorf("turn 2 has no clip on disk, got audio %q", got)
	}
}
