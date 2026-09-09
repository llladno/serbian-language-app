package store

import (
	"database/sql"
	"errors"
	"testing"
	"time"
)

func mkUser(t *testing.T, s *Store, name string) string {
	t.Helper()
	id, err := s.CreateUser(name)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestSessionRoundTrip(t *testing.T) {
	s := newStore(t)
	uid := mkUser(t, s, "Гриша")

	if err := s.CreateSession("hash-1", uid, "Firefox/1.0", day0, day0.Add(sessionSlide)); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	got, err := s.SessionByHash("hash-1", day0.Add(time.Hour))
	if err != nil {
		t.Fatalf("SessionByHash: %v", err)
	}
	if got.UserID != uid || got.UserAgent != "Firefox/1.0" || got.TokenHash != "hash-1" {
		t.Errorf("session = %+v", got)
	}

	if _, err := s.SessionByHash("unknown", day0); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("unknown hash err = %v, want sql.ErrNoRows", err)
	}
}

func TestSessionExpiry(t *testing.T) {
	s := newStore(t)
	uid := mkUser(t, s, "Гриша")

	if err := s.CreateSession("hash-exp", uid, "", day0, day0.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SessionByHash("hash-exp", day0); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expired session err = %v, want sql.ErrNoRows", err)
	}
}

func TestTouchSessionCap(t *testing.T) {
	s := newStore(t)
	uid := mkUser(t, s, "Гриша")

	if err := s.CreateSession("hash-old", uid, "", day0, day0.Add(sessionSlide)); err != nil {
		t.Fatal(err)
	}
	// Backdate creation so the 90-day hard cap bites.
	created := day0.Add(-100 * 24 * time.Hour)
	if _, err := s.db.Exec(`UPDATE sessions SET created_at = ? WHERE token_hash = ?`,
		created.UTC().Format(time.RFC3339), "hash-old"); err != nil {
		t.Fatal(err)
	}

	if err := s.TouchSession("hash-old", day0); err != nil {
		t.Fatalf("TouchSession: %v", err)
	}

	var expStr, seenStr string
	if err := s.db.QueryRow(`SELECT expires_at, last_seen_at FROM sessions WHERE token_hash = ?`, "hash-old").
		Scan(&expStr, &seenStr); err != nil {
		t.Fatal(err)
	}
	exp, _ := time.Parse(time.RFC3339, expStr)
	hardCap := created.Add(sessionMaxAge)
	if exp.After(hardCap.Add(time.Second)) {
		t.Errorf("expires_at %s past hard cap %s", exp, hardCap)
	}
	if exp.Before(hardCap.Add(-time.Second)) {
		t.Errorf("expires_at %s well short of hard cap %s (want it pinned there)", exp, hardCap)
	}
	if seen, _ := time.Parse(time.RFC3339, seenStr); !seen.Equal(day0) {
		t.Errorf("last_seen_at = %s, want %s", seen, day0)
	}
}

func TestTouchSessionSlides(t *testing.T) {
	s := newStore(t)
	uid := mkUser(t, s, "Гриша")

	if err := s.CreateSession("hash-fresh", uid, "", day0, day0.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	touchAt := day0.Add(30 * time.Minute) // still live (expires day0+1h)
	if err := s.TouchSession("hash-fresh", touchAt); err != nil {
		t.Fatal(err)
	}
	got, err := s.SessionByHash("hash-fresh", day0.Add(10*24*time.Hour))
	if err != nil {
		t.Fatalf("session should still be live after a touch: %v", err)
	}
	exp, _ := time.Parse(time.RFC3339, got.ExpiresAt)
	if want := touchAt.Add(sessionSlide); !exp.Equal(want) {
		t.Errorf("expires_at = %s, want slid to %s", exp, want)
	}
}

// TestTouchSessionExpiredNoop covers the liveness guard: touching a session
// whose expires_at is already in the past must not slide it back to life.
func TestTouchSessionExpiredNoop(t *testing.T) {
	s := newStore(t)
	uid := mkUser(t, s, "Гриша")

	if err := s.CreateSession("hash-dead", uid, "", day0, day0.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.TouchSession("hash-dead", day0); err != nil {
		t.Fatalf("TouchSession on expired session should be a no-op nil, got %v", err)
	}
	// expires_at untouched, session still dead
	var expStr string
	if err := s.db.QueryRow(`SELECT expires_at FROM sessions WHERE token_hash = ?`, "hash-dead").
		Scan(&expStr); err != nil {
		t.Fatal(err)
	}
	if exp, _ := time.Parse(time.RFC3339, expStr); !exp.Equal(day0.Add(-time.Hour)) {
		t.Errorf("expires_at = %s, want unchanged %s", exp, day0.Add(-time.Hour))
	}
	if _, err := s.SessionByHash("hash-dead", day0); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expired session revived: err = %v, want sql.ErrNoRows", err)
	}

	// an unknown token hash is likewise a silent no-op
	if err := s.TouchSession("no-such-hash", day0); err != nil {
		t.Errorf("TouchSession on unknown hash = %v, want nil", err)
	}
}

func TestDeleteUserSessionsExcept(t *testing.T) {
	s := newStore(t)
	uid := mkUser(t, s, "Гриша")
	other := mkUser(t, s, "Оля")

	for _, h := range []string{"a", "b", "c"} {
		if err := s.CreateSession(h, uid, "", day0, day0.Add(sessionSlide)); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.CreateSession("z", other, "", day0, day0.Add(sessionSlide)); err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteUserSessionsExcept(uid, "b"); err != nil {
		t.Fatalf("DeleteUserSessionsExcept: %v", err)
	}
	list, err := s.ListUserSessions(uid)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].TokenHash != "b" {
		t.Errorf("remaining sessions = %+v, want just [b]", list)
	}
	if otherList, err := s.ListUserSessions(other); err != nil || len(otherList) != 1 {
		t.Errorf("other user's session touched: %+v %v", otherList, err)
	}
}

func TestDeleteExpiredSessions(t *testing.T) {
	s := newStore(t)
	uid := mkUser(t, s, "Гриша")

	live := []string{"live-1", "live-2"}
	dead := []string{"dead-1", "dead-2"}
	for _, h := range live {
		if err := s.CreateSession(h, uid, "", day0, day0.Add(sessionSlide)); err != nil {
			t.Fatal(err)
		}
	}
	for _, h := range dead {
		if err := s.CreateSession(h, uid, "", day0, day0.Add(-time.Hour)); err != nil {
			t.Fatal(err)
		}
	}

	n, err := s.DeleteExpiredSessions(day0)
	if err != nil {
		t.Fatalf("DeleteExpiredSessions: %v", err)
	}
	if n != 2 {
		t.Errorf("swept %d, want 2", n)
	}
	list, err := s.ListUserSessions(uid)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("remaining = %d, want 2 live", len(list))
	}
}

func TestDeleteSessionAndDeleteUserSessions(t *testing.T) {
	s := newStore(t)
	uid := mkUser(t, s, "Гриша")
	for _, h := range []string{"s1", "s2"} {
		if err := s.CreateSession(h, uid, "", day0, day0.Add(sessionSlide)); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.DeleteSession("s1"); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.ListUserSessions(uid); len(list) != 1 {
		t.Errorf("after DeleteSession: %d, want 1", len(list))
	}
	if err := s.DeleteUserSessions(uid); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.ListUserSessions(uid); len(list) != 0 {
		t.Errorf("after DeleteUserSessions: %d, want 0", len(list))
	}
}
