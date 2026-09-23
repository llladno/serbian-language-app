package store

import (
	"fmt"
	"time"
)

// Donation is one payment event from the Tribute webhook — a new donation, a
// recurring one, or a cancellation (see internal/api/tribute.go). Read-only
// from here on; the admin panel lists them directly against this repo's
// Postgres, same as SupportMessage. ListDonations exists for tests — no
// handler in this repo calls it.
type Donation struct {
	ID               int64
	UserID           string // "" when the donor's telegram id matched no account
	TelegramUserID   string
	TelegramUsername string // "" if Tribute didn't send one
	AmountMinorUnits int64
	Currency         string
	EventType        string // "newDonation" | "recurrentDonation" | "cancelledDonation" | ...
	TributeEventID   string
	RawPayload       string
	CreatedAt        time.Time
}

// CreateDonation stores one Tribute webhook event. Idempotent on
// tributeEventID (via the unique index in migration 011): a retried webhook
// delivery for an event already stored is silently ignored rather than
// double-counted.
func (s *Store) CreateDonation(d Donation, at time.Time) error {
	if d.TelegramUserID == "" {
		return fmt.Errorf("empty telegram_user_id")
	}
	if d.TributeEventID == "" {
		return fmt.Errorf("empty tribute event id")
	}
	if _, err := s.db.Exec(`INSERT INTO donations
		(user_id, telegram_user_id, telegram_username, amount_minor_units, currency, event_type, tribute_event_id, raw_payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (tribute_event_id) DO NOTHING`,
		nullIf(d.UserID), d.TelegramUserID, nullIf(d.TelegramUsername), d.AmountMinorUnits,
		d.Currency, d.EventType, d.TributeEventID, d.RawPayload, at.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("create donation: %w", err)
	}
	return nil
}

// ListDonations returns every donation, newest first.
func (s *Store) ListDonations() ([]Donation, error) {
	rows, err := s.db.Query(`SELECT id, COALESCE(user_id,''), telegram_user_id, COALESCE(telegram_username,''),
		amount_minor_units, currency, event_type, tribute_event_id, raw_payload, created_at
		FROM donations ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list donations: %w", err)
	}
	defer rows.Close()
	var out []Donation
	for rows.Next() {
		var d Donation
		var createdAt string
		if err := rows.Scan(&d.ID, &d.UserID, &d.TelegramUserID, &d.TelegramUsername,
			&d.AmountMinorUnits, &d.Currency, &d.EventType, &d.TributeEventID, &d.RawPayload, &createdAt); err != nil {
			return nil, fmt.Errorf("list donations: %w", err)
		}
		t, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("list donations: parse created_at: %w", err)
		}
		d.CreatedAt = t
		out = append(out, d)
	}
	return out, rows.Err()
}
