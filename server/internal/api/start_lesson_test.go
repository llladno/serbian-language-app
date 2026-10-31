package api

import (
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/content"
)

func TestSessionOffersFirstLessonUntilLearningStarts(t *testing.T) {
	var course func() *content.Course
	h, st, _ := newAuthAPI(t, func(d *Deps) { course = d.Course })
	uid := registerAndVerify(t, h, st, "alice@example.com", "password123", "Alice")
	want := course().Phases[0].Lessons[0]

	rr := anon(h, "POST", "/api/auth/login", loginBody("alice@example.com", "password123"))
	if rr.Code != 200 {
		t.Fatalf("login = %d %s", rr.Code, rr.Body)
	}
	if got := decodeBody[sessionUserDTO](t, rr).StartLesson; got != want {
		t.Fatalf("login start_lesson = %q, want %q for an account that has not started", got, want)
	}
	cookie := grabSessionCookie(t, rr)
	if got := decodeBody[sessionUserDTO](t, doCookie(h, cookie, "GET", "/api/auth/session", "")).StartLesson; got != want {
		t.Fatalf("session start_lesson = %q, want %q", got, want)
	}

	// Opening a lesson is enough: from then on the app opens on the course map.
	if err := st.User(uid).SetLessonStatus(want, "in_progress", time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := decodeBody[sessionUserDTO](t, doCookie(h, cookie, "GET", "/api/auth/session", "")).StartLesson; got != "" {
		t.Fatalf("start_lesson after opening a lesson = %q, want empty", got)
	}
}
