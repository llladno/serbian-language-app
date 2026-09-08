package content

import "testing"

func TestDifficultyOrder(t *testing.T) {
	bad := Step{ID: "x.2", Kind: "practice", Exercises: []Exercise{
		{ID: "x.2.1", Type: "translate"},
		{ID: "x.2.2", Type: "choice"},
	}}
	if err := checkDifficultyOrder("x.yaml", bad); err == nil {
		t.Fatal("translate before choice should be rejected")
	}

	bad.Mixed = true
	if err := checkDifficultyOrder("x.yaml", bad); err != nil {
		t.Fatalf("mixed step should pass: %v", err)
	}

	bad.Mixed = false
	bad.Kind = "checkpoint"
	if err := checkDifficultyOrder("x.yaml", bad); err != nil {
		t.Fatalf("checkpoint should be exempt: %v", err)
	}

	good := Step{ID: "x.3", Kind: "practice", Exercises: []Exercise{
		{ID: "x.3.1", Type: "choice"},
		{ID: "x.3.2", Type: "fill_blank"},
		{ID: "x.3.3", Type: "word_bank"},
		{ID: "x.3.4", Type: "translate"},
	}}
	if err := checkDifficultyOrder("x.yaml", good); err != nil {
		t.Fatalf("ascending step should pass: %v", err)
	}
}

func TestValidateWordBank(t *testing.T) {
	if err := validateWordBank([]string{"Ja", "sam"}, []string{"Ja sam Ana"}); err == nil {
		t.Fatal("bank missing 'Ana' should be rejected")
	}
	if err := validateWordBank([]string{"Ja", "sam", "Ana", "Marko"}, []string{"Ja sam Ana"}); err != nil {
		t.Fatalf("bank covering the answer (with a distractor) should pass: %v", err)
	}
}

func TestManifestRejectsDifficultyRegression(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"lessons/90.yaml": `lesson: "90"
title: t
steps:
  - id: "90.1"
    kind: practice
    exercises:
      - {id: "90.1.1", type: translate, prompt: p, accept: ["a"]}
      - {id: "90.1.2", type: choice, prompt: p, options: ["a","b"], answer: "a"}
`,
	})
	if _, err := parseManifest(dir, "lessons/90.yaml"); err == nil {
		t.Fatal("manifest with a difficulty regression should fail to parse")
	}
}
