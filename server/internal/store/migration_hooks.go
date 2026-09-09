package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
)

func init() {
	registerHook(2, migrate002)
	registerHook(3, migrate003)
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
