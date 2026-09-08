package content

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mustCopyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
	if err != nil {
		t.Fatalf("copy tree: %v", err)
	}
}

func TestWatchReloadsOnNestedFragmentChange(t *testing.T) {
	dir := t.TempDir()
	mustCopyTree(t, "testdata/content", dir)

	get, _, err := Watch(dir)
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	before := get().Lessons["90"].Steps[0].Markdown
	if before == "" {
		t.Fatal("fixture lesson 90 step 0 has no markdown")
	}

	frag := filepath.Join(dir, "lessons", "90", "1-intro.md")
	if err := os.WriteFile(frag, []byte("# Intro\n\nIzmenjeno. *(изменено)*\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if get().Lessons["90"].Steps[0].Markdown != before {
			return // reloaded
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("content did not reload after nested fragment change")
}
