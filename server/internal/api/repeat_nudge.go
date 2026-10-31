package api

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
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

// repeatIntroText is the one-off message to a learner who studies but never opened «Повторить».
// It has no opt-out button: it is sent once and the learner's study switch already covers it.
const repeatIntroText = "Пройденные уроки можно повторять 🔁\n\n" +
	"В приложении есть раздел «Повторить»: мы собираем пятнадцать заданий из пройденных уроков и сначала ставим те, где были ошибки. " +
	"Минут пять, и слова держатся в голове заметно дольше. Hajde (давай), загляни!"

// repeatIntroPerSweep caps how many introductions one reminder sweep queues. A sweep runs every 20
// minutes, the outbox sends 25 messages a second against Telegram's 30, so a batch this size drains
// in a few seconds and the whole audience takes a couple of hours, inside each learner's daytime.
var repeatIntroPerSweep = 150

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

// maybeNudgeRepeat writes the nudge to one learner if it is due. info is their nudge state as the
// sweep read it; otherToday: another study message already went out to them today; lastActivity is
// their latest review or attempt.
func (h handlers) maybeNudgeRepeat(tu store.TelegramUser, info store.RepeatNudgeInfo, now, lastActivity time.Time, otherToday bool) {
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
	token, err := h.Store.RecordRepeatNudge(tu.UserID, variant, kind, now, info.LastRepeat)
	if err != nil {
		log.Printf("reminder sweep: record repeat nudge for %s: %v", tu.UserID, err)
		return
	}
	if token == "" {
		return // another sweep got there first
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
		&telegram.InlineButton{Label: repeatNudgeLabel, RepeatNudgeURL: repeatURL(h.Config.AppBaseURL, token)},
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

// repeatURL is the address the button under a «Повторить» message opens. The token says which message
// it came from, so RepeatClickTracker can count the press.
func repeatURL(base, token string) string { return base + "/repeat?n=" + token }

// maybeIntroRepeat queues the one-off introduction to one learner if it is due and the sweep still has
// room, and reports whether it did. info is their nudge state as the sweep read it.
func (h handlers) maybeIntroRepeat(tu store.TelegramUser, info store.RepeatNudgeInfo, now, lastActivity time.Time, otherToday bool) bool {
	loc := h.Store.User(tu.UserID).Location()
	if !repeat.IntroDue(repeat.IntroInput{
		Now:                now.In(loc),
		LastRepeat:         info.LastRepeat,
		LastActivity:       lastActivity,
		AlreadySent:        !info.LastNudge.IsZero(), // for someone who never repeated, any row is the intro
		OtherLearningToday: otherToday,
	}) {
		return false
	}
	// Recorded before it is queued, like the nudge: a failure must not turn into a message every sweep.
	token, err := h.Store.RecordRepeatNudge(tu.UserID, 0, store.NudgeIntro, now, info.LastRepeat)
	if err != nil {
		log.Printf("reminder sweep: record repeat intro for %s: %v", tu.UserID, err)
		return false
	}
	if token == "" {
		return false
	}
	h.EnqueueTelegramMessage(tu.ChatID, repeatIntroText,
		&telegram.InlineButton{Label: repeatNudgeLabel, WebAppURL: repeatURL(h.Config.AppBaseURL, token)},
		telegram.PriorityNormal)
	return true
}

var clickToken = regexp.MustCompile(`^[0-9a-f]{16}$`)

// RepeatClickTracker counts a press on the button under a «Повторить» message. The button opens
// /repeat?n=<token>; this wrapper notes the token when the page is requested and then serves the
// page whatever happens: counting must never get in the way of opening the screen.
func RepeatClickTracker(deps Deps, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/repeat" {
			if n := r.URL.Query().Get("n"); clickToken.MatchString(n) {
				if _, err := deps.Store.NoteRepeatClick(n, deps.Now()); err != nil {
					log.Printf("repeat click: %v", err)
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
