package content

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watch loads dir into an atomic snapshot and reloads it on file changes
// (300ms debounce). The returned get func always returns the last snapshot
// that parsed cleanly; stale reports whether the most recent reload failed.
func Watch(dir string) (get func() *Course, stale func() bool, err error) {
	first, err := Load(dir)
	if err != nil {
		return nil, nil, err
	}
	var snap atomic.Pointer[Course]
	var bad atomic.Bool
	snap.Store(first)

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}
	// Watch dir and every subdirectory (lessons/NN/ fragments live nested).
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			_ = w.Add(p)
		}
		return nil
	})

	go func() {
		var timer *time.Timer
		reload := func() {
			c, err := Load(dir)
			if err != nil {
				log.Printf("content reload failed: %v", err)
				bad.Store(true)
				return
			}
			snap.Store(c)
			bad.Store(false)
			log.Printf("content reloaded (%d lessons, %d vocab)", len(c.Lessons), len(c.Vocab))
		}
		for {
			select {
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				// A new subdirectory (e.g. lessons/NN/) must be watched too.
				if ev.Op&fsnotify.Create != 0 {
					if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
						_ = w.Add(ev.Name)
					}
				}
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(300*time.Millisecond, reload)
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				log.Printf("content watcher error: %v", err)
			}
		}
	}()

	return snap.Load, bad.Load, nil
}
