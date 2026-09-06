package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadValidFixture(t *testing.T) {
	c, err := Load("testdata/content")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Title != "Српски језик" {
		t.Errorf("title = %q", c.Title)
	}
	if len(c.Phases) != 1 || len(c.Phases[0].Lessons) != 2 {
		t.Fatalf("phases: %+v", c.Phases)
	}
	if c.Lessons["01"].Planned {
		t.Error("01 should not be planned")
	}
	if !c.Lessons["02"].Planned {
		t.Error("02 should be planned")
	}
	if !strings.Contains(c.Lessons["01"].Markdown, "svet") {
		t.Error("markdown not loaded")
	}
	if len(c.Exercises["01"]) != 1 {
		t.Fatalf("blocks: %d", len(c.Exercises["01"]))
	}
	if got := c.Exercises["01"][0].Exercises; len(got) != 3 {
		t.Fatalf("exercises: %d", len(got))
	}
	conj := c.Exercises["01"][0].Exercises[2]
	if len(conj.AcceptForms) != 6 || conj.AcceptForms[1][0] != "govoriš" {
		t.Errorf("conjugate accept: %+v", conj.AcceptForms)
	}
	if len(c.Vocab) != 2 || len(c.FalseFriends) != 1 {
		t.Errorf("vocab=%d ff=%d", len(c.Vocab), len(c.FalseFriends))
	}
}

func baseTree() map[string]string {
	return map[string]string{
		"course.yaml":        "title: x\nphases: []\nlessons:\n  \"01\": {title: t, subtitle: s, file: lessons/01.md}\n",
		"lessons/01.md":      "# t\n",
		"vocab.yaml":         "[]\n",
		"false-friends.yaml": "[]\n",
	}
}

func TestLoadRejectsExerciseForUnknownLesson(t *testing.T) {
	dir := t.TempDir()
	tree := baseTree()
	tree["exercises/99.yaml"] = "lesson: \"99\"\nblocks: []\n"
	writeTree(t, dir, tree)
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "99") {
		t.Fatalf("want error mentioning 99, got %v", err)
	}
}

func TestLoadRejectsMissingAccept(t *testing.T) {
	dir := t.TempDir()
	tree := baseTree()
	tree["exercises/01.yaml"] = "lesson: \"01\"\nblocks:\n  - id: A\n    title: A\n    exercises:\n      - id: \"01-A-1\"\n        type: translate\n        prompt: p\n        accept: []\n"
	writeTree(t, dir, tree)
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "01-A-1") {
		t.Fatalf("want error mentioning 01-A-1, got %v", err)
	}
}

func TestLoadRejectsDuplicateVocabID(t *testing.T) {
	dir := t.TempDir()
	tree := baseTree()
	tree["vocab.yaml"] = "- {id: x, latin: x, ru: a}\n- {id: x, latin: y, ru: b}\n"
	writeTree(t, dir, tree)
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), `"x"`) {
		t.Fatalf("want error mentioning x, got %v", err)
	}
}

func TestLoadRejectsConjugateArityMismatch(t *testing.T) {
	dir := t.TempDir()
	tree := baseTree()
	tree["exercises/01.yaml"] = "lesson: \"01\"\nblocks:\n  - id: A\n    title: A\n    exercises:\n      - id: \"01-A-1\"\n        type: conjugate\n        prompt: govoriti\n        forms: [ja, ti, on, mi, vi, oni]\n        accept:\n          - [govorim]\n          - [govoriš]\n"
	writeTree(t, dir, tree)
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "01-A-1") {
		t.Fatalf("want arity error mentioning 01-A-1, got %v", err)
	}
}

func TestLoadRejectsMissingMarkdownFile(t *testing.T) {
	dir := t.TempDir()
	tree := baseTree()
	delete(tree, "lessons/01.md")
	writeTree(t, dir, tree)
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "lessons/01.md") {
		t.Fatalf("want missing-file error, got %v", err)
	}
}
