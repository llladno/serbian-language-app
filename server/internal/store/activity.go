package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// lockUserTx takes a row lock on the user for the rest of the transaction, so
// two transactions for the same learner run one after the other. It is how a
// check-then-write on per-user state (is this the first attempt? is the balance
// enough?) stays correct under concurrent requests. Postgres only: SQLite
// already serialises writers. No table references users, so nothing else can
// be waiting on this row while holding a lock this transaction needs.
func lockUserTx(tx *dbtx, userID string) error {
	if !tx.pg {
		return nil
	}
	var id string
	if err := tx.QueryRow(`SELECT id FROM users WHERE id = ? FOR UPDATE`, userID).Scan(&id); err != nil {
		return fmt.Errorf("lock user %s: %w", userID, err)
	}
	return nil
}

// recordActionTx counts one action toward the user's daily goal and, if this
// action is the one that met the goal, pays the streak drip — in the caller's
// transaction, so the action and its payout commit together.
//
// "One action" is deliberately narrow: the first attempt at a given exercise
// (see AddAttempt) and every SRS review (see GradeCard). Re-solving a finished
// lesson is worth nothing, which is what stops the daily goal, the answer
// streak and every counting quest from being farmed on lesson 00.
func recordActionTx(tx *dbtx, userID string, now time.Time) error {
	set, err := economySettings(tx)
	if err != nil {
		return err
	}
	loc := userLocation(tx, userID)
	day := now.In(loc).Format(dateFmt)

	// The goal is frozen into the row by the first action of the day, so a
	// goal changed later by an admin applies from the next day on and cannot
	// make today cross twice.
	var actions, goal int
	err = tx.QueryRow(`INSERT INTO user_daily_activity (user_id, day, actions, goal)
		VALUES (?, ?, 1, ?)
		ON CONFLICT (user_id, day) DO UPDATE SET actions = user_daily_activity.actions + 1
		RETURNING actions, goal`, userID, day, set.DailyGoal).Scan(&actions, &goal)
	if err != nil {
		return fmt.Errorf("record action: %w", err)
	}

	// Pay exactly on the crossing, not on every action after it.
	if actions != goal {
		return nil
	}
	streak, err := streakDaysTx(tx, userID, loc, now)
	if err != nil {
		return err
	}
	drip := set.Drip.DripFor(streak)
	if drip <= 0 {
		return nil
	}
	_, err = addLedgerEntryTx(tx, LedgerEntry{
		UserID:         userID,
		Amount:         drip,
		Kind:           "streak_daily",
		Ref:            day,
		IdempotencyKey: "streak_daily:" + userID + ":" + day,
	}, now)
	// Belt and braces: the crossing happens once per row, but the unique key
	// is the real guarantee that a day pays once, so a duplicate is "already
	// paid", not a failure.
	if err != nil && !errors.Is(err, ErrDuplicateEntry) {
		return err
	}
	return nil
}

// recordAnswerTx is replaced by the answer-streak implementation in a later
// task; until then a first attempt only counts toward the daily goal.
func recordAnswerTx(tx *dbtx, userID string, correct bool, now time.Time) error { return nil }

// streakDaysTx counts consecutive active days in the user's own timezone. A day
// is active when it met that day's goal or when a purchased repair covers it.
//
// The streak ends today if today is active, otherwise yesterday: it only
// breaks once a whole day has passed with nothing done, so it does not read 0
// at midnight, before the first action of the new day.
func streakDaysTx(q querier, userID string, loc *time.Location, now time.Time) (int, error) {
	rows, err := q.Query(`SELECT day FROM user_daily_activity WHERE user_id = ? AND actions >= goal
		UNION SELECT day FROM streak_repairs WHERE user_id = ?`, userID, userID)
	if err != nil {
		return 0, fmt.Errorf("streak days: %w", err)
	}
	defer rows.Close()
	active := map[string]bool{}
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return 0, fmt.Errorf("scan active day: %w", err)
		}
		active[d] = true
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate active days: %w", err)
	}

	// Walk calendar dates, not instants: take today's civil date in the user's
	// zone and step it in UTC, where a day is always 24 hours, so a DST change
	// or a zone with no local midnight cannot skip or repeat a date.
	y, m, d := now.In(loc).Date()
	cur := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	if !active[cur.Format(dateFmt)] {
		cur = cur.AddDate(0, 0, -1)
	}
	streak := 0
	for ; active[cur.Format(dateFmt)]; cur = cur.AddDate(0, 0, -1) {
		streak++
	}
	return streak, nil
}

// StreakDays counts consecutive days that met the daily goal, ending today —
// or yesterday while today has not yet met it.
//
// This replaces the pre-currency definition ("any day with at least one SRS
// review or answer"): a day now needs real work rather than one tap, only the
// first attempt at an exercise counts, and the day boundary follows the user's
// timezone instead of the server's.
func (u *UserStore) StreakDays(now time.Time) (int, error) {
	return streakDaysTx(u.db, u.user, userLocation(u.db, u.user), now)
}

// RepairStreak marks one missed day as active. The caller is responsible for
// having consumed the entitlement that paid for it (see SpendStreakRepair);
// this function only enforces that the day is in the past, is not already
// active, and is inside the configured window.
func (u *UserStore) RepairStreak(day string, now time.Time) error {
	set, err := economySettings(u.db)
	if err != nil {
		return err
	}
	loc := userLocation(u.db, u.user)
	parsed, err := time.ParseInLocation(dateFmt, day, loc)
	if err != nil {
		return fmt.Errorf("repair streak: bad day %q: %w", day, err)
	}
	// Today is not a missed day yet — it can still be done — so only strictly
	// earlier days qualify.
	if day >= now.In(loc).Format(dateFmt) {
		return fmt.Errorf("repair streak: %s is not in the past", day)
	}
	// The window is measured from the end of the missed day.
	deadline := parsed.AddDate(0, 0, 1).Add(time.Duration(set.RepairWindowHours) * time.Hour)
	if now.After(deadline) {
		return fmt.Errorf("repair streak: %s is outside the %dh window", day, set.RepairWindowHours)
	}
	var actions, goal int
	err = u.db.QueryRow(`SELECT actions, goal FROM user_daily_activity WHERE user_id = ? AND day = ?`,
		u.user, day).Scan(&actions, &goal)
	switch {
	case err == nil && actions >= goal:
		return fmt.Errorf("repair streak: %s is already active", day)
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("repair streak: read %s: %w", day, err)
	}
	if _, err := u.db.Exec(`INSERT INTO streak_repairs (user_id, day, created_at) VALUES (?, ?, ?)
		ON CONFLICT (user_id, day) DO NOTHING`,
		u.user, day, now.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("repair streak: %w", err)
	}
	return nil
}
