// Package config loads process configuration from environment variables. It is
// the single place that reads os.Getenv, so the rest of the server takes a
// Config value and never touches the environment directly.
package config

import (
	"os"
	"strings"

	"github.com/grisha/serbian-app/server/internal/mail"
)

// defaultAppBaseURL is used when APP_BASE_URL is unset or empty.
const defaultAppBaseURL = "http://localhost:8080"

// Config is the resolved process configuration.
type Config struct {
	// AppBaseURL is the public origin the app is served from, without a
	// trailing slash. It drives absolute links in email and the cookie
	// security mode. From APP_BASE_URL, default defaultAppBaseURL.
	AppBaseURL string
	// SMTP holds outbound mail settings. A zero Host means email is disabled
	// and callers fall back to a log-only mailer.
	SMTP mail.SMTPConfig
	// TelegramBotToken enables Telegram sign-in when non-empty. From
	// TELEGRAM_BOT_TOKEN.
	TelegramBotToken string
	// TelegramBotUsername and TelegramWebhookSecret pin the bot's @username
	// and the webhook secret instead of discovering/generating them via a
	// live call to api.telegram.org at startup (see main.go). Both are
	// static once chosen — the username doesn't change, and the secret is
	// only ever compared against what was registered with Telegram's
	// setWebhook — so when a host's outbound access to api.telegram.org is
	// unreliable (e.g. blocked, as happens for some Russian hosts), set
	// these from TELEGRAM_BOT_USERNAME / TELEGRAM_WEBHOOK_SECRET and
	// register the webhook once from a network that can reach Telegram;
	// the app no longer needs its own boot-time call to succeed.
	TelegramBotUsername   string
	TelegramWebhookSecret string
	// TelegramAPIBase overrides the Bot API origin (default
	// https://api.telegram.org) for every call the app makes, including
	// the outbox worker's sendMessage — not just the boot-time calls
	// TelegramBotUsername/TelegramWebhookSecret bypass. Needed on top of
	// those two: pinning them only skips the boot-time GetMe/SetWebhook
	// calls, it does nothing for the ongoing sendMessage calls the outbox
	// worker makes for as long as the process runs, which hit the exact
	// same outbound block. Point this at a small reverse-proxy (e.g. a
	// Cloudflare Worker forwarding to https://api.telegram.org) reachable
	// from a blocked host. From TELEGRAM_API_BASE; empty keeps the real
	// Telegram origin.
	TelegramAPIBase string
}

// Load reads the configuration from the environment. Missing variables take
// their documented defaults; it never fails.
func Load() Config {
	base := os.Getenv("APP_BASE_URL")
	if base == "" {
		base = defaultAppBaseURL
	}
	// Trim trailing slashes so path joins never double up.
	base = strings.TrimRight(base, "/")

	return Config{
		AppBaseURL: base,
		SMTP: mail.SMTPConfig{
			Host:     os.Getenv("SMTP_HOST"),
			Port:     os.Getenv("SMTP_PORT"),
			User:     os.Getenv("SMTP_USER"),
			Pass:     os.Getenv("SMTP_PASS"),
			From:     os.Getenv("SMTP_FROM"),
			FromName: os.Getenv("SMTP_FROM_NAME"),
		},
		TelegramBotToken:      os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramBotUsername:   os.Getenv("TELEGRAM_BOT_USERNAME"),
		TelegramWebhookSecret: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
		TelegramAPIBase:       os.Getenv("TELEGRAM_API_BASE"),
	}
}

// Secure reports whether the app is served over HTTPS, which gates
// Secure-flagged and __Host- cookies.
func (c Config) Secure() bool {
	return strings.HasPrefix(c.AppBaseURL, "https://")
}

// CookieName is the session cookie name: the locked-down "__Host-session" when
// served over HTTPS, plain "session" otherwise (the __Host- prefix requires a
// secure origin).
func (c Config) CookieName() string {
	if c.Secure() {
		return "__Host-session"
	}
	return "session"
}

// TelegramEnabled reports whether Telegram sign-in is configured.
func (c Config) TelegramEnabled() bool {
	return c.TelegramBotToken != ""
}

// TelegramBotID returns the bot's numeric id — the segment before ":" in
// TELEGRAM_BOT_TOKEN — for the client-side Telegram Login Widget, which opens
// its auth popup with this id (not the token). Not a secret: once a bot
// exists, its id is public (part of the bot's own API surface, same as its
// t.me link). Empty when Telegram sign-in is not configured or the token is
// malformed.
func (c Config) TelegramBotID() string {
	id, _, ok := strings.Cut(c.TelegramBotToken, ":")
	if !ok || id == "" {
		return ""
	}
	return id
}

// SMTPEnabled reports whether outbound SMTP mail is configured.
func (c Config) SMTPEnabled() bool {
	return c.SMTP.Host != ""
}
