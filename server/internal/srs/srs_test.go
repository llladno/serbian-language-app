package srs

import (
	"testing"
	"time"
)

const day = 24 * time.Hour

var now = time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

func TestNewCardGoodThenGood(t *testing.T) {
	c := Schedule(NewCard(), Good, now)
	if c.IntervalDays != 1 || c.State != Review {
		t.Fatalf("after first good: %+v", c)
	}
	c = Schedule(c, Good, now.Add(day))
	if c.IntervalDays != 3 { // round(1 * 2.5) = 3
		t.Errorf("interval = %d, want 3", c.IntervalDays)
	}
}

func TestNewCardEasy(t *testing.T) {
	c := Schedule(NewCard(), Easy, now)
	if c.IntervalDays != 4 || c.State != Review {
		t.Errorf("%+v", c)
	}
}

func TestNewCardAgain(t *testing.T) {
	c := Schedule(NewCard(), Again, now)
	if c.State != Learning {
		t.Errorf("state = %s", c.State)
	}
	if c.Due != now {
		t.Errorf("due = %v, want %v", c.Due, now)
	}
}

func TestReviewAgainIsLapse(t *testing.T) {
	c := Card{Ease: 2.5, IntervalDays: 10, Reps: 3, State: Review}
	c = Schedule(c, Again, now)
	if c.Lapses != 1 || c.IntervalDays != 1 || c.State != Learning {
		t.Fatalf("%+v", c)
	}
	if c.Ease > 2.31 {
		t.Errorf("ease not decremented: %v", c.Ease)
	}
}

func TestReviewHardGoodEasy(t *testing.T) {
	base := Card{Ease: 2.5, IntervalDays: 10, Reps: 1, State: Review}

	h := Schedule(base, Hard, now)
	if h.IntervalDays != 12 || !approx(h.Ease, 2.35) {
		t.Errorf("hard: %+v", h)
	}
	g := Schedule(base, Good, now)
	if g.IntervalDays != 25 || !approx(g.Ease, 2.5) {
		t.Errorf("good: %+v", g)
	}
	e := Schedule(base, Easy, now)
	if e.IntervalDays != 34 || !approx(e.Ease, 2.65) { // round(10 * 2.65 * 1.3)
		t.Errorf("easy: %+v", e)
	}
}

func TestEaseFloorAndIntervalCap(t *testing.T) {
	c := Card{Ease: 1.3, IntervalDays: 300, Reps: 9, State: Review}
	c = Schedule(c, Again, now)
	if c.Ease < 1.3 {
		t.Errorf("ease below floor: %v", c.Ease)
	}
	c2 := Card{Ease: 3.0, IntervalDays: 300, Reps: 9, State: Review}
	c2 = Schedule(c2, Easy, now)
	if c2.IntervalDays > 365 {
		t.Errorf("interval above cap: %d", c2.IntervalDays)
	}
}

func TestDueIsIntervalDaysAhead(t *testing.T) {
	c := Schedule(Card{Ease: 2.5, IntervalDays: 2, Reps: 1, State: Review}, Good, now)
	want := addDays(now, c.IntervalDays)
	if !c.Due.Equal(want) {
		t.Errorf("due = %v, want %v", c.Due, want)
	}
}

func approx(a, b float64) bool {
	d := a - b
	return d < 1e-9 && d > -1e-9
}
