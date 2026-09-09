package store

import (
	"database/sql"
	"testing"
)

// mustOpenRaw opens a bare in-memory SQLite database wrapped as *database, with
// no schema applied, so a migration can be driven step by step from a test.
func mustOpenRaw(t *testing.T) *database {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1) // :memory: is per-connection; keep one conn
	db.Exec(`PRAGMA foreign_keys=ON`)
	t.Cleanup(func() { db.Close() })
	return &database{sqlDB: db, pg: false}
}

// mustExec runs a statement against the store or fails the test.
func mustExec(t *testing.T, s *Store, q string, a ...any) {
	t.Helper()
	if _, err := s.db.Exec(q, a...); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
}

// openPreMigration002Store returns a Store whose database has migration 001
// applied but not 002, on whichever backend testDSN() selects. On Postgres it
// drops and recreates the "public" schema first, so migrate002's row-copy path
// (the []any scan-then-reinsert) is exercised against a real pgx connection; a
// t.Cleanup rebuilds the full schema afterwards for the rest of the suite.
func openPreMigration002Store(t *testing.T) *Store {
	t.Helper()
	dsn := testDSN()
	if !IsPostgresDSN(dsn) {
		s := &Store{db: mustOpenRaw(t)}
		if err := s.runMigrationsUpTo(1); err != nil {
			t.Fatalf("migrate up to 1: %v", err)
		}
		return s
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open pg: %v", err)
	}
	db.SetMaxOpenConns(4)
	reset := func() {
		if _, err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
			t.Fatalf("reset public schema: %v", err)
		}
	}
	reset()
	t.Cleanup(func() {
		reset()
		restored, err := Open(dsn) // reapplies 001+002 for the next test
		if err != nil {
			t.Fatalf("restore schema: %v", err)
		}
		restored.Close()
		db.Close()
	})
	s := &Store{db: &database{sqlDB: db, pg: true}}
	if err := s.runMigrationsUpTo(1); err != nil {
		t.Fatalf("migrate up to 1: %v", err)
	}
	return s
}

// queryInts / queryStrings collect a single-column result set, in row order.
func queryInts(t *testing.T, s *Store, q string, a ...any) []int {
	t.Helper()
	rows, err := s.db.Query(q, a...)
	if err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, n)
	}
	return out
}

func queryStrings(t *testing.T, s *Store, q string, a ...any) []string {
	t.Helper()
	rows, err := s.db.Query(q, a...)
	if err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, v)
	}
	return out
}

// openRawV1 creates a database with the v1 (accountless) schema so the
// migration path can be exercised.
func openRawV1(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`
CREATE TABLE srs_cards (
	card_id TEXT PRIMARY KEY, kind TEXT NOT NULL, ref_id TEXT NOT NULL,
	ease REAL NOT NULL DEFAULT 2.5, interval_days INTEGER NOT NULL DEFAULT 0,
	reps INTEGER NOT NULL DEFAULT 0, lapses INTEGER NOT NULL DEFAULT 0,
	state TEXT NOT NULL DEFAULT 'new', due TEXT, updated_at TEXT NOT NULL
);
CREATE TABLE reviews (
	id INTEGER PRIMARY KEY AUTOINCREMENT, card_id TEXT NOT NULL,
	grade INTEGER NOT NULL, reviewed_at TEXT NOT NULL
);
CREATE TABLE attempts (
	id INTEGER PRIMARY KEY AUTOINCREMENT, exercise_id TEXT NOT NULL, lesson TEXT NOT NULL,
	block TEXT NOT NULL, answer TEXT NOT NULL, correct INTEGER NOT NULL, attempted_at TEXT NOT NULL
);
CREATE TABLE lesson_progress (
	lesson TEXT PRIMARY KEY, status TEXT NOT NULL, started_at TEXT, completed_at TEXT
);
PRAGMA user_version = 1;
`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
