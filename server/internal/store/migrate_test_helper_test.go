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
