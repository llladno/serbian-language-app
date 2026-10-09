package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
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
// failing the request. The currency-name defaults are exactly the values
// migrations 013 + 017 leave behind, so a table that lost those rows (never
// migrated, or an admin deleted them) behaves identically to a freshly seeded
// one instead of surfacing blank display strings.
const (
	defaultDailyGoal         = 10
	defaultRepairWindowHours = 48
	defaultDrip              = `[[1,1],[30,2],[100,3]]`
	defaultCurrencyNameOne   = "зёрнышко"
	defaultCurrencyNameFew   = "зёрнышка"
	defaultCurrencyNameMany  = "зёрнышек"
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

// querier is the read/write surface shared by *database and *dbtx, so helpers
// can run either standalone or inside someone else's transaction.
type querier interface {
	Exec(q string, a ...any) (sql.Result, error)
	Query(q string, a ...any) (*sql.Rows, error)
	QueryRow(q string, a ...any) *sql.Row
}

// EconomySettings reads and parses every setting in one query.
func (s *Store) EconomySettings() (Settings, error) { return economySettings(s.db) }

// economySettings is EconomySettings against any querier, so a counted action
// can read the settings inside its own transaction.
func economySettings(q querier) (Settings, error) {
	rows, err := q.Query(`SELECT key, value FROM economy_settings`)
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

	str := func(key, def string) string {
		if v, ok := raw[key]; ok && v != "" {
			return v
		}
		return def
	}

	// positiveOrDefault floors at def for anything <= 0. Used only for
	// daily_goal: a goal of 0 can never be met, so a later task's "pay the
	// drip when actions == goal" would silently never fire again. That
	// makes 0 a misconfiguration to correct, not an intent to honor.
	positiveOrDefault := func(key string, def int) int {
		n, err := strconv.Atoi(raw[key])
		if err != nil || n <= 0 {
			return def
		}
		return n
	}
	// nonNegativeOrDefault accepts 0 — unlike daily_goal, a repair window of
	// 0 hours is a coherent admin intent ("no repairs": the deadline is the
	// end of the missed day, so every repair request is refused, which is
	// exactly what disabling looks like). Only a negative or unparseable
	// value is treated as a misconfiguration.
	nonNegativeOrDefault := func(key string, def int) int {
		n, err := strconv.Atoi(raw[key])
		if err != nil || n < 0 {
			return def
		}
		return n
	}

	ladder, err := economy.ParseLadder(raw["streak_drip"])
	if err != nil {
		ladder, _ = economy.ParseLadder(defaultDrip)
	}
	return Settings{
		CurrencyNameOne:  str("currency_name_one", defaultCurrencyNameOne),
		CurrencyNameFew:  str("currency_name_few", defaultCurrencyNameFew),
		CurrencyNameMany: str("currency_name_many", defaultCurrencyNameMany),
		DailyGoal:        positiveOrDefault("daily_goal", defaultDailyGoal),
		Drip:             ladder,
		RepairWindowHours: nonNegativeOrDefault(
			"streak_repair_window_hours", defaultRepairWindowHours),
		// No fallback: empty is already a meaningful value here — a later
		// task treats an empty channel as "hide the subscription quest
		// entirely", so inventing a default channel would be worse than
		// leaving it blank. Do not "fix" this for consistency with the
		// other string fields above.
		TelegramChannel: raw["telegram_channel"],
	}, nil
}

// fallbackTZ is the course's home timezone: the audience is Serbia and
// Russia, so it is never more than a couple of hours off, and it is what the
// backfill uses for history where no real timezone is known.
const fallbackTZ = "Europe/Belgrade"

// UserLocation returns the user's timezone, never nil. An unknown user, an
// unreadable row or a name the runtime cannot load all degrade to
// Europe/Belgrade, and finally to UTC if even that is unavailable (a Go build
// without tzdata). An unknown user (sql.ErrNoRows) is an ordinary, expected
// miss and stays silent; any other scan error — a genuine connectivity or
// driver failure — is logged before falling back, since this function backs
// every counted action and a systemic DB read problem must leave a trace an
// operator can find, not just a silently wrong date.
func (s *Store) UserLocation(userID string) *time.Location {
	return userLocation(s.db, userID)
}

// userLocation is UserLocation against any querier, so a counted action can
// resolve the timezone inside its own transaction. Same fallbacks.
func userLocation(q querier, userID string) *time.Location {
	var name string
	if err := q.QueryRow(`SELECT timezone FROM users WHERE id = ?`, userID).Scan(&name); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("user location: scan timezone for user %q: %v", userID, err)
		}
		name = fallbackTZ
	}
	if loc, err := time.LoadLocation(name); err == nil {
		return loc
	}
	if loc, err := time.LoadLocation(fallbackTZ); err == nil {
		return loc
	}
	return time.UTC
}

// SetUserTimezone stores an IANA timezone name after checking the runtime can
// load it. Days already written to user_daily_activity are never recomputed,
// so changing this cannot rewrite past streaks.
func (s *Store) SetUserTimezone(userID, tz string) error {
	if _, err := time.LoadLocation(tz); err != nil {
		return fmt.Errorf("unknown timezone %q: %w", tz, err)
	}
	if _, err := s.db.Exec(`UPDATE users SET timezone = ? WHERE id = ?`, tz, userID); err != nil {
		return fmt.Errorf("set timezone: %w", err)
	}
	return nil
}
