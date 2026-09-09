package store

import (
	"database/sql"
	"slices"
	"testing"

	"github.com/grisha/serbian-app/server/internal/auth"
)

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

// TestMigration002RekeysStateTablesMultiUser drives migrate002's row-copy path
// (scan every legacy row into []any through the driver, re-insert into *_new
// with the resolved user_id) on the backend production actually uses. With
// TEST_DATABASE_URL set it runs against real Postgres (pgx/v5) — the empty-table
// PG coverage in TestMigration002Schema never enters those copy loops. On SQLite
// it runs too, as a fresh in-memory db. Seeds multi-row / multi-user name-keyed
// data across all 5 state tables plus orphan rows (a user_name with no users
// row), then asserts the re-key, the orphan drop, row-order preservation, and
// the identities FK.
func TestMigration002RekeysStateTablesMultiUser(t *testing.T) {
	s := openPreMigration002Store(t) // migration 001 applied, 002 pending

	// two real accounts + an orphan name that owns state rows but no users row
	mustExec(t, s, `INSERT INTO users (name, created_at) VALUES (?, ?)`, "Гриша", "2026-09-06T10:00:00Z")
	mustExec(t, s, `INSERT INTO users (name, created_at) VALUES (?, ?)`, "Оля", "2026-09-07T10:00:00Z")
	const orphan = "Никита"

	// srs_cards — REAL ease, INTEGER counters, nullable TEXT due, all through []any
	seedCard := func(name, cardID string, ease float64, interval, reps, lapses int, state string, due any) {
		mustExec(t, s, `INSERT INTO srs_cards
			(user_name, card_id, kind, ref_id, ease, interval_days, reps, lapses, state, due, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			name, cardID, "vocab", "ref-"+cardID, ease, interval, reps, lapses, state, due, "2026-09-08T08:00:00Z")
	}
	seedCard("Гриша", "g1", 2.5, 0, 0, 0, "new", nil)
	seedCard("Гриша", "g2", 2.25, 6, 3, 1, "review", "2026-09-20")
	seedCard("Оля", "o1", 1.75, 1, 5, 4, "learning", "2026-09-09")
	seedCard(orphan, "x1", 2.5, 0, 0, 0, "new", nil)

	// reviews — fresh autoincrement id in *_new; insert order must survive
	for _, r := range []struct {
		name, card string
		grade      int
	}{
		{"Гриша", "g1", 5}, {"Оля", "o1", 4}, {"Гриша", "g2", 3}, {orphan, "x1", 2}, {"Оля", "o1", 5},
	} {
		mustExec(t, s, `INSERT INTO reviews (user_name, card_id, grade, reviewed_at) VALUES (?, ?, ?, ?)`,
			r.name, r.card, r.grade, "2026-09-08T09:00:00Z")
	}

	// attempts — fresh autoincrement id + INTEGER correct
	for _, a := range []struct {
		name, ex, lesson, block, answer string
		correct                         int
	}{
		{"Гриша", "01-A-1", "01", "A", "a0", 0},
		{"Оля", "02-B-1", "02", "B", "b0", 1},
		{"Гриша", "01-A-1", "01", "A", "a1", 1},
		{orphan, "09-Z-1", "09", "Z", "z0", 0},
	} {
		mustExec(t, s, `INSERT INTO attempts
			(user_name, exercise_id, lesson, block, answer, correct, attempted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			a.name, a.ex, a.lesson, a.block, a.answer, a.correct, "2026-09-08T09:00:00Z")
	}

	// composite-PK tables — nullable timestamps
	mustExec(t, s, `INSERT INTO lesson_progress (user_name, lesson, status, started_at, completed_at) VALUES (?, ?, ?, ?, ?)`,
		"Гриша", "01", "done", "2026-09-06T10:00:00Z", "2026-09-06T11:00:00Z")
	mustExec(t, s, `INSERT INTO lesson_progress (user_name, lesson, status, started_at, completed_at) VALUES (?, ?, ?, ?, ?)`,
		"Оля", "02", "in_progress", "2026-09-07T10:00:00Z", nil)
	mustExec(t, s, `INSERT INTO lesson_progress (user_name, lesson, status, started_at, completed_at) VALUES (?, ?, ?, ?, ?)`,
		orphan, "03", "done", "2026-09-06T10:00:00Z", "2026-09-06T11:00:00Z")

	mustExec(t, s, `INSERT INTO lesson_step_progress (user_name, lesson, step, status, completed_at) VALUES (?, ?, ?, ?, ?)`,
		"Гриша", "01", "01.1", "done", "2026-09-06T10:30:00Z")
	mustExec(t, s, `INSERT INTO lesson_step_progress (user_name, lesson, step, status, completed_at) VALUES (?, ?, ?, ?, ?)`,
		"Оля", "02", "02.1", "in_progress", nil)
	mustExec(t, s, `INSERT INTO lesson_step_progress (user_name, lesson, step, status, completed_at) VALUES (?, ?, ?, ?, ?)`,
		orphan, "03", "03.1", "done", nil)

	// --- apply migration 002 (002_auth.sql + migrate002 hook) ---
	if err := s.runMigrations(); err != nil {
		t.Fatalf("runMigrations: %v", err)
	}

	// resolve the minted ids
	idByName := map[string]string{}
	rows, err := s.db.Query(`SELECT id, name FROM users`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			t.Fatal(err)
		}
		if len(id) < 4 || id[:4] != "usr_" {
			t.Errorf("user %q got non-usr id %q", name, id)
		}
		idByName[name] = id
	}
	rows.Close()
	if len(idByName) != 2 {
		t.Fatalf("users after migrate = %v, want Гриша + Оля only", idByName)
	}
	grisha, olya := idByName["Гриша"], idByName["Оля"]

	// (a) every surviving state row carries its former owner's new id
	assertOwner := func(table, where, want string) {
		t.Helper()
		var got string
		if err := s.db.QueryRow(`SELECT user_id FROM ` + table + ` WHERE ` + where).Scan(&got); err != nil {
			t.Fatalf("%s WHERE %s: %v", table, where, err)
		}
		if got != want {
			t.Errorf("%s WHERE %s: user_id = %q, want %q", table, where, got, want)
		}
	}
	assertOwner("srs_cards", "card_id = 'g1'", grisha)
	assertOwner("srs_cards", "card_id = 'g2'", grisha)
	assertOwner("srs_cards", "card_id = 'o1'", olya)
	assertOwner("reviews", "grade = 3", grisha)
	assertOwner("attempts", "answer = 'b0'", olya)
	assertOwner("lesson_progress", "lesson = '01'", grisha)
	assertOwner("lesson_progress", "lesson = '02'", olya)
	assertOwner("lesson_step_progress", "lesson = '01' AND step = '01.1'", grisha)
	assertOwner("lesson_step_progress", "lesson = '02' AND step = '02.1'", olya)

	// REAL / INTEGER / non-null + NULL TEXT all round-tripped through the []any scan
	var ease float64
	var interval, reps, lapses int
	var due sql.NullString
	if err := s.db.QueryRow(`SELECT ease, interval_days, reps, lapses, due FROM srs_cards WHERE card_id = 'g2'`).
		Scan(&ease, &interval, &reps, &lapses, &due); err != nil {
		t.Fatal(err)
	}
	if ease != 2.25 || interval != 6 || reps != 3 || lapses != 1 || !due.Valid || due.String != "2026-09-20" {
		t.Errorf("g2 copied wrong: ease=%v interval=%d reps=%d lapses=%d due=%v", ease, interval, reps, lapses, due)
	}
	if err := s.db.QueryRow(`SELECT due FROM srs_cards WHERE card_id = 'g1'`).Scan(&due); err != nil {
		t.Fatal(err)
	}
	if due.Valid {
		t.Errorf("g1 due = %q, want NULL", due.String)
	}

	// (b) orphan-keyed rows dropped from every state table
	for _, q := range []string{
		`SELECT COUNT(*) FROM srs_cards WHERE card_id = 'x1'`,
		`SELECT COUNT(*) FROM reviews WHERE grade = 2`,
		`SELECT COUNT(*) FROM attempts WHERE lesson = '09'`,
		`SELECT COUNT(*) FROM lesson_progress WHERE lesson = '03'`,
		`SELECT COUNT(*) FROM lesson_step_progress WHERE lesson = '03'`,
	} {
		var n int
		if err := s.db.QueryRow(q).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("orphan row survived: %s -> %d", q, n)
		}
	}

	// row order preserved after id re-assignment (store.go treats MAX(id) as latest)
	if got := queryInts(t, s, `SELECT grade FROM reviews ORDER BY id`); !slices.Equal(got, []int{5, 4, 3, 5}) {
		t.Errorf("reviews grades in id order = %v, want [5 4 3 5]", got)
	}
	if got := queryStrings(t, s, `SELECT answer FROM attempts ORDER BY id`); !slices.Equal(got, []string{"a0", "b0", "a1"}) {
		t.Errorf("attempts answers in id order = %v, want [a0 b0 a1]", got)
	}

	// (c) identities FK rejects an unknown user_id, accepts a real one
	if _, err := s.db.Exec(`INSERT INTO identities (id, user_id, provider, provider_uid, created_at)
		VALUES (?, ?, ?, ?, ?)`, "idn_bad", "usr_missing", "password", "p1", "2026-09-09T00:00:00Z"); err == nil {
		t.Error("identities insert with unknown user_id succeeded, want FK violation")
	}
	if _, err := s.db.Exec(`INSERT INTO identities (id, user_id, provider, provider_uid, created_at)
		VALUES (?, ?, ?, ?, ?)`, "idn_ok", grisha, "password", "p2", "2026-09-09T00:00:00Z"); err != nil {
		t.Errorf("identities insert with real user_id failed: %v", err)
	}
}

func TestMigration003Accounts(t *testing.T) {
	s := &Store{db: mustOpenRaw(t)}
	if err := s.runMigrationsUpTo(2); err != nil {
		t.Fatal(err)
	}

	seed := func(name string) string {
		id := auth.NewUserID()
		mustExec(t, s, `INSERT INTO users (id, name, created_at) VALUES (?, ?, '2026-09-06T10:00:00Z')`, id, name)
		return id
	}
	grishaID := seed("Гриша")
	seed("Алина")
	seed("DeployCheck")
	chk2ID := seed("chk2")
	mustExec(t, s, `INSERT INTO lesson_progress (user_id, lesson, status) VALUES (?, '01', 'done')`, chk2ID)

	if err := s.runMigrations(); err != nil {
		t.Fatal(err)
	} // applies version 3

	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE name IN ('DeployCheck','chk2')`).Scan(&n)
	if n != 0 {
		t.Fatalf("junk users left: %d", n)
	}
	s.db.QueryRow(`SELECT COUNT(*) FROM lesson_progress`).Scan(&n)
	if n != 0 {
		t.Fatalf("junk state left: %d", n)
	}
	var puid string
	if err := s.db.QueryRow(
		`SELECT provider_uid FROM identities WHERE user_id = ? AND provider = 'telegram'`, grishaID,
	).Scan(&puid); err != nil {
		t.Fatalf("Гриша telegram identity: %v", err)
	}
	if puid != "pending:llladnooo" {
		t.Fatalf("provider_uid = %q", puid)
	}

	// idempotent: re-running the runner does not re-fire the hook or duplicate identities
	if err := s.runMigrations(); err != nil {
		t.Fatalf("re-run: %v", err)
	}
	s.db.QueryRow(`SELECT COUNT(*) FROM identities WHERE user_id = ? AND provider = 'telegram'`, grishaID).Scan(&n)
	if n != 1 {
		t.Fatalf("telegram identities for Гриша = %d, want 1", n)
	}
}
