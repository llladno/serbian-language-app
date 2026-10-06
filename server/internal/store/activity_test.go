package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"sync"
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

func TestDayCountsFirstAttemptsAndDueReviewsOnly(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Актив")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)

	answer(t, u, "01.1", true, at)
	answer(t, u, "01.1", true, at) // same exercise again: must not count
	answer(t, u, "01.2", false, at)

	// A review counts only when the queue would have served the card. The first
	// grade of a new card does; grading the same card again straight away is
	// ahead of its schedule (Good moved it to tomorrow) and earns nothing,
	// though it is still recorded.
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
	if actions != 3 {
		t.Fatalf("actions = %d, want 3 (two first attempts + the due review; the repeated attempt and the early re-grade must not count)", actions)
	}
	if goal != 10 {
		t.Fatalf("goal = %d, want the seeded 10", goal)
	}
	var reviews int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM reviews WHERE user_id = ?`, id).Scan(&reviews); err != nil {
		t.Fatalf("count reviews: %v", err)
	}
	if reviews != 2 {
		t.Fatalf("%d reviews recorded, want 2: an uncounted review is still a review", reviews)
	}
}

// The regression guard for review farming: ten grades of one card must not
// meet the daily goal, extend the streak or pay the drip, yet each grade is
// still recorded and the schedule still advances.
func TestGradingOneCardRepeatedlyDoesNotMeetTheGoal(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Фермер")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)
	if err := u.EnsureCards([]CardSeed{{CardID: "vocab:x", Kind: "vocab", RefID: "x"}}); err != nil {
		t.Fatalf("seed card: %v", err)
	}

	var last srs.Card
	for i := 0; i < 10; i++ {
		c, err := u.GradeCard("vocab:x", srs.Good, at)
		if err != nil {
			t.Fatalf("grade %d: %v", i, err)
		}
		last = c
	}

	var actions int
	if err := s.db.QueryRow(`SELECT actions FROM user_daily_activity WHERE user_id = ? AND day = ?`,
		id, "2026-09-29").Scan(&actions); err != nil {
		t.Fatalf("activity row: %v", err)
	}
	if actions != 1 {
		t.Fatalf("actions = %d after ten grades of one card, want 1 (only the first, new-card grade counts)", actions)
	}
	if got, _ := u.StreakDays(at); got != 0 {
		t.Fatalf("streak = %d, want 0", got)
	}
	if bal, _ := s.Balance(id); bal != 0 {
		t.Fatalf("balance = %d, want 0", bal)
	}
	var reviews int
	s.db.QueryRow(`SELECT COUNT(*) FROM reviews WHERE user_id = ?`, id).Scan(&reviews)
	if reviews != 10 {
		t.Fatalf("%d reviews recorded, want 10: grading ahead of schedule stays permitted", reviews)
	}
	if last.State != srs.Review || last.Reps != 10 {
		t.Fatalf("schedule did not advance as before: state %q reps %d, want review/10", last.State, last.Reps)
	}
}

func TestOverdueReviewCounts(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Долг")
	u := s.User(id)
	if err := u.EnsureCards([]CardSeed{{CardID: "vocab:x", Kind: "vocab", RefID: "x"}}); err != nil {
		t.Fatalf("seed card: %v", err)
	}
	// Learned on the 20th, due the 21st, reviewed on the 29th: overdue, so the
	// queue serves it and the review counts.
	if _, err := u.GradeCard("vocab:x", srs.Good, time.Date(2026, 9, 20, 10, 0, 0, 0, loc)); err != nil {
		t.Fatal(err)
	}
	if _, err := u.GradeCard("vocab:x", srs.Good, time.Date(2026, 9, 29, 10, 0, 0, 0, loc)); err != nil {
		t.Fatal(err)
	}
	var actions int
	if err := s.db.QueryRow(`SELECT actions FROM user_daily_activity WHERE user_id = ? AND day = ?`,
		id, "2026-09-29").Scan(&actions); err != nil {
		t.Fatalf("activity row: %v", err)
	}
	if actions != 1 {
		t.Fatalf("actions = %d, want 1 for an overdue review", actions)
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

// Meaningful on Postgres only: SQLite serialises writers, so the race this
// guards against cannot occur there.
func TestConcurrentFirstAttemptsCountOnce(t *testing.T) {
	if !IsPostgresDSN(testDSN()) {
		t.Skip("the first-attempt race needs concurrent writers; SQLite serialises them")
	}
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Двойной клик")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)

	const clicks = 12
	var wg sync.WaitGroup
	errs := make(chan error, clicks)
	start := make(chan struct{})
	for i := 0; i < clicks; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- u.AddAttempt(Attempt{ExerciseID: "01.1", Lesson: "01", Block: "a", Answer: "x", Correct: true}, at)
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("attempt: %v", err)
		}
	}

	var actions int
	if err := s.db.QueryRow(`SELECT actions FROM user_daily_activity WHERE user_id = ? AND day = ?`,
		id, "2026-09-29").Scan(&actions); err != nil {
		t.Fatalf("activity row: %v", err)
	}
	if actions != 1 {
		t.Fatalf("actions = %d after %d simultaneous first attempts at one exercise, want 1", actions, clicks)
	}
	var rows int
	s.db.QueryRow(`SELECT COUNT(*) FROM attempts WHERE user_id = ?`, id).Scan(&rows)
	if rows != clicks {
		t.Fatalf("%d attempt rows, want %d (every answer is still recorded)", rows, clicks)
	}
}

func TestServedByQueue(t *testing.T) {
	now := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	day := func(d int) time.Time { return time.Date(2026, 9, d, 0, 0, 0, 0, time.UTC) }
	cases := []struct {
		name string
		c    srs.Card
		want bool
	}{
		{"new card", srs.Card{State: srs.New}, true},
		{"learning, due today", srs.Card{State: srs.Learning, Due: day(29)}, true},
		{"learning, no due date", srs.Card{State: srs.Learning}, true},
		{"review, overdue", srs.Card{State: srs.Review, Due: day(20)}, true},
		{"review, due today", srs.Card{State: srs.Review, Due: day(29)}, true},
		{"review, due tomorrow", srs.Card{State: srs.Review, Due: day(30)}, false},
		{"learning, due tomorrow", srs.Card{State: srs.Learning, Due: day(30)}, false},
	}
	for _, tc := range cases {
		if got := servedByQueue(tc.c, now); got != tc.want {
			t.Errorf("%s: servedByQueue = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// Again leaves a card due today, so the queue keeps serving it: "due" alone
// would let one card be graded Again over and over. Regression guard for that.
func TestRepeatedAgainOnOneCardCountsOnce(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Снова")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)
	if err := u.EnsureCards([]CardSeed{{CardID: "vocab:x", Kind: "vocab", RefID: "x"}}); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		c, err := u.GradeCard("vocab:x", srs.Again, at)
		if err != nil {
			t.Fatalf("grade %d: %v", i, err)
		}
		if c.State != srs.Learning {
			t.Fatalf("grade %d: state %q, want the card to stay learning (due today)", i, c.State)
		}
	}

	var actions int
	if err := s.db.QueryRow(`SELECT actions FROM user_daily_activity WHERE user_id = ? AND day = ?`,
		id, "2026-09-29").Scan(&actions); err != nil {
		t.Fatalf("activity row: %v", err)
	}
	if actions != 1 {
		t.Fatalf("actions = %d after ten Again grades of one card, want exactly 1", actions)
	}
	if got, _ := u.StreakDays(at); got != 0 {
		t.Fatalf("streak = %d, want 0", got)
	}
	var reviews int
	s.db.QueryRow(`SELECT COUNT(*) FROM reviews WHERE user_id = ?`, id).Scan(&reviews)
	if reviews != 10 {
		t.Fatalf("%d reviews recorded, want 10: uncounted grades are still recorded", reviews)
	}
}

// A learner who genuinely fails a hard card has done real work: the single
// Again still counts.
func TestAgainOnAHardCardStillCounts(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Трудное")
	u := s.User(id)
	if err := u.EnsureCards([]CardSeed{{CardID: "vocab:x", Kind: "vocab", RefID: "x"}}); err != nil {
		t.Fatal(err)
	}
	// Learned earlier, now a due review card that the learner gets wrong.
	if _, err := u.GradeCard("vocab:x", srs.Good, time.Date(2026, 9, 28, 10, 0, 0, 0, loc)); err != nil {
		t.Fatal(err)
	}
	if _, err := u.GradeCard("vocab:x", srs.Again, time.Date(2026, 9, 29, 10, 0, 0, 0, loc)); err != nil {
		t.Fatal(err)
	}
	var actions int
	if err := s.db.QueryRow(`SELECT actions FROM user_daily_activity WHERE user_id = ? AND day = ?`,
		id, "2026-09-29").Scan(&actions); err != nil {
		t.Fatalf("activity row: %v", err)
	}
	if actions != 1 {
		t.Fatalf("actions = %d, want 1 for a failed due review", actions)
	}
}

func TestTenDifferentCardsGradedAgainMeetTheGoal(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Десять")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)
	var seeds []CardSeed
	for i := 0; i < 10; i++ {
		w := "w" + strconv.Itoa(i)
		seeds = append(seeds, CardSeed{CardID: "vocab:" + w, Kind: "vocab", RefID: w})
	}
	if err := u.EnsureCards(seeds); err != nil {
		t.Fatal(err)
	}
	for _, c := range seeds {
		if _, err := u.GradeCard(c.CardID, srs.Again, at); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := u.StreakDays(at); got != 1 {
		t.Fatalf("streak = %d, want 1: ten different due cards meet the goal", got)
	}
	if bal, _ := s.Balance(id); bal != 1 {
		t.Fatalf("balance = %d, want the day-one drip of 1", bal)
	}
}

// The once-a-day limit is per learner-local day, not per UTC day and not
// forever: the same card counts once on each of two consecutive days, even
// when 23:30 and 00:30 local fall on the same UTC date.
func TestSameCardCountsOncePerLocalDay(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Два дня")
	u := s.User(id)
	if err := u.EnsureCards([]CardSeed{{CardID: "vocab:x", Kind: "vocab", RefID: "x"}}); err != nil {
		t.Fatal(err)
	}
	grades := []time.Time{
		time.Date(2026, 9, 29, 22, 0, 0, 0, loc),
		time.Date(2026, 9, 29, 23, 30, 0, 0, loc), // same local day: uncounted
		time.Date(2026, 9, 30, 0, 30, 0, 0, loc),  // next local day, same UTC day
		time.Date(2026, 9, 30, 9, 0, 0, 0, loc),   // uncounted
	}
	for _, at := range grades {
		if _, err := u.GradeCard("vocab:x", srs.Again, at); err != nil {
			t.Fatal(err)
		}
	}
	for _, day := range []string{"2026-09-29", "2026-09-30"} {
		var actions int
		if err := s.db.QueryRow(`SELECT actions FROM user_daily_activity WHERE user_id = ? AND day = ?`,
			id, day).Scan(&actions); err != nil {
			t.Fatalf("activity row %s: %v", day, err)
		}
		if actions != 1 {
			t.Fatalf("%s: actions = %d, want 1", day, actions)
		}
	}
}

// Postgres only, for the same reason as the first-attempt race: two
// simultaneous grades of one card must not both find "no review today".
func TestConcurrentGradesOfOneCardCountOnce(t *testing.T) {
	if !IsPostgresDSN(testDSN()) {
		t.Skip("the race needs concurrent writers; SQLite serialises them")
	}
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Двойное касание")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)
	if err := u.EnsureCards([]CardSeed{{CardID: "vocab:x", Kind: "vocab", RefID: "x"}}); err != nil {
		t.Fatal(err)
	}

	const taps = 12
	var wg sync.WaitGroup
	errs := make(chan error, taps)
	start := make(chan struct{})
	for i := 0; i < taps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := u.GradeCard("vocab:x", srs.Again, at)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("grade: %v", err)
		}
	}
	var actions int
	if err := s.db.QueryRow(`SELECT actions FROM user_daily_activity WHERE user_id = ? AND day = ?`,
		id, "2026-09-29").Scan(&actions); err != nil {
		t.Fatalf("activity row: %v", err)
	}
	if actions != 1 {
		t.Fatalf("actions = %d after %d simultaneous grades of one card, want 1", actions, taps)
	}
}

func TestAnswerStreakTracksCurrentAndBest(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Серия")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)

	answer(t, u, "01.1", true, at)
	answer(t, u, "01.2", true, at)
	answer(t, u, "01.3", true, at)

	cur, best, err := u.AnswerStreak()
	if err != nil {
		t.Fatalf("answer streak: %v", err)
	}
	if cur != 3 || best != 3 {
		t.Fatalf("current, best = %d, %d; want 3, 3", cur, best)
	}

	answer(t, u, "01.4", false, at) // breaks the run
	cur, best, _ = u.AnswerStreak()
	if cur != 0 || best != 3 {
		t.Fatalf("after a wrong answer: current, best = %d, %d; want 0, 3", cur, best)
	}

	answer(t, u, "01.5", true, at)
	cur, best, _ = u.AnswerStreak()
	if cur != 1 || best != 3 {
		t.Fatalf("after restarting: current, best = %d, %d; want 1, 3", cur, best)
	}
}

func TestAnswerStreakIgnoresRepeatsAndReviews(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Повтор")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)

	answer(t, u, "01.1", true, at)
	answer(t, u, "01.1", true, at) // repeat
	answer(t, u, "01.1", true, at) // repeat

	// An SRS review is not an answer either, whatever the grade.
	if err := u.EnsureCards([]CardSeed{{CardID: "vocab:x", Kind: "vocab", RefID: "x"}}); err != nil {
		t.Fatalf("seed card: %v", err)
	}
	if _, err := u.GradeCard("vocab:x", srs.Again, at); err != nil {
		t.Fatalf("grade: %v", err)
	}

	cur, best, _ := u.AnswerStreak()
	if cur != 1 || best != 1 {
		t.Fatalf("repeats or a review moved the streak: current, best = %d, %d; want 1, 1", cur, best)
	}
}

// seedAttempt inserts a raw attempt row, as history written before the
// currency feature existed (no counting, no streak bookkeeping).
func seedAttempt(t *testing.T, s *Store, userID, exID string, correct bool, at string) {
	t.Helper()
	c := 0
	if correct {
		c = 1
	}
	mustExec(t, s, `INSERT INTO attempts (user_id, exercise_id, lesson, block, answer, correct, attempted_at)
		VALUES (?, ?, '01', 'a', 'x', ?, ?)`, userID, exID, c, at)
}

// seedReview inserts a raw review row for the same purpose.
func seedReview(t *testing.T, s *Store, userID, cardID, at string) {
	t.Helper()
	mustExec(t, s, `INSERT INTO reviews (user_id, card_id, grade, reviewed_at) VALUES (?, ?, 3, ?)`,
		userID, cardID, at)
}

// activityOn returns the backfilled action count for a day, or -1 when there
// is no row at all.
func activityOn(t *testing.T, s *Store, userID, day string) int {
	t.Helper()
	var n int
	err := s.db.QueryRow(`SELECT actions FROM user_daily_activity WHERE user_id = ? AND day = ?`,
		userID, day).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return -1
	}
	if err != nil {
		t.Fatalf("activity on %s: %v", day, err)
	}
	return n
}

func TestBackfillReconstructsActivityAndAnswerStreak(t *testing.T) {
	s := openStoreAtVersion(t, 14) // everything up to and including the seed, but not the backfill
	id, _ := s.CreateUser("История")

	// Two days of history, written before the backfill version runs: 10 first
	// attempts on the 27th (one wrong at the end), 10 reviews on the 28th.
	for i := 0; i < 10; i++ {
		seedAttempt(t, s, id, "01."+strconv.Itoa(i), i != 9, "2026-09-27T09:00:00Z")
	}
	for i := 0; i < 10; i++ {
		seedReview(t, s, id, "vocab:"+strconv.Itoa(i), "2026-09-28T09:00:00Z")
	}

	if err := s.runMigrations(); err != nil {
		t.Fatalf("migrate to head: %v", err)
	}

	if got := activityOn(t, s, id, "2026-09-27"); got != 10 {
		t.Fatalf("27th actions = %d, want 10", got)
	}
	if got := activityOn(t, s, id, "2026-09-28"); got != 10 {
		t.Fatalf("28th actions = %d, want 10", got)
	}

	u := s.User(id)
	cur, best, err := u.AnswerStreak()
	if err != nil {
		t.Fatalf("answer streak: %v", err)
	}
	if best != 9 {
		t.Fatalf("best = %d, want 9 (nine correct, then one wrong)", best)
	}
	if cur != 0 {
		t.Fatalf("current = %d, want 0 (the last answer was wrong)", cur)
	}

	// The point of the backfill: two goal-meeting days are a two-day streak,
	// not zero, on the first day the feature is live.
	streak, err := u.StreakDays(time.Date(2026, 9, 29, 10, 0, 0, 0, belgrade(t)))
	if err != nil {
		t.Fatalf("streak days: %v", err)
	}
	if streak != 2 {
		t.Fatalf("streak = %d, want 2", streak)
	}

	// The backfill must not mint currency: it reconstructs history only. Check
	// the ledger itself, not just the balance, so a +N/-N pair cannot hide.
	if bal, _ := s.Balance(id); bal != 0 {
		t.Fatalf("balance = %d after backfill, want 0", bal)
	}
	var ledgerRows int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM currency_ledger`).Scan(&ledgerRows); err != nil {
		t.Fatalf("count ledger: %v", err)
	}
	if ledgerRows != 0 {
		t.Fatalf("currency_ledger has %d rows after backfill, want 0", ledgerRows)
	}
}

// Ten reviews of one card on one day are one action, not ten: the per-card,
// per-local-day cap is the half of the live counting rule the backfill can
// reconstruct, and the half that stops inflated streaks being gifted.
func TestBackfillCountsOneReviewPerCardPerDay(t *testing.T) {
	s := openStoreAtVersion(t, 14)
	id, _ := s.CreateUser("Зубрила")
	for i := 0; i < 10; i++ {
		seedReview(t, s, id, "vocab:1", "2026-09-28T09:"+fmt.Sprintf("%02d", i)+":00Z")
	}
	// A second card the same day adds one more; the same card on the next day
	// is a new day's action.
	seedReview(t, s, id, "vocab:2", "2026-09-28T10:00:00Z")
	seedReview(t, s, id, "vocab:1", "2026-09-29T09:00:00Z")

	if err := s.runMigrations(); err != nil {
		t.Fatalf("migrate to head: %v", err)
	}
	if got := activityOn(t, s, id, "2026-09-28"); got != 2 {
		t.Fatalf("28th actions = %d, want 2 (ten grades of one card + one of another)", got)
	}
	if got := activityOn(t, s, id, "2026-09-29"); got != 1 {
		t.Fatalf("29th actions = %d, want 1", got)
	}
}

// The per-card buckets are cut at Belgrade midnight, the same zone the day
// rows use, so the two halves of the reconstruction agree on where a day ends.
// Belgrade is UTC+2 in September: 22:30Z on the 27th is already the 28th.
func TestBackfillBucketsReviewsInBelgradeDays(t *testing.T) {
	s := openStoreAtVersion(t, 14)
	id, _ := s.CreateUser("Полуночник")

	// Same UTC day, different Belgrade days: two actions.
	seedReview(t, s, id, "vocab:1", "2026-09-27T21:30:00Z") // 27th 23:30 local
	seedReview(t, s, id, "vocab:1", "2026-09-27T22:30:00Z") // 28th 00:30 local
	// Different UTC days, same Belgrade day (the 29th): one action.
	seedReview(t, s, id, "vocab:2", "2026-09-28T22:30:00Z") // 29th 00:30 local
	seedReview(t, s, id, "vocab:2", "2026-09-29T21:30:00Z") // 29th 23:30 local

	if err := s.runMigrations(); err != nil {
		t.Fatalf("migrate to head: %v", err)
	}
	if got := activityOn(t, s, id, "2026-09-27"); got != 1 {
		t.Fatalf("27th actions = %d, want 1", got)
	}
	if got := activityOn(t, s, id, "2026-09-28"); got != 1 {
		t.Fatalf("28th actions = %d, want 1", got)
	}
	if got := activityOn(t, s, id, "2026-09-29"); got != 1 {
		t.Fatalf("29th actions = %d, want 1 (two grades of one card inside one local day)", got)
	}
}

// Only the first attempt at an exercise counts, for the day count and for the
// answer streak alike, and "first" means the earliest attempt, not the one with
// the lowest correct flag.
func TestBackfillReplaysFirstAttemptsOnly(t *testing.T) {
	s := openStoreAtVersion(t, 14)
	id, _ := s.CreateUser("Повторщик")

	seedAttempt(t, s, id, "01.1", true, "2026-09-27T09:00:00Z")
	seedAttempt(t, s, id, "01.2", true, "2026-09-27T09:01:00Z")
	seedAttempt(t, s, id, "01.3", true, "2026-09-27T09:02:00Z")
	// Re-solves: 01.1 gets answered wrong afterwards, on a later day too. A
	// MIN(correct) over the exercise would read 01.1 as wrong and break the run.
	seedAttempt(t, s, id, "01.1", false, "2026-09-27T09:03:00Z")
	seedAttempt(t, s, id, "01.2", true, "2026-09-28T09:00:00Z")
	// And the reverse: wrong first, right on the retry — the first one stands.
	seedAttempt(t, s, id, "01.4", false, "2026-09-28T10:00:00Z")
	seedAttempt(t, s, id, "01.4", true, "2026-09-28T10:01:00Z")

	if err := s.runMigrations(); err != nil {
		t.Fatalf("migrate to head: %v", err)
	}
	if got := activityOn(t, s, id, "2026-09-27"); got != 3 {
		t.Fatalf("27th actions = %d, want 3 (re-solves do not count)", got)
	}
	if got := activityOn(t, s, id, "2026-09-28"); got != 1 {
		t.Fatalf("28th actions = %d, want 1 (only 01.4's first attempt)", got)
	}
	cur, best, err := s.User(id).AnswerStreak()
	if err != nil {
		t.Fatalf("answer streak: %v", err)
	}
	if best != 3 || cur != 0 {
		t.Fatalf("streak = (cur %d, best %d), want (0, 3)", cur, best)
	}
}

// Frozen goals come from the seeded daily_goal, the same value a live first
// action of the day would freeze into its row, and users without any history
// get no rows at all.
func TestBackfillFreezesTheConfiguredGoalAndSkipsEmptyUsers(t *testing.T) {
	s := openStoreAtVersion(t, 14)
	t.Cleanup(func() { restoreEconomySettings(t, s) })
	mustExec(t, s, `UPDATE economy_settings SET value = '3' WHERE key = 'daily_goal'`)
	id, _ := s.CreateUser("Целеустремлённый")
	idle, _ := s.CreateUser("Тихоня")
	seedReview(t, s, id, "vocab:1", "2026-09-28T09:00:00Z")

	if err := s.runMigrations(); err != nil {
		t.Fatalf("migrate to head: %v", err)
	}
	var goal int
	if err := s.db.QueryRow(`SELECT goal FROM user_daily_activity WHERE user_id = ? AND day = ?`,
		id, "2026-09-28").Scan(&goal); err != nil {
		t.Fatalf("read goal: %v", err)
	}
	if goal != 3 {
		t.Fatalf("goal = %d, want the configured 3", goal)
	}
	var rows int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM user_daily_activity WHERE user_id = ?`, idle).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 {
		t.Fatalf("idle user has %d activity rows, want 0", rows)
	}
}

// Version 15 must run on a database that has already moved past it: the dev
// database applied 013 and 016 long before the backfill existed. The runner
// applies every unrecorded version in ascending order, so 015 still fires.
func TestBackfillRunsOnADatabaseAlreadyAtVersion016(t *testing.T) {
	s := openStoreAtVersion(t, 14)
	// Apply 016 by hand, out of order, exactly as that database got it.
	body, err := migrationFS.ReadFile("migrations/016_reviews_card_day_idx.sql")
	if err != nil {
		t.Fatalf("read 016: %v", err)
	}
	for _, stmt := range splitSQL(string(body)) {
		mustExec(t, s, stmt)
	}
	mustExec(t, s, `INSERT INTO schema_migrations (version, applied_at) VALUES (16, '2026-09-30T00:00:00Z')`)

	id, _ := s.CreateUser("Старожил")
	seedReview(t, s, id, "vocab:1", "2026-09-28T09:00:00Z")

	if err := s.runMigrations(); err != nil {
		t.Fatalf("migrate to head: %v", err)
	}
	var applied int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 15`).Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if applied != 1 {
		t.Fatalf("version 15 recorded %d times, want 1", applied)
	}
	if got := activityOn(t, s, id, "2026-09-28"); got != 1 {
		t.Fatalf("28th actions = %d, want 1", got)
	}
}

// Rows the live code already wrote survive the backfill untouched. On a
// database that ran Task 5's counting before this hook existed (dev did), the
// reconstruction is the weaker source: it keeps the live row even when its own
// count is higher, because lowering a count a learner was already paid for is
// the worse failure. This pins that, since the hook only ever runs once and a
// regression here could not be caught later.
func TestBackfillNeverOverwritesLiveRows(t *testing.T) {
	s := openStoreAtVersion(t, 14)
	id, _ := s.CreateUser("Ветеран")

	// History that would reconstruct as 10 actions on the 27th and best = 9.
	for i := 0; i < 10; i++ {
		seedAttempt(t, s, id, "01."+strconv.Itoa(i), i != 9, "2026-09-27T09:00:00Z")
	}
	// What "the live code already recorded" looks like: a partial day with a
	// goal from a different settings generation, and a streak still at zero.
	mustExec(t, s, `INSERT INTO user_daily_activity (user_id, day, actions, goal) VALUES (?, ?, ?, ?)`,
		id, "2026-09-27", 3, 99)
	mustExec(t, s, `INSERT INTO user_answer_streak (user_id, current, best, updated_at) VALUES (?, ?, ?, ?)`,
		id, 0, 0, "2026-09-27T09:00:00Z")

	if err := s.runMigrations(); err != nil {
		t.Fatalf("migrate to head: %v", err)
	}

	if got := activityOn(t, s, id, "2026-09-27"); got != 3 {
		t.Fatalf("27th actions = %d, want 3 (the live row, not the backfill's 10)", got)
	}
	var goal int
	if err := s.db.QueryRow(`SELECT goal FROM user_daily_activity WHERE user_id = ? AND day = ?`,
		id, "2026-09-27").Scan(&goal); err != nil {
		t.Fatalf("read goal: %v", err)
	}
	if goal != 99 {
		t.Fatalf("goal = %d, want 99 (the live row's frozen goal)", goal)
	}
	cur, best, err := s.User(id).AnswerStreak()
	if err != nil {
		t.Fatalf("answer streak: %v", err)
	}
	if cur != 0 || best != 0 {
		t.Fatalf("answer streak = (%d, %d), want (0, 0): the live row stands", cur, best)
	}
}
