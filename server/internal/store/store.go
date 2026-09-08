// Package store persists mutable app state (SRS schedule, exercise
// attempts, lesson progress), scoped per account name.
//
// Two backends are supported behind one code path: PostgreSQL (production,
// selected by a "postgres://" / "postgresql://" DSN) and SQLite (local dev
// and tests, any other DSN incl. ":memory:"). Query strings are written with
// "?" placeholders and rebound to "$N" for Postgres; see rebind.
package store

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // "pgx" driver
	_ "modernc.org/sqlite"

	"github.com/grisha/serbian-app/server/internal/srs"
)

const (
	dateFmt     = "2006-01-02"
	schemaVer   = 2
	legacyOwner = "Гриша" // v1 rows (no account) are migrated to this account
)

// Store owns the database connection and the account list.
type Store struct {
	db *database
}

// UserStore is a per-account view over the state tables.
type UserStore struct {
	db   *database
	user string
}

// IsPostgresDSN reports whether dsn selects the Postgres backend.
func IsPostgresDSN(dsn string) bool {
	return strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://")
}

// database wraps *sql.DB, rebinding "?" placeholders to "$N" on Postgres so
// the rest of the package can use one placeholder style.
type database struct {
	sqlDB *sql.DB
	pg    bool
}

func (d *database) Exec(q string, a ...any) (sql.Result, error) {
	return d.sqlDB.Exec(rebind(q, d.pg), a...)
}
func (d *database) Query(q string, a ...any) (*sql.Rows, error) {
	return d.sqlDB.Query(rebind(q, d.pg), a...)
}
func (d *database) QueryRow(q string, a ...any) *sql.Row {
	return d.sqlDB.QueryRow(rebind(q, d.pg), a...)
}
func (d *database) Begin() (*dbtx, error) {
	tx, err := d.sqlDB.Begin()
	if err != nil {
		return nil, err
	}
	return &dbtx{tx: tx, pg: d.pg}, nil
}
func (d *database) Close() error { return d.sqlDB.Close() }

// dbtx is the transaction-scoped counterpart of database.
type dbtx struct {
	tx *sql.Tx
	pg bool
}

func (t *dbtx) Exec(q string, a ...any) (sql.Result, error) {
	return t.tx.Exec(rebind(q, t.pg), a...)
}
func (t *dbtx) Query(q string, a ...any) (*sql.Rows, error) {
	return t.tx.Query(rebind(q, t.pg), a...)
}
func (t *dbtx) QueryRow(q string, a ...any) *sql.Row {
	return t.tx.QueryRow(rebind(q, t.pg), a...)
}
func (t *dbtx) Prepare(q string) (*sql.Stmt, error) { return t.tx.Prepare(rebind(q, t.pg)) }
func (t *dbtx) Commit() error                       { return t.tx.Commit() }
func (t *dbtx) Rollback() error                     { return t.tx.Rollback() }

// rebind converts "?" placeholders to "$1, $2, …" for Postgres. It skips
// question marks inside single-quoted string literals. No-op for SQLite.
func rebind(q string, pg bool) string {
	if !pg || !strings.ContainsRune(q, '?') {
		return q
	}
	var b strings.Builder
	b.Grow(len(q) + 8)
	n, inQuote := 0, false
	for i := 0; i < len(q); i++ {
		c := q[i]
		switch {
		case c == '\'':
			inQuote = !inQuote
			b.WriteByte(c)
		case c == '?' && !inQuote:
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// execScript runs a multi-statement SQL string one statement at a time
// (the Postgres wire protocol rejects multiple commands per Exec).
func (d *database) execScript(script string) error {
	for _, stmt := range strings.Split(script, ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := d.sqlDB.Exec(stmt); err != nil {
			return fmt.Errorf("%s: %w", firstLine(stmt), err)
		}
	}
	return nil
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

// Open connects to the database named by dsn and applies the schema.
// A "postgres://" / "postgresql://" dsn uses PostgreSQL; anything else is
// treated as a SQLite path (":memory:" included).
func Open(dsn string) (*Store, error) {
	pg := IsPostgresDSN(dsn)
	driver := "sqlite"
	if pg {
		driver = "pgx"
	}
	sqlDB, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	if pg {
		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	} else {
		sqlDB.SetMaxOpenConns(1) // modernc sqlite + WAL: single writer
		if _, err := sqlDB.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`); err != nil {
			return nil, err
		}
	}
	s := &Store{db: &database{sqlDB: sqlDB, pg: pg}}
	if err := s.applySchema(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

// schemaSQL returns the CREATE statements for the given backend. The only
// dialect difference is the autoincrement id column.
func schemaSQL(pg bool) string {
	id := "INTEGER PRIMARY KEY AUTOINCREMENT"
	if pg {
		id = "BIGINT GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY"
	}
	return `
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
	id          ` + id + `,
	user_name   TEXT NOT NULL,
	card_id     TEXT NOT NULL,
	grade       INTEGER NOT NULL,
	reviewed_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS attempts (
	id           ` + id + `,
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
CREATE TABLE IF NOT EXISTS lesson_step_progress (
	user_name    TEXT NOT NULL,
	lesson       TEXT NOT NULL,
	step         TEXT NOT NULL,
	status       TEXT NOT NULL,
	completed_at TEXT,
	PRIMARY KEY (user_name, lesson, step)
);
CREATE INDEX IF NOT EXISTS reviews_user ON reviews(user_name);
CREATE INDEX IF NOT EXISTS attempts_user ON attempts(user_name);
`
}

func (s *Store) applySchema() error {
	if s.db.pg {
		// Fresh or existing Postgres: the CREATE ... IF NOT EXISTS script is
		// idempotent. No v1 legacy path — that only ever existed on SQLite.
		return s.db.execScript(schemaSQL(true))
	}

	var ver int
	_ = s.db.QueryRow(`PRAGMA user_version`).Scan(&ver)
	if ver == 1 {
		if err := s.migrateV1toV2(); err != nil {
			return fmt.Errorf("migrate v1->v2: %w", err)
		}
	}
	if _, err := s.db.Exec(schemaSQL(false)); err != nil {
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
		schemaSQL(false),
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
	_, err := s.db.Exec(
		`INSERT INTO users (name, created_at) VALUES (?, ?) ON CONFLICT (name) DO NOTHING`,
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

// UserProgress is a one-line summary of one account's progress.
type UserProgress struct {
	Name          string
	LessonsDone   int
	CardsKnown    int
	TotalCards    int
	StreakDays    int
	ReviewedToday int
	LastActive    string // ISO date; "" if never active
}

// AllUsersProgress returns a progress summary per account, best first
// (lessons done, then cards known).
func (s *Store) AllUsersProgress(today time.Time) ([]UserProgress, error) {
	names, err := s.ListUsers()
	if err != nil {
		return nil, err
	}
	lessons := map[string]int{}
	scanCount(s.db, `SELECT user_name, COUNT(*) FROM lesson_progress WHERE status='done' GROUP BY user_name`, lessons)

	known := map[string]int{}
	total := map[string]int{}
	if rows, err := s.db.Query(`SELECT user_name, COUNT(*),
		SUM(CASE WHEN state='review' AND interval_days>=7 THEN 1 ELSE 0 END)
		FROM srs_cards GROUP BY user_name`); err == nil {
		for rows.Next() {
			var n string
			var t, k int
			if rows.Scan(&n, &t, &k) == nil {
				total[n] = t
				known[n] = k
			}
		}
		rows.Close()
	}

	reviewedToday := map[string]int{}
	scanCount(s.db, `SELECT user_name, COUNT(*) FROM reviews WHERE substr(reviewed_at,1,10)='`+
		today.Format(dateFmt)+`' GROUP BY user_name`, reviewedToday)

	lastActive := map[string]string{}
	if rows, err := s.db.Query(`SELECT user_name, MAX(d) FROM (
		SELECT user_name, substr(reviewed_at,1,10) d FROM reviews
		UNION ALL SELECT user_name, substr(attempted_at,1,10) FROM attempts
	) GROUP BY user_name`); err == nil {
		for rows.Next() {
			var n, d string
			if rows.Scan(&n, &d) == nil {
				lastActive[n] = d
			}
		}
		rows.Close()
	}

	out := make([]UserProgress, 0, len(names))
	for _, n := range names {
		streak, _ := s.User(n).StreakDays(today)
		out = append(out, UserProgress{
			Name: n, LessonsDone: lessons[n], CardsKnown: known[n], TotalCards: total[n],
			StreakDays: streak, ReviewedToday: reviewedToday[n], LastActive: lastActive[n],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LessonsDone != out[j].LessonsDone {
			return out[i].LessonsDone > out[j].LessonsDone
		}
		return out[i].CardsKnown > out[j].CardsKnown
	})
	return out, nil
}

func scanCount(db *database, q string, into map[string]int) {
	rows, err := db.Query(q)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		var v int
		if rows.Scan(&k, &v) == nil {
			into[k] = v
		}
	}
}

// ---- per-account state ----

// EnsureCards inserts any missing cards in state "new".
func (u *UserStore) EnsureCards(seeds []CardSeed) error {
	tx, err := u.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT INTO srs_cards (user_name, card_id, kind, ref_id, updated_at)
		VALUES (?, ?, ?, ?, ?) ON CONFLICT (user_name, card_id) DO NOTHING`)
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

// ActivateCard makes a card immediately reviewable: it upserts the card and, if
// it is still "new", moves it to "learning" due today so it turns up in the next
// review session without waiting for the daily new-card draw. A card the learner
// has already started (learning/review) is left untouched. Returns whether it
// changed anything.
func (u *UserStore) ActivateCard(seed CardSeed, now time.Time) (bool, error) {
	tx, err := u.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`INSERT INTO srs_cards (user_name, card_id, kind, ref_id, updated_at)
		VALUES (?, ?, ?, ?, ?) ON CONFLICT (user_name, card_id) DO NOTHING`,
		u.user, seed.CardID, seed.Kind, seed.RefID, now.UTC().Format(time.RFC3339)); err != nil {
		return false, err
	}

	var state string
	if err := tx.QueryRow(`SELECT state FROM srs_cards WHERE user_name = ? AND card_id = ?`,
		u.user, seed.CardID).Scan(&state); err != nil {
		return false, err
	}
	if state != string(srs.New) {
		return false, tx.Commit()
	}

	if _, err := tx.Exec(`UPDATE srs_cards SET state=?, due=?, updated_at=? WHERE user_name=? AND card_id=?`,
		string(srs.Learning), now.Format(dateFmt), now.UTC().Format(time.RFC3339), u.user, seed.CardID); err != nil {
		return false, err
	}
	return true, tx.Commit()
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
GROUP BY exercise_id, lesson
HAVING COUNT(*) >= 2 AND SUM(1 - correct) * 2 >= COUNT(*)
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

// SetStepStatus upserts one lesson step's status ("in_progress" | "done").
// completed_at is stamped once, on the first transition to "done".
func (u *UserStore) SetStepStatus(lesson, step, status string, now time.Time) error {
	var completedCol any
	if status == "done" {
		completedCol = now.UTC().Format(time.RFC3339)
	}
	_, err := u.db.Exec(`
INSERT INTO lesson_step_progress (user_name, lesson, step, status, completed_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(user_name, lesson, step) DO UPDATE SET
	status = excluded.status,
	completed_at = COALESCE(lesson_step_progress.completed_at, excluded.completed_at)`,
		u.user, lesson, step, status, completedCol)
	return err
}

// StepStatuses returns step id -> status for one lesson.
func (u *UserStore) StepStatuses(lesson string) (map[string]string, error) {
	rows, err := u.db.Query(`SELECT step, status FROM lesson_step_progress WHERE user_name = ? AND lesson = ?`, u.user, lesson)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var step, status string
		if err := rows.Scan(&step, &status); err != nil {
			return nil, err
		}
		out[step] = status
	}
	return out, rows.Err()
}

// AttemptSummary is the learner's most recent answer to one exercise.
type AttemptSummary struct {
	Answer  string
	Correct bool
}

// LessonAttempts returns the latest attempt per exercise for a lesson.
func (u *UserStore) LessonAttempts(lesson string) (map[string]AttemptSummary, error) {
	rows, err := u.db.Query(`
SELECT a.exercise_id, a.answer, a.correct
FROM attempts a
JOIN (SELECT exercise_id, MAX(id) AS mid FROM attempts
      WHERE user_name = ? AND lesson = ? GROUP BY exercise_id) last
  ON a.id = last.mid`, u.user, lesson)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]AttemptSummary{}
	for rows.Next() {
		var id string
		var s AttemptSummary
		var c int
		if err := rows.Scan(&id, &s.Answer, &c); err != nil {
			return nil, err
		}
		s.Correct = c != 0
		out[id] = s
	}
	return out, nil
}

// ResetLesson clears all attempts and progress for one lesson.
func (u *UserStore) ResetLesson(lesson string) error {
	tx, err := u.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM attempts WHERE user_name = ? AND lesson = ?`, u.user, lesson); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM lesson_progress WHERE user_name = ? AND lesson = ?`, u.user, lesson); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM lesson_step_progress WHERE user_name = ? AND lesson = ?`, u.user, lesson); err != nil {
		return err
	}
	return tx.Commit()
}

// ResetExercises clears every attempt and lesson-progress row for the account
// (SRS word cards are left untouched).
func (u *UserStore) ResetExercises() error {
	tx, err := u.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM attempts WHERE user_name = ?`, u.user); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM lesson_progress WHERE user_name = ?`, u.user); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM lesson_step_progress WHERE user_name = ?`, u.user); err != nil {
		return err
	}
	return tx.Commit()
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
