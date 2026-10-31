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
	// through Т-Касса (the only payment service). Everyone else gets neither the
	// screens nor the endpoints. Empty means nobody. Matched case-insensitively.
	// From TARIFFS_EMAILS, a comma-separated list.
	TariffsEmails []string
	// TBankEmails is a second list with the same effect as TariffsEmails, kept
	// because it was set before ЮKassa was dropped and there is one service now.
	// From TBANK_EMAILS.
	TBankEmails []string
	// TariffsForAll opens the tariffs to every account with a verified email
	// address. An account with no email (Telegram only) still sees nothing unless
	// this is on: the receipt has to go somewhere (it is asked for when paying).
	// From TARIFFS_FOR_ALL (1, true or yes).
	TariffsForAll bool
	// RepeatNudge switches on the Telegram nudge for learners who study but stopped using
	// «Повторить». Off unless set, so a deploy does not start it: it starts when this is
	// turned on, and stops when it is turned off. From REPEAT_NUDGE (1, true or yes).
	RepeatNudge bool
	// RepeatIntro switches on the one-off introduction of «Повторить» to every learner who has
	// studied but never opened it: a capped batch per reminder sweep until all have had it. Off
	// unless set. From REPEAT_INTRO (1, true or yes).
	RepeatIntro bool
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
	// PaymentsDev swaps every payment service for a stand-in cash desk that the developer
	// operates with two buttons (the page /dev/pay), so the whole flow from the order to
	// the choice of a level can be tried on a local machine without a bank. It changes
	// only who takes the money: who may pay and what happens after are the real code. From
	// PAYMENTS_DEV (1, true or yes), and never on an https base: the production site is
	// served over https, so a .env copied there cannot turn it on.
	PaymentsDev bool
	// TariffLevelCooldownDays is how long after paying for a level the next one
	// cannot be bought for rubles: one level a month. Levels bought for зёрнышки
	// and the year are not held back. 0 turns the limit off. From
	// TARIFF_LEVEL_COOLDOWN_DAYS, default 30.
	TariffLevelCooldownDays int
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
		TariffsForAll:         parseFlag(os.Getenv("TARIFFS_FOR_ALL")),
		RepeatNudge:           parseFlag(os.Getenv("REPEAT_NUDGE")),
		RepeatIntro:           parseFlag(os.Getenv("REPEAT_INTRO")),
		TBankTerminalKey:      strings.TrimSpace(os.Getenv("TBANK_TERMINAL_KEY")),
		TBankPassword:         strings.TrimSpace(os.Getenv("TBANK_PASSWORD")),
		TBankTaxation:         taxationOr(os.Getenv("TBANK_TAXATION"), "usn_income"),
		TBankAPIBase:          strings.TrimSpace(os.Getenv("TBANK_API_BASE")),
		PaymentsDev:           parseFlag(os.Getenv("PAYMENTS_DEV")) && !strings.HasPrefix(base, "https://"),

		TariffLevelCooldownDays: daysOr(os.Getenv("TARIFF_LEVEL_COOLDOWN_DAYS"), 30),
	}
}

// daysOr reads a number of days from 0 to 365, falling back to def for anything
// else (empty, not a number, negative, absurdly long).
func daysOr(raw string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 0 || n > 365 {
		return def
	}
	return n
}

// parseFlag reads an on/off setting: 1, true or yes (any case) is on, anything
// else, including empty, is off.
func parseFlag(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes":
		return true
	}
	return false
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

// Payment services an account can be sent to.
const (
	ProviderTBank = "tbank"
	// ProviderDev is the stand-in cash desk of PAYMENTS_DEV: no bank, no money.
	ProviderDev = "dev"
)

// PaymentProviderFor says which payment service the account with this email
// pays through: ProviderTBank for an address in TARIFFS_EMAILS or TBANK_EMAILS
// and for everyone else when the tariffs are open to all (TARIFFS_FOR_ALL), and
// "" otherwise (those accounts see no payments at all). An account with no email
// (Telegram only) has no list entry to match, so it pays through Т-Касса when the
// tariffs are open to all and sees nothing otherwise; the address for its receipt
// is asked for when it pays.
// For an account that has an email the caller must already have established that
// it is verified: an address someone merely typed at registration proves nothing.
func (c Config) PaymentProviderFor(email string) string {
	p := c.realProviderFor(email)
	if p != "" && c.PaymentsDev {
		// Who may pay stays as configured; only the cash desk is replaced.
		return ProviderDev
	}
	return p
}

func (c Config) realProviderFor(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		if c.TariffsForAll {
			return ProviderTBank
		}
		return ""
	}
	for _, e := range c.TBankEmails {
		if e == email {
			return ProviderTBank
		}
	}
	for _, e := range c.TariffsEmails {
		if e == email {
			return ProviderTBank
		}
	}
	if c.TariffsForAll {
		return ProviderTBank
	}
	return ""
}

// TariffsAllowed reports whether the account with this email may use the paid
// tariffs at all.
func (c Config) TariffsAllowed(email string) bool {
	return c.PaymentProviderFor(email) != ""
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
