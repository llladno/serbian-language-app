// Package outbox drains bot_outbox at a pace that stays under Telegram's
// documented Bot API limits (core.telegram.org/bots/faq: 1 msg/s per chat,
// 30 msg/s bulk) — server/main.go runs ProcessNext from a ticker at
// time.Second/25. This is the only code in serbian-app that actually calls
// the Bot API to send a message; everything else (the /start webhook,
// reminders, admin broadcasts) enqueues into bot_outbox instead.
package outbox

import (
	"time"

	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
)

// ProcessNext sends at most one queued message — the oldest, highest
// priority pending row. ok is false if the queue was empty (nothing to
// do). On a Telegram 429, the row is left pending and retryAfter tells the
// caller how long to pause before calling ProcessNext again; any other
// send error marks the row permanently failed (no retry — the usual cause
// is the user having blocked the bot).
func ProcessNext(st *store.Store, botToken string, now time.Time) (ok bool, retryAfter time.Duration, err error) {
	msg, found, err := st.NextPendingOutboxMessage()
	if err != nil {
		return false, 0, err
	}
	if !found {
		return false, 0, nil
	}

	var button *telegram.InlineButton
	if msg.Button.Type != "" {
		button = &telegram.InlineButton{Label: msg.Button.Label}
		switch msg.Button.Type {
		case "web_app":
			button.WebAppURL = msg.Button.Target
		case "url":
			button.URL = msg.Button.Target
		}
	}

	sendErr := telegram.SendMessageWithButton(botToken, msg.ChatID, msg.Text, button)
	if sendErr == nil {
		if err := st.MarkOutboxSent(msg.ID, now); err != nil {
			return true, 0, err
		}
		return true, 0, nil
	}
	if wait, limited := telegram.RateLimited(sendErr); limited {
		return true, wait, nil
	}
	if err := st.MarkOutboxFailed(msg.ID, sendErr.Error()); err != nil {
		return true, 0, err
	}
	return true, 0, nil
}
