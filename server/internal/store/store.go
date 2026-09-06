// Package store persists mutable app state (SRS schedule, exercise
// attempts, lesson progress) in SQLite, scoped per account name.
package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/grisha/serbian-app/server/internal/srs"
)

const (
	dateFmt     = "2006-01-02"
	schemaVer   = 2
	legacyOwner = "Гриша" // v1 rows (no account) are migrated to this account
)

// Store owns the SQLite connection and the account list.
type Store struct {
	db *sql.DB
}

// UserStore is a per-account view over the state tables.
type UserStore struct {
	db   *sql.DB
	user string
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

// DayActivity is the number of reviews + exercise attempts on one date.
type DayActivity struct {
	Date  string
	Count int
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

const schemaSQL = `
CREATE TABLE IF NOT EXISTS users (
	name       TEXT PRIMARY KEY,
	created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS srs_cards (
	user_name     TEXT NOT NULL,
	card_id       TEXT NOT NULL,
	kind          TEXT NOT NULL,
	ref_id        TEXT NOT NULL,
	ease          REAL NOT NULL DEFAULT 2.5,
	interval_days INTEGER NOT NULL DEFAULT 0,
	reps          INTEGER NOT NULL DEFAULT 0,
	lapses        INTEGER NOT NULL DEFAULT 0,
	state         TEXT NOT NULL DEFAULT 'new',
	due           TEXT,
	updated_at    TEXT NOT NULL,
	PRIMARY KEY (user_name, card_id)
);
CREATE TABLE IF NOT EXISTS reviews (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	user_name   TEXT NOT NULL,
	card_id     TEXT NOT NULL,
	grade       INTEGER NOT NULL,
	reviewed_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS attempts (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	user_name    TEXT NOT NULL,
	exercise_id  TEXT NOT NULL,
	lesson       TEXT NOT NULL,
	block        TEXT NOT NULL,
	answer       TEXT NOT NULL,
	correct      INTEGER NOT NULL,
	attempted_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS lesson_progress (
	user_name    TEXT NOT NULL,
	lesson       TEXT NOT NULL,
	status       TEXT NOT NULL,
	started_at   TEXT,
	completed_at TEXT,
	PRIMARY KEY (user_name, lesson)
);
CREATE INDEX IF NOT EXISTS reviews_user ON reviews(user_name);
CREATE INDEX IF NOT EXISTS attempts_user ON attempts(user_name);
`

func (s *Store) applySchema() error {
	var ver int
	_ = s.db.QueryRow(`PRAGMA user_version`).Scan(&ver)

	if ver == 1 {
		if err := s.migrateV1toV2(); err != nil {
			return fmt.Errorf("migrate v1->v2: %w", err)
		}
	}
	if _, err := s.db.Exec(schemaSQL); err != nil {
		return err
	}
	if _, err := s.db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaVer)); err != nil {
		return err
	}
	return nil
}

// migrateV1toV2 moves the accountless v1 tables under the legacy account.
func (s *Store) migrateV1toV2() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmts := []string{
		`ALTER TABLE srs_cards RENAME TO srs_cards_v1`,
		`ALTER TABLE reviews RENAME TO reviews_v1`,
		`ALTER TABLE attempts RENAME TO attempts_v1`,
		`ALTER TABLE lesson_progress RENAME TO lesson_progress_v1`,
		schemaSQL,
		`INSERT INTO users (name, created_at) VALUES ('` + legacyOwner + `', '` +
			time.Now().UTC().Format(time.RFC3339) + `')`,
		`INSERT INTO srs_cards (user_name, card_id, kind, ref_id, ease, interval_days, reps, lapses, state, due, updated_at)
			SELECT '` + legacyOwner + `', card_id, kind, ref_id, ease, interval_days, reps, lapses, state, due, updated_at FROM srs_cards_v1`,
		`INSERT INTO reviews (user_name, card_id, grade, reviewed_at)
			SELECT '` + legacyOwner + `', card_id, grade, reviewed_at FROM reviews_v1`,
		`INSERT INTO attempts (user_name, exercise_id, lesson, block, answer, correct, attempted_at)
			SELECT '` + legacyOwner + `', exercise_id, lesson, block, answer, correct, attempted_at FROM attempts_v1`,
		`INSERT INTO lesson_progress (user_name, lesson, status, started_at, completed_at)
			SELECT '` + legacyOwner + `', lesson, status, started_at, completed_at FROM lesson_progress_v1`,
		`DROP TABLE srs_cards_v1`,
		`DROP TABLE reviews_v1`,
		`DROP TABLE attempts_v1`,
		`DROP TABLE lesson_progress_v1`,
	}
	for _, q := range stmts {
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("%s: %w", firstLine(q), err)
		}
	}
	return tx.Commit()
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// ---- account management ----

// NormalizeName trims and collapses whitespace in an account name.
func NormalizeName(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// EnsureUser creates the account if it does not exist. Returns the
// normalized name.
func (s *Store) EnsureUser(name string) (string, error) {
	n := NormalizeName(name)
	if n == "" {
		return "", fmt.Errorf("empty name")
	}
	if len([]rune(n)) > 40 {
		return "", fmt.Errorf("name too long")
	}
	_, err := s.db.Exec(`INSERT OR IGNORE INTO users (name, created_at) VALUES (?, ?)`,
		n, time.Now().UTC().Format(time.RFC3339))
	return n, err
}

// UserExists reports whether an account with this exact (normalized) name exists.
func (s *Store) UserExists(name string) (bool, error) {
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM users WHERE name = ?`, NormalizeName(name)).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// ListUsers returns all account names, oldest first.
func (s *Store) ListUsers() ([]string, error) {
	rows, err := s.db.Query(`SELECT name FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

// User returns a per-account view of the state tables.
func (s *Store) User(name string) *UserStore {
	return &UserStore{db: s.db, user: NormalizeName(name)}
}

// ---- per-account state ----

// EnsureCards inserts any missing cards in state "new".
func (u *UserStore) EnsureCards(seeds []CardSeed) error {
	tx, err := u.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO srs_cards (user_name, card_id, kind, ref_id, updated_at) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	nowISO := time.Now().UTC().Format(time.RFC3339)
	for _, sd := range seeds {
		if _, err := stmt.Exec(u.user, sd.CardID, sd.Kind, sd.RefID, nowISO); err != nil {
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
func (u *UserStore) DueQueue(today time.Time, newLimit int) ([]CardRow, error) {
	todayStr := today.Format(dateFmt)
	var out []CardRow

	rows, err := u.db.Query(`SELECT `+cardCols+` FROM srs_cards
		WHERE user_name = ? AND state IN ('learning','review') AND (due IS NULL OR due <= ?)
		ORDER BY due ASC`, u.user, todayStr)
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
		nrows, err := u.db.Query(`SELECT `+cardCols+` FROM srs_cards
			WHERE user_name = ? AND state = 'new' ORDER BY RANDOM() LIMIT ?`, u.user, newLimit)
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
func (u *UserStore) GradeCard(cardID string, g srs.Grade, now time.Time) (srs.Card, error) {
	tx, err := u.db.Begin()
	if err != nil {
		return srs.Card{}, err
	}
	defer tx.Rollback()

	c, err := scanCard(tx.QueryRow(`SELECT `+cardCols+` FROM srs_cards WHERE user_name = ? AND card_id = ?`, u.user, cardID))
	if err != nil {
		return srs.Card{}, fmt.Errorf("load card %s: %w", cardID, err)
	}
	updated := srs.Schedule(c.Card, g, now)

	var dueStr any
	if !updated.Due.IsZero() {
		dueStr = updated.Due.Format(dateFmt)
	}
	if _, err := tx.Exec(`UPDATE srs_cards SET ease=?, interval_days=?, reps=?, lapses=?, state=?, due=?, updated_at=? WHERE user_name=? AND card_id=?`,
		updated.Ease, updated.IntervalDays, updated.Reps, updated.Lapses, string(updated.State), dueStr,
		now.UTC().Format(time.RFC3339), u.user, cardID); err != nil {
		return srs.Card{}, err
	}
	if _, err := tx.Exec(`INSERT INTO reviews (user_name, card_id, grade, reviewed_at) VALUES (?, ?, ?, ?)`,
		u.user, cardID, int(g), now.UTC().Format(time.RFC3339)); err != nil {
		return srs.Card{}, err
	}
	return updated, tx.Commit()
}

// ReviewedToday counts reviews logged on today's date.
func (u *UserStore) ReviewedToday(today time.Time) (int, error) {
	var n int
	err := u.db.QueryRow(`SELECT COUNT(*) FROM reviews WHERE user_name = ? AND substr(reviewed_at,1,10) = ?`,
		u.user, today.Format(dateFmt)).Scan(&n)
	return n, err
}

// StreakDays counts consecutive days (ending today) with at least one review.
func (u *UserStore) StreakDays(today time.Time) (int, error) {
	rows, err := u.db.Query(`SELECT DISTINCT substr(reviewed_at,1,10) AS d FROM reviews WHERE user_name = ? ORDER BY d DESC`, u.user)
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
func (u *UserStore) AddAttempt(a Attempt, now time.Time) error {
	correct := 0
	if a.Correct {
		correct = 1
	}
	_, err := u.db.Exec(`INSERT INTO attempts (user_name, exercise_id, lesson, block, answer, correct, attempted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		u.user, a.ExerciseID, a.Lesson, a.Block, a.Answer, correct, now.UTC().Format(time.RFC3339))
	return err
}

// WeakExercises returns exercises failed at least half the time (>= 2 attempts).
func (u *UserStore) WeakExercises(limit int) ([]WeakExercise, error) {
	rows, err := u.db.Query(`
SELECT exercise_id, lesson, SUM(1 - correct) AS wrong, COUNT(*) AS total
FROM attempts
WHERE user_name = ?
GROUP BY exercise_id
HAVING total >= 2 AND wrong * 2 >= total
ORDER BY wrong DESC, total DESC
LIMIT ?`, u.user, limit)
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
func (u *UserStore) SetLessonStatus(lesson, status string, now time.Time) error {
	iso := now.UTC().Format(time.RFC3339)
	var completedCol any
	if status == "done" {
		completedCol = iso
	}
	_, err := u.db.Exec(`
INSERT INTO lesson_progress (user_name, lesson, status, started_at, completed_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(user_name, lesson) DO UPDATE SET
	status = excluded.status,
	started_at = COALESCE(lesson_progress.started_at, excluded.started_at),
	completed_at = COALESCE(excluded.completed_at, lesson_progress.completed_at)`,
		u.user, lesson, status, iso, completedCol)
	return err
}

// LessonStatuses returns lesson id -> status for every tracked lesson.
func (u *UserStore) LessonStatuses() (map[string]string, error) {
	rows, err := u.db.Query(`SELECT lesson, status FROM lesson_progress WHERE user_name = ?`, u.user)
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
func (u *UserStore) LessonStatus(lesson string) (string, error) {
	var st string
	err := u.db.QueryRow(`SELECT status FROM lesson_progress WHERE user_name = ? AND lesson = ?`, u.user, lesson).Scan(&st)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return st, err
}

// ActivityByDay returns per-day activity counts on or after `since`
// (reviews and exercise attempts combined), oldest first.
func (u *UserStore) ActivityByDay(since time.Time) ([]DayActivity, error) {
	sinceISO := since.Format(time.RFC3339)
	rows, err := u.db.Query(`
SELECT d, SUM(n) FROM (
	SELECT substr(reviewed_at,1,10) AS d, COUNT(*) AS n FROM reviews WHERE user_name = ? AND reviewed_at >= ? GROUP BY d
	UNION ALL
	SELECT substr(attempted_at,1,10) AS d, COUNT(*) AS n FROM attempts WHERE user_name = ? AND attempted_at >= ? GROUP BY d
)
GROUP BY d ORDER BY d ASC`, u.user, sinceISO, u.user, sinceISO)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DayActivity
	for rows.Next() {
		var a DayActivity
		if err := rows.Scan(&a.Date, &a.Count); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// NewCount returns how many cards are still in the "new" state.
func (u *UserStore) NewCount() (int, error) {
	var n int
	err := u.db.QueryRow(`SELECT COUNT(*) FROM srs_cards WHERE user_name = ? AND state = 'new'`, u.user).Scan(&n)
	return n, err
}

// DueCount returns how many learning/review cards are due on or before today.
func (u *UserStore) DueCount(today time.Time) (int, error) {
	var n int
	err := u.db.QueryRow(`SELECT COUNT(*) FROM srs_cards
		WHERE user_name = ? AND state IN ('learning','review') AND (due IS NULL OR due <= ?)`,
		u.user, today.Format(dateFmt)).Scan(&n)
	return n, err
}

// CardStats returns the total number of cards and how many are "known"
// (in review with an interval of at least a week).
func (u *UserStore) CardStats() (total, known int, err error) {
	if err = u.db.QueryRow(`SELECT COUNT(*) FROM srs_cards WHERE user_name = ?`, u.user).Scan(&total); err != nil {
		return
	}
	err = u.db.QueryRow(`SELECT COUNT(*) FROM srs_cards WHERE user_name = ? AND state = 'review' AND interval_days >= 7`, u.user).Scan(&known)
	return
}
