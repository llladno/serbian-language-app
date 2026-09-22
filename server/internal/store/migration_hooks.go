package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
	"github.com/grisha/serbian-app/server/internal/telegram"
)

func init() {
	registerHook(2, migrate002)
	registerHook(3, migrate003)
	registerHook(4, migrate004)
	registerHook(7, migrate007)
}

// migrate002 fills users_new with a generated id per legacy row, re-keys the
// five state tables from user_name to user_id, then swaps *_new into place.
// Runs inside migration 002's transaction, after 002_auth.sql.
func migrate002(tx *dbtx, pg bool) error {
	rows, err := tx.Query(`SELECT name, created_at FROM users ORDER BY created_at`)
	if err != nil {
		return fmt.Errorf("read legacy users: %w", err)
	}
	type u struct{ name, created string }
	var legacy []u
	for rows.Next() {
		var x u
		if err := rows.Scan(&x.name, &x.created); err != nil {
			rows.Close()
			return fmt.Errorf("scan legacy user: %w", err)
		}
		legacy = append(legacy, x)
	}
	rows.Close()

	idByName := map[string]string{}
	for _, x := range legacy {
		id := auth.NewUserID()
		idByName[x.name] = id
		if _, err := tx.Exec(`INSERT INTO users_new (id, name, created_at) VALUES (?, ?, ?)`,
			id, x.name, x.created); err != nil {
			return fmt.Errorf("mint user id for %s: %w", x.name, err)
		}
	}

	// state tables: copy rows whose user_name resolves to a known id.
	// reviews/attempts get fresh autoincrement ids in *_new, assigned in scan
	// order; store.go reads MAX(id) as "the latest attempt", so their source
	// SELECT must be ordered by the old id (an unordered Postgres seqscan
	// could otherwise scramble it). The 3 composite-PK tables have no id.
	copies := []struct{ dst, src, cols, orderBy string }{
		{"srs_cards_new", "srs_cards", "card_id, kind, ref_id, ease, interval_days, reps, lapses, state, due, updated_at", ""},
		{"reviews_new", "reviews", "card_id, grade, reviewed_at", " ORDER BY id"},
		{"attempts_new", "attempts", "exercise_id, lesson, block, answer, correct, attempted_at", " ORDER BY id"},
		{"lesson_progress_new", "lesson_progress", "lesson, status, started_at, completed_at", ""},
		{"lesson_step_progress_new", "lesson_step_progress", "lesson, step, status, completed_at", ""},
	}
	for _, c := range copies {
		src, err := tx.Query(`SELECT user_name, ` + c.cols + ` FROM ` + c.src + c.orderBy)
		if err != nil {
			return fmt.Errorf("read %s: %w", c.src, err)
		}
		var batch [][]any
		ncol := len(strings.Split(c.cols, ", "))
		for src.Next() {
			vals := make([]any, ncol+1)
			ptrs := make([]any, ncol+1)
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := src.Scan(ptrs...); err != nil {
				src.Close()
				return fmt.Errorf("scan %s: %w", c.src, err)
			}
			batch = append(batch, vals)
		}
		src.Close()
		ph := "?" + strings.Repeat(", ?", ncol) // ncol+1 placeholders: user_id + ncol copied cols
		ins := `INSERT INTO ` + c.dst + ` (user_id, ` + c.cols + `) VALUES (` + ph + `)`
		var copied, dropped int
		for _, vals := range batch {
			// A []byte here (some drivers return TEXT as bytes) would coerce to
			// "" and silently drop every row — hard-fail instead.
			name, ok := vals[0].(string)
			if !ok {
				return fmt.Errorf("copy %s: user_name is %T, want string", c.src, vals[0])
			}
			uid, ok := idByName[name]
			if !ok {
				dropped++ // orphaned row (deleted account) — dropped
				continue
			}
			args := append([]any{uid}, vals[1:]...)
			if _, err := tx.Exec(ins, args...); err != nil {
				return fmt.Errorf("copy %s: %w", c.src, err)
			}
			copied++
		}
		// The orphan drop is the one irreversible step — log it so the deploy
		// log shows exactly what was and wasn't carried over.
		log.Printf("migrate002: %s: copied %d rows, dropped %d orphaned", c.src, copied, dropped)
	}

	// swap state tables + users
	for _, name := range []string{"users", "srs_cards", "reviews", "attempts", "lesson_progress", "lesson_step_progress"} {
		if _, err := tx.Exec(`DROP TABLE ` + name); err != nil {
			return fmt.Errorf("drop %s: %w", name, err)
		}
		if _, err := tx.Exec(`ALTER TABLE ` + name + `_new RENAME TO ` + name); err != nil {
			return fmt.Errorf("swap %s: %w", name, err)
		}
	}
	for _, q := range []string{
		`CREATE INDEX reviews_user ON reviews(user_id)`,
		`CREATE INDEX attempts_user ON attempts(user_id)`,
	} {
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("recreate state index: %w", err)
		}
	}

	// auth tables — created here, after users is in place, so FKs resolve
	for _, q := range auth002Tables {
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("create auth tables: %w", err)
		}
	}
	return nil
}

var junkAccounts = []string{"DeployCheck", "ProbaPG", "chk2", "kbcheck"}

var legacyTelegram = map[string]string{
	"Гриша": "llladnooo",
	"Алина": "alinsssk",
}

// migrate003 removes seed/test accounts and attaches the two real accounts to
// their Telegram identity (provider_uid "pending:<username>" until first login).
// Idempotent at the runner level (version 3 is recorded once); also guards the
// identity insert so a manual re-run cannot duplicate.
func migrate003(tx *dbtx, pg bool) error {
	stateTables := []string{"srs_cards", "reviews", "attempts", "lesson_progress", "lesson_step_progress"}
	for _, name := range junkAccounts {
		var id string
		switch err := tx.QueryRow(`SELECT id FROM users WHERE name = ?`, name).Scan(&id); {
		case errors.Is(err, sql.ErrNoRows):
			continue // account not present — fine
		case err != nil:
			return fmt.Errorf("lookup junk account %q: %w", name, err)
		}
		for _, tbl := range stateTables {
			if _, err := tx.Exec(`DELETE FROM `+tbl+` WHERE user_id = ?`, id); err != nil {
				return fmt.Errorf("delete %s for %q: %w", tbl, name, err)
			}
		}
		if _, err := tx.Exec(`DELETE FROM identities WHERE user_id = ?`, id); err != nil {
			return fmt.Errorf("delete identities for %q: %w", name, err)
		}
		if _, err := tx.Exec(`DELETE FROM users WHERE id = ?`, id); err != nil {
			return fmt.Errorf("delete user %q: %w", name, err)
		}
	}
	for name, username := range legacyTelegram {
		var id string
		switch err := tx.QueryRow(`SELECT id FROM users WHERE name = ?`, name).Scan(&id); {
		case errors.Is(err, sql.ErrNoRows):
			continue
		case err != nil:
			return fmt.Errorf("lookup %q: %w", name, err)
		}
		var exists int
		switch err := tx.QueryRow(
			`SELECT 1 FROM identities WHERE user_id = ? AND provider = 'telegram'`, id).Scan(&exists); {
		case errors.Is(err, sql.ErrNoRows):
			// no telegram identity yet — create it below
		case err != nil:
			return fmt.Errorf("check telegram identity for %q: %w", name, err)
		default:
			continue // already linked
		}
		if _, err := tx.Exec(`INSERT INTO identities
			(id, user_id, provider, provider_uid, tg_username, created_at)
			VALUES (?, ?, 'telegram', ?, ?, ?)`,
			auth.NewIdentityID(), id, "pending:"+username, username,
			time.Now().UTC().Format(time.RFC3339)); err != nil {
			return fmt.Errorf("link telegram for %q: %w", name, err)
		}
	}
	return nil
}

// auth002Tables is the DDL for identities/sessions/email_tokens + indexes,
// portable across SQLite and Postgres (no {{.AutoID}} needed — all TEXT PKs).
var auth002Tables = []string{
	`CREATE TABLE identities (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		provider TEXT NOT NULL, provider_uid TEXT NOT NULL,
		email TEXT, password_hash TEXT, email_verified_at TEXT, tg_username TEXT,
		created_at TEXT NOT NULL,
		UNIQUE (provider, provider_uid))`,
	`CREATE TABLE sessions (
		token_hash TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at TEXT NOT NULL, last_seen_at TEXT NOT NULL,
		expires_at TEXT NOT NULL, user_agent TEXT)`,
	`CREATE TABLE email_tokens (
		token_hash TEXT PRIMARY KEY,
		identity_id TEXT NOT NULL REFERENCES identities(id) ON DELETE CASCADE,
		kind TEXT NOT NULL, created_at TEXT NOT NULL,
		expires_at TEXT NOT NULL, used_at TEXT)`,
	`CREATE INDEX identities_user ON identities(user_id)`,
	`CREATE INDEX sessions_user ON sessions(user_id)`,
	`CREATE INDEX sessions_expires ON sessions(expires_at)`,
	`CREATE INDEX email_tokens_ident ON email_tokens(identity_id)`,
}

// lessonSplit is one contiguous run of a pre-split lesson's local step
// numbers (the digits after the dot in a step id like "04.11") that became
// its own lesson. See docs/superpowers/specs/2026-09-21-split-lessons-00-12-design.md:
// levels 1-2 ("00".."12") were each cut into 2-3 shorter lessons ("00".."29")
// without touching any teach/practice/reading/checkpoint/dialogue content,
// only regrouping and renumbering steps.
type lessonSplit struct {
	old        string
	startLocal int
	endLocal   int
	new        string
}

var lessonSplits = []lessonSplit{
	{"00", 1, 2, "00"}, {"00", 3, 5, "01"},
	{"01", 1, 4, "02"}, {"01", 5, 9, "03"},
	{"02", 1, 4, "04"}, {"02", 5, 10, "05"},
	{"03", 1, 4, "06"}, {"03", 5, 9, "07"},
	{"04", 1, 6, "08"}, {"04", 7, 10, "09"}, {"04", 11, 17, "10"},
	{"05", 1, 4, "11"}, {"05", 5, 10, "12"}, {"05", 11, 18, "13"},
	{"06", 1, 4, "14"}, {"06", 5, 8, "15"},
	{"07", 1, 6, "16"}, {"07", 7, 14, "17"},
	{"08", 1, 6, "18"}, {"08", 7, 14, "19"},
	{"09", 1, 4, "20"}, {"09", 5, 8, "21"}, {"09", 9, 16, "22"},
	{"10", 1, 6, "23"}, {"10", 7, 10, "24"}, {"10", 11, 18, "25"},
	{"11", 1, 6, "26"}, {"11", 7, 13, "27"},
	{"12", 1, 4, "28"}, {"12", 5, 8, "29"},
}

// splitLessons lists the old lesson ids handled by migrate004, and for each,
// every new lesson id it was cut into (in order) — used to fan out a
// whole-lesson lesson_progress row across its new parts.
var splitLessons = map[string][]string{}

func init() {
	for _, s := range lessonSplits {
		splitLessons[s.old] = append(splitLessons[s.old], s.new)
	}
}

// remapLessonStep maps an old (lesson, local step number) pair — e.g. ("04",
// 11) from step id "04.11" — to the new (lesson, local step number) it became
// after the split, e.g. ("10", 1) for new step id "10.1".
func remapLessonStep(oldLesson string, local int) (newLesson string, newLocal int, ok bool) {
	for _, s := range lessonSplits {
		if s.old == oldLesson && local >= s.startLocal && local <= s.endLocal {
			return s.new, local - s.startLocal + 1, true
		}
	}
	return "", 0, false
}

// remapStepID maps an old step id ("04.11") to its new one ("10.1").
func remapStepID(step string) (string, bool) {
	lesson, localStr, found := strings.Cut(step, ".")
	if !found {
		return "", false
	}
	local, err := strconv.Atoi(localStr)
	if err != nil {
		return "", false
	}
	newLesson, newLocal, ok := remapLessonStep(lesson, local)
	if !ok {
		return "", false
	}
	return newLesson + "." + strconv.Itoa(newLocal), true
}

// remapExerciseID maps an old exercise (or dialogue-turn-exercise) id
// ("04.11.1") to its new one ("10.1.1") — the lesson.step prefix is
// renumbered, the trailing exercise-local suffix is untouched.
func remapExerciseID(exID string) (string, bool) {
	parts := strings.SplitN(exID, ".", 3)
	if len(parts) != 3 {
		return "", false
	}
	newStep, ok := remapStepID(parts[0] + "." + parts[1])
	if !ok {
		return "", false
	}
	return newStep + "." + parts[2], true
}

// migrate004 relabels every row keyed by a pre-split lesson/step id ("00"
// through "12") to match the new content/ layout ("00" through "29"). It
// touches the three tables keyed by lesson or step id:
//
//   - lesson_step_progress: renumbered 1:1, no data loss.
//   - attempts: a pure history log — lesson/block/exercise_id renumbered 1:1.
//   - lesson_progress: one row per whole OLD lesson has no equivalent single
//     NEW lesson (a lesson became 2-3), so its status/timestamps are copied
//     to every new part; the next real step of activity in any one of them
//     naturally corrects an over-optimistic "done" once the learner reaches
//     material that used to live further into the original lesson.
//
// Every table is read into memory, the old-id rows are deleted, then the
// remapped rows are (re)inserted — so a computed new id that happens to
// collide with an old id string still in the same table (e.g. new lesson
// "04" vs. old lesson "04") can never clash mid-migration.
func migrate004(tx *dbtx, pg bool) error {
	oldIDs := make([]string, 0, len(splitLessons))
	for id := range splitLessons {
		oldIDs = append(oldIDs, id)
	}
	inClause, args := sqlInStrings("lesson", oldIDs)

	if err := migrate004StepProgress(tx, inClause, args); err != nil {
		return err
	}
	if err := migrate004Attempts(tx, inClause, args); err != nil {
		return err
	}
	if err := migrate004LessonProgress(tx, inClause, args); err != nil {
		return err
	}
	return nil
}

// sqlInClause builds a "col IN (?, ?, ...)" fragment and its args.
func sqlInStrings(col string, vals []string) (string, []any) {
	ph := make([]string, len(vals))
	args := make([]any, len(vals))
	for i, v := range vals {
		ph[i] = "?"
		args[i] = v
	}
	return col + " IN (" + strings.Join(ph, ", ") + ")", args
}

func migrate004StepProgress(tx *dbtx, where string, args []any) error {
	type row struct {
		userID, lesson, step, status string
		completedAt                  sql.NullString
	}
	rows, err := tx.Query(`SELECT user_id, lesson, step, status, completed_at FROM lesson_step_progress WHERE `+where, args...)
	if err != nil {
		return fmt.Errorf("read lesson_step_progress: %w", err)
	}
	var got []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.userID, &r.lesson, &r.step, &r.status, &r.completedAt); err != nil {
			rows.Close()
			return fmt.Errorf("scan lesson_step_progress: %w", err)
		}
		got = append(got, r)
	}
	rows.Close()

	if _, err := tx.Exec(`DELETE FROM lesson_step_progress WHERE `+where, args...); err != nil {
		return fmt.Errorf("clear lesson_step_progress: %w", err)
	}
	skipped := 0
	for _, r := range got {
		newStep, ok := remapStepID(r.step)
		if !ok {
			skipped++
			continue
		}
		newLesson, _, _ := strings.Cut(newStep, ".")
		if _, err := tx.Exec(`INSERT INTO lesson_step_progress (user_id, lesson, step, status, completed_at)
			VALUES (?, ?, ?, ?, ?)`, r.userID, newLesson, newStep, r.status, r.completedAt); err != nil {
			return fmt.Errorf("insert lesson_step_progress %s: %w", newStep, err)
		}
	}
	log.Printf("migrate004: lesson_step_progress: relabeled %d rows, skipped %d unmappable", len(got)-skipped, skipped)
	return nil
}

func migrate004Attempts(tx *dbtx, where string, args []any) error {
	type row struct {
		userID, exerciseID, lesson, block, answer, attemptedAt string
		correct                                                int
	}
	rows, err := tx.Query(`SELECT user_id, exercise_id, lesson, block, answer, correct, attempted_at
		FROM attempts WHERE `+where+` ORDER BY id`, args...)
	if err != nil {
		return fmt.Errorf("read attempts: %w", err)
	}
	var got []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.userID, &r.exerciseID, &r.lesson, &r.block, &r.answer, &r.correct, &r.attemptedAt); err != nil {
			rows.Close()
			return fmt.Errorf("scan attempts: %w", err)
		}
		got = append(got, r)
	}
	rows.Close()

	if _, err := tx.Exec(`DELETE FROM attempts WHERE `+where, args...); err != nil {
		return fmt.Errorf("clear attempts: %w", err)
	}
	skipped := 0
	for _, r := range got {
		newExID, ok := remapExerciseID(r.exerciseID)
		if !ok {
			skipped++
			continue
		}
		newBlock, ok := remapStepID(r.block)
		if !ok {
			skipped++
			continue
		}
		newLesson, _, _ := strings.Cut(newBlock, ".")
		if _, err := tx.Exec(`INSERT INTO attempts (user_id, exercise_id, lesson, block, answer, correct, attempted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			r.userID, newExID, newLesson, newBlock, r.answer, r.correct, r.attemptedAt); err != nil {
			return fmt.Errorf("insert attempt %s: %w", newExID, err)
		}
	}
	log.Printf("migrate004: attempts: relabeled %d rows, skipped %d unmappable", len(got)-skipped, skipped)
	return nil
}

func migrate004LessonProgress(tx *dbtx, where string, args []any) error {
	type row struct {
		userID, lesson, status string
		startedAt, completedAt sql.NullString
	}
	rows, err := tx.Query(`SELECT user_id, lesson, status, started_at, completed_at FROM lesson_progress WHERE `+where, args...)
	if err != nil {
		return fmt.Errorf("read lesson_progress: %w", err)
	}
	var got []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.userID, &r.lesson, &r.status, &r.startedAt, &r.completedAt); err != nil {
			rows.Close()
			return fmt.Errorf("scan lesson_progress: %w", err)
		}
		got = append(got, r)
	}
	rows.Close()

	if _, err := tx.Exec(`DELETE FROM lesson_progress WHERE `+where, args...); err != nil {
		return fmt.Errorf("clear lesson_progress: %w", err)
	}
	fanned := 0
	for _, r := range got {
		for _, newLesson := range splitLessons[r.lesson] {
			if _, err := tx.Exec(`INSERT INTO lesson_progress (user_id, lesson, status, started_at, completed_at)
				VALUES (?, ?, ?, ?, ?)`, r.userID, newLesson, r.status, r.startedAt, r.completedAt); err != nil {
				return fmt.Errorf("insert lesson_progress %s: %w", newLesson, err)
			}
			fanned++
		}
	}
	log.Printf("migrate004: lesson_progress: relabeled %d rows into %d rows", len(got), fanned)
	return nil
}

// migrate007 seeds bot_messages with telegram.DefaultMessages so
// ucimo-content-admin's editor always has something to show. Runs once,
// inside migration 007's transaction, right after 007_bot_messages.sql
// creates the table.
func migrate007(tx *dbtx, pg bool) error {
	now := time.Now().UTC().Format(time.RFC3339)
	for key, text := range telegram.DefaultMessages {
		if _, err := tx.Exec(`INSERT INTO bot_messages (key, text, updated_at) VALUES (?, ?, ?)`,
			string(key), text, now); err != nil {
			return fmt.Errorf("seed bot message %s: %w", key, err)
		}
	}
	return nil
}
