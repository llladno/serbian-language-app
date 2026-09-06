// Package store persists mutable app state (SRS schedule, exercise
// attempts, lesson progress) in SQLite.
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/grisha/serbian-app/server/internal/srs"
)

const dateFmt = "2006-01-02"

// Store wraps the SQLite connection.
type Store struct {
	db *sql.DB
}

// CardSeed identifies a card that should exist for a given content item.
type CardSeed struct {
	CardID string
	Kind   string // "vocab" | "ff"
	RefID  string
}

// CardRow is a stored SRS card with its scheduling state.
type CardRow struct {
	CardID string
	Kind   string
	RefID  string
	srs.Card
}

// Attempt is one recorded exercise answer.
type Attempt struct {
	ExerciseID string
	Lesson     string
	Block      string
	Answer     string
	Correct    bool
}

// WeakExercise is an exercise the learner keeps getting wrong.
type WeakExercise struct {
	ExerciseID string
	Lesson     string
	Wrong      int
	Total      int
}

// Open opens (creating if needed) the database at path and applies the schema.
// path may be ":memory:".
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // modernc sqlite + WAL: keep it simple, single writer
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`); err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.applySchema(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) applySchema() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS srs_cards (
	card_id       TEXT PRIMARY KEY,
	kind          TEXT NOT NULL,
	ref_id        TEXT NOT NULL,
	ease          REAL NOT NULL DEFAULT 2.5,
	interval_days INTEGER NOT NULL DEFAULT 0,
	reps          INTEGER NOT NULL DEFAULT 0,
	lapses        INTEGER NOT NULL DEFAULT 0,
	state         TEXT NOT NULL DEFAULT 'new',
	due           TEXT,
	updated_at    TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS reviews (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	card_id     TEXT NOT NULL,
	grade       INTEGER NOT NULL,
	reviewed_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS attempts (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	exercise_id  TEXT NOT NULL,
	lesson       TEXT NOT NULL,
	block        TEXT NOT NULL,
	answer       TEXT NOT NULL,
	correct      INTEGER NOT NULL,
	attempted_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS lesson_progress (
	lesson       TEXT PRIMARY KEY,
	status       TEXT NOT NULL,
	started_at   TEXT,
	completed_at TEXT
);
PRAGMA user_version = 1;
`)
	return err
}

// EnsureCards inserts any missing cards in state "new".
func (s *Store) EnsureCards(seeds []CardSeed) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO srs_cards (card_id, kind, ref_id, updated_at) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	nowISO := time.Now().UTC().Format(time.RFC3339)
	for _, sd := range seeds {
		if _, err := stmt.Exec(sd.CardID, sd.Kind, sd.RefID, nowISO); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func scanCard(row interface {
	Scan(...any) error
}) (CardRow, error) {
	var c CardRow
	var due sql.NullString
	var state string
	if err := row.Scan(&c.CardID, &c.Kind, &c.RefID, &c.Ease, &c.IntervalDays,
		&c.Reps, &c.Lapses, &state, &due); err != nil {
		return CardRow{}, err
	}
	c.State = srs.State(state)
	if due.Valid && due.String != "" {
		if t, err := time.Parse(dateFmt, due.String); err == nil {
			c.Due = t
		}
	}
	return c, nil
}

const cardCols = `card_id, kind, ref_id, ease, interval_days, reps, lapses, state, due`

// DueQueue returns due learning/review cards plus up to newLimit new cards.
func (s *Store) DueQueue(today time.Time, newLimit int) ([]CardRow, error) {
	todayStr := today.Format(dateFmt)
	var out []CardRow

	rows, err := s.db.Query(`SELECT `+cardCols+` FROM srs_cards
		WHERE state IN ('learning','review') AND (due IS NULL OR due <= ?)
		ORDER BY due ASC`, todayStr)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, c)
	}
	rows.Close()

	if newLimit > 0 {
		nrows, err := s.db.Query(`SELECT `+cardCols+` FROM srs_cards
			WHERE state = 'new' ORDER BY card_id LIMIT ?`, newLimit)
		if err != nil {
			return nil, err
		}
		defer nrows.Close()
		for nrows.Next() {
			c, err := scanCard(nrows)
			if err != nil {
				return nil, err
			}
			out = append(out, c)
		}
	}
	return out, nil
}

// GradeCard applies srs.Schedule to the card, persists it, and logs a review.
func (s *Store) GradeCard(cardID string, g srs.Grade, now time.Time) (srs.Card, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return srs.Card{}, err
	}
	defer tx.Rollback()

	c, err := scanCard(tx.QueryRow(`SELECT `+cardCols+` FROM srs_cards WHERE card_id = ?`, cardID))
	if err != nil {
		return srs.Card{}, fmt.Errorf("load card %s: %w", cardID, err)
	}
	updated := srs.Schedule(c.Card, g, now)

	var dueStr any
	if !updated.Due.IsZero() {
		dueStr = updated.Due.Format(dateFmt)
	}
	if _, err := tx.Exec(`UPDATE srs_cards SET ease=?, interval_days=?, reps=?, lapses=?, state=?, due=?, updated_at=? WHERE card_id=?`,
		updated.Ease, updated.IntervalDays, updated.Reps, updated.Lapses, string(updated.State), dueStr,
		now.UTC().Format(time.RFC3339), cardID); err != nil {
		return srs.Card{}, err
	}
	if _, err := tx.Exec(`INSERT INTO reviews (card_id, grade, reviewed_at) VALUES (?, ?, ?)`,
		cardID, int(g), now.UTC().Format(time.RFC3339)); err != nil {
		return srs.Card{}, err
	}
	return updated, tx.Commit()
}

// ReviewedToday counts reviews logged on today's date.
func (s *Store) ReviewedToday(today time.Time) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM reviews WHERE substr(reviewed_at,1,10) = ?`,
		today.Format(dateFmt)).Scan(&n)
	return n, err
}

// StreakDays counts consecutive days (ending today) with at least one review.
func (s *Store) StreakDays(today time.Time) (int, error) {
	rows, err := s.db.Query(`SELECT DISTINCT substr(reviewed_at,1,10) AS d FROM reviews ORDER BY d DESC`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	days := map[string]bool{}
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return 0, err
		}
		days[d] = true
	}
	streak := 0
	for cur := today; days[cur.Format(dateFmt)]; cur = cur.AddDate(0, 0, -1) {
		streak++
	}
	return streak, nil
}

// AddAttempt records one exercise answer.
func (s *Store) AddAttempt(a Attempt, now time.Time) error {
	correct := 0
	if a.Correct {
		correct = 1
	}
	_, err := s.db.Exec(`INSERT INTO attempts (exercise_id, lesson, block, answer, correct, attempted_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		a.ExerciseID, a.Lesson, a.Block, a.Answer, correct, now.UTC().Format(time.RFC3339))
	return err
}

// WeakExercises returns exercises failed at least half the time (>= 2 attempts).
func (s *Store) WeakExercises(limit int) ([]WeakExercise, error) {
	rows, err := s.db.Query(`
SELECT exercise_id, lesson, SUM(1 - correct) AS wrong, COUNT(*) AS total
FROM attempts
GROUP BY exercise_id
HAVING total >= 2 AND wrong * 2 >= total
ORDER BY wrong DESC, total DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WeakExercise
	for rows.Next() {
		var w WeakExercise
		if err := rows.Scan(&w.ExerciseID, &w.Lesson, &w.Wrong, &w.Total); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, nil
}

// SetLessonStatus upserts a lesson's progress status.
func (s *Store) SetLessonStatus(lesson, status string, now time.Time) error {
	iso := now.UTC().Format(time.RFC3339)
	var startedCol, completedCol any
	if status == "done" {
		completedCol = iso
	}
	startedCol = iso
	_, err := s.db.Exec(`
INSERT INTO lesson_progress (lesson, status, started_at, completed_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(lesson) DO UPDATE SET
	status = excluded.status,
	started_at = COALESCE(lesson_progress.started_at, excluded.started_at),
	completed_at = COALESCE(excluded.completed_at, lesson_progress.completed_at)`,
		lesson, status, startedCol, completedCol)
	return err
}

// LessonStatuses returns lesson id -> status for every tracked lesson.
func (s *Store) LessonStatuses() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT lesson, status FROM lesson_progress`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var l, st string
		if err := rows.Scan(&l, &st); err != nil {
			return nil, err
		}
		out[l] = st
	}
	return out, nil
}

// LessonStatus returns one lesson's status, or "" if untracked.
func (s *Store) LessonStatus(lesson string) (string, error) {
	var st string
	err := s.db.QueryRow(`SELECT status FROM lesson_progress WHERE lesson = ?`, lesson).Scan(&st)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return st, err
}

// CardStats returns the total number of cards and how many are "known"
// (in review with an interval of at least a week).
func (s *Store) CardStats() (total, known int, err error) {
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM srs_cards`).Scan(&total); err != nil {
		return
	}
	err = s.db.QueryRow(`SELECT COUNT(*) FROM srs_cards WHERE state = 'review' AND interval_days >= 7`).Scan(&known)
	return
}
