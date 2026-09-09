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
