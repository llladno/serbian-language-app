package store

import (
	"strconv"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/srs"
)

// belgrade is the location the seeds default to; tests that don't care about
// timezones use it so day boundaries are predictable.
func belgrade(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Belgrade")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	return loc
}

// answer solves one exercise for the first time, which is the only kind of
// attempt that counts as an action.
func answer(t *testing.T, u *UserStore, exID string, correct bool, at time.Time) {
	t.Helper()
	if err := u.AddAttempt(Attempt{
		ExerciseID: exID, Lesson: "01", Block: "a", Answer: "x", Correct: correct,
	}, at); err != nil {
		t.Fatalf("attempt %s: %v", exID, err)
	}
}

func TestDayCountsOnlyFirstAttemptsAndAllReviews(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Актив")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)

	answer(t, u, "01.1", true, at)
	answer(t, u, "01.1", true, at) // same exercise again: must not count
	answer(t, u, "01.2", false, at)

	// Every SRS review counts, including repeated reviews of the same card:
	// the scheduler decides when a card is due, so they cannot be farmed.
	if err := u.EnsureCards([]CardSeed{{CardID: "vocab:x", Kind: "vocab", RefID: "x"}}); err != nil {
		t.Fatalf("seed card: %v", err)
	}
	if _, err := u.GradeCard("vocab:x", srs.Good, at); err != nil {
		t.Fatalf("grade: %v", err)
	}
	if _, err := u.GradeCard("vocab:x", srs.Good, at); err != nil {
		t.Fatalf("grade again: %v", err)
	}

	var actions, goal int
	err := s.db.QueryRow(`SELECT actions, goal FROM user_daily_activity WHERE user_id = ? AND day = ?`,
		id, "2026-09-29").Scan(&actions, &goal)
	if err != nil {
		t.Fatalf("activity row: %v", err)
	}
	if actions != 4 {
		t.Fatalf("actions = %d, want 4 (two first attempts + two reviews; the repeated attempt must not count)", actions)
	}
	if goal != 10 {
		t.Fatalf("goal = %d, want the seeded 10", goal)
	}
}

func TestDailyDripPaysOnceWhenTheGoalIsMet(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Капля")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)

	for i := 0; i < 9; i++ {
		answer(t, u, "01."+strconv.Itoa(i), true, at)
	}
	if bal, _ := s.Balance(id); bal != 0 {
		t.Fatalf("balance = %d before the goal is met, want 0", bal)
	}

	answer(t, u, "01.9", true, at) // tenth action meets the goal
	if bal, _ := s.Balance(id); bal != 1 {
		t.Fatalf("balance = %d after meeting the goal, want 1", bal)
	}

	// More actions the same day must not pay again.
	answer(t, u, "01.10", true, at)
	answer(t, u, "01.11", true, at)
	if bal, _ := s.Balance(id); bal != 1 {
		t.Fatalf("balance = %d after extra actions, want 1", bal)
	}
}

func TestStreakCountsConsecutiveDaysAndRepairs(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Стрик")
	u := s.User(id)

	// Three consecutive days that met the goal, then a gap, then today.
	days := []string{"2026-09-25", "2026-09-26", "2026-09-27", "2026-09-29"}
	for _, d := range days {
		if _, err := s.db.Exec(`INSERT INTO user_daily_activity (user_id, day, actions, goal) VALUES (?, ?, ?, ?)`,
			id, d, 10, 10); err != nil {
			t.Fatalf("seed %s: %v", d, err)
		}
	}
	now := time.Date(2026, 9, 29, 20, 0, 0, 0, loc)

	got, err := u.StreakDays(now)
	if err != nil {
		t.Fatalf("streak: %v", err)
	}
	if got != 1 {
		t.Fatalf("streak = %d, want 1 (the 28th is missing)", got)
	}

	if err := u.RepairStreak("2026-09-28", now); err != nil {
		t.Fatalf("repair: %v", err)
	}
	got, err = u.StreakDays(now)
	if err != nil {
		t.Fatalf("streak after repair: %v", err)
	}
	if got != 5 {
		t.Fatalf("streak after repair = %d, want 5", got)
	}
}

func TestRepairOutsideTheWindowIsRejected(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Поздно")
	u := s.User(id)
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, loc)

	// The 25th ended four days ago; the window is 48 hours.
	if err := u.RepairStreak("2026-09-25", now); err == nil {
		t.Fatal("a repair four days late was accepted; want an error")
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM streak_repairs WHERE user_id = ?`, id).Scan(&n); err != nil {
		t.Fatalf("count repairs: %v", err)
	}
	if n != 0 {
		t.Fatalf("%d repair rows written by a rejected repair, want 0", n)
	}

	// A future day is not repairable either.
	if err := u.RepairStreak("2026-09-30", now); err == nil {
		t.Fatal("a future day was accepted; want an error")
	}
}

func TestDayUnderGoalDoesNotExtendTheStreak(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Недобор")
	u := s.User(id)
	if _, err := s.db.Exec(`INSERT INTO user_daily_activity (user_id, day, actions, goal) VALUES (?, ?, ?, ?)`,
		id, "2026-09-29", 9, 10); err != nil {
		t.Fatalf("seed: %v", err)
	}
	now := time.Date(2026, 9, 29, 20, 0, 0, 0, loc)
	if got, _ := u.StreakDays(now); got != 0 {
		t.Fatalf("streak = %d, want 0 (9 of 10 actions)", got)
	}
}

func TestDayIsStampedInTheUsersTimezone(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Владивосток")
	if err := s.SetUserTimezone(id, "Asia/Vladivostok"); err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	u := s.User(id)

	// 23:30 UTC on the 29th is already 09:30 on the 30th in Vladivostok.
	at := time.Date(2026, 9, 29, 23, 30, 0, 0, time.UTC)
	answer(t, u, "01.1", true, at)

	var day string
	if err := s.db.QueryRow(`SELECT day FROM user_daily_activity WHERE user_id = ?`, id).Scan(&day); err != nil {
		t.Fatalf("activity row: %v", err)
	}
	if day != "2026-09-30" {
		t.Fatalf("day = %q, want 2026-09-30", day)
	}
}

func TestChangingTimezoneDoesNotRewritePastDays(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Переезд")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 23, 30, 0, 0, time.UTC)
	answer(t, u, "01.1", true, at)

	if err := s.SetUserTimezone(id, "Asia/Vladivostok"); err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	var day string
	if err := s.db.QueryRow(`SELECT day FROM user_daily_activity WHERE user_id = ?`, id).Scan(&day); err != nil {
		t.Fatalf("activity row: %v", err)
	}
	if day != "2026-09-30" {
		t.Fatalf("day = %q — the already-stamped row moved when the timezone changed", day)
	}
	_ = u
}

func TestStreakSurvivesUntilTheFirstActionOfTheDay(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Полночь")
	u := s.User(id)

	// Thirty consecutive active days ending on the 28th.
	last := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 30; i++ {
		d := last.AddDate(0, 0, -i).Format(dateFmt)
		if _, err := s.db.Exec(`INSERT INTO user_daily_activity (user_id, day, actions, goal) VALUES (?, ?, 10, 10)`,
			id, d); err != nil {
			t.Fatalf("seed %s: %v", d, err)
		}
	}

	// One minute into the 29th, nothing done yet: still 30, not 0.
	justAfterMidnight := time.Date(2026, 9, 29, 0, 1, 0, 0, loc)
	if got, _ := u.StreakDays(justAfterMidnight); got != 30 {
		t.Fatalf("streak at 00:01 = %d, want 30", got)
	}
	// A whole day with nothing done finally breaks it.
	if got, _ := u.StreakDays(time.Date(2026, 9, 30, 0, 1, 0, 0, loc)); got != 0 {
		t.Fatalf("streak a full day later = %d, want 0", got)
	}
}

func TestStreakDayBoundaryFollowsTheUsersTimezone(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Граница")
	if err := s.SetUserTimezone(id, "Asia/Vladivostok"); err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	u := s.User(id)
	if _, err := s.db.Exec(`INSERT INTO user_daily_activity (user_id, day, actions, goal) VALUES (?, '2026-09-30', 10, 10)`, id); err != nil {
		t.Fatal(err)
	}
	// 23:30 UTC on the 29th is the 30th in Vladivostok: that day is today, so
	// the streak is 1. Read in UTC it would be "yesterday" of the 30th's end.
	if got, _ := u.StreakDays(time.Date(2026, 9, 29, 23, 30, 0, 0, time.UTC)); got != 1 {
		t.Fatalf("streak = %d, want 1", got)
	}
}

func TestDripGrowsWithTheStreak(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Лесенка")
	u := s.User(id)

	// 29 earlier days: meeting the goal today makes the 30th, which the default
	// ladder [[1,1],[30,2],[100,3]] pays 2 for.
	today := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= 29; i++ {
		d := today.AddDate(0, 0, -i).Format(dateFmt)
		if _, err := s.db.Exec(`INSERT INTO user_daily_activity (user_id, day, actions, goal) VALUES (?, ?, 10, 10)`,
			id, d); err != nil {
			t.Fatalf("seed %s: %v", d, err)
		}
	}
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)
	for i := 0; i < 10; i++ {
		answer(t, u, "01."+strconv.Itoa(i), true, at)
	}
	if bal, _ := s.Balance(id); bal != 2 {
		t.Fatalf("balance = %d, want 2 (the 30th day of the streak)", bal)
	}
}

func TestRepairIsOnlyForPastDays(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Сегодня")
	u := s.User(id)
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, loc)

	// Today can still be done, so it is not a "missed" day.
	if err := u.RepairStreak("2026-09-29", now); err == nil {
		t.Fatal("repairing today was accepted; want an error")
	}
	// A day that already met its goal needs no repair.
	if _, err := s.db.Exec(`INSERT INTO user_daily_activity (user_id, day, actions, goal) VALUES (?, '2026-09-28', 10, 10)`, id); err != nil {
		t.Fatal(err)
	}
	if err := u.RepairStreak("2026-09-28", now); err == nil {
		t.Fatal("repairing an already-active day was accepted; want an error")
	}
	// Yesterday, missed, inside the window: accepted, and idempotent.
	if err := u.RepairStreak("2026-09-27", now); err != nil {
		t.Fatalf("repair of yesterday-but-one: %v", err)
	}
	if err := u.RepairStreak("2026-09-27", now); err != nil {
		t.Fatalf("repeating a repair: %v", err)
	}
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM streak_repairs WHERE user_id = ?`, id).Scan(&n)
	if n != 1 {
		t.Fatalf("%d repair rows, want 1", n)
	}
}
