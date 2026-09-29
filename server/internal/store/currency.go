package store

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrDuplicateEntry is returned when a ledger row with the same
// idempotency_key already exists. Callers treat it as "already done", not as
// a failure: it is exactly what stops a double-clicked claim or a retried
// request from crediting twice.
var ErrDuplicateEntry = errors.New("duplicate ledger entry")

// LedgerEntry is one row of currency_ledger. The ledger is append-only — a
// correction is a new row, never an update.
type LedgerEntry struct {
	ID             int64
	UserID         string
	Amount         int64 // >0 credit, <0 debit, never 0
	Kind           string
	Ref            string
	IdempotencyKey string
	Comment        string
	CreatedBy      string // "" = system, "admin" = manual adjustment
	CreatedAt      time.Time
}

// AddLedgerEntry appends one entry and returns its id.
func (s *Store) AddLedgerEntry(e LedgerEntry, now time.Time) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin ledger entry: %w", err)
	}
	defer tx.Rollback()
	id, err := addLedgerEntryTx(tx, e, now)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit ledger entry: %w", err)
	}
	return id, nil
}

// addLedgerEntryTx is the transaction-scoped form, used by claim and purchase
// so the credit and its side effects commit together.
func addLedgerEntryTx(tx *dbtx, e LedgerEntry, now time.Time) (int64, error) {
	if e.UserID == "" {
		return 0, errors.New("ledger entry: empty user id")
	}
	if e.Amount == 0 {
		return 0, errors.New("ledger entry: zero amount")
	}
	if e.IdempotencyKey == "" {
		return 0, errors.New("ledger entry: empty idempotency key")
	}

	// RETURNING works via QueryRow on both backends (verified against
	// modernc.org/sqlite v1.58.0 as well as Postgres), so no dialect branch
	// is needed here.
	var id int64
	err := tx.QueryRow(`INSERT INTO currency_ledger
		(user_id, amount, kind, ref, idempotency_key, comment, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`,
		e.UserID, e.Amount, e.Kind, e.Ref, e.IdempotencyKey, e.Comment, e.CreatedBy,
		now.UTC().Format(time.RFC3339)).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicateEntry
		}
		return 0, fmt.Errorf("insert ledger entry: %w", err)
	}
	return id, nil
}

// isUniqueViolation reports whether err is a unique-index conflict on either
// backend. Both drivers only expose this in the message text, so match on it
// rather than pulling in driver-specific error types.
func isUniqueViolation(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique index")
}

// Balance is always computed, never stored. See the design doc.
func (s *Store) Balance(userID string) (int64, error) {
	var bal int64
	err := s.db.QueryRow(`SELECT COALESCE(SUM(amount), 0) FROM currency_ledger WHERE user_id = ?`,
		userID).Scan(&bal)
	if err != nil {
		return 0, fmt.Errorf("balance: %w", err)
	}
	return bal, nil
}

// balanceTx is the transaction-scoped form; a purchase reads the balance
// inside the same transaction that debits it.
func balanceTx(tx *dbtx, userID string) (int64, error) {
	var bal int64
	err := tx.QueryRow(`SELECT COALESCE(SUM(amount), 0) FROM currency_ledger WHERE user_id = ?`,
		userID).Scan(&bal)
	if err != nil {
		return 0, fmt.Errorf("balance: %w", err)
	}
	return bal, nil
}

// ListLedger returns one user's entries, newest first.
func (s *Store) ListLedger(userID string, limit, offset int) ([]LedgerEntry, error) {
	rows, err := s.db.Query(`SELECT id, user_id, amount, kind, ref, idempotency_key,
		comment, created_by, created_at
		FROM currency_ledger WHERE user_id = ? ORDER BY id DESC LIMIT ? OFFSET ?`,
		userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list ledger: %w", err)
	}
	defer rows.Close()
	var out []LedgerEntry
	for rows.Next() {
		var e LedgerEntry
		var created string
		if err := rows.Scan(&e.ID, &e.UserID, &e.Amount, &e.Kind, &e.Ref,
			&e.IdempotencyKey, &e.Comment, &e.CreatedBy, &created); err != nil {
			return nil, fmt.Errorf("scan ledger entry: %w", err)
		}
		t, err := time.Parse(time.RFC3339, created)
		if err != nil {
			return nil, fmt.Errorf("list ledger: parse created_at: %w", err)
		}
		e.CreatedAt = t
		out = append(out, e)
	}
	return out, rows.Err()
}
