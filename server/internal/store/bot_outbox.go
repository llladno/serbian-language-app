package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// Outbox message priorities — lower runs first. See telegram.PriorityHigh/
// PriorityNormal, which this package intentionally does not import (this
// table's row shape stays plain strings/ints, not Telegram's wire types).
const (
	PriorityHigh   = 0
	PriorityNormal = 1
)

// OutboxButton is the inline button (if any) attached to a queued message.
// Type is "web_app", "url", or "" for no button.
type OutboxButton struct {
	Label  string
	Type   string
	Target string
}

// OutboxMessage is one row dequeued from bot_outbox.
type OutboxMessage struct {
	ID     int64
	ChatID int64
	Text   string
	Button OutboxButton
}

// EnqueueBotMessage queues text for chatID at priority (PriorityHigh or
// PriorityNormal). internal/outbox.ProcessNext is the only thing that ever
// actually calls the Bot API — everything else just enqueues.
func (s *Store) EnqueueBotMessage(chatID int64, text string, button OutboxButton, priority int, at time.Time) error {
	_, err := s.db.Exec(`INSERT INTO bot_outbox
		(chat_id, text, button_label, button_type, button_target, priority, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 'pending', ?)`,
		strconv.FormatInt(chatID, 10), text, button.Label, button.Type, button.Target, priority,
		at.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("enqueue bot message: %w", err)
	}
	return nil
}

// NextPendingOutboxMessage returns the oldest highest-priority pending
// message, ok=false if the queue is empty.
func (s *Store) NextPendingOutboxMessage() (OutboxMessage, bool, error) {
	row := s.db.QueryRow(`SELECT id, chat_id, text, button_label, button_type, button_target
		FROM bot_outbox WHERE status = 'pending' ORDER BY priority ASC, created_at ASC LIMIT 1`)
	var m OutboxMessage
	var chatID string
	if err := row.Scan(&m.ID, &chatID, &m.Text, &m.Button.Label, &m.Button.Type, &m.Button.Target); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return OutboxMessage{}, false, nil
		}
		return OutboxMessage{}, false, fmt.Errorf("next outbox message: %w", err)
	}
	parsed, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return OutboxMessage{}, false, fmt.Errorf("next outbox message: bad chat_id %q: %w", chatID, err)
	}
	m.ChatID = parsed
	return m, true, nil
}

// MarkOutboxSent records a successful send.
func (s *Store) MarkOutboxSent(id int64, at time.Time) error {
	_, err := s.db.Exec(`UPDATE bot_outbox SET status = 'sent', sent_at = ? WHERE id = ?`,
		at.UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("mark outbox sent: %w", err)
	}
	return nil
}

// MarkOutboxFailed records a permanent failure. Never called for a 429 —
// internal/outbox.ProcessNext leaves those rows pending and retries later.
func (s *Store) MarkOutboxFailed(id int64, errText string) error {
	_, err := s.db.Exec(`UPDATE bot_outbox SET status = 'failed', error = ? WHERE id = ?`,
		errText, id)
	if err != nil {
		return fmt.Errorf("mark outbox failed: %w", err)
	}
	return nil
}
