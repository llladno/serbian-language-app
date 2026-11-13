package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// PersonalDiscount is a percent off the tariffs for one learner, until ExpiresAt. The owner makes it in the admin
// panel (which writes the table itself); the app only reads it and prices the learner's order with it.
type PersonalDiscount struct {
	ID        string
	UserID    string
	Percent   int
	Note      string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Bounds of a personal discount. The admin enforces them too; the app ignores a row outside them rather than trust it.
const (
	MinDiscountPercent = 1
	MaxDiscountPercent = 90
)

// ActiveDiscount is the discount userID can use right now: not taken back, not expired and not yet spent by a paid
// order. With several, the biggest wins (then the one that lasts longest). ok is false when there is none.
func (s *Store) ActiveDiscount(userID string, now time.Time) (d PersonalDiscount, ok bool, err error) {
	var created, expires string
	err = s.db.QueryRow(`SELECT d.id, d.user_id, d.percent, d.note, d.created_at, d.expires_at
		FROM personal_discounts d
		WHERE d.user_id = ? AND d.canceled_at = '' AND d.expires_at > ? AND d.percent >= ? AND d.percent <= ?
		  AND NOT EXISTS (SELECT 1 FROM payments p WHERE p.discount_id = d.id AND p.status = ?)
		ORDER BY d.percent DESC, d.expires_at DESC LIMIT 1`,
		userID, now.UTC().Format(time.RFC3339), MinDiscountPercent, MaxDiscountPercent, PaymentSucceeded).
		Scan(&d.ID, &d.UserID, &d.Percent, &d.Note, &created, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return PersonalDiscount{}, false, nil
	}
	if err != nil {
		return PersonalDiscount{}, false, fmt.Errorf("active discount: %w", err)
	}
	d.CreatedAt, _ = time.Parse(time.RFC3339, created)
	d.ExpiresAt, _ = time.Parse(time.RFC3339, expires)
	return d, true, nil
}

// CreateDiscount stores a discount. The admin panel does the same with its own INSERT; this is for tests and tooling.
func (s *Store) CreateDiscount(d PersonalDiscount) error {
	if _, err := s.db.Exec(`INSERT INTO personal_discounts (id, user_id, percent, note, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		d.ID, d.UserID, d.Percent, d.Note, d.CreatedAt.UTC().Format(time.RFC3339), d.ExpiresAt.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("create discount: %w", err)
	}
	return nil
}

// DiscountedKopecks is price with percent off, rounded to a whole ruble (prices here are in whole rubles, and the
// receipt should not carry stray kopecks) and never below one ruble. A percent outside the bounds changes nothing.
func DiscountedKopecks(price int64, percent int) int64 {
	if percent < MinDiscountPercent || percent > MaxDiscountPercent {
		return price
	}
	v := (price*int64(100-percent)/100 + 50) / 100 * 100
	if v < 100 {
		v = 100
	}
	return v
}
