package telegram

import "strings"

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
}

// Substitute replaces every "{name}" placeholder in text with name.
func Substitute(text, name string) string {
	return strings.ReplaceAll(text, "{name}", name)
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
