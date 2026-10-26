package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// RepeatCandidate is an exercise the learner has answered at least once, with what
// «Повторить» needs to rank it.
type RepeatCandidate struct {
	ExerciseID  string
	Lesson      string
	LastCorrect bool
	// LastTypo: the latest answer was right only because a one-letter slip was forgiven.
	LastTypo bool
	Wrong    int
	Total    int
	LastAt   time.Time
}

// RepeatCandidates lists every exercise the learner has attempted, with the outcome of
// the latest attempt and the lifetime wrong/total counts.
func (u *UserStore) RepeatCandidates() ([]RepeatCandidate, error) {
	rows, err := u.db.Query(`
SELECT a.exercise_id, a.lesson, a.correct, a.typo, a.attempted_at, w.wrong, w.total
FROM attempts a
JOIN (SELECT exercise_id, MAX(id) AS last_id, SUM(1 - correct) AS wrong, COUNT(*) AS total
      FROM attempts WHERE user_id = ? GROUP BY exercise_id) w ON a.id = w.last_id`, u.user)
	if err != nil {
		return nil, fmt.Errorf("repeat candidates: %w", err)
	}
	defer rows.Close()
	var out []RepeatCandidate
	for rows.Next() {
		var c RepeatCandidate
		var correct, typo int
		var at string
		if err := rows.Scan(&c.ExerciseID, &c.Lesson, &correct, &typo, &at, &c.Wrong, &c.Total); err != nil {
			return nil, fmt.Errorf("repeat candidates: %w", err)
		}
		c.LastCorrect, c.LastTypo = correct == 1, typo == 1
		c.LastAt, _ = time.Parse(time.RFC3339, at)
		out = append(out, c)
	}
	return out, rows.Err()
}

// Location is the learner's timezone (the one their days, streak and daily goal follow).
func (u *UserStore) Location() *time.Location { return userLocation(u.db, u.user) }

// Today is the learner's calendar date at now, in their own timezone.
func (u *UserStore) Today(now time.Time) string { return now.In(u.Location()).Format(dateFmt) }

// RepeatSession is one learner's batch for one day. Keys are "lesson|exerciseID".
type RepeatSession struct {
	ID           int64
	Day          string
	Keys         []string
	AfterAttempt int64 // attempts with a bigger id were made after the batch started
	StartedAt    time.Time
}

// RepeatSession returns the batch stored for day, nil when there is none.
func (u *UserStore) RepeatSession(day string) (*RepeatSession, error) {
	var s RepeatSession
	var raw, started string
	err := u.db.QueryRow(`SELECT id, day, exercise_ids, after_attempt, started_at FROM repeat_sessions WHERE user_id = ? AND day = ?`,
		u.user, day).Scan(&s.ID, &s.Day, &raw, &s.AfterAttempt, &started)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("repeat session: %w", err)
	}
	if err := json.Unmarshal([]byte(raw), &s.Keys); err != nil {
		return nil, fmt.Errorf("repeat session: bad batch: %w", err)
	}
	s.StartedAt, _ = time.Parse(time.RFC3339, started)
	return &s, nil
}

// StartRepeatSession stores keys as the batch for day. A second call for the same day
// changes nothing and returns the batch already stored (two tabs, a double tap).
func (u *UserStore) StartRepeatSession(day string, keys []string, now time.Time) (*RepeatSession, error) {
	raw, _ := json.Marshal(keys)
	if _, err := u.db.Exec(`INSERT INTO repeat_sessions (user_id, day, exercise_ids, after_attempt, started_at)
		VALUES (?, ?, ?, (SELECT COALESCE(MAX(id), 0) FROM attempts WHERE user_id = ?), ?)
		ON CONFLICT (user_id, day) DO NOTHING`, u.user, day, string(raw), u.user, now.UTC().Format(time.RFC3339)); err != nil {
		return nil, fmt.Errorf("start repeat session: %w", err)
	}
	return u.RepeatSession(day)
}

// RepeatPending returns the keys of the batch that still have no correct answer recorded after
// the batch started: what is left to do.
func (u *UserStore) RepeatPending(s RepeatSession) ([]string, error) {
	var pending []string
	for _, k := range s.Keys {
		_, ex, _ := strings.Cut(k, "|")
		var n int
		if err := u.db.QueryRow(`SELECT COUNT(*) FROM attempts WHERE user_id = ? AND exercise_id = ? AND correct = 1 AND id > ?`,
			u.user, ex, s.AfterAttempt).Scan(&n); err != nil {
			return nil, fmt.Errorf("repeat pending: %w", err)
		}
		if n == 0 {
			pending = append(pending, k)
		}
	}
	return pending, nil
}

// RepeatSessionDone reports whether every exercise of the batch has been answered right.
func (u *UserStore) RepeatSessionDone(s RepeatSession) (bool, error) {
	pending, err := u.RepeatPending(s)
	return len(pending) == 0, err
}

// HasUnlimitedRepeat: a running year, or at least one level paid for in rubles. A level
// bought with зёрнышки does not count.
func (s *Store) HasUnlimitedRepeat(userID string, now time.Time) (bool, error) {
	if _, ok, err := s.User(userID).AllAccessUntil(now); err != nil || ok {
		return ok, err
	}
	return s.HasPaidLevel(userID)
}
