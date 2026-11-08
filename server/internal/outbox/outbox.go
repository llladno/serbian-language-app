// Package outbox drains bot_outbox at a pace that stays under Telegram's
// documented Bot API limits (core.telegram.org/bots/faq: 1 msg/s per chat,
// 30 msg/s bulk) — server/main.go runs ProcessNext from a ticker at
// time.Second/25. This is the only code in serbian-app that actually calls
// the Bot API to send a message; everything else (the /start webhook,
// reminders, admin broadcasts) enqueues into bot_outbox instead.
package outbox

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
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

	// Claim the row before sending: another worker may have read it too, and a message must never go out twice.
	claimed, err := st.ClaimOutboxMessage(msg.ID)
	if err != nil {
		return false, 0, err
	}
	if !claimed {
		return true, 0, nil // somebody else has it: look at the next one
	}

	button := buttonFromRow(msg.Button)

	var sendErr error
	if msg.Photo != "" {
		sendErr = sendPhoto(st, botToken, msg, button, now)
	} else {
		sendErr = telegram.SendMessageWithButton(botToken, msg.ChatID, msg.Text, button)
	}
	if sendErr == nil {
		if err := st.MarkOutboxSent(msg.ID, now); err != nil {
			return true, 0, err
		}
		return true, 0, nil
	}
	if wait, limited := telegram.RateLimited(sendErr); limited {
		if err := st.ReleaseOutboxMessage(msg.ID); err != nil {
			return true, 0, err
		}
		return true, wait, nil
	}
	if err := st.MarkOutboxFailed(msg.ID, sendErr.Error()); err != nil {
		return true, 0, err
	}
	return true, 0, nil
}

// sendPhoto sends a queued photo. Telegram downloads the image from its address once; the file_id it answers with is kept (in
// the table the lead-magnet files use, under a key made from the address), and every later send goes out by that id. A cached id
// Telegram no longer accepts falls back to the address.
func sendPhoto(st *store.Store, botToken string, msg store.OutboxMessage, button *telegram.InlineButton, now time.Time) error {
	sum := sha256.Sum256([]byte(msg.Photo))
	slug := "photo:" + hex.EncodeToString(sum[:8])
	if id, ok, err := st.LeadMagnetFileID(slug, msg.Photo); err != nil {
		log.Printf("outbox: photo file id lookup: %v", err)
	} else if ok && id != "" {
		if _, err := telegram.SendPhoto(botToken, msg.ChatID, telegram.Photo{FileID: id}, msg.Text, button); err == nil {
			return nil
		} else if _, limited := telegram.RateLimited(err); limited {
			return err
		} else {
			log.Printf("outbox: resend photo by file_id failed, fetching again: %v", err)
		}
	}
	id, err := telegram.SendPhoto(botToken, msg.ChatID, telegram.Photo{URL: msg.Photo}, msg.Text, button)
	if err != nil {
		return err
	}
	if id != "" {
		if saveErr := st.SaveLeadMagnetFileID(slug, msg.Photo, id, now); saveErr != nil {
			log.Printf("outbox: save photo file id: %v", saveErr)
		}
	}
	return nil
}

// ButtonRow is how a button is stored in bot_outbox: one (label, type, target)
// triple, whatever the keyboard. Presets (the channel pair, the settings
// toggles) keep their whole state in target.
func ButtonRow(b *telegram.InlineButton) store.OutboxButton {
	if b == nil {
		return store.OutboxButton{}
	}
	ob := store.OutboxButton{Label: b.Label}
	switch {
	case b.Prefs != nil:
		ob.Type, ob.Target = "prefs", b.Prefs.Encode()
	case b.PrefOff != "":
		ob.Type, ob.Target = "pref_off", string(b.PrefOff)
	case b.RepeatNudgeURL != "":
		ob.Type, ob.Target = "repeat_nudge", b.RepeatNudgeURL
	case b.CampaignURL != "":
		ob.Type, ob.Target = "campaign", b.CampaignURL
	case b.ChannelURL != "":
		ob.Type, ob.Target = "channel_check", b.ChannelURL
	case b.WebAppURL != "":
		ob.Type, ob.Target = "web_app", b.WebAppURL
	default:
		ob.Type, ob.Target = "url", b.URL
	}
	return ob
}

// buttonFromRow is ButtonRow's inverse, used when the row is finally sent.
func buttonFromRow(ob store.OutboxButton) *telegram.InlineButton {
	if ob.Type == "" {
		return nil
	}
	b := &telegram.InlineButton{Label: ob.Label}
	switch ob.Type {
	case "web_app":
		b.WebAppURL = ob.Target
	case "url":
		b.URL = ob.Target
	case "repeat_nudge":
		b.RepeatNudgeURL = ob.Target
	case "campaign":
		b.CampaignURL = ob.Target
	case "channel_check":
		b.ChannelURL = ob.Target
	case "pref_off":
		b.PrefOff = telegram.PrefCategory(ob.Target)
	case "prefs":
		st := telegram.DecodePrefsState(ob.Target)
		b.Prefs = &st
	}
	return b
}
