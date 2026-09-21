package store

import (
	"fmt"
	"strings"
	"time"
)

// SupportMessage is one user-submitted note from the profile page's support
// card — a bug report, a wish, or a plain question. Read-only from here on;
// the admin panel lists them, nothing marks them resolved.
type SupportMessage struct {
	ID        int64
	UserID    string
	Message   string
	CreatedAt time.Time
}

// CreateSupportMessage stores one support message for userID. message is
// trimmed; blank input is rejected rather than silently stored.
func (s *Store) CreateSupportMessage(userID, message string, at time.Time) error {
	msg := strings.TrimSpace(message)
	if msg == "" {
		return fmt.Errorf("empty message")
	}
	if _, err := s.db.Exec(`INSERT INTO support_messages (user_id, message, created_at) VALUES (?, ?, ?)`,
		userID, msg, at.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("create support message: %w", err)
	}
	return nil
}

// ListSupportMessages returns every support message, newest first.
func (s *Store) ListSupportMessages() ([]SupportMessage, error) {
	rows, err := s.db.Query(`SELECT id, user_id, message, created_at FROM support_messages ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list support messages: %w", err)
	}
	defer rows.Close()
	var out []SupportMessage
	for rows.Next() {
		var m SupportMessage
		var createdAt string
		if err := rows.Scan(&m.ID, &m.UserID, &m.Message, &createdAt); err != nil {
			return nil, fmt.Errorf("list support messages: %w", err)
		}
		t, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("list support messages: parse created_at: %w", err)
		}
		m.CreatedAt = t
		out = append(out, m)
	}
	return out, rows.Err()
}
