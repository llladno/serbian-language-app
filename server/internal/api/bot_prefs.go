package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
)

// Which Telegram messages an account accepts. One switch per category (see
// migration 023); service messages (sign-in, a payment received, access opened)
// are not switchable and are not here. The same choices can be made in three
// places: the profile (these endpoints), a "stop" button under a reminder, and
// the bot's /stop and /settings commands.

type botPrefsDTO struct {
	Learning bool `json:"learning"`
	Billing  bool `json:"billing"`
	News     bool `json:"news"`
}

func prefsDTO(p store.BotPrefs) botPrefsDTO {
	return botPrefsDTO{Learning: p.Learning, Billing: p.Billing, News: p.News}
}

func prefsState(p store.BotPrefs) telegram.PrefsState {
	return telegram.PrefsState{Learning: p.Learning, Billing: p.Billing, News: p.News}
}

func storePrefs(s telegram.PrefsState) store.BotPrefs {
	return store.BotPrefs{Learning: s.Learning, Billing: s.Billing, News: s.News}
}

func (h handlers) getBotPrefs(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	p, err := h.Store.BotPrefs(ac.UserID)
	if err != nil {
		log.Printf("bot prefs: get: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, prefsDTO(p))
}

// putBotPrefs applies the fields that are present and leaves the rest, so a
// single toggle never has to resend (and possibly overwrite) the other two.
func (h handlers) putBotPrefs(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	var req struct {
		Learning *bool `json:"learning"`
		Billing  *bool `json:"billing"`
		News     *bool `json:"news"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	p, err := h.Store.BotPrefs(ac.UserID)
	if err != nil {
		log.Printf("bot prefs: load: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	if req.Learning != nil {
		p.Learning = *req.Learning
	}
	if req.Billing != nil {
		p.Billing = *req.Billing
	}
	if req.News != nil {
		p.News = *req.News
	}
	if err := h.Store.SetBotPrefs(ac.UserID, p, h.Now()); err != nil {
		log.Printf("bot prefs: save: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, prefsDTO(p))
}

// userForTelegram finds the account a Telegram user id is linked to.
func (h handlers) userForTelegram(tgID int64) (string, bool) {
	id, err := h.Store.IdentityByProviderUID("telegram", strconv.FormatInt(tgID, 10))
	if err != nil {
		return "", false
	}
	return id.UserID, true
}

// botStop is /stop: every optional category off at once. It is the one thing a
// person who is annoyed by the bot can type without finding the settings.
func (h handlers) botStop(chatID, tgID int64, name string) {
	uid, ok := h.userForTelegram(tgID)
	if !ok {
		h.sendBotMessage(chatID, telegram.MsgPrefsNoAccount, name, nil)
		return
	}
	if err := h.Store.SetBotPrefs(uid, store.BotPrefs{}, h.Now()); err != nil {
		log.Printf("telegram webhook: /stop: %v", err)
		return
	}
	h.sendBotMessage(chatID, telegram.MsgPrefsStopped, name, nil)
}

// botSettings is /settings: the three switches as buttons under one message.
func (h handlers) botSettings(chatID, tgID int64, name string) {
	uid, ok := h.userForTelegram(tgID)
	if !ok {
		h.sendBotMessage(chatID, telegram.MsgPrefsNoAccount, name, nil)
		return
	}
	p, err := h.Store.BotPrefs(uid)
	if err != nil {
		log.Printf("telegram webhook: /settings: %v", err)
		return
	}
	st := prefsState(p)
	h.sendBotMessage(chatID, telegram.MsgPrefsMenu, name, &telegram.InlineButton{Prefs: &st})
}

// prefPressed handles the two opt-out buttons and reports whether data was one
// of them. Unlike messages, the answer and the keyboard redraw go straight to
// the Bot API: Telegram wants the answer within seconds.
func (h handlers) prefPressed(queryID, data string, tgID, chatID, messageID int64) bool {
	token := h.Config.TelegramBotToken
	answer := func(text string) {
		if err := telegram.AnswerCallbackQuery(token, queryID, text); err != nil {
			log.Printf("telegram webhook: answer callback: %v", err)
		}
	}
	off, isOff := telegram.ParsePrefOff(data)
	cat, on, isToggle := telegram.ParsePref(data)
	if !isOff && !isToggle {
		return false
	}
	if isOff {
		cat, on = off, false
	}

	uid, ok := h.userForTelegram(tgID)
	if !ok {
		answer("Сначала привяжи Telegram в приложении")
		return true
	}
	cur, err := h.Store.BotPrefs(uid)
	if err != nil {
		log.Printf("telegram webhook: load prefs: %v", err)
		answer("Не получилось, попробуй ещё раз")
		return true
	}
	next := prefsState(cur).With(cat, on)
	if err := h.Store.SetBotPrefs(uid, storePrefs(next), h.Now()); err != nil {
		log.Printf("telegram webhook: save prefs: %v", err)
		answer("Не получилось, попробуй ещё раз")
		return true
	}

	if on {
		answer("Включено: " + telegram.PrefTitles[cat])
	} else {
		answer("Выключено: " + telegram.PrefTitles[cat])
	}
	if messageID == 0 {
		return true
	}
	var editErr error
	if isOff {
		editErr = telegram.RemoveKeyboard(token, chatID, messageID)
	} else {
		editErr = telegram.EditPrefsKeyboard(token, chatID, messageID, next)
	}
	if editErr != nil {
		log.Printf("telegram webhook: redraw keyboard: %v", editErr)
	}
	return true
}
