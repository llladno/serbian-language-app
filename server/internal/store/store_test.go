package store

import (
	"os"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/srs"
)

var day0 = time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)

// testDSN picks the backend under test. Set TEST_DATABASE_URL to a
// "postgres://" URL to run the suite against a real Postgres (the schema
// is truncated between tests); otherwise an in-memory SQLite db is used.
func testDSN() string {
	if dsn := os.Getenv("TEST_DATABASE_URL"); IsPostgresDSN(dsn) {
		return dsn
	}
	return ":memory:"
}

func newStore(t *testing.T) *Store {
	t.Helper()
	dsn := testDSN()
	s, err := Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if IsPostgresDSN(dsn) {
		truncate := func() {
			s.db.Exec(`TRUNCATE users, srs_cards, reviews, attempts, lesson_progress`)
		}
		truncate()
		t.Cleanup(func() { truncate(); s.Close() })
	} else {
		t.Cleanup(func() { s.Close() })
	}
	return s
}

func newUser(t *testing.T) (*Store, *UserStore) {
	t.Helper()
	s := newStore(t)
	if _, err := s.EnsureUser("Гриша"); err != nil {
		t.Fatal(err)
	}
	return s, s.User("Гриша")
}

func TestEnsureCardsIdempotent(t *testing.T) {
	_, u := newUser(t)
	seeds := []CardSeed{{"vocab:zdravo", "vocab", "zdravo"}}
	if err := u.EnsureCards(seeds); err != nil {
		t.Fatal(err)
	}
	if err := u.EnsureCards(seeds); err != nil {
		t.Fatal(err)
	}
	total, _, err := u.CardStats()
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
}

func TestAccountsAreIsolated(t *testing.T) {
	s, a := newUser(t)
	s.EnsureUser("Оля")
	b := s.User("Оля")

	a.EnsureCards([]CardSeed{{"vocab:x", "vocab", "x"}, {"vocab:y", "vocab", "y"}})
	b.EnsureCards([]CardSeed{{"vocab:x", "vocab", "x"}})
	a.GradeCard("vocab:x", srs.Good, day0)
	a.SetLessonStatus("01", "done", day0)

	at, _, _ := a.CardStats()
	bt, _, _ := b.CardStats()
	if at != 2 || bt != 1 {
		t.Errorf("card counts a=%d b=%d, want 2 and 1", at, bt)
	}
	if n, _ := a.ReviewedToday(day0); n != 1 {
		t.Errorf("a reviewed = %d", n)
	}
	if n, _ := b.ReviewedToday(day0); n != 0 {
		t.Errorf("b reviewed = %d, want 0 (isolation leak)", n)
	}
	am, _ := a.LessonStatuses()
	bm, _ := b.LessonStatuses()
	if am["01"] != "done" || bm["01"] != "" {
		t.Errorf("lesson status leaked: a=%v b=%v", am, bm)
	}
}

func TestListUsers(t *testing.T) {
	s, _ := newUser(t)
	s.EnsureUser("Оля")
	names, err := s.ListUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "Гриша" || names[1] != "Оля" {
		t.Errorf("users = %v", names)
	}
}

func TestEnsureUserNormalizesAndValidates(t *testing.T) {
	s, _ := newUser(t)
	n, err := s.EnsureUser("  Марко   Краљевић  ")
	if err != nil || n != "Марко Краљевић" {
		t.Errorf("normalize: %q %v", n, err)
	}
	if _, err := s.EnsureUser("   "); err == nil {
		t.Error("empty name should fail")
	}
}

func TestDueQueueLimitsNewAndOrdersOverdueFirst(t *testing.T) {
	_, u := newUser(t)
	u.EnsureCards([]CardSeed{
		{"vocab:a", "vocab", "a"},
		{"vocab:b", "vocab", "b"},
		{"vocab:c", "vocab", "c"},
	})
	if _, err := u.db.Exec(`UPDATE srs_cards SET state='review', interval_days=3, due='2026-09-01' WHERE card_id='vocab:a'`); err != nil {
		t.Fatal(err)
	}
	q, err := u.DueQueue(day0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(q) != 2 {
		t.Fatalf("queue len = %d, want 2 (1 overdue + 1 new)", len(q))
	}
	if q[0].CardID != "vocab:a" {
		t.Errorf("first card = %s, want overdue vocab:a", q[0].CardID)
	}
	if q[1].State != srs.New {
		t.Errorf("second card state = %s, want new", q[1].State)
	}
}

func TestGradeCardPersistsAndLogsReview(t *testing.T) {
	_, u := newUser(t)
	u.EnsureCards([]CardSeed{{"vocab:x", "vocab", "x"}})
	c, err := u.GradeCard("vocab:x", srs.Good, day0)
	if err != nil {
		t.Fatal(err)
	}
	if c.State != srs.Review {
		t.Errorf("state = %s", c.State)
	}
	n, err := u.ReviewedToday(day0)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("reviewed today = %d, want 1", n)
	}
	q, _ := u.DueQueue(day0, 0)
	if len(q) != 0 {
		t.Errorf("expected empty due queue, got %d", len(q))
	}
}

func TestAddAttemptAndWeakExercises(t *testing.T) {
	_, u := newUser(t)
	u.AddAttempt(Attempt{"02-C-1", "02", "C", "x", false}, day0)
	u.AddAttempt(Attempt{"02-C-1", "02", "C", "y", false}, day0)
	u.AddAttempt(Attempt{"02-C-1", "02", "C", "radi", true}, day0)
	u.AddAttempt(Attempt{"02-B-1", "02", "B", "ok", true}, day0)

	w, err := u.WeakExercises(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(w) != 1 {
		t.Fatalf("weak = %+v", w)
	}
	if w[0].ExerciseID != "02-C-1" || w[0].Wrong != 2 || w[0].Total != 3 {
		t.Errorf("weak[0] = %+v", w[0])
	}
}

func TestLessonStatusRoundTrip(t *testing.T) {
	_, u := newUser(t)
	if err := u.SetLessonStatus("01", "in_progress", day0); err != nil {
		t.Fatal(err)
	}
	if err := u.SetLessonStatus("01", "done", day0.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	m, err := u.LessonStatuses()
	if err != nil {
		t.Fatal(err)
	}
	if m["01"] != "done" {
		t.Errorf("statuses = %+v", m)
	}
	one, _ := u.LessonStatus("01")
	if one != "done" {
		t.Errorf("LessonStatus = %q", one)
	}
	if miss, _ := u.LessonStatus("99"); miss != "" {
		t.Errorf("unknown lesson status = %q, want empty", miss)
	}
}

func TestActivityByDay(t *testing.T) {
	_, u := newUser(t)
	u.EnsureCards([]CardSeed{{"vocab:x", "vocab", "x"}})
	u.GradeCard("vocab:x", srs.Good, day0)
	u.AddAttempt(Attempt{"01-A-1", "01", "A", "z", true}, day0)
	u.AddAttempt(Attempt{"01-A-2", "01", "A", "z", false}, day0.AddDate(0, 0, -3))

	acts, err := u.ActivityByDay(day0.AddDate(0, 0, -7))
	if err != nil {
		t.Fatal(err)
	}
	byDate := map[string]int{}
	for _, a := range acts {
		byDate[a.Date] = a.Count
	}
	if byDate[day0.Format("2006-01-02")] != 2 {
		t.Errorf("today count = %d, want 2 (%+v)", byDate[day0.Format("2006-01-02")], acts)
	}
	if byDate[day0.AddDate(0, 0, -3).Format("2006-01-02")] != 1 {
		t.Errorf("three days ago count wrong: %+v", acts)
	}
}

func TestStreakDays(t *testing.T) {
	_, u := newUser(t)
	u.EnsureCards([]CardSeed{{"vocab:x", "vocab", "x"}})
	u.GradeCard("vocab:x", srs.Good, day0)
	if _, err := u.db.Exec(`INSERT INTO reviews (user_name, card_id, grade, reviewed_at) VALUES ('Гриша', 'vocab:x', 2, '2026-09-05T09:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	streak, err := u.StreakDays(day0)
	if err != nil {
		t.Fatal(err)
	}
	if streak != 2 {
		t.Errorf("streak = %d, want 2", streak)
	}
	if got, _ := u.StreakDays(day0.AddDate(0, 0, 2)); got != 0 {
		t.Errorf("streak after gap = %d, want 0", got)
	}
}

func TestMigrationV1toV2(t *testing.T) {
	if IsPostgresDSN(testDSN()) {
		t.Skip("v1->v2 migration is a SQLite-only legacy path")
	}
	dir := t.TempDir() + "/v1.db"
	db, err := openRawV1(dir)
	if err != nil {
		t.Fatal(err)
	}
	// seed a v1 row set
	if _, err := db.Exec(`INSERT INTO srs_cards (card_id, kind, ref_id, state, interval_days, updated_at) VALUES ('vocab:z','vocab','z','review',9,'x')`); err != nil {
		t.Fatal(err)
	}
	db.Exec(`INSERT INTO reviews (card_id, grade, reviewed_at) VALUES ('vocab:z', 2, '2026-09-06T10:00:00Z')`)
	db.Exec(`INSERT INTO lesson_progress (lesson, status, started_at) VALUES ('01','done','x')`)
	db.Close()

	s, err := Open(dir)
	if err != nil {
		t.Fatalf("open (migrate): %v", err)
	}
	defer s.Close()

	names, _ := s.ListUsers()
	if len(names) != 1 || names[0] != legacyOwner {
		t.Fatalf("users after migrate = %v", names)
	}
	u := s.User(legacyOwner)
	total, _, _ := u.CardStats()
	if total != 1 {
		t.Errorf("migrated cards = %d, want 1", total)
	}
	m, _ := u.LessonStatuses()
	if m["01"] != "done" {
		t.Errorf("migrated lesson status = %v", m)
	}
	if n, _ := u.ReviewedToday(time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)); n != 1 {
		t.Errorf("migrated reviews = %d, want 1", n)
	}
}

func TestImportSQLite(t *testing.T) {
	if !IsPostgresDSN(testDSN()) {
		t.Skip("import target is Postgres; set TEST_DATABASE_URL")
	}
	// build a small SQLite source
	srcPath := t.TempDir() + "/src.db"
	src, err := Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	src.EnsureUser("Гриша")
	src.EnsureUser("Оля")
	u := src.User("Гриша")
	u.EnsureCards([]CardSeed{{"vocab:x", "vocab", "x"}, {"vocab:y", "vocab", "y"}})
	u.GradeCard("vocab:x", srs.Good, day0)
	u.AddAttempt(Attempt{"01-A-1", "01", "A", "z", true}, day0)
	u.SetLessonStatus("01", "done", day0)
	src.Close()

	dst := newStore(t) // truncated Postgres
	n, err := ImportSQLite(dst, srcPath)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if n < 6 {
		t.Errorf("imported %d rows, want >= 6", n)
	}
	names, _ := dst.ListUsers()
	if len(names) != 2 {
		t.Fatalf("users after import = %v", names)
	}
	du := dst.User("Гриша")
	total, _, _ := du.CardStats()
	if total != 2 {
		t.Errorf("cards = %d, want 2", total)
	}
	if m, _ := du.LessonStatuses(); m["01"] != "done" {
		t.Errorf("lesson status = %v", m)
	}
	if got, err := du.ReviewedToday(day0); err != nil || got != 1 {
		t.Errorf("reviewed today = %d (%v), want 1", got, err)
	}
	// second run is a no-op
	if n2, err := ImportSQLite(dst, srcPath); err != nil || n2 != 0 {
		t.Errorf("second import = %d (%v), want 0", n2, err)
	}
}
