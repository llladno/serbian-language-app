package store

import (
	"fmt"
	"strings"
	"time"
)

// The places that sell the paid tariffs, and what is logged about each: a showing (view), a press on the way
// to the tariffs (click) and the banner being closed (dismiss). The admin's «Воронка тарифов» counts the rows (migration 032).
const (
	TariffEventView  = "view"
	TariffEventClick = "click"
	// TariffEventDismiss is a learner closing a place that sells a tariff without going on (the cross, "Не сейчас").
	TariffEventDismiss = "dismiss"
	// TariffEventEarn is a learner taking the free way out of the window of a paid level or story: «Заработать зёрнышки».
	TariffEventEarn = "earn"
	// TariffEventClaimView and TariffEventClaim belong to the window of a paid level only: зёрнышки already earned and
	// not yet taken were offered to the learner (claim_view), and the learner took them there (claim). The detail says
	// which wording the learner got ("v=button" or "v=link") and how many were waiting ("n=76"). They are their own kinds,
	// not a detail of view/click, so the admin's counts of showings and presses of the tariff stay what they were.
	TariffEventClaimView = "claim_view"
	TariffEventClaim     = "claim"

	// The steps of paying, reported under TariffSourceCheckout: the tariffs screen opened, a pay button pressed,
	// the window that asks where the receipt goes shown / answered / closed, and an order that was refused.
	TariffEventOpen          = "open"
	TariffEventPress         = "press"
	TariffEventReceiptView   = "receipt_view"
	TariffEventReceiptSend   = "receipt_send"
	TariffEventReceiptCancel = "receipt_cancel"
	TariffEventError         = "error"

	// TariffSourceProfileBanner is the banner at the top of the profile.
	TariffSourceProfileBanner = "profile_banner"
	// TariffSourceRepeatLimit is the window that says today's «Повторить» is used up.
	TariffSourceRepeatLimit = "repeat_limit"
	// TariffSourceUnlockModal is the window of a paid level, when зёрнышек are short and a payment is offered.
	TariffSourceUnlockModal = "unlock_modal"
	// TariffSourceStoryModal is the window of a paid story, when зёрнышек are short and a payment is offered.
	TariffSourceStoryModal = "story_modal"
	// TariffSourceCheckout is the tariffs screen and the way from its pay button to the bank, not a place that sells.
	TariffSourceCheckout = "checkout"
)

// maxTariffDetail is the longest detail kept, in characters.
const maxTariffDetail = 64

// tariffEventDedupe is how long the same event of the same learner is taken to be one: a reload or a component
// mounted twice is not a second showing, a showing after a pause is.
const tariffEventDedupe = 30 * time.Second

// ValidTariffEvent reports whether kind and source are ones the app reports: the places that sell take view,
// click, dismiss and earn; the checkout takes the steps of paying.
func ValidTariffEvent(kind, source string) bool {
	switch source {
	case TariffSourceProfileBanner, TariffSourceRepeatLimit, TariffSourceUnlockModal, TariffSourceStoryModal:
		switch kind {
		case TariffEventView, TariffEventClick, TariffEventDismiss, TariffEventEarn:
			return true
		case TariffEventClaimView, TariffEventClaim:
			return source == TariffSourceUnlockModal
		}
	case TariffSourceCheckout:
		switch kind {
		case TariffEventOpen, TariffEventPress, TariffEventReceiptView, TariffEventReceiptSend, TariffEventReceiptCancel, TariffEventError:
			return true
		}
	}
	return false
}

// cleanTariffDetail keeps a detail short and plain: letters, digits and _ : . , = - only, at most maxTariffDetail
// characters. Anything else (a stray space, a quote, markup) is dropped rather than refusing the event.
func cleanTariffDetail(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if b.Len() >= maxTariffDetail {
			break
		}
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '_', r == ':', r == '.', r == ',', r == '=', r == '-':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// RecordTariffEvent logs one showing of a tariff, or one press on the way to it, with no detail. See
// RecordTariffEventDetail.
func (s *Store) RecordTariffEvent(userID, kind, source string, now time.Time) (bool, error) {
	return s.RecordTariffEventDetail(userID, kind, source, "", now)
}

// RecordTariffEventDetail logs one event with a few words of detail (cleaned by cleanTariffDetail). It reports
// whether a row was written: the same event (same detail too) of the same learner within tariffEventDedupe is
// ignored. An unknown kind or source is an error and writes nothing.
func (s *Store) RecordTariffEventDetail(userID, kind, source, detail string, now time.Time) (bool, error) {
	if !ValidTariffEvent(kind, source) {
		return false, fmt.Errorf("tariff event: unknown kind %q or source %q", kind, source)
	}
	detail = cleanTariffDetail(detail)
	at := now.UTC().Format(time.RFC3339)
	since := now.Add(-tariffEventDedupe).UTC().Format(time.RFC3339)
	res, err := s.db.Exec(`INSERT INTO tariff_events (user_id, kind, source, detail, created_at)
		SELECT ?, ?, ?, ?, ?
		WHERE NOT EXISTS (SELECT 1 FROM tariff_events
			WHERE user_id = ? AND kind = ? AND source = ? AND detail = ? AND created_at > ?)`,
		userID, kind, source, detail, at, userID, kind, source, detail, since)
	if err != nil {
		return false, fmt.Errorf("record tariff event: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("record tariff event: %w", err)
	}
	return n > 0, nil
}
