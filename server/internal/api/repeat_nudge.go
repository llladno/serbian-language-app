package api

import (
	"fmt"
	"log"
	"time"

	"github.com/grisha/serbian-app/server/internal/repeat"
	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
)

// The Telegram nudge for a learner who still studies but has stopped using «Повторить». The rules
// are in repeat.NudgeDue: once per quiet spell, only after the learner has used the button before,
// only in their daytime, and never to the control half. Texts speak as «мы», open with what
// catches the eye in a notification preview, carry at most one Serbian word and two emoji.

const repeatNudgeVariants = 3

// repeatNudgeLabel is the text of the button that opens «Повторить».
const repeatNudgeLabel = "Повторить"

// daysWord is «день / дня / дней» for n.
func daysWord(n int) string {
	switch {
	case n%100 >= 11 && n%100 <= 14:
		return "дней"
	case n%10 == 1:
		return "день"
	case n%10 >= 2 && n%10 <= 4:
		return "дня"
	}
	return "дней"
}

// repeatNudgeText is text number variant (0-based) for a learner called name who has not repeated
// for days days. An empty name drops the greeting rather than leaving a gap.
func repeatNudgeText(variant int, name string, days int) string {
	span := fmt.Sprintf("%d %s без повторения", days, daysWord(days))
	switch variant {
	case 0:
		lead := span
		if name != "" {
			lead = name + ", " + span
		}
		return lead + " 😬 Мы переживаем: выученные слова уже начинают забываться, а у кассы в Белграде они понадобятся именно тогда, когда не вспомнятся.\n" +
			"Пятнадцать заданий, минут пять, и всё на месте. Мы всё подготовили. Hajde (давай)!"
	case 1:
		return "Давай повторим сербский язык 😏 Спорим, ты всё ещё помнишь выученное? Проверим за пять минут: мы подготовили пятнадцать заданий.\n" +
			"Всё вспомнил — bravo (молодец)! Что-то забылось — вернём вместе, пока не ушло совсем."
	}
	return "Мы скучаем по тебе 🥺 " + span + ", и выученные слова уже начали забываться.\n" +
		"Давай вернём их: пятнадцать заданий, минут пять. Polako (не спеша), мы всё подготовили."
}

// maybeNudgeRepeat writes the nudge to one learner if it is due. otherToday: another study message
// already went out to them today. lastActivity is their latest review or attempt.
func (h handlers) maybeNudgeRepeat(tu store.TelegramUser, now, lastActivity time.Time, otherToday bool) {
	info, err := h.Store.RepeatNudgeInfo(tu.UserID)
	if err != nil {
		log.Printf("reminder sweep: repeat nudge info for %s: %v", tu.UserID, err)
		return
	}
	loc := h.Store.User(tu.UserID).Location()
	if !repeat.NudgeDue(repeat.NudgeInput{
		Now:                now.In(loc),
		LastRepeat:         info.LastRepeat,
		LastActivity:       lastActivity,
		LastNudge:          info.LastNudge,
		OptedOut:           info.OptedOut,
		OtherLearningToday: otherToday,
	}) {
		return
	}
	variant := repeat.NudgeVariant(tu.UserID, info.Sent, repeatNudgeVariants)
	kind := store.NudgeSent
	if repeat.InControl(tu.UserID) {
		kind = store.NudgeControl
	}
	// Recorded before it is queued: a failure here must not turn into a message sent every sweep.
	if err := h.Store.RecordRepeatNudge(tu.UserID, variant, kind, now); err != nil {
		log.Printf("reminder sweep: record repeat nudge for %s: %v", tu.UserID, err)
		return
	}
	if kind == store.NudgeControl {
		return
	}
	var name string
	if u, err := h.Store.UserByID(tu.UserID); err == nil {
		name = u.Name
	}
	days := int(now.Sub(info.LastRepeat) / (24 * time.Hour))
	h.EnqueueTelegramMessage(tu.ChatID, repeatNudgeText(variant, name, days),
		&telegram.InlineButton{Label: repeatNudgeLabel, RepeatNudgeURL: h.Config.AppBaseURL + "/repeat"},
		telegram.PriorityNormal)
}

// repeatNudgeOffPressed handles «Не про повторения» under the nudge: it switches off this nudge
// only, the other study messages stay. Like prefPressed it answers straight from the webhook.
func (h handlers) repeatNudgeOffPressed(queryID string, tgID, chatID, messageID int64) {
	token := h.Config.TelegramBotToken
	answer := func(text string) {
		if err := telegram.AnswerCallbackQuery(token, queryID, text); err != nil {
			log.Printf("telegram webhook: answer callback: %v", err)
		}
	}
	uid, ok := h.userForTelegram(tgID)
	if !ok {
		answer("Сначала привяжи Telegram в приложении")
		return
	}
	if err := h.Store.SetRepeatNudgeOff(uid); err != nil {
		log.Printf("telegram webhook: repeat nudge off: %v", err)
		answer("Не получилось, попробуй ещё раз")
		return
	}
	answer("Больше не напомню про повторения")
	if messageID == 0 {
		return
	}
	if err := telegram.RemoveKeyboard(token, chatID, messageID); err != nil {
		log.Printf("telegram webhook: redraw keyboard: %v", err)
	}
}
