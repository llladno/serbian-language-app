package store

import (
	"testing"
	"time"
)

func TestRepeatCandidatesSeeLastAttemptAndTypo(t *testing.T) {
	_, u := newUser(t)
	u.AddAttempt(Attempt{"01-A-1", "01", "A", "x", false, false}, day0)                // wrong, never retried
	u.AddAttempt(Attempt{"01-A-2", "01", "A", "sin", true, true}, day0)                // right with a slip
	u.AddAttempt(Attempt{"01-A-3", "01", "A", "x", false, false}, day0)                // wrong once …
	u.AddAttempt(Attempt{"01-A-3", "01", "A", "ok", true, false}, day0.Add(time.Hour)) // … then fixed
	got, err := u.RepeatCandidates()
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]RepeatCandidate{}
	for _, c := range got {
		by[c.ExerciseID] = c
	}
	if len(by) != 3 {
		t.Fatalf("candidates = %+v", got)
	}
	if c := by["01-A-1"]; c.LastCorrect || c.Wrong != 1 || c.Total != 1 || c.Lesson != "01" {
		t.Errorf("01-A-1 = %+v", c)
	}
	if c := by["01-A-2"]; !c.LastCorrect || !c.LastTypo {
		t.Errorf("01-A-2 should be a typo: %+v", c)
	}
	if c := by["01-A-3"]; !c.LastCorrect || c.LastTypo || c.Wrong != 1 || c.Total != 2 {
		t.Errorf("01-A-3 = %+v", c)
	}
	if !by["01-A-3"].LastAt.Equal(day0.Add(time.Hour)) {
		t.Errorf("LastAt = %v, want the latest attempt", by["01-A-3"].LastAt)
	}
}

func TestRepeatSessionIsStoredOncePerDayAndKnowsWhenItIsDone(t *testing.T) {
	_, u := newUser(t)
	u.AddAttempt(Attempt{"01-A-1", "01", "A", "x", false, false}, day0)
	keys := []string{"01|01-A-1", "01|01-A-2"}
	s, err := u.StartRepeatSession("2026-10-23", keys, day0)
	if err != nil || s == nil {
		t.Fatalf("start: %v %v", s, err)
	}
	again, _ := u.StartRepeatSession("2026-10-23", []string{"02|other"}, day0.Add(time.Minute))
	if again.ID != s.ID || len(again.Keys) != 2 {
		t.Fatalf("second start must return the stored batch, got %+v", again)
	}
	if done, _ := u.RepeatSessionDone(*s); done {
		t.Fatal("nothing answered yet")
	}
	u.AddAttempt(Attempt{"01-A-1", "01", "A", "ok", true, false}, day0.Add(time.Hour))
	if done, _ := u.RepeatSessionDone(*s); done {
		t.Fatal("one of two answered")
	}
	if pending, _ := u.RepeatPending(*s); len(pending) != 1 || pending[0] != "01|01-A-2" {
		t.Fatalf("pending = %v, want only the exercise not answered yet", pending)
	}
	u.AddAttempt(Attempt{"01-A-2", "01", "A", "ok", true, false}, day0.Add(2*time.Hour))
	if done, _ := u.RepeatSessionDone(*s); !done {
		t.Fatal("both answered right after the start: done")
	}
	if got, _ := u.RepeatSession("2026-10-24"); got != nil {
		t.Fatalf("no session for another day: %+v", got)
	}
}

func TestRepeatSessionIgnoresAnswersBeforeItStarted(t *testing.T) {
	_, u := newUser(t)
	u.AddAttempt(Attempt{"01-A-1", "01", "A", "ok", true, false}, day0) // answered long before
	s, _ := u.StartRepeatSession("2026-10-23", []string{"01|01-A-1"}, day0.Add(48*time.Hour))
	if done, _ := u.RepeatSessionDone(*s); done {
		t.Fatal("an answer from before the batch started must not finish it")
	}
}

func TestTodayFollowsTheLearnersTimezone(t *testing.T) {
	s, u := newUser(t)
	if err := s.SetUserTimezone(u.user, "Pacific/Auckland"); err != nil {
		t.Fatal(err)
	}
	// 23:00 UTC on the 5th is already the 6th in Auckland.
	if got := u.Today(time.Date(2026, 10, 5, 23, 0, 0, 0, time.UTC)); got != "2026-10-06" {
		t.Errorf("Today = %q, want the local date 2026-10-06", got)
	}
}

func TestUnlimitedRepeatNeedsAYearOrAPaidLevel(t *testing.T) {
	s := newStore(t)
	free, _ := s.CreateUser("Бесплатный")
	if ok, err := s.HasUnlimitedRepeat(free, day0); err != nil || ok {
		t.Fatalf("a learner who paid nothing: unlimited = %v, %v", ok, err)
	}

	year, _ := s.CreateUser("Годовой")
	id := newPayment(t, s, year, TariffYear, "", 100000)
	if _, err := s.CompletePayment(id, day0); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.HasUnlimitedRepeat(year, day0.Add(24*time.Hour)); !ok {
		t.Error("a running year must lift the limit")
	}
	if ok, _ := s.HasUnlimitedRepeat(year, day0.AddDate(1, 0, 1)); ok {
		t.Error("an ended year must not")
	}

	level, _ := s.CreateUser("Уровневый")
	id = newPayment(t, s, level, TariffLevel, "2", 24900)
	if _, err := s.CompletePayment(id, day0); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.HasUnlimitedRepeat(level, day0.AddDate(2, 0, 0)); !ok {
		t.Error("a level paid for in rubles lifts the limit for good")
	}

	seeds, _ := s.CreateUser("Зёрнышки")
	// A level opened with зёрнышки is an entitlement, not a payment.
	if err := s.User(seeds).GrantEntitlement("phase_unlock", "2", 1, day0); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.HasUnlimitedRepeat(seeds, day0); ok {
		t.Error("a level bought with зёрнышки must not lift the limit")
	}
}
