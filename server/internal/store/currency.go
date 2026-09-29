package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/grisha/serbian-app/server/internal/economy"
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
// so the credit and its side effects commit together. A duplicate
// idempotency_key is reported as ErrDuplicateEntry without failing the
// INSERT statement itself (see the ON CONFLICT DO NOTHING below) — so, unlike
// a driver-level constraint-violation error, it does not poison the
// enclosing transaction on Postgres. A caller that gets ErrDuplicateEntry may
// keep using tx afterwards and commit it.
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

	// ON CONFLICT DO NOTHING means a duplicate key never fails the
	// statement — it just returns no row, which we read as ErrDuplicateEntry
	// below. currency_ledger has exactly one unique index (idempotency_key,
	// whose entire purpose is this check), so DO NOTHING can't be hiding any
	// other conflict. RETURNING works via QueryRow on both backends
	// (verified against modernc.org/sqlite v1.58.0 as well as Postgres) even
	// when combined with ON CONFLICT DO NOTHING, so no dialect branch is
	// needed here.
	var id int64
	err := tx.QueryRow(`INSERT INTO currency_ledger
		(user_id, amount, kind, ref, idempotency_key, comment, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id`,
		e.UserID, e.Amount, e.Kind, e.Ref, e.IdempotencyKey, e.Comment, e.CreatedBy,
		now.UTC().Format(time.RFC3339)).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrDuplicateEntry
		}
		return 0, fmt.Errorf("insert ledger entry: %w", err)
	}
	return id, nil
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

// Defaults used when a setting is missing or unparseable. The admin panel can
// write anything into economy_settings, so every read falls back rather than
// failing the request.
const (
	defaultDailyGoal         = 10
	defaultRepairWindowHours = 48
	defaultDrip              = `[[1,1],[30,2],[100,3]]`
)

// Settings is the parsed economy_settings table.
type Settings struct {
	CurrencyNameOne   string
	CurrencyNameFew   string
	CurrencyNameMany  string
	DailyGoal         int
	Drip              economy.Ladder
	RepairWindowHours int
	TelegramChannel   string
}

// EconomySettings reads and parses every setting in one query.
func (s *Store) EconomySettings() (Settings, error) {
	rows, err := s.db.Query(`SELECT key, value FROM economy_settings`)
	if err != nil {
		return Settings{}, fmt.Errorf("economy settings: %w", err)
	}
	defer rows.Close()
	raw := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return Settings{}, fmt.Errorf("scan economy setting: %w", err)
		}
		raw[k] = v
	}
	if err := rows.Err(); err != nil {
		return Settings{}, fmt.Errorf("iterate economy settings: %w", err)
	}

	atoi := func(key string, def int) int {
		n, err := strconv.Atoi(raw[key])
		if err != nil || n <= 0 {
			return def
		}
		return n
	}
	ladder, err := economy.ParseLadder(raw["streak_drip"])
	if err != nil {
		ladder, _ = economy.ParseLadder(defaultDrip)
	}
	return Settings{
		CurrencyNameOne:   raw["currency_name_one"],
		CurrencyNameFew:   raw["currency_name_few"],
		CurrencyNameMany:  raw["currency_name_many"],
		DailyGoal:         atoi("daily_goal", defaultDailyGoal),
		Drip:              ladder,
		RepairWindowHours: atoi("streak_repair_window_hours", defaultRepairWindowHours),
		TelegramChannel:   raw["telegram_channel"],
	}, nil
}
