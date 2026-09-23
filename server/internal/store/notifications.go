package store

import (
	"fmt"
	"time"
)

// Notification is one row a user sees in the bell dropdown — the
// notifications/notification_recipients join for a single recipient.
type Notification struct {
	ID        int64
	Text      string
	CreatedAt time.Time
	Read      bool
}

// notificationRetention is how long a *read* notification keeps showing up
// in ListNotificationsForUser after it was read. Unread notifications have
// no expiry.
const notificationRetention = 30 * 24 * time.Hour

// ListNotificationsForUser returns userID's notifications, newest first,
// capped at 50: every unread one, plus read ones from the last 30 days.
func (s *Store) ListNotificationsForUser(userID string, now time.Time) ([]Notification, error) {
	cutoff := now.Add(-notificationRetention).UTC().Format(time.RFC3339)
	rows, err := s.db.Query(`
		SELECT n.id, n.text, n.created_at, nr.read_at
		FROM notifications n
		JOIN notification_recipients nr ON nr.notification_id = n.id
		WHERE nr.user_id = ? AND (nr.read_at = '' OR nr.read_at > ?)
		ORDER BY n.created_at DESC
		LIMIT 50`, userID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()
	var out []Notification
	for rows.Next() {
		var n Notification
		var createdAt, readAt string
		if err := rows.Scan(&n.ID, &n.Text, &createdAt, &readAt); err != nil {
			return nil, fmt.Errorf("list notifications: %w", err)
		}
		t, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("list notifications: parse created_at: %w", err)
		}
		n.CreatedAt = t
		n.Read = readAt != ""
		out = append(out, n)
	}
	return out, rows.Err()
}

// MarkNotificationsRead marks every currently-unread notification of userID
// as read at now. Idempotent — a second call touches zero rows and returns
// no error.
func (s *Store) MarkNotificationsRead(userID string, now time.Time) error {
	_, err := s.db.Exec(`UPDATE notification_recipients SET read_at = ? WHERE user_id = ? AND read_at = ''`,
		now.UTC().Format(time.RFC3339), userID)
	if err != nil {
		return fmt.Errorf("mark notifications read: %w", err)
	}
	return nil
}

// SeedNotificationForTest creates one notification with a single recipient,
// bypassing the normal write path — production notifications are always
// created by ucimo-content-admin writing directly into Postgres (see
// docs/superpowers/specs/2026-09-23-notifications-design.md), so there is no
// ordinary Go writer to seed fixtures with. Exported only so
// internal/api's tests can set up fixtures too; mirrors
// telegram.SetAPIBaseForTesting's "exported test seam" pattern.
func (s *Store) SeedNotificationForTest(userID, text string, at time.Time) (int64, error) {
	createdAt := at.UTC().Format(time.RFC3339)
	if _, err := s.db.Exec(`INSERT INTO notifications (text, created_at) VALUES (?, ?)`, text, createdAt); err != nil {
		return 0, fmt.Errorf("seed notification: %w", err)
	}
	var id int64
	if err := s.db.QueryRow(`SELECT id FROM notifications WHERE text = ? AND created_at = ? ORDER BY id DESC LIMIT 1`,
		text, createdAt).Scan(&id); err != nil {
		return 0, fmt.Errorf("seed notification: find id: %w", err)
	}
	if _, err := s.db.Exec(`INSERT INTO notification_recipients (notification_id, user_id, read_at) VALUES (?, ?, '')`,
		id, userID); err != nil {
		return 0, fmt.Errorf("seed notification: recipient: %w", err)
	}
	return id, nil
}

// SeedNotificationReadAtForTest backdates one recipient row's read_at. Test
// seam only, like SeedNotificationForTest.
func (s *Store) SeedNotificationReadAtForTest(notificationID int64, userID string, readAt time.Time) error {
	_, err := s.db.Exec(`UPDATE notification_recipients SET read_at = ? WHERE notification_id = ? AND user_id = ?`,
		readAt.UTC().Format(time.RFC3339), notificationID, userID)
	if err != nil {
		return fmt.Errorf("seed notification read_at: %w", err)
	}
	return nil
}
