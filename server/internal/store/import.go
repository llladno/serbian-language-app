package store

import (
	"database/sql"
	"fmt"
)

// ImportSQLite copies every row from a SQLite database file into dst
// (intended: a fresh Postgres store). It is a no-op — returning 0, nil —
// when dst already has accounts, so it is safe to wire on every boot.
//
// A v1 (accountless) SQLite file is migrated to v2 in place first, so its
// rows land under the legacy account.
func ImportSQLite(dst *Store, sqlitePath string) (int, error) {
	var existing int
	if err := dst.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&existing); err != nil {
		return 0, fmt.Errorf("check dst: %w", err)
	}
	if existing > 0 {
		return 0, nil
	}

	src, err := Open(sqlitePath)
	if err != nil {
		return 0, fmt.Errorf("open sqlite %s: %w", sqlitePath, err)
	}
	defer src.Close()

	tx, err := dst.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	type table struct {
		read  string
		write string
		cols  int
	}
	tables := []table{
		{`SELECT name, created_at FROM users`,
			`INSERT INTO users (name, created_at) VALUES (?, ?)`, 2},
		{`SELECT user_name, card_id, kind, ref_id, ease, interval_days, reps, lapses, state, due, updated_at FROM srs_cards`,
			`INSERT INTO srs_cards (user_name, card_id, kind, ref_id, ease, interval_days, reps, lapses, state, due, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, 11},
		{`SELECT user_name, card_id, grade, reviewed_at FROM reviews`,
			`INSERT INTO reviews (user_name, card_id, grade, reviewed_at) VALUES (?, ?, ?, ?)`, 4},
		{`SELECT user_name, exercise_id, lesson, block, answer, correct, attempted_at FROM attempts`,
			`INSERT INTO attempts (user_name, exercise_id, lesson, block, answer, correct, attempted_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, 7},
		{`SELECT user_name, lesson, status, started_at, completed_at FROM lesson_progress`,
			`INSERT INTO lesson_progress (user_name, lesson, status, started_at, completed_at) VALUES (?, ?, ?, ?, ?)`, 5},
	}

	total := 0
	for _, tbl := range tables {
		rows, err := src.db.Query(tbl.read)
		if err != nil {
			return 0, fmt.Errorf("read: %w", err)
		}
		vals := make([]any, tbl.cols)
		ptrs := make([]any, tbl.cols)
		for i := range vals {
			var v sql.NullString
			vals[i] = &v
			ptrs[i] = &v
		}
		var batch [][]any
		for rows.Next() {
			if err := rows.Scan(ptrs...); err != nil {
				rows.Close()
				return 0, err
			}
			rec := make([]any, tbl.cols)
			for i := range ptrs {
				ns := ptrs[i].(*sql.NullString)
				if ns.Valid {
					rec[i] = ns.String
				} else {
					rec[i] = nil
				}
			}
			batch = append(batch, rec)
		}
		rows.Close()
		for _, rec := range batch {
			if _, err := tx.Exec(tbl.write, rec...); err != nil {
				return 0, fmt.Errorf("write %s: %w", firstLine(tbl.write), err)
			}
			total++
		}
	}
	return total, tx.Commit()
}
