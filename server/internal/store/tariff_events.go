package store

import (
	"fmt"
	"time"
)

// The places that sell the paid tariffs, and what is logged about each: a showing (view) and a press on the way
// to the tariffs (click). The admin's «Воронка тарифов» counts the rows (migration 032).
const (
	TariffEventView  = "view"
	TariffEventClick = "click"

	// TariffSourceProfileBanner is the banner at the top of the profile.
	TariffSourceProfileBanner = "profile_banner"
	// TariffSourceRepeatLimit is the window that says today's «Повторить» is used up.
	TariffSourceRepeatLimit = "repeat_limit"
	// TariffSourceUnlockModal is the window of a paid level, when зёрнышек are short and a payment is offered.
	TariffSourceUnlockModal = "unlock_modal"
)

// tariffEventDedupe is how long the same event of the same learner is taken to be one: a reload or a component
// mounted twice is not a second showing, a showing after a pause is.
const tariffEventDedupe = 30 * time.Second

// ValidTariffEvent reports whether kind and source are ones the app reports.
func ValidTariffEvent(kind, source string) bool {
	switch kind {
	case TariffEventView, TariffEventClick:
	default:
		return false
	}
	switch source {
	case TariffSourceProfileBanner, TariffSourceRepeatLimit, TariffSourceUnlockModal:
		return true
	}
	return false
}

// RecordTariffEvent logs one showing of a tariff, or one press on the way to it. It reports whether a row was
// written: the same event of the same learner within tariffEventDedupe is ignored. An unknown kind or source is
// an error and writes nothing.
func (s *Store) RecordTariffEvent(userID, kind, source string, now time.Time) (bool, error) {
	if !ValidTariffEvent(kind, source) {
		return false, fmt.Errorf("tariff event: unknown kind %q or source %q", kind, source)
	}
	at := now.UTC().Format(time.RFC3339)
	since := now.Add(-tariffEventDedupe).UTC().Format(time.RFC3339)
	res, err := s.db.Exec(`INSERT INTO tariff_events (user_id, kind, source, created_at)
		SELECT ?, ?, ?, ?
		WHERE NOT EXISTS (SELECT 1 FROM tariff_events
			WHERE user_id = ? AND kind = ? AND source = ? AND created_at > ?)`,
		userID, kind, source, at, userID, kind, source, since)
	if err != nil {
		return false, fmt.Errorf("record tariff event: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("record tariff event: %w", err)
	}
	return n > 0, nil
}
