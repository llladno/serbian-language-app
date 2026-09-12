package config

import "testing"

// clearEnv sets every variable Load reads to empty for the duration of the
// test, so a value in the real environment cannot leak in.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"APP_BASE_URL",
		"SMTP_HOST", "SMTP_PORT", "SMTP_USER", "SMTP_PASS", "SMTP_FROM", "SMTP_FROM_NAME",
		"TELEGRAM_BOT_TOKEN",
	} {
		t.Setenv(k, "")
	}
}

func TestLoadDefaultAppBaseURL(t *testing.T) {
	clearEnv(t)
	c := Load()
	if c.AppBaseURL != "http://localhost:8080" {
		t.Fatalf("AppBaseURL default: got %q, want %q", c.AppBaseURL, "http://localhost:8080")
	}
	if c.Secure() {
		t.Error("Secure() on http base: got true, want false")
	}
	if got := c.CookieName(); got != "session" {
		t.Errorf("CookieName() on http base: got %q, want %q", got, "session")
	}
}

func TestLoadTrimsTrailingSlash(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_BASE_URL", "https://srpski.example/")
	c := Load()
	if c.AppBaseURL != "https://srpski.example" {
		t.Fatalf("AppBaseURL: got %q, want %q (trailing slash trimmed)", c.AppBaseURL, "https://srpski.example")
	}
}

func TestSecureAndCookieName(t *testing.T) {
	tests := []struct {
		base       string
		wantSecure bool
		wantCookie string
	}{
		{"http://localhost:8080", false, "session"},
		{"https://srpski.example", true, "__Host-session"},
	}
	for _, tt := range tests {
		clearEnv(t)
		t.Setenv("APP_BASE_URL", tt.base)
		c := Load()
		if c.Secure() != tt.wantSecure {
			t.Errorf("Secure() for %q: got %v, want %v", tt.base, c.Secure(), tt.wantSecure)
		}
		if got := c.CookieName(); got != tt.wantCookie {
			t.Errorf("CookieName() for %q: got %q, want %q", tt.base, got, tt.wantCookie)
		}
	}
}

func TestSMTPEnabledAndFields(t *testing.T) {
	clearEnv(t)
	c := Load()
	if c.SMTPEnabled() {
		t.Fatal("SMTPEnabled() with no SMTP_HOST: got true, want false")
	}

	clearEnv(t)
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "465")
	t.Setenv("SMTP_USER", "postmaster@srpski.example")
	t.Setenv("SMTP_PASS", "hunter2")
	t.Setenv("SMTP_FROM", "no-reply@srpski.example")
	t.Setenv("SMTP_FROM_NAME", "Српски")
	c = Load()
	if !c.SMTPEnabled() {
		t.Fatal("SMTPEnabled() with SMTP_HOST set: got false, want true")
	}
	if c.SMTP.Host != "smtp.example.com" ||
		c.SMTP.Port != "465" ||
		c.SMTP.User != "postmaster@srpski.example" ||
		c.SMTP.Pass != "hunter2" ||
		c.SMTP.From != "no-reply@srpski.example" ||
		c.SMTP.FromName != "Српски" {
		t.Errorf("SMTP fields did not propagate: %+v", c.SMTP)
	}
}

func TestTelegramEnabled(t *testing.T) {
	clearEnv(t)
	if Load().TelegramEnabled() {
		t.Fatal("TelegramEnabled() with no token: got true, want false")
	}

	clearEnv(t)
	t.Setenv("TELEGRAM_BOT_TOKEN", "123456:abcdef")
	c := Load()
	if !c.TelegramEnabled() {
		t.Fatal("TelegramEnabled() with token set: got false, want true")
	}
	if c.TelegramBotToken != "123456:abcdef" {
		t.Errorf("TelegramBotToken: got %q, want %q", c.TelegramBotToken, "123456:abcdef")
	}
}

func TestTelegramBotID(t *testing.T) {
	clearEnv(t)
	if got := Load().TelegramBotID(); got != "" {
		t.Fatalf("TelegramBotID() with no token: got %q, want empty", got)
	}

	clearEnv(t)
	t.Setenv("TELEGRAM_BOT_TOKEN", "123456:abcdef-secret")
	if got := Load().TelegramBotID(); got != "123456" {
		t.Fatalf("TelegramBotID(): got %q, want %q", got, "123456")
	}

	clearEnv(t)
	t.Setenv("TELEGRAM_BOT_TOKEN", "malformed-no-colon")
	if got := Load().TelegramBotID(); got != "" {
		t.Fatalf("TelegramBotID() on malformed token: got %q, want empty", got)
	}
}
