package store

import (
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/srs"
)

func mustOpen(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

var day0 = time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)

func TestEnsureCardsIdempotent(t *testing.T) {
	s := mustOpen(t)
	seeds := []CardSeed{{"vocab:zdravo", "vocab", "zdravo"}}
	if err := s.EnsureCards(seeds); err != nil {
		t.Fatal(err)
	}
	if err := s.EnsureCards(seeds); err != nil {
		t.Fatal(err)
	}
	total, _, err := s.CardStats()
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
}

func TestDueQueueLimitsNewAndOrdersOverdueFirst(t *testing.T) {
	s := mustOpen(t)
	s.EnsureCards([]CardSeed{
		{"vocab:a", "vocab", "a"},
		{"vocab:b", "vocab", "b"},
		{"vocab:c", "vocab", "c"},
	})
	// Promote 'a' to a review card due in the past.
	if _, err := s.db.Exec(`UPDATE srs_cards SET state='review', interval_days=3, due='2026-09-01' WHERE card_id='vocab:a'`); err != nil {
		t.Fatal(err)
	}
	q, err := s.DueQueue(day0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(q) != 2 {
		t.Fatalf("queue len = %d, want 2 (1 overdue + 1 new)", len(q))
	}
	if q[0].CardID != "vocab:a" {
		t.Errorf("first card = %s, want overdue vocab:a", q[0].CardID)
	}
	if q[1].State != srs.New {
		t.Errorf("second card state = %s, want new", q[1].State)
	}
}

func TestGradeCardPersistsAndLogsReview(t *testing.T) {
	s := mustOpen(t)
	s.EnsureCards([]CardSeed{{"vocab:x", "vocab", "x"}})
	c, err := s.GradeCard("vocab:x", srs.Good, day0)
	if err != nil {
		t.Fatal(err)
	}
	if c.State != srs.Review {
		t.Errorf("state = %s", c.State)
	}
	n, err := s.ReviewedToday(day0)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("reviewed today = %d, want 1", n)
	}
	// The card should no longer be due today.
	q, _ := s.DueQueue(day0, 0)
	if len(q) != 0 {
		t.Errorf("expected empty due queue, got %d", len(q))
	}
}

func TestAddAttemptAndWeakExercises(t *testing.T) {
	s := mustOpen(t)
	s.AddAttempt(Attempt{"02-C-1", "02", "C", "x", false}, day0)
	s.AddAttempt(Attempt{"02-C-1", "02", "C", "y", false}, day0)
	s.AddAttempt(Attempt{"02-C-1", "02", "C", "radi", true}, day0)
	s.AddAttempt(Attempt{"02-B-1", "02", "B", "ok", true}, day0)

	w, err := s.WeakExercises(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(w) != 1 {
		t.Fatalf("weak = %+v", w)
	}
	if w[0].ExerciseID != "02-C-1" || w[0].Wrong != 2 || w[0].Total != 3 {
		t.Errorf("weak[0] = %+v", w[0])
	}
}

func TestLessonStatusRoundTrip(t *testing.T) {
	s := mustOpen(t)
	if err := s.SetLessonStatus("01", "in_progress", day0); err != nil {
		t.Fatal(err)
	}
	if err := s.SetLessonStatus("01", "done", day0.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	m, err := s.LessonStatuses()
	if err != nil {
		t.Fatal(err)
	}
	if m["01"] != "done" {
		t.Errorf("statuses = %+v", m)
	}
	one, _ := s.LessonStatus("01")
	if one != "done" {
		t.Errorf("LessonStatus = %q", one)
	}
	if miss, _ := s.LessonStatus("99"); miss != "" {
		t.Errorf("unknown lesson status = %q, want empty", miss)
	}
}

func TestActivityByDay(t *testing.T) {
	s := mustOpen(t)
	s.EnsureCards([]CardSeed{{"vocab:x", "vocab", "x"}})
	s.GradeCard("vocab:x", srs.Good, day0)
	s.AddAttempt(Attempt{"01-A-1", "01", "A", "z", true}, day0)
	s.AddAttempt(Attempt{"01-A-2", "01", "A", "z", false}, day0.AddDate(0, 0, -3))

	acts, err := s.ActivityByDay(day0.AddDate(0, 0, -7))
	if err != nil {
		t.Fatal(err)
	}
	byDate := map[string]int{}
	for _, a := range acts {
		byDate[a.Date] = a.Count
	}
	if byDate[day0.Format("2006-01-02")] != 2 { // 1 review + 1 attempt
		t.Errorf("today count = %d, want 2 (%+v)", byDate[day0.Format("2006-01-02")], acts)
	}
	if byDate[day0.AddDate(0, 0, -3).Format("2006-01-02")] != 1 {
		t.Errorf("three days ago count wrong: %+v", acts)
	}
}

func TestStreakDays(t *testing.T) {
	s := mustOpen(t)
	s.EnsureCards([]CardSeed{{"vocab:x", "vocab", "x"}})
	// reviews today and yesterday -> streak 2; nothing two days ago -> stop.
	s.GradeCard("vocab:x", srs.Good, day0)
	if _, err := s.db.Exec(`INSERT INTO reviews (card_id, grade, reviewed_at) VALUES ('vocab:x', 2, '2026-09-05T09:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	streak, err := s.StreakDays(day0)
	if err != nil {
		t.Fatal(err)
	}
	if streak != 2 {
		t.Errorf("streak = %d, want 2", streak)
	}
	// A gap day resets the streak to 0.
	if got, _ := s.StreakDays(day0.AddDate(0, 0, 2)); got != 0 {
		t.Errorf("streak after gap = %d, want 0", got)
	}
}
