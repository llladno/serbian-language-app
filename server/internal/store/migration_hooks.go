package store

import (
	"strings"

	"github.com/grisha/serbian-app/server/internal/auth"
)

func init() {
	registerHook(2, migrate002)
}

// migrate002 fills users_new with a generated id per legacy row, re-keys the
// five state tables from user_name to user_id, then swaps *_new into place.
// Runs inside migration 002's transaction, after 002_auth.sql.
func migrate002(tx *dbtx, pg bool) error {
	rows, err := tx.Query(`SELECT name, created_at FROM users ORDER BY created_at`)
	if err != nil {
		return err
	}
	type u struct{ name, created string }
	var legacy []u
	for rows.Next() {
		var x u
		if err := rows.Scan(&x.name, &x.created); err != nil {
			rows.Close()
			return err
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
			return err
		}
	}

	// state tables: copy rows whose user_name resolves to a known id
	copies := []struct{ dst, src, cols string }{
		{"srs_cards_new", "srs_cards", "card_id, kind, ref_id, ease, interval_days, reps, lapses, state, due, updated_at"},
		{"reviews_new", "reviews", "card_id, grade, reviewed_at"},
		{"attempts_new", "attempts", "exercise_id, lesson, block, answer, correct, attempted_at"},
		{"lesson_progress_new", "lesson_progress", "lesson, status, started_at, completed_at"},
		{"lesson_step_progress_new", "lesson_step_progress", "lesson, step, status, completed_at"},
	}
	for _, c := range copies {
		src, err := tx.Query(`SELECT user_name, ` + c.cols + ` FROM ` + c.src)
		if err != nil {
			return err
		}
		var batch [][]any
		ncol := len(splitCols(c.cols))
		for src.Next() {
			vals := make([]any, ncol+1)
			ptrs := make([]any, ncol+1)
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := src.Scan(ptrs...); err != nil {
				src.Close()
				return err
			}
			batch = append(batch, vals)
		}
		src.Close()
		ph := "?" + strings0(", ?", ncol) // ncol+1 placeholders: user_id + ncol copied cols
		ins := `INSERT INTO ` + c.dst + ` (user_id, ` + c.cols + `) VALUES (` + ph + `)`
		for _, vals := range batch {
			name, _ := vals[0].(string)
			uid, ok := idByName[name]
			if !ok {
				continue // orphaned row (deleted account) — dropped
			}
			args := append([]any{uid}, vals[1:]...)
			if _, err := tx.Exec(ins, args...); err != nil {
				return err
			}
		}
	}

	// swap state tables + users
	for _, name := range []string{"users", "srs_cards", "reviews", "attempts", "lesson_progress", "lesson_step_progress"} {
		if _, err := tx.Exec(`DROP TABLE ` + name); err != nil {
			return err
		}
		if _, err := tx.Exec(`ALTER TABLE ` + name + `_new RENAME TO ` + name); err != nil {
			return err
		}
	}
	for _, q := range []string{
		`CREATE INDEX reviews_user ON reviews(user_id)`,
		`CREATE INDEX attempts_user ON attempts(user_id)`,
	} {
		if _, err := tx.Exec(q); err != nil {
			return err
		}
	}

	// auth tables — created here, after users is in place, so FKs resolve
	for _, q := range auth002Tables {
		if _, err := tx.Exec(q); err != nil {
			return err
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

// splitCols splits a ", "-separated column list.
func splitCols(s string) []string { return strings.Split(s, ", ") }

// strings0 repeats sep n times (n placeholders' worth of separators).
func strings0(sep string, n int) string { return strings.Repeat(sep, n) }
