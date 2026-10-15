package telegram

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// PrefCategory is one kind of message a person can switch off. The store keeps
// one column per category (migration 023); service messages (sign-in, a
// payment received) are deliberately not here, they cannot be switched off.
type PrefCategory string

const (
	PrefLearning PrefCategory = "learning" // study reminders
	PrefBilling  PrefCategory = "billing"  // reminders about a paid tariff
	PrefNews     PrefCategory = "news"     // new levels, offers, announcements
)

// PrefCategories is the order the settings screen lists them in.
var PrefCategories = []PrefCategory{PrefLearning, PrefBilling, PrefNews}

// PrefTitles is what each category is called to the person.
var PrefTitles = map[PrefCategory]string{
	PrefLearning: "Напоминания об учёбе",
	PrefBilling:  "Про тариф и оплату",
	PrefNews:     "Новые уровни и предложения",
}

// PrefsState is which categories are on.
type PrefsState struct {
	Learning, Billing, News bool
}

// On reports whether category c is on.
func (p PrefsState) On(c PrefCategory) bool {
	switch c {
	case PrefLearning:
		return p.Learning
	case PrefBilling:
		return p.Billing
	case PrefNews:
		return p.News
	}
	return false
}

// With returns p with category c set to on.
func (p PrefsState) With(c PrefCategory, on bool) PrefsState {
	switch c {
	case PrefLearning:
		p.Learning = on
	case PrefBilling:
		p.Billing = on
	case PrefNews:
		p.News = on
	}
	return p
}

// Encode/DecodePrefsState carry the state through the outbox's single "target"
// column: one 0/1 per category, in PrefCategories order ("101").
func (p PrefsState) Encode() string {
	var b strings.Builder
	for _, c := range PrefCategories {
		if p.On(c) {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
	}
	return b.String()
}

func DecodePrefsState(s string) PrefsState {
	var p PrefsState
	for i, c := range PrefCategories {
		if i < len(s) && s[i] == '1' {
			p = p.With(c, true)
		}
	}
	return p
}

// The two callback_data shapes the webhook answers.
//
//	prefoff:<category>        the "stop these" button under a reminder
//	pref:<category>:<0|1>     a toggle on the settings screen (the value to set)
const (
	CallbackPrefOff = "prefoff:"
	CallbackPref    = "pref:"
	PrefOffLabel    = "🔕 Не напоминать"
)

// ParsePrefOff reads "prefoff:<category>".
func ParsePrefOff(data string) (PrefCategory, bool) {
	rest, ok := strings.CutPrefix(data, CallbackPrefOff)
	if !ok {
		return "", false
	}
	return knownCategory(rest)
}

// ParsePref reads "pref:<category>:<0|1>".
func ParsePref(data string) (c PrefCategory, on bool, ok bool) {
	rest, found := strings.CutPrefix(data, CallbackPref)
	if !found {
		return "", false, false
	}
	name, val, found := strings.Cut(rest, ":")
	if !found || (val != "0" && val != "1") {
		return "", false, false
	}
	c, ok = knownCategory(name)
	if !ok {
		return "", false, false
	}
	return c, val == "1", true
}

func knownCategory(name string) (PrefCategory, bool) {
	for _, c := range PrefCategories {
		if string(c) == name {
			return c, true
		}
	}
	return "", false
}

// prefsRows is the settings keyboard: one row per category, a tick when it is
// on, and a press sets the opposite.
func prefsRows(p PrefsState) [][]map[string]any {
	rows := make([][]map[string]any, 0, len(PrefCategories))
	for _, c := range PrefCategories {
		mark, next := "⬜", "1"
		if p.On(c) {
			mark, next = "✅", "0"
		}
		rows = append(rows, []map[string]any{{
			"text":          mark + " " + PrefTitles[c],
			"callback_data": CallbackPref + string(c) + ":" + next,
		}})
	}
	return rows
}

// EditPrefsKeyboard redraws the settings keyboard under an existing message, so
// a toggle shows its new state without sending anything new.
func EditPrefsKeyboard(botToken string, chatID int64, messageID int64, p PrefsState) error {
	return editKeyboard(botToken, chatID, messageID, prefsRows(p))
}

// RemoveKeyboard takes the buttons off a message, used once "stop these" has
// been pressed so it cannot be pressed twice.
func RemoveKeyboard(botToken string, chatID int64, messageID int64) error {
	return editKeyboard(botToken, chatID, messageID, [][]map[string]any{})
}

func editKeyboard(botToken string, chatID, messageID int64, rows [][]map[string]any) error {
	markup, err := json.Marshal(map[string]any{"inline_keyboard": rows})
	if err != nil {
		return fmt.Errorf("telegram editMessageReplyMarkup: marshal: %w", err)
	}
	params := url.Values{
		"chat_id":      {fmt.Sprintf("%d", chatID)},
		"message_id":   {fmt.Sprintf("%d", messageID)},
		"reply_markup": {string(markup)},
	}
	return call(botToken, "editMessageReplyMarkup", params, nil)
}

// IsCommand reports whether text is exactly the command "/name", optionally
// "/name@botname", with nothing after it.
func IsCommand(text, name string) bool {
	fields := strings.Fields(text)
	if len(fields) != 1 {
		return false
	}
	cmd, _, _ := strings.Cut(fields[0], "@")
	return cmd == "/"+name
}
