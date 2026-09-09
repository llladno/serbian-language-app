package store

import "testing"

func TestMigrationsIdempotent(t *testing.T) {
	s := newStore(t) // newStore already calls Open -> runMigrations
	// second Open on the same in-memory db is a fresh db, so instead
	// re-run the runner directly and assert no error / no dup rows.
	if err := s.runMigrations(); err != nil {
		t.Fatalf("re-run: %v", err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("schema_migrations empty")
	}
	// core tables from 001 exist
	for _, tbl := range []string{"users", "srs_cards", "reviews", "attempts", "lesson_progress", "lesson_step_progress"} {
		rows, err := s.db.Query(`SELECT 1 FROM ` + tbl + ` LIMIT 1`)
		if err != nil {
			t.Fatalf("table %s missing: %v", tbl, err)
		}
		rows.Close() // release the pooled conn (SQLite pool is size 1)
	}
}

func TestMigrationsRecordVersions(t *testing.T) {
	s := newStore(t)
	rows, err := s.db.Query(`SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []int
	for rows.Next() {
		var v int
		_ = rows.Scan(&v)
		got = append(got, v)
	}
	if len(got) == 0 || got[0] != 1 {
		t.Fatalf("versions = %v, want first = 1", got)
	}
}

// queryOK runs q and reports whether it prepared/executed without error,
// releasing the pooled connection (SQLite pool size is 1).
func queryOK(s *Store, q string) error {
	rows, err := s.db.Query(q)
	if err != nil {
		return err
	}
	return rows.Close()
}

func TestMigration002Schema(t *testing.T) {
	s := newStore(t)
	// new tables
	for _, tbl := range []string{"identities", "sessions", "email_tokens"} {
		if err := queryOK(s, `SELECT 1 FROM `+tbl+` LIMIT 1`); err != nil {
			t.Fatalf("table %s missing: %v", tbl, err)
		}
	}
	// users has id, state tables have user_id
	if err := queryOK(s, `SELECT id, name, created_at FROM users LIMIT 1`); err != nil {
		t.Fatalf("users.id missing: %v", err)
	}
	for _, tbl := range []string{"srs_cards", "reviews", "attempts", "lesson_progress", "lesson_step_progress"} {
		if err := queryOK(s, `SELECT user_id FROM `+tbl+` LIMIT 1`); err != nil {
			t.Fatalf("%s.user_id missing: %v", tbl, err)
		}
	}
}

func TestMigration002RekeysExistingData(t *testing.T) {
	// Build a pre-002 db by hand (SQLite in-memory): run only 001, seed
	// name-keyed rows, then run the rest.
	s := &Store{db: mustOpenRaw(t)}
	if err := s.runMigrationsUpTo(1); err != nil {
		t.Fatal(err)
	}
	mustExec(t, s, `INSERT INTO users (name, created_at) VALUES ('Гриша', '2026-09-06T10:00:00Z')`)
	mustExec(t, s, `INSERT INTO lesson_progress (user_name, lesson, status) VALUES ('Гриша', '01', 'done')`)
	if err := s.runMigrations(); err != nil {
		t.Fatal(err)
	}

	var uid string
	if err := s.db.QueryRow(`SELECT id FROM users WHERE name='Гриша'`).Scan(&uid); err != nil {
		t.Fatal(err)
	}
	if len(uid) < 4 || uid[:4] != "usr_" {
		t.Fatalf("bad user id %q", uid)
	}
	var got string
	if err := s.db.QueryRow(`SELECT user_id FROM lesson_progress WHERE lesson='01'`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != uid {
		t.Fatalf("progress user_id = %q, want %q", got, uid)
	}
}
