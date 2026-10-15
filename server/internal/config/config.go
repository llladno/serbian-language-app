// Package config loads process configuration from environment variables. It is
// the single place that reads os.Getenv, so the rest of the server takes a
// Config value and never touches the environment directly.
package config

import (
	"os"
	"strconv"
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
	// TelegramWebhookURL overrides the URL registered with Telegram's
	// setWebhook (default AppBaseURL + "/api/telegram/webhook", i.e.
	// Telegram delivers straight to this app). Set this when that direct
	// route is itself unreliable (inbound, not outbound — a separate
	// problem from TelegramAPIBase) — point it at a reverse-proxy relay
	// (e.g. a Cloudflare Worker forwarding to AppBaseURL +
	// "/api/telegram/webhook") that Telegram can reach even when this host
	// can't be reached directly. The app's own handler at
	// /api/telegram/webhook is unaffected either way — only where Telegram
	// is told to deliver updates changes. From TELEGRAM_WEBHOOK_URL.
	TelegramWebhookURL string
	// TelegramAdminIDs are the Telegram user ids allowed to run the bot's /stats command
	// (the lead-magnet funnel). Anyone can run /id to learn theirs. From
	// TELEGRAM_ADMIN_IDS, a comma-separated list; empty disables /stats.
	TelegramAdminIDs []int64
	// TributeAPIKey authenticates the Tribute donation webhook
	// (POST /api/tribute/webhook): Tribute signs every request with
	// HMAC-SHA256(key=TributeAPIKey, msg=body) in the "trbt-signature"
	// header. Empty disables the webhook (falls through as a no-op 200).
	// From TRIBUTE_API_KEY.
	TributeAPIKey string

	// TariffsEmails are the accounts, by VERIFIED email address, that see and can
	// use the paid tariffs (the level and year pages, offers, orders), paying
	// through ЮKassa. Everyone else gets neither the screens nor the endpoints.
	// Empty means nobody. Matched case-insensitively. From TARIFFS_EMAILS, a
	// comma-separated list.
	TariffsEmails []string
	// TBankEmails are accounts that see the same tariffs but pay through Т-Касса
	// (Т-Банк) instead. An address in both lists pays through Т-Касса. From
	// TBANK_EMAILS.
	TBankEmails []string
	// YooKassaShopID and YooKassaSecretKey are the credentials of the ЮKassa shop
	// that takes the payments: the test shop's while testing, the real one's at
	// launch. Both empty means payments are not connected and orders are refused.
	// From YOOKASSA_SHOP_ID and YOOKASSA_SECRET_KEY.
	YooKassaShopID    string
	YooKassaSecretKey string
	// YooKassaTaxSystem is the receipt's tax_system_code (1..6); 0 leaves it out,
	// which ЮKassa accepts for a shop with a single tax system. YOOKASSA_TAX_SYSTEM.
	YooKassaTaxSystem int
	// YooKassaAPIBase overrides the ЮKassa API address (tests). YOOKASSA_API_BASE.
	YooKassaAPIBase string
	// TBankTerminalKey and TBankPassword are the Т-Касса terminal's credentials:
	// the test terminal's (its key ends in DEMO) while testing, the real one's at
	// launch. Both empty means Т-Касса is not connected. From TBANK_TERMINAL_KEY
	// and TBANK_PASSWORD.
	TBankTerminalKey string
	TBankPassword    string
	// TBankTaxation is the tax system printed on the receipt: osn, usn_income,
	// usn_income_outcome, esn or patent. From TBANK_TAXATION, default usn_income.
	TBankTaxation string
	// TBankAPIBase overrides the Т-Касса API address (tests). TBANK_API_BASE.
	TBankAPIBase string
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
		TelegramWebhookURL:    os.Getenv("TELEGRAM_WEBHOOK_URL"),
		TelegramAdminIDs:      parseIDList(os.Getenv("TELEGRAM_ADMIN_IDS")),
		TributeAPIKey:         os.Getenv("TRIBUTE_API_KEY"),
		TariffsEmails:         parseEmailList(os.Getenv("TARIFFS_EMAILS")),
		TBankEmails:           parseEmailList(os.Getenv("TBANK_EMAILS")),
		YooKassaShopID:        strings.TrimSpace(os.Getenv("YOOKASSA_SHOP_ID")),
		YooKassaSecretKey:     strings.TrimSpace(os.Getenv("YOOKASSA_SECRET_KEY")),
		YooKassaTaxSystem:     parseSmallInt(os.Getenv("YOOKASSA_TAX_SYSTEM")),
		YooKassaAPIBase:       strings.TrimSpace(os.Getenv("YOOKASSA_API_BASE")),
		TBankTerminalKey:      strings.TrimSpace(os.Getenv("TBANK_TERMINAL_KEY")),
		TBankPassword:         strings.TrimSpace(os.Getenv("TBANK_PASSWORD")),
		TBankTaxation:         taxationOr(os.Getenv("TBANK_TAXATION"), "usn_income"),
		TBankAPIBase:          strings.TrimSpace(os.Getenv("TBANK_API_BASE")),
	}
}

// taxationOr reads a receipt tax system, falling back to def for anything the
// bank would not accept.
func taxationOr(raw, def string) string {
	switch v := strings.ToLower(strings.TrimSpace(raw)); v {
	case "osn", "usn_income", "usn_income_outcome", "esn", "patent":
		return v
	}
	return def
}

// parseIDList reads "123, 456" into ids, skipping anything that is not a number.
func parseIDList(raw string) []int64 {
	var ids []int64
	for _, part := range strings.Split(raw, ",") {
		if id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// parseEmailList reads "a@x.com, B@y.org" into lower-cased addresses.
func parseEmailList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if e := strings.ToLower(strings.TrimSpace(part)); e != "" {
			out = append(out, e)
		}
	}
	return out
}

func parseSmallInt(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 0 || n > 6 {
		return 0
	}
	return n
}

// Payment services an account can be sent to.
const (
	ProviderYooKassa = "yookassa"
	ProviderTBank    = "tbank"
)

// PaymentProviderFor says which payment service the account with this email
// pays through: ProviderTBank for an address in TBANK_EMAILS, ProviderYooKassa
// for one in TARIFFS_EMAILS, "" for everyone else (who see no payments at all).
// The caller must already have established that the email is verified: an
// address someone merely typed at registration proves nothing.
func (c Config) PaymentProviderFor(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return ""
	}
	for _, e := range c.TBankEmails {
		if e == email {
			return ProviderTBank
		}
	}
	for _, e := range c.TariffsEmails {
		if e == email {
			return ProviderYooKassa
		}
	}
	return ""
}

// TariffsAllowed reports whether the account with this email may use the paid
// tariffs at all (through either payment service).
func (c Config) TariffsAllowed(email string) bool {
	return c.PaymentProviderFor(email) != ""
}

// YooKassaConfigured reports whether ЮKassa payments are connected.
func (c Config) YooKassaConfigured() bool {
	return c.YooKassaShopID != "" && c.YooKassaSecretKey != ""
}

// TBankConfigured reports whether Т-Касса payments are connected.
func (c Config) TBankConfigured() bool {
	return c.TBankTerminalKey != "" && c.TBankPassword != ""
}

// IsTelegramAdmin reports whether tgID may run the bot's admin commands.
func (c Config) IsTelegramAdmin(tgID int64) bool {
	for _, id := range c.TelegramAdminIDs {
		if id == tgID {
			return true
		}
	}
	return false
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
