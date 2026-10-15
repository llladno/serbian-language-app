package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// BotPrefs is which categories of Telegram message one account accepts. See
// migration 023 for what each category covers and why News starts off.
type BotPrefs struct {
	Learning bool
	Billing  bool
	News     bool
}

// DefaultBotPrefs is what an account that never opened its settings gets.
var DefaultBotPrefs = BotPrefs{Learning: true, Billing: true, News: false}

// BotPrefs reads userID's choices, or the defaults when it has made none.
func (s *Store) BotPrefs(userID string) (BotPrefs, error) {
	var learning, billing, news int
	err := s.db.QueryRow(`SELECT learning, billing, news FROM bot_prefs WHERE user_id = ?`, userID).
		Scan(&learning, &billing, &news)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultBotPrefs, nil
	}
	if err != nil {
		return BotPrefs{}, fmt.Errorf("bot prefs: %w", err)
	}
	return BotPrefs{Learning: learning != 0, Billing: billing != 0, News: news != 0}, nil
}

// SetBotPrefs stores userID's choices, replacing any earlier ones.
func (s *Store) SetBotPrefs(userID string, p BotPrefs, at time.Time) error {
	b2i := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	if _, err := s.db.Exec(`INSERT INTO bot_prefs (user_id, learning, billing, news, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET learning = excluded.learning,
			billing = excluded.billing, news = excluded.news, updated_at = excluded.updated_at`,
		userID, b2i(p.Learning), b2i(p.Billing), b2i(p.News), at.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("set bot prefs: %w", err)
	}
	return nil
}
