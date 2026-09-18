package store

import (
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
)

func TestTelegramLinkedUsers(t *testing.T) {
	s := newStore(t)
	linked, err := s.CreateUser("Гриша")
	if err != nil {
		t.Fatal(err)
	}
	pending, err := s.CreateUser("Оля")
	if err != nil {
		t.Fatal(err)
	}
	none, err := s.CreateUser("Никто")
	if err != nil {
		t.Fatal(err)
	}
	_ = none

	must := func(in Identity) {
		t.Helper()
		if err := s.CreateIdentity(in); err != nil {
			t.Fatalf("CreateIdentity: %v", err)
		}
	}
	must(Identity{ID: auth.NewIdentityID(), UserID: linked, Provider: "telegram", ProviderUID: "555111", TgUsername: "grisha", CreatedAt: day0.Format(time.RFC3339)})
	must(Identity{ID: auth.NewIdentityID(), UserID: pending, Provider: "telegram", ProviderUID: "pending:olya", CreatedAt: day0.Format(time.RFC3339)})

	got, err := s.TelegramLinkedUsers()
	if err != nil {
		t.Fatalf("TelegramLinkedUsers: %v", err)
	}
	if len(got) != 1 || got[0].UserID != linked || got[0].ChatID != 555111 {
		t.Fatalf("TelegramLinkedUsers = %+v, want one entry {%s 555111}", got, linked)
	}
}

func TestLastActivityByUser(t *testing.T) {
	s, u := newUser(t)
	if _, err := s.CreateUser("Оля"); err != nil { // never active — must not appear
		t.Fatal(err)
	}

	if err := u.EnsureCards([]CardSeed{{CardID: "vocab:zdravo", Kind: "vocab", RefID: "zdravo"}}); err != nil {
		t.Fatal(err)
	}
	reviewAt := day0.Add(2 * time.Hour)
	if _, err := u.GradeCard("vocab:zdravo", 2, reviewAt); err != nil {
		t.Fatalf("GradeCard: %v", err)
	}
	attemptAt := day0.Add(5 * time.Hour) // later than the review — should win
	if err := u.AddAttempt(Attempt{ExerciseID: "e1", Lesson: "01", Block: "practice", Answer: "x", Correct: true}, attemptAt); err != nil {
		t.Fatalf("AddAttempt: %v", err)
	}

	got, err := s.LastActivityByUser()
	if err != nil {
		t.Fatalf("LastActivityByUser: %v", err)
	}
	last, ok := got[u.user]
	if !ok {
		t.Fatalf("no activity recorded for %s", u.user)
	}
	if !last.Equal(attemptAt.UTC().Truncate(time.Second)) {
		t.Fatalf("last activity = %v, want %v", last, attemptAt.UTC())
	}
	if len(got) != 1 {
		t.Fatalf("LastActivityByUser returned %d users, want 1 (idle account must be absent): %+v", len(got), got)
	}
}

func TestBotReminderState(t *testing.T) {
	s, u := newUser(t)

	st, err := s.BotReminderState(u.user)
	if err != nil {
		t.Fatalf("BotReminderState (no row): %v", err)
	}
	if st.AllDoneDate != "" || !st.InactiveSentAt.IsZero() {
		t.Fatalf("zero state = %+v, want empty", st)
	}

	if err := s.SetBotReminderAllDoneDate(u.user, "2026-09-17"); err != nil {
		t.Fatalf("SetBotReminderAllDoneDate: %v", err)
	}
	sentAt := day0.Add(30 * time.Hour)
	if err := s.SetBotReminderInactiveSentAt(u.user, sentAt); err != nil {
		t.Fatalf("SetBotReminderInactiveSentAt: %v", err)
	}

	st, err = s.BotReminderState(u.user)
	if err != nil {
		t.Fatalf("BotReminderState: %v", err)
	}
	if st.AllDoneDate != "2026-09-17" {
		t.Fatalf("AllDoneDate = %q, want 2026-09-17", st.AllDoneDate)
	}
	if !st.InactiveSentAt.Equal(sentAt.UTC()) {
		t.Fatalf("InactiveSentAt = %v, want %v", st.InactiveSentAt, sentAt.UTC())
	}

	// updating one field must not clobber the other
	if err := s.SetBotReminderAllDoneDate(u.user, "2026-09-18"); err != nil {
		t.Fatalf("SetBotReminderAllDoneDate (2nd): %v", err)
	}
	st, err = s.BotReminderState(u.user)
	if err != nil {
		t.Fatalf("BotReminderState: %v", err)
	}
	if st.AllDoneDate != "2026-09-18" {
		t.Fatalf("AllDoneDate after update = %q, want 2026-09-18", st.AllDoneDate)
	}
	if !st.InactiveSentAt.Equal(sentAt.UTC()) {
		t.Fatalf("InactiveSentAt clobbered by all-done update: %v, want %v", st.InactiveSentAt, sentAt.UTC())
	}
}
