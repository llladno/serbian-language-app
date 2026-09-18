package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// TelegramUser is one account reachable through the bot: a numeric chat id
// resolved from an attached "telegram" identity (provider_uid is the tg id
// once AttachPendingTelegram has run — a still-pending "pending:<username>"
// row has no chat to send to and is excluded).
type TelegramUser struct {
	UserID string
	ChatID int64
}

// TelegramLinkedUsers returns every account with a usable Telegram chat,
// for the reminder sweep (internal/api RunReminderSweep) to iterate.
func (s *Store) TelegramLinkedUsers() ([]TelegramUser, error) {
	rows, err := s.db.Query(`SELECT user_id, provider_uid FROM identities
		WHERE provider = 'telegram' AND provider_uid NOT LIKE 'pending:%'`)
	if err != nil {
		return nil, fmt.Errorf("telegram linked users: %w", err)
	}
	defer rows.Close()
	var out []TelegramUser
	for rows.Next() {
		var userID, uid string
		if err := rows.Scan(&userID, &uid); err != nil {
			return nil, fmt.Errorf("telegram linked users: %w", err)
		}
		chatID, err := strconv.ParseInt(uid, 10, 64)
		if err != nil {
			continue // shouldn't happen outside "pending:...", but never crash the sweep over one bad row
		}
		out = append(out, TelegramUser{UserID: userID, ChatID: chatID})
	}
	return out, rows.Err()
}

// LastActivityByUser returns, per user id, the timestamp of their most
// recent review or exercise attempt (whichever is later) — the same two
// tables that back the streak/last-active dashboard stats, but keyed to the
// instant rather than the calendar day so the reminder sweep can compare
// against a rolling 24h window. A user absent from the result has never
// logged either kind of activity.
func (s *Store) LastActivityByUser() (map[string]time.Time, error) {
	rows, err := s.db.Query(`SELECT user_id, MAX(t) FROM (
		SELECT user_id, reviewed_at AS t FROM reviews
		UNION ALL SELECT user_id, attempted_at AS t FROM attempts
	) AS activity GROUP BY user_id`)
	if err != nil {
		return nil, fmt.Errorf("last activity by user: %w", err)
	}
	defer rows.Close()
	out := map[string]time.Time{}
	for rows.Next() {
		var id, ts string
		if err := rows.Scan(&id, &ts); err != nil {
			return nil, fmt.Errorf("last activity by user: %w", err)
		}
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			continue
		}
		out[id] = t
	}
	return out, rows.Err()
}

// BotReminderState is one account's dedup bookkeeping for the reminder
// sweep. The zero value (no bot_reminders row yet) means neither kind of
// message has ever been sent.
type BotReminderState struct {
	AllDoneDate    string // "" or "YYYY-MM-DD": last date the "all done" congrats went out
	InactiveSentAt time.Time
}

// BotReminderState reads userID's dedup state, defaulting to the zero value
// when no row exists yet (first sweep for this account).
func (s *Store) BotReminderState(userID string) (BotReminderState, error) {
	var allDone sql.NullString
	var inactive sql.NullString
	err := s.db.QueryRow(`SELECT all_done_date, inactive_sent_at FROM bot_reminders WHERE user_id = ?`,
		userID).Scan(&allDone, &inactive)
	if errors.Is(err, sql.ErrNoRows) {
		return BotReminderState{}, nil
	}
	if err != nil {
		return BotReminderState{}, fmt.Errorf("bot reminder state: %w", err)
	}
	st := BotReminderState{AllDoneDate: allDone.String}
	if inactive.String != "" {
		if t, err := time.Parse(time.RFC3339, inactive.String); err == nil {
			st.InactiveSentAt = t
		}
	}
	return st, nil
}

// SetBotReminderAllDoneDate records dateStr (today, "YYYY-MM-DD") as the last
// date the "all done" congrats was sent to userID.
func (s *Store) SetBotReminderAllDoneDate(userID, dateStr string) error {
	if _, err := s.db.Exec(`INSERT INTO bot_reminders (user_id, all_done_date) VALUES (?, ?)
		ON CONFLICT (user_id) DO UPDATE SET all_done_date = excluded.all_done_date`,
		userID, dateStr); err != nil {
		return fmt.Errorf("set bot reminder all-done date: %w", err)
	}
	return nil
}

// SetBotReminderInactiveSentAt records at as the last time the "come back"
// nudge was sent to userID.
func (s *Store) SetBotReminderInactiveSentAt(userID string, at time.Time) error {
	if _, err := s.db.Exec(`INSERT INTO bot_reminders (user_id, inactive_sent_at) VALUES (?, ?)
		ON CONFLICT (user_id) DO UPDATE SET inactive_sent_at = excluded.inactive_sent_at`,
		userID, at.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("set bot reminder inactive-sent-at: %w", err)
	}
	return nil
}
