package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// BotMessageText returns the current text for key, ok=false if no row
// exists (fall back to telegram.DefaultMessages — only expected for a key
// added to that map after migration 007 last seeded the table).
func (s *Store) BotMessageText(key string) (text string, ok bool, err error) {
	row := s.db.QueryRow(`SELECT text FROM bot_messages WHERE key = ?`, key)
	if err := row.Scan(&text); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("bot message text: %w", err)
	}
	return text, true, nil
}

// SetBotMessageText overwrites key's text (inserts if the row is somehow
// missing). Production edits come from ucimo-content-admin writing
// directly against Postgres, not through this method — it exists for tests
// and any future in-process tooling.
func (s *Store) SetBotMessageText(key, text string, at time.Time) error {
	_, err := s.db.Exec(`INSERT INTO bot_messages (key, text, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (key) DO UPDATE SET text = excluded.text, updated_at = excluded.updated_at`,
		key, text, at.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("set bot message text: %w", err)
	}
	return nil
}
