package telegram

import (
	"strings"
	"unicode"
)

// MessageKey identifies one editable bot reply template, stored in
// serbian-app's bot_messages table and edited from ucimo-content-admin's
// "Бот" → "Сообщения" page.
type MessageKey string

const (
	MsgStartGreeting        MessageKey = "start_greeting"
	MsgLoginSuccessNew      MessageKey = "login_success_new"
	MsgLoginSuccessExisting MessageKey = "login_success_existing"
	MsgLoginTokenExpired    MessageKey = "login_token_expired"
	MsgLoginError           MessageKey = "login_error"
	MsgLinkSuccess          MessageKey = "link_success"
	MsgLinkTaken            MessageKey = "link_taken"
	MsgLinkError            MessageKey = "link_error"
	// The channel-subscription quest, when the learner was sent to the bot to
	// link Telegram for it: the prompt that follows the link, and the two
	// answers to "check".
	MsgChannelPrompt MessageKey = "channel_prompt"
	MsgChannelOK     MessageKey = "channel_ok"
	MsgChannelNotYet MessageKey = "channel_not_yet"
	// Opting out: the answer to /stop, the settings screen behind /settings,
	// and the answer from someone whose Telegram is not linked to an account.
	MsgPrefsStopped   MessageKey = "prefs_stopped"
	MsgPrefsMenu      MessageKey = "prefs_menu"
	MsgPrefsNoAccount MessageKey = "prefs_no_account"
)

// SupportURL is where every "написать в поддержку" button and @-mention
// points — the same account the website's support card links to
// (web/src/lib/supportModal.ts's SUPPORT_TELEGRAM_URL).
const SupportURL = "https://t.me/ucimosupport"

// Outbox priorities — lower runs first in internal/outbox.ProcessNext.
// High is every direct reaction to something a user just did (the /start
// webhook flow); Normal is everything else (periodic reminders, admin
// broadcasts).
const (
	PriorityHigh   = 0
	PriorityNormal = 1
)

// DefaultMessages holds the fallback text for every MessageKey. Used by
// server/internal/api's sendBotMessage only when bot_messages has no row
// for a key (in normal operation migration 007 has already seeded all of
// them — this only matters for a key added to this map after that
// migration last ran). Also exactly what migrate007
// (server/internal/store/migration_hooks.go) seeds the table with.
var DefaultMessages = map[MessageKey]string{
	MsgStartGreeting: "Zdravo (привет), {name}! 👋 Это бот Учимо — сервиса для изучения сербского с нуля.\n\n" +
		"Жми на кнопку ниже, чтобы открыть приложение и начать учиться.",
	MsgLoginSuccessNew: "Готово, {name}! 🎉 Регистрация прошла успешно — добро пожаловать в Учимо.\n\n" +
		"Заходи на сайт ucimo.ru или сразу открывай приложение здесь, в Telegram, — и начинай первый урок!",
	MsgLoginSuccessExisting: "С возвращением, {name}! 👋 Ты уже с нами — продолжай изучать сербский.\n\n" +
		"Открывай приложение и вперёд: тебя ждут уроки и повторение слов.",
	MsgLoginTokenExpired: "Кажется, эта ссылка уже устарела 🙈\n\n" +
		"Зайди на ucimo.ru и попробуй войти через Telegram ещё раз.",
	MsgLoginError: "Упс, что-то пошло не так 😔\n\n" +
		"Попробуй войти ещё раз с сайта ucimo.ru. Если не получится — напиши нам: @ucimosupport",
	MsgLinkSuccess: "Готово! 🎉 Telegram привязан к твоему аккаунту, {name}.\n\n" +
		"Теперь можно входить в Учимо и через Telegram — возвращайся на сайт.",
	MsgLinkTaken: "Этот Telegram уже привязан к другому аккаунту Учимо.\n\n" +
		"Если это ошибка — напиши нам: @ucimosupport",
	MsgLinkError: "Упс, не получилось привязать Telegram 😔\n\n" +
		"Попробуй ещё раз с сайта, а если не поможет — напиши нам: @ucimosupport",
	MsgChannelPrompt: "Готово, {name}! Telegram привязан 🎉\n\n" +
		"Осталось подписаться на канал: нажми «Подписаться на канал», а потом «Проверить подписку». " +
		"Награду заберёшь в приложении.",
	MsgChannelOK: "Подписку вижу, {name} ✅\n\n" +
		"Возвращайся в приложение и забирай награду.",
	MsgChannelNotYet: "Пока не вижу подписки, {name} 🤔\n\n" +
		"Подпишись на канал и нажми «Проверить подписку» ещё раз.",
	MsgPrefsStopped: "Хорошо, {name}, больше не пишу про учёбу, тариф и новости.\n\n" +
		"Служебные сообщения всё равно придут: вход, оплата, открытый доступ. Вернуть всё обратно — /settings.",
	MsgPrefsMenu: "Что мне можно тебе писать, {name}? Нажми, чтобы включить или выключить.\n\n" +
		"Вход и оплату я пишу всегда. Остановить всё сразу — /stop.",
	MsgPrefsNoAccount: "Пока ничего не пишу: этот Telegram не привязан ни к одному аккаунту Учимо. " +
		"Привязать можно в приложении, в профиле.",
}

// Substitute replaces every "{name}" placeholder in text with name.
func Substitute(text, name string) string {
	return strings.ReplaceAll(text, "{name}", name)
}

// GreetingName turns a first name as people type it (Telegram profile, registration form) into one a message can
// open with («Гриша, ...»), or "" when it can't be trusted to read like a name. Only the first word counts, it must be
// 2 to 20 letters (a hyphen or apostrophe between letters is fine), so handles (`gri_92`), emoji, digits and
// «Иван | дизайн» come out empty, and the caller drops the name instead of printing junk. A name typed all lower case
// or all upper case gets a capital first letter; mixed case (McDonald) is left alone.
func GreetingName(raw string) string {
	words := strings.Fields(raw)
	if len(words) == 0 {
		return ""
	}
	runes := []rune(words[0])
	if len(runes) < 2 || len(runes) > 20 {
		return ""
	}
	lower, upper := 0, 0
	for i, r := range runes {
		switch {
		case unicode.IsLetter(r):
			if unicode.IsLower(r) {
				lower++
			} else if unicode.IsUpper(r) {
				upper++
			}
		case (r == '-' || r == '\'' || r == '’') && i > 0 && i < len(runes)-1:
		default:
			return ""
		}
	}
	if lower == 0 || upper == 0 || (upper == 1 && unicode.IsUpper(runes[0])) {
		// all lower, all upper, or a plain «Иван»: normalise to a capital and lower case.
		for i, r := range runes {
			if i == 0 {
				runes[i] = unicode.ToUpper(r)
			} else {
				runes[i] = unicode.ToLower(r)
			}
		}
		return string(runes)
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// DisplayName picks what to greet a Telegram user by: their first name,
// else "@" + their username, else the generic fallback "друг" (Telegram
// guarantees at least one of first_name/username is non-empty for a real
// user, but never both are absent and no name at all is defensively
// handled anyway).
func DisplayName(firstName, username string) string {
	if firstName != "" {
		return firstName
	}
	if username != "" {
		return "@" + username
	}
	return "друг"
}
