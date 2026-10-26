package store

import (
	"database/sql"
	"fmt"
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
	// cap at 2: this test is only about the user_name -> user_id rekey: a
	// later migrate004 run would relabel lesson "01" itself and the query
	// below would find nothing.
	if err := s.runMigrationsUpTo(2); err != nil {
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
	// Capped at 2: a later migrate004 run would relabel these lesson ids
	// ("01".."03", "09") out from under the assertions below, which are only
	// about the user_name -> user_id rekey.
	if err := s.runMigrationsUpTo(2); err != nil {
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

// TestMigration004RelabelsLessons seeds progress against the pre-split lesson
// ids ("00".."12") and checks migrate004 relabels it to the post-split ids
// ("00".."29") from the 2026-09-21 lesson-split (see
// docs/superpowers/specs/2026-09-21-split-lessons-00-12-design.md): a
// whole-lesson lesson_progress row fans out across every new part, while
// lesson_step_progress and attempts follow the exact step/exercise renumbering.
func TestMigration004RelabelsLessons(t *testing.T) {
	s := &Store{db: mustOpenRaw(t)}
	if err := s.runMigrationsUpTo(3); err != nil {
		t.Fatal(err)
	}
	uid := auth.NewUserID()
	mustExec(t, s, `INSERT INTO users (id, name, created_at) VALUES (?, 'Гриша', '2026-09-06T10:00:00Z')`, uid)

	// old lesson "02" ("Ko si ti") split into new "04" + "05" — a whole-lesson
	// status must fan out to both.
	mustExec(t, s, `INSERT INTO lesson_progress (user_id, lesson, status, started_at, completed_at)
		VALUES (?, '02', 'done', '2026-09-06T10:00:00Z', '2026-09-06T10:20:00Z')`, uid)

	// old "04.11" (Který čas, first step of the 3rd of 3 parts) -> new "10.1"
	mustExec(t, s, `INSERT INTO lesson_step_progress (user_id, lesson, step, status, completed_at)
		VALUES (?, '04', '04.11', 'done', '2026-09-06T11:00:00Z')`, uid)
	// old "05.9" (2nd part of the split, local 9 of range 5-10) -> new "12.5"
	mustExec(t, s, `INSERT INTO lesson_step_progress (user_id, lesson, step, status, completed_at)
		VALUES (?, '05', '05.9', 'in_progress', NULL)`, uid)

	// old "09.16.3" (dialogue turn 3 of step 09.16, 3rd of 3 parts) -> "22.8.3"
	mustExec(t, s, `INSERT INTO attempts (user_id, exercise_id, lesson, block, answer, correct, attempted_at)
		VALUES (?, '09.16.3', '09', '09.16', 'Jedan sok, molim.', 1, '2026-09-06T12:00:00Z')`, uid)

	if err := s.runMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	} // applies version 4

	// lesson_progress fanned out, old id gone
	rows, err := s.db.Query(`SELECT lesson, status FROM lesson_progress WHERE user_id = ? ORDER BY lesson`, uid)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for rows.Next() {
		var lesson, status string
		if err := rows.Scan(&lesson, &status); err != nil {
			t.Fatal(err)
		}
		got[lesson] = status
	}
	rows.Close()
	if want := map[string]string{"04": "done", "05": "done"}; len(got) != len(want) || got["04"] != want["04"] || got["05"] != want["05"] {
		t.Errorf("lesson_progress after migrate = %v, want %v", got, want)
	}

	// lesson_step_progress renumbered
	assertStep := func(lesson, step, wantStatus string) {
		t.Helper()
		var status string
		if err := s.db.QueryRow(`SELECT status FROM lesson_step_progress WHERE user_id = ? AND lesson = ? AND step = ?`,
			uid, lesson, step).Scan(&status); err != nil {
			t.Fatalf("lesson_step_progress %s/%s: %v", lesson, step, err)
		}
		if status != wantStatus {
			t.Errorf("lesson_step_progress %s/%s status = %q, want %q", lesson, step, status, wantStatus)
		}
	}
	assertStep("10", "10.1", "done")
	assertStep("12", "12.5", "in_progress")
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM lesson_step_progress WHERE lesson IN ('04','05')`).Scan(&n)
	if n != 0 {
		t.Errorf("old-id lesson_step_progress rows left: %d", n)
	}

	// attempts renumbered
	var lesson, block, exID string
	if err := s.db.QueryRow(`SELECT lesson, block, exercise_id FROM attempts WHERE user_id = ?`, uid).
		Scan(&lesson, &block, &exID); err != nil {
		t.Fatalf("attempts: %v", err)
	}
	if lesson != "22" || block != "22.8" || exID != "22.8.3" {
		t.Errorf("attempt = lesson=%q block=%q exercise_id=%q, want 22/22.8/22.8.3", lesson, block, exID)
	}

	// idempotent: re-running does not re-fire the hook or duplicate rows
	if err := s.runMigrations(); err != nil {
		t.Fatalf("re-run: %v", err)
	}
	s.db.QueryRow(`SELECT COUNT(*) FROM lesson_progress WHERE user_id = ?`, uid).Scan(&n)
	if n != 2 {
		t.Fatalf("lesson_progress rows after re-run = %d, want 2", n)
	}
}

func TestMigration012Schema(t *testing.T) {
	s := newStore(t)
	if err := queryOK(s, `SELECT id, text, created_at FROM broadcasts LIMIT 1`); err != nil {
		t.Fatalf("broadcasts table missing: %v", err)
	}
	if err := queryOK(s, `SELECT broadcast_id FROM bot_outbox LIMIT 1`); err != nil {
		t.Fatalf("bot_outbox.broadcast_id missing: %v", err)
	}
}

// Migration 019 fills the quest's channel in on a database that never had one,
// and leaves a channel somebody already chose alone.
func TestMigration019FillsOnlyAnEmptyChannel(t *testing.T) {
	channelAfter := func(t *testing.T, before string) string {
		t.Helper()
		s := openStoreAtVersion(t, 18)
		mustExec(t, s, `UPDATE economy_settings SET value = ? WHERE key = 'telegram_channel'`, before)
		if err := s.runMigrations(); err != nil {
			t.Fatalf("run migrations: %v", err)
		}
		var v string
		if err := s.db.QueryRow(`SELECT value FROM economy_settings WHERE key = 'telegram_channel'`).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}

	if got := channelAfter(t, ""); got != "@ucimosrb" {
		t.Errorf("empty channel became %q, want @ucimosrb", got)
	}
	if got := channelAfter(t, "@pockets_money"); got != "@pockets_money" {
		t.Errorf("a chosen channel became %q, want it kept", got)
	}
}

// Migration 020 moves a database seeded before Level 4 existed to the new
// prices, rewards and quests, leaves anything edited in the admin panel alone,
// and writes nothing on a database the seed has not run on yet.
func TestMigration020Level4Economy(t *testing.T) {
	seedOld := func(t *testing.T) *Store {
		t.Helper()
		s := openStoreAtVersion(t, 19)
		mustExec(t, s, `INSERT INTO economy_settings (key, value, updated_at) VALUES ('economy_seeded', '1', 'x')`)
		mustExec(t, s, `INSERT INTO economy_settings (key, value, updated_at) VALUES ('economy_lessons_seeded', '1', 'x')`)
		for _, p := range []struct {
			ref, title string
			price      int
		}{{"4", "Уровень 4 — Мнения и жизнь", 500}, {"5", "Уровень 5 — Уверенно", 1000}} {
			mustExec(t, s, `INSERT INTO products (kind, ref, title, description, price, created_at, updated_at)
				VALUES ('phase_unlock', ?, ?, '', ?, 'x', 'x')`, p.ref, p.title, p.price)
		}
		mustExec(t, s, `INSERT INTO quests (kind, target, param, title, reward, created_at, updated_at)
			VALUES ('telegram_subscribed', 1, '', 'Подписаться на канал', 15, 'x', 'x')`)
		mustExec(t, s, `INSERT INTO quests (kind, target, param, title, reward, created_at, updated_at)
			VALUES ('streak_days', 7, '', 'Стрик 7 дней', 10, 'x', 'x')`)
		return s
	}
	count := func(t *testing.T, s *Store, q string, a ...any) int {
		t.Helper()
		var n int
		if err := s.db.QueryRow(q, a...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	t.Run("old seed values are replaced", func(t *testing.T) {
		s := seedOld(t)
		if err := s.runMigrationsUpTo(20); err != nil {
			t.Fatalf("run migrations: %v", err)
		}
		if got := count(t, s, `SELECT price FROM products WHERE kind = 'phase_unlock' AND ref = '4'`); got != 230 {
			t.Errorf("level 4 price = %d, want 230", got)
		}
		if got := count(t, s, `SELECT price FROM products WHERE kind = 'phase_unlock' AND ref = '5'`); got != 375 {
			t.Errorf("level 5 price = %d, want 375", got)
		}
		if got := count(t, s, `SELECT COUNT(*) FROM products WHERE title LIKE '%Падежи в жизни%'`); got != 1 {
			t.Errorf("level 4 title was not renamed")
		}
		if got := count(t, s, `SELECT reward FROM quests WHERE kind = 'telegram_subscribed'`); got != 20 {
			t.Errorf("telegram reward = %d, want 20", got)
		}
		if got := count(t, s, `SELECT reward FROM quests WHERE kind = 'streak_days' AND target = 7`); got != 60 {
			t.Errorf("7-day streak reward = %d, want 60", got)
		}
		if got := count(t, s, `SELECT reward FROM quests WHERE kind = 'streak_days' AND target = 3`); got != 20 {
			t.Errorf("3-day streak reward = %d, want 20", got)
		}
		if got := count(t, s, `SELECT reward FROM quests WHERE kind = 'phase_completed' AND param = '4'`); got != 22 {
			t.Errorf("level 4 completion reward = %d, want 22", got)
		}
		if got := count(t, s, `SELECT COUNT(*) FROM quests WHERE kind = 'lesson_completed'`); got != 18 {
			t.Errorf("lesson reward rows = %d, want 18 (lessons 42..59)", got)
		}
		if got := count(t, s, `SELECT SUM(reward) FROM quests WHERE kind = 'lesson_completed'`); got != 88 {
			t.Errorf("lesson rewards add up to %d, want 88 (14 x 4 + 4 x 8)", got)
		}
	})

	t.Run("admin edits survive", func(t *testing.T) {
		s := seedOld(t)
		mustExec(t, s, `UPDATE products SET price = 700 WHERE kind = 'phase_unlock' AND ref = '4'`)
		mustExec(t, s, `UPDATE quests SET reward = 99 WHERE kind = 'streak_days' AND target = 7`)
		if err := s.runMigrationsUpTo(20); err != nil {
			t.Fatalf("run migrations: %v", err)
		}
		if got := count(t, s, `SELECT price FROM products WHERE kind = 'phase_unlock' AND ref = '4'`); got != 700 {
			t.Errorf("an edited price became %d, want 700 kept", got)
		}
		if got := count(t, s, `SELECT reward FROM quests WHERE kind = 'streak_days' AND target = 7`); got != 99 {
			t.Errorf("an edited reward became %d, want 99 kept", got)
		}
	})

	t.Run("an unseeded database gets nothing", func(t *testing.T) {
		s := openStoreAtVersion(t, 19)
		if err := s.runMigrationsUpTo(20); err != nil {
			t.Fatalf("run migrations: %v", err)
		}
		if got := count(t, s, `SELECT COUNT(*) FROM quests`); got != 0 {
			t.Errorf("%d quests on a database the seed has not run on, want 0", got)
		}
	})

	t.Run("running twice adds nothing", func(t *testing.T) {
		s := seedOld(t)
		if err := s.runMigrationsUpTo(20); err != nil {
			t.Fatal(err)
		}
		before := count(t, s, `SELECT COUNT(*) FROM quests`)
		if err := s.runMigrationsUpTo(20); err != nil {
			t.Fatal(err)
		}
		if after := count(t, s, `SELECT COUNT(*) FROM quests`); after != before {
			t.Errorf("quests went from %d to %d on a second run", before, after)
		}
	})
}

// Migration 021 renames the currency from пёрышки to зёрнышки on a database that
// still holds 017's names, and leaves a name somebody typed into the admin alone.
func TestMigration021RenamesOnlyTheOldDefault(t *testing.T) {
	nameAfter := func(t *testing.T, before string) string {
		t.Helper()
		s := openStoreAtVersion(t, 20)
		mustExec(t, s, `UPDATE economy_settings SET value = ? WHERE key = 'currency_name_many'`, before)
		if err := s.runMigrations(); err != nil {
			t.Fatalf("run migrations: %v", err)
		}
		var v string
		if err := s.db.QueryRow(`SELECT value FROM economy_settings WHERE key = 'currency_name_many'`).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	if got := nameAfter(t, "пёрышек"); got != "зёрнышек" {
		t.Errorf("old default became %q, want зёрнышек", got)
	}
	if got := nameAfter(t, "орешков"); got != "орешков" {
		t.Errorf("a chosen name became %q, want it kept", got)
	}
}

// Migration 022 adds the invite-a-friend quest to a database that has already
// been seeded (a fresh one gets it from the seed), exactly once, and writes
// nothing where the seed has not run yet.
func TestMigration022InviteQuest(t *testing.T) {
	count := func(t *testing.T, s *Store) int {
		t.Helper()
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM quests WHERE kind = 'friends_invited'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	t.Run("seeded database gets the quest once", func(t *testing.T) {
		s := openStoreAtVersion(t, 21)
		mustExec(t, s, `INSERT INTO economy_settings (key, value, updated_at) VALUES ('economy_seeded', '1', 'x')`)
		if err := s.runMigrations(); err != nil {
			t.Fatalf("run migrations: %v", err)
		}
		if got := count(t, s); got != 1 {
			t.Fatalf("invite quests = %d, want 1", got)
		}
		var reward int
		if err := s.db.QueryRow(`SELECT reward FROM quests WHERE kind = 'friends_invited'`).Scan(&reward); err != nil || reward != 30 {
			t.Fatalf("reward = %d (%v), want 30", reward, err)
		}
	})

	t.Run("unseeded database is left to the seed", func(t *testing.T) {
		s := openStoreAtVersion(t, 21)
		if err := s.runMigrations(); err != nil {
			t.Fatalf("run migrations: %v", err)
		}
		if got := count(t, s); got != 0 {
			t.Fatalf("invite quests = %d, want 0 before the seed", got)
		}
	})

	t.Run("an existing invite quest is not duplicated", func(t *testing.T) {
		s := openStoreAtVersion(t, 21)
		mustExec(t, s, `INSERT INTO economy_settings (key, value, updated_at) VALUES ('economy_seeded', '1', 'x')`)
		mustExec(t, s, `INSERT INTO quests (kind, target, param, title, reward, created_at, updated_at)
			VALUES ('friends_invited', 1, '', 'Пригласить друга', 50, 'x', 'x')`)
		if err := s.runMigrations(); err != nil {
			t.Fatalf("run migrations: %v", err)
		}
		if got := count(t, s); got != 1 {
			t.Fatalf("invite quests = %d, want 1", got)
		}
	})
}

// Production sits at migration 012 with real learners in it. Upgrading that
// database to the current schema must keep every row, record every version
// once, and be harmless to run again; the economy is then seeded on top, as at
// boot, and must end up with the final prices and the invite quest. Run it
// against Postgres (TEST_DATABASE_URL) as well: that is what production is.
func TestUpgradeFromProductionAtMigration012(t *testing.T) {
	s := openStoreAtVersion(t, 12)
	const ts = "2026-09-20T10:00:00Z"
	for _, u := range []struct{ id, name, email, tg string }{
		{"usr_a", "Анна", "anna@example.com", ""},
		{"usr_b", "Борис", "", "boris_tg"},
		{"usr_c", "Вера", "vera@example.com", ""},
	} {
		mustExec(t, s, `INSERT INTO users (id, name, created_at) VALUES (?, ?, ?)`, u.id, u.name, ts)
		if u.email != "" {
			mustExec(t, s, `INSERT INTO identities (id, user_id, provider, provider_uid, email, password_hash, email_verified_at, created_at)
				VALUES (?, ?, 'password', ?, ?, 'x', ?, ?)`, "idn_"+u.id, u.id, u.email, u.email, ts, ts)
		} else {
			mustExec(t, s, `INSERT INTO identities (id, user_id, provider, provider_uid, tg_username, created_at)
				VALUES (?, ?, 'telegram', ?, ?, ?)`, "idn_"+u.id, u.id, "1001", u.tg, ts)
		}
	}
	mustExec(t, s, `INSERT INTO lesson_progress (user_id, lesson, status, started_at, completed_at) VALUES ('usr_a', '05', 'done', ?, ?)`, ts, ts)
	mustExec(t, s, `INSERT INTO lesson_progress (user_id, lesson, status, started_at) VALUES ('usr_a', '06', 'in_progress', ?)`, ts)
	mustExec(t, s, `INSERT INTO srs_cards (user_id, card_id, kind, ref_id, state, due, updated_at) VALUES ('usr_a', 'vocab:dan', 'vocab', 'dan', 'review', ?, ?)`, ts, ts)
	mustExec(t, s, `INSERT INTO reviews (user_id, card_id, grade, reviewed_at) VALUES ('usr_a', 'vocab:dan', 4, ?)`, ts)
	mustExec(t, s, `INSERT INTO attempts (user_id, exercise_id, lesson, block, answer, correct, attempted_at) VALUES ('usr_a', '05.2.1', '05', '05.2', 'x', 1, ?)`, ts)

	if err := s.runMigrations(); err != nil {
		t.Fatalf("upgrade 012 -> latest: %v", err)
	}
	if err := s.runMigrations(); err != nil {
		t.Fatalf("second run: %v", err)
	}

	count := func(q string, a ...any) int {
		t.Helper()
		var n int
		if err := s.db.QueryRow(q, a...).Scan(&n); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		return n
	}
	if got := count(`SELECT COUNT(*) FROM users`); got != 3 {
		t.Errorf("users = %d, want 3", got)
	}
	if got := count(`SELECT COUNT(*) FROM identities`); got != 3 {
		t.Errorf("identities = %d, want 3", got)
	}
	if got := count(`SELECT COUNT(*) FROM lesson_progress WHERE user_id = 'usr_a'`); got != 2 {
		t.Errorf("lesson_progress = %d, want 2", got)
	}
	if got := count(`SELECT COUNT(*) FROM srs_cards`); got != 1 {
		t.Errorf("srs_cards = %d, want 1", got)
	}
	if got := count(`SELECT COUNT(*) FROM reviews`); got != 1 {
		t.Errorf("reviews = %d, want 1", got)
	}
	// each version is recorded exactly once, ending at the newest migration
	if got, want := count(`SELECT COUNT(*) FROM schema_migrations`), count(`SELECT COUNT(DISTINCT version) FROM schema_migrations`); got != want {
		t.Errorf("schema_migrations has repeated versions: %d rows, %d distinct", got, want)
	}
	if got := count(`SELECT MAX(version) FROM schema_migrations`); got != 32 { // 32: tariff_events
		t.Errorf("latest version = %d, want 32", got)
	}
	// every account that existed before the script setting reads in Latin, as before
	if got, want := count(`SELECT COUNT(*) FROM users WHERE script = 'latin'`), count(`SELECT COUNT(*) FROM users`); got != want || want == 0 {
		t.Errorf("users on the latin default = %d of %d, want all of a non-empty set", got, want)
	}
	// the two tariffs are seeded, nobody has paid, and an old entitlement row has no end date
	if got := count(`SELECT COUNT(*) FROM tariffs`); got != 2 {
		t.Errorf("tariffs = %d rows, want level and year", got)
	}
	if got := count(`SELECT COUNT(*) FROM payments`); got != 0 {
		t.Errorf("payments has %d rows after upgrade, want none", got)
	}
	if got := count(`SELECT COUNT(*) FROM user_entitlements WHERE expires_at <> ''`); got != 0 {
		t.Errorf("%d old entitlements got an end date, want none", got)
	}
	// message preferences exist and are empty: nobody has opted out of anything
	// yet, and "no row" is what means the defaults
	if got := count(`SELECT COUNT(*) FROM bot_prefs`); got != 0 {
		t.Errorf("bot_prefs has %d rows after upgrade, want none", got)
	}
	// the new columns are there, and an existing learner has no code until asked
	if got := count(`SELECT COUNT(*) FROM users WHERE referral_code IS NOT NULL OR referred_by IS NOT NULL`); got != 0 {
		t.Errorf("existing users got referral data: %d", got)
	}
	// migrations leave an unseeded economy alone: the seed owns the first fill
	if got := count(`SELECT COUNT(*) FROM quests`); got != 0 {
		t.Errorf("quests before the seed = %d, want 0", got)
	}

	// boot seeds the economy once; the result is the designed one
	if err := s.SeedEconomyDefaults(nil, day0); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if got := count(`SELECT price FROM products WHERE kind = 'phase_unlock' AND ref = '1b'`); got != 40 {
		t.Errorf("level 2 price = %d, want 40", got)
	}
	if got := count(`SELECT price FROM products WHERE kind = 'phase_unlock' AND ref = '4'`); got != 370 {
		t.Errorf("level 5 price = %d, want 370", got)
	}
	if got := count(`SELECT price FROM products WHERE kind = 'phase_unlock' AND ref = '5'`); got != 480 {
		t.Errorf("level 6 price = %d, want 480", got)
	}
	if got := count(`SELECT reward FROM quests WHERE kind = 'friends_invited'`); got != 30 {
		t.Errorf("invite quest reward = %d, want 30", got)
	}
	if got := count(`SELECT COUNT(*) FROM quests WHERE kind = 'friends_invited'`); got != 1 {
		t.Errorf("invite quests = %d, want exactly 1", got)
	}
	if got := count(`SELECT COUNT(*) FROM economy_settings WHERE key = 'currency_name_many' AND value = 'зёрнышек'`); got != 1 {
		t.Errorf("currency name was not seeded as зёрнышки")
	}
}

// Production applied 024 (the lead magnet) from another branch before 023 (bot
// preferences) existed. A migration below the newest recorded version must still
// run: the runner goes by "not recorded yet", not by "newer than the last".
func TestAMigrationBelowTheNewestRecordedVersionStillRuns(t *testing.T) {
	s := newStore(t)
	if _, err := s.db.Exec(`DROP TABLE bot_prefs`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`DELETE FROM schema_migrations WHERE version = 23`); err != nil {
		t.Fatal(err)
	}
	if err := s.runMigrations(); err != nil {
		t.Fatalf("runMigrations: %v", err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM bot_prefs`).Scan(&n); err != nil {
		t.Fatalf("bot_prefs was not recreated: %v", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 23`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("version 23 recorded %d times (%v), want once", n, err)
	}
}

// Migration 030 makes Level 2 the first paid level. A database the seed ran on
// before it ends up with exactly what a fresh seed writes, a learner who had
// started a level that used to be free keeps it, and anything edited in the admin
// panel is left alone.
func TestMigration030Level2Paywall(t *testing.T) {
	// seedOld is the production database as it stood before: Level 1 was sixteen
	// numbered lessons that paid 2, and 4 on every fourth, and only Levels 4 and 5
	// were for sale.
	seedOld := func(t *testing.T) *Store {
		t.Helper()
		s := openStoreAtVersion(t, 29)
		mustExec(t, s, `INSERT INTO economy_settings (key, value, updated_at) VALUES ('economy_seeded', '1', 'x')`)
		mustExec(t, s, `INSERT INTO economy_settings (key, value, updated_at) VALUES ('economy_lessons_seeded', '1', 'x')`)
		for _, p := range []struct {
			ref, title, desc string
			price            int
			sort             int
		}{
			{"4", "Уровень 4 — Падежи в жизни", "Открывает уровень 4: падежи, прилагательные, местоимения", 230, 10},
			{"5", "Уровень 5 — Жизнь на сербском", "Открывает уровень 5: чувства, звонки, работа, документы", 375, 11},
		} {
			mustExec(t, s, `INSERT INTO products (kind, ref, title, description, price, sort_order, created_at, updated_at)
				VALUES ('phase_unlock', ?, ?, ?, ?, ?, 'x', 'x')`, p.ref, p.title, p.desc, p.price, p.sort)
		}
		for _, q := range []struct {
			param, title string
			reward, sort int
		}{
			{"1", "Уровень 1 на 100%", 10, 70}, {"2", "Уровень 2 на 100%", 19, 71},
			{"3", "Уровень 3 на 100%", 30, 72}, {"4", "Уровень 4 на 100%", 22, 73},
		} {
			mustExec(t, s, `INSERT INTO quests (kind, target, param, title, description, reward, sort_order, created_at, updated_at)
				VALUES ('phase_completed', 100, ?, ?, 'старое', ?, ?, 'x', 'x')`, q.param, q.title, q.reward, q.sort)
		}
		for n := 0; n <= 15; n++ {
			reward := 2
			if (n+1)%4 == 0 {
				reward = 4
			}
			id := fmt.Sprintf("%02d", n)
			mustExec(t, s, `INSERT INTO quests (kind, target, param, title, reward, sort_order, created_at, updated_at)
				VALUES ('lesson_completed', 1, ?, ?, ?, ?, 'x', 'x')`, id, "Урок "+id, reward, 101+n)
		}
		return s
	}
	// snapshot is what the economy looks like after a migration or a fresh seed:
	// Level 1 and 2 lesson rewards, the level bonuses, the level products.
	snapshot := func(t *testing.T, s *Store) map[string]string {
		t.Helper()
		out := map[string]string{}
		rows, err := s.db.Query(`SELECT kind, param, reward, title FROM quests
			WHERE kind = 'phase_completed' OR (kind = 'lesson_completed' AND param < '16')`)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var kind, param, title string
			var reward int
			if err := rows.Scan(&kind, &param, &reward, &title); err != nil {
				t.Fatal(err)
			}
			if kind == "phase_completed" {
				out["quest:"+param] = fmt.Sprintf("%d %s", reward, title)
			} else {
				out["lesson:"+param] = fmt.Sprint(reward)
			}
		}
		rows.Close()
		prows, err := s.db.Query(`SELECT ref, price, title, sort_order FROM products WHERE kind = 'phase_unlock'`)
		if err != nil {
			t.Fatal(err)
		}
		for prows.Next() {
			var ref, title string
			var price, sort int
			if err := prows.Scan(&ref, &price, &title, &sort); err != nil {
				t.Fatal(err)
			}
			out["product:"+ref] = fmt.Sprintf("%d %s %d", price, title, sort)
		}
		prows.Close()
		return out
	}
	count := func(t *testing.T, s *Store, q string, a ...any) int {
		t.Helper()
		var n int
		if err := s.db.QueryRow(q, a...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	t.Run("an old database ends up where a fresh seed starts", func(t *testing.T) {
		s := seedOld(t)
		if err := s.runMigrations(); err != nil {
			t.Fatalf("run migrations: %v", err)
		}
		fresh := newStore(t)
		if err := fresh.SeedEconomyDefaults(realPhases(t), seedNow); err != nil {
			t.Fatal(err)
		}
		got, want := snapshot(t, s), snapshot(t, fresh)
		for k, w := range want {
			// the old fixture holds only the phase_unlock rows 4 and 5 as products
			// and no other quests, so compare what the migration is responsible for
			if g, ok := got[k]; !ok {
				t.Errorf("%s: missing after the migration, a fresh seed has %q", k, w)
			} else if g != w {
				t.Errorf("%s = %q after the migration, a fresh seed has %q", k, g, w)
			}
		}
		for k := range got {
			if _, ok := want[k]; !ok {
				t.Errorf("%s exists after the migration but not in a fresh seed", k)
			}
		}
	})

	t.Run("admin edits survive", func(t *testing.T) {
		s := seedOld(t)
		mustExec(t, s, `UPDATE products SET price = 333 WHERE kind = 'phase_unlock' AND ref = '4'`)
		mustExec(t, s, `UPDATE quests SET reward = 7 WHERE kind = 'lesson_completed' AND param = '02'`)
		mustExec(t, s, `UPDATE quests SET title = 'Мой уровень' WHERE kind = 'phase_completed' AND param = '2'`)
		if err := s.runMigrations(); err != nil {
			t.Fatalf("run migrations: %v", err)
		}
		if got := count(t, s, `SELECT price FROM products WHERE kind = 'phase_unlock' AND ref = '4'`); got != 333 {
			t.Errorf("an edited price became %d, want 333 kept", got)
		}
		if got := count(t, s, `SELECT reward FROM quests WHERE kind = 'lesson_completed' AND param = '02'`); got != 7 {
			t.Errorf("an edited reward became %d, want 7 kept", got)
		}
		var title string
		if err := s.db.QueryRow(`SELECT title FROM quests WHERE kind = 'phase_completed' AND param = '2'`).Scan(&title); err != nil {
			t.Fatal(err)
		}
		if title != "Мой уровень" {
			t.Errorf("an edited title became %q, want it kept", title)
		}
	})

	t.Run("a started level stays open", func(t *testing.T) {
		s := seedOld(t)
		for _, p := range []struct{ user, lesson string }{
			{"usr_a", "08"}, // began Level 2
			{"usr_b", "17"}, // began Level 3
			{"usr_c", "31"}, // began Level 4
			{"usr_d", "05"}, // only Level 1: nothing to keep
			{"usr_e", "42"}, // Level 5 was already for sale
			{"usr_a", "16"}, // and Level 3 too
		} {
			mustExec(t, s, `INSERT INTO lesson_progress (user_id, lesson, status) VALUES (?, ?, 'in_progress')`, p.user, p.lesson)
		}
		if err := s.runMigrations(); err != nil {
			t.Fatalf("run migrations: %v", err)
		}
		has := func(user, phase string) bool {
			return count(t, s, `SELECT COUNT(*) FROM user_entitlements WHERE user_id = ? AND kind = 'phase_unlock' AND ref = ?`, user, phase) == 1
		}
		for _, want := range []struct {
			user, phase string
			open        bool
		}{
			{"usr_a", "1b", true}, {"usr_a", "2", true}, {"usr_a", "3", false},
			{"usr_b", "2", true}, {"usr_b", "1b", false},
			{"usr_c", "3", true},
			{"usr_d", "1b", false}, {"usr_d", "2", false},
			{"usr_e", "4", false}, {"usr_e", "3", false},
		} {
			if has(want.user, want.phase) != want.open {
				t.Errorf("%s / level %s: open = %v, want %v", want.user, want.phase, !want.open, want.open)
			}
		}
	})

	// Levels 4 and 5 of the old numbering are Levels 5 and 6 now: the ids of the phases did not change, so what a
	// learner paid for stays theirs under the new names, and only the prices for the ones who did not buy move.
	t.Run("a level bought before stays bought, under its new number and price", func(t *testing.T) {
		s := seedOld(t)
		for _, e := range []struct{ user, phase string }{
			{"usr_old4", "4"}, // paid 230 for «Уровень 4», it is «Уровень 5» now
			{"usr_old5", "5"}, // paid 375 for «Уровень 5», it is «Уровень 6» now
			{"usr_both", "4"}, {"usr_both", "5"},
		} {
			mustExec(t, s, `INSERT INTO user_entitlements (user_id, kind, ref, qty, updated_at) VALUES (?, 'phase_unlock', ?, 1, 'x')`, e.user, e.phase)
			mustExec(t, s, `INSERT INTO lesson_progress (user_id, lesson, status) VALUES (?, ?, 'done')`, e.user, map[string]string{"4": "42", "5": "60"}[e.phase])
		}
		if err := s.runMigrations(); err != nil {
			t.Fatalf("run migrations: %v", err)
		}
		has := func(user, phase string) bool {
			return count(t, s, `SELECT COUNT(*) FROM user_entitlements WHERE user_id = ? AND kind = 'phase_unlock' AND ref = ?`, user, phase) == 1
		}
		for _, want := range []struct{ user, phase string }{{"usr_old4", "4"}, {"usr_old5", "5"}, {"usr_both", "4"}, {"usr_both", "5"}} {
			if !has(want.user, want.phase) {
				t.Errorf("%s lost the level with id %s that they had bought", want.user, want.phase)
			}
		}
		// the buyers were not given anything else
		if has("usr_old5", "4") || has("usr_old4", "5") {
			t.Error("a purchase of one level opened the other")
		}
		var title string
		var price int
		if err := s.db.QueryRow(`SELECT title, price FROM products WHERE kind = 'phase_unlock' AND ref = '5'`).Scan(&title, &price); err != nil {
			t.Fatal(err)
		}
		if title != "Уровень 6 — Жизнь на сербском" || price != 480 {
			t.Errorf("old level 5 is %q at %d, want «Уровень 6» at 480", title, price)
		}
	})

	// "Started" means any trace of the learner in a lesson, not only the progress row: the answers they gave and the
	// steps they went through count too, so nobody who began a level loses it to the new prices.
	t.Run("a level started only by answering or by a step stays open too", func(t *testing.T) {
		s := seedOld(t)
		mustExec(t, s, `INSERT INTO lesson_step_progress (user_id, lesson, step, status) VALUES ('usr_step', '09', '1', 'in_progress')`)
		mustExec(t, s, `INSERT INTO attempts (user_id, exercise_id, lesson, block, answer, correct, attempted_at)
			VALUES ('usr_ans', '18.1.1', '18', 1, 'x', 0, '2026-10-20T10:00:00Z')`)
		mustExec(t, s, `INSERT INTO attempts (user_id, exercise_id, lesson, block, answer, correct, attempted_at)
			VALUES ('usr_ans2', '33.2.1', '33', 2, 'x', 1, '2026-10-20T10:00:00Z')`)
		mustExec(t, s, `INSERT INTO attempts (user_id, exercise_id, lesson, block, answer, correct, attempted_at)
			VALUES ('usr_free', '03.1.1', '03', 1, 'x', 1, '2026-10-20T10:00:00Z')`)
		if err := s.runMigrations(); err != nil {
			t.Fatalf("run migrations: %v", err)
		}
		has := func(user, phase string) bool {
			return count(t, s, `SELECT COUNT(*) FROM user_entitlements WHERE user_id = ? AND kind = 'phase_unlock' AND ref = ?`, user, phase) == 1
		}
		if !has("usr_step", "1b") {
			t.Error("a learner who only went through a step of Level 2 lost it")
		}
		if !has("usr_ans", "2") {
			t.Error("a learner who only answered in Level 3 lost it")
		}
		if !has("usr_ans2", "3") {
			t.Error("a learner who only answered in Level 4 lost it")
		}
		if has("usr_free", "1b") || has("usr_free", "2") {
			t.Error("a learner who only worked in the free Level 1 was given a paid level")
		}
	})

	t.Run("an unseeded database gets nothing", func(t *testing.T) {
		s := openStoreAtVersion(t, 29)
		mustExec(t, s, `INSERT INTO lesson_progress (user_id, lesson, status) VALUES ('usr_a', '08', 'in_progress')`)
		if err := s.runMigrations(); err != nil {
			t.Fatalf("run migrations: %v", err)
		}
		for _, table := range []string{"quests", "products", "user_entitlements"} {
			if got := count(t, s, `SELECT COUNT(*) FROM `+table); got != 0 {
				t.Errorf("%d rows in %s on a database the seed has not run on, want 0", got, table)
			}
		}
	})

	t.Run("running twice changes nothing", func(t *testing.T) {
		s := seedOld(t)
		mustExec(t, s, `INSERT INTO lesson_progress (user_id, lesson, status) VALUES ('usr_a', '08', 'in_progress')`)
		if err := s.runMigrations(); err != nil {
			t.Fatal(err)
		}
		before := fmt.Sprint(snapshot(t, s)) + fmt.Sprint(count(t, s, `SELECT COUNT(*) FROM quests`), count(t, s, `SELECT COUNT(*) FROM user_entitlements`))
		if err := s.runMigrations(); err != nil {
			t.Fatal(err)
		}
		after := fmt.Sprint(snapshot(t, s)) + fmt.Sprint(count(t, s, `SELECT COUNT(*) FROM quests`), count(t, s, `SELECT COUNT(*) FROM user_entitlements`))
		if before != after {
			t.Errorf("a second run changed the economy:\n before %s\n after  %s", before, after)
		}
	})
}
