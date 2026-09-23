package store

import (
	"testing"
	"time"
)

func TestListNotificationsForUserOnlyOwnRows(t *testing.T) {
	s := newStore(t)
	if _, err := s.SeedNotificationForTest("user-a", "for A", day0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SeedNotificationForTest("user-b", "for B", day0); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListNotificationsForUser("user-a", day0)
	if err != nil {
		t.Fatalf("ListNotificationsForUser: %v", err)
	}
	if len(got) != 1 || got[0].Text != "for A" {
		t.Fatalf("ListNotificationsForUser(user-a) = %+v, want just \"for A\"", got)
	}
}

func TestListNotificationsForUserOrdersNewestFirst(t *testing.T) {
	s := newStore(t)
	if _, err := s.SeedNotificationForTest("user-a", "older", day0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SeedNotificationForTest("user-a", "newer", day0.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListNotificationsForUser("user-a", day0.Add(time.Hour))
	if err != nil {
		t.Fatalf("ListNotificationsForUser: %v", err)
	}
	if len(got) != 2 || got[0].Text != "newer" || got[1].Text != "older" {
		t.Fatalf("ListNotificationsForUser order = %+v, want [newer, older]", got)
	}
}

func TestListNotificationsForUserFiltersReadPastRetention(t *testing.T) {
	s := newStore(t)
	readLongAgo, err := s.SeedNotificationForTest("user-a", "read long ago", day0)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SeedNotificationReadAtForTest(readLongAgo, "user-a", day0); err != nil {
		t.Fatal(err)
	}
	readRecently, err := s.SeedNotificationForTest("user-a", "read recently", day0)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SeedNotificationReadAtForTest(readRecently, "user-a", day0.Add(29*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SeedNotificationForTest("user-a", "still unread", day0); err != nil {
		t.Fatal(err)
	}

	now := day0.Add(35 * 24 * time.Hour) // read_at cutoff = now - 30d = day0 + 5d
	got, err := s.ListNotificationsForUser("user-a", now)
	if err != nil {
		t.Fatalf("ListNotificationsForUser: %v", err)
	}
	texts := map[string]bool{}
	for _, n := range got {
		texts[n.Text] = true
	}
	if texts["read long ago"] {
		t.Errorf("got %+v, \"read long ago\" (read 35 days before now) should have expired", got)
	}
	if !texts["read recently"] {
		t.Errorf("got %+v, \"read recently\" (read 6 days before now) should still be present", got)
	}
	if !texts["still unread"] {
		t.Errorf("got %+v, unread notifications should never expire", got)
	}
}

func TestMarkNotificationsReadIsIdempotentAndScopedToUser(t *testing.T) {
	s := newStore(t)
	if _, err := s.SeedNotificationForTest("user-a", "for A", day0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SeedNotificationForTest("user-b", "for B", day0); err != nil {
		t.Fatal(err)
	}

	if err := s.MarkNotificationsRead("user-a", day0.Add(time.Hour)); err != nil {
		t.Fatalf("MarkNotificationsRead: %v", err)
	}
	gotA, err := s.ListNotificationsForUser("user-a", day0.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(gotA) != 1 || !gotA[0].Read {
		t.Fatalf("user-a's notification = %+v, want Read=true", gotA)
	}
	gotB, err := s.ListNotificationsForUser("user-b", day0.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(gotB) != 1 || gotB[0].Read {
		t.Fatalf("user-b's notification = %+v, want Read=false (mark-read must not leak across users)", gotB)
	}

	// Idempotent: calling again must not error or panic on the zero-row update.
	if err := s.MarkNotificationsRead("user-a", day0.Add(2*time.Hour)); err != nil {
		t.Fatalf("MarkNotificationsRead (second call): %v", err)
	}
}
