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

func TestPaymentProviderIsChosenByTheAccountsEmail(t *testing.T) {
	clearEnv(t)
	t.Setenv("TARIFFS_EMAILS", "Greg@Example.com, both@example.com")
	t.Setenv("TBANK_EMAILS", " man@example.com ,BOTH@example.com")
	c := Load()
	for email, want := range map[string]string{
		// either list opens the tariffs, and the only service is Т-Касса
		"greg@example.com":   ProviderTBank,
		"GREG@EXAMPLE.COM":   ProviderTBank,
		"man@example.com":    ProviderTBank,
		"both@example.com":   ProviderTBank,
		"other@example.com":  "",
		"":                   "",
		"  ":                 "",
		"greg@example.com.x": "",
	} {
		if got := c.PaymentProviderFor(email); got != want {
			t.Errorf("PaymentProviderFor(%q) = %q, want %q", email, got, want)
		}
		if got := c.TariffsAllowed(email); got != (want != "") {
			t.Errorf("TariffsAllowed(%q) = %t", email, got)
		}
	}
}

func TestNobodyIsListedByDefault(t *testing.T) {
	clearEnv(t)
	t.Setenv("TARIFFS_EMAILS", "")
	t.Setenv("TBANK_EMAILS", "")
	if c := Load(); c.TariffsAllowed("anyone@example.com") || c.TBankConfigured() {
		t.Errorf("a bare environment lists someone or connects a service: %+v", c)
	}
}

func TestTBankSettings(t *testing.T) {
	clearEnv(t)
	t.Setenv("TBANK_TERMINAL_KEY", " 123DEMO ")
	t.Setenv("TBANK_PASSWORD", "pw")
	t.Setenv("TBANK_TAXATION", "")
	c := Load()
	if !c.TBankConfigured() || c.TBankTerminalKey != "123DEMO" || c.TBankPassword != "pw" {
		t.Errorf("credentials: %+v", c)
	}
	if c.TBankTaxation != "usn_income" {
		t.Errorf("TBankTaxation default = %q, want usn_income", c.TBankTaxation)
	}
	t.Setenv("TBANK_TAXATION", " Patent ")
	if got := Load().TBankTaxation; got != "patent" {
		t.Errorf("TBANK_TAXATION=Patent -> %q", got)
	}
	t.Setenv("TBANK_TAXATION", "whatever")
	if got := Load().TBankTaxation; got != "usn_income" {
		t.Errorf("a tax system the bank would refuse -> %q, want the default", got)
	}
	t.Setenv("TBANK_PASSWORD", "")
	if Load().TBankConfigured() {
		t.Error("configured without a password")
	}
}

func TestLevelCooldownDefaultsToAMonthAndZeroTurnsItOff(t *testing.T) {
	clearEnv(t)
	for raw, want := range map[string]int{
		"":     30, // unset: one level a month
		"0":    0,  // off
		"7":    7,
		" 14 ": 14,
		"365":  365,
		"-1":   30, // anything not a sensible number of days falls back
		"abc":  30,
		"400":  30,
	} {
		t.Setenv("TARIFF_LEVEL_COOLDOWN_DAYS", raw)
		if got := Load().TariffLevelCooldownDays; got != want {
			t.Errorf("TARIFF_LEVEL_COOLDOWN_DAYS=%q -> %d, want %d", raw, got, want)
		}
	}
}

func TestTariffsForAllOpensThemThroughTBankForListedAndUnlistedAlike(t *testing.T) {
	clearEnv(t)
	t.Setenv("TARIFFS_EMAILS", "greg@example.com")
	t.Setenv("TBANK_EMAILS", "man@example.com")
	t.Setenv("TARIFFS_FOR_ALL", "1")
	c := Load()
	for email, want := range map[string]string{
		"someone@example.com": ProviderTBank, // everyone else pays through Т-Касса
		"SOMEONE@Example.com": ProviderTBank,
		"greg@example.com":    ProviderTBank,
		"man@example.com":     ProviderTBank,
		"":                    ProviderTBank, // no email (Telegram only): pays through Т-Касса, naming a receipt address then
		"   ":                 ProviderTBank,
	} {
		if got := c.PaymentProviderFor(email); got != want {
			t.Errorf("PaymentProviderFor(%q) = %q, want %q", email, got, want)
		}
	}
	if !c.TariffsAllowed("someone@example.com") || !c.TariffsAllowed("") {
		t.Error("TariffsAllowed does not follow the provider")
	}
}

func TestAnAccountWithoutAnEmailGetsNothingWhileTheTariffsAreClosed(t *testing.T) {
	clearEnv(t)
	t.Setenv("TARIFFS_EMAILS", "greg@example.com")
	c := Load()
	if got := c.PaymentProviderFor(""); got != "" || c.TariffsAllowed("") {
		t.Errorf("PaymentProviderFor(\"\") = %q with the tariffs closed, want none", got)
	}
}

func TestTariffsStayClosedUnlessTheSwitchIsOn(t *testing.T) {
	clearEnv(t)
	for raw, want := range map[string]bool{
		"": false, "0": false, "no": false, "off": false, "maybe": false,
		"1": true, "true": true, " TRUE ": true, "Yes": true,
	} {
		t.Setenv("TARIFFS_FOR_ALL", raw)
		if got := Load().TariffsForAll; got != want {
			t.Errorf("TARIFFS_FOR_ALL=%q -> %t, want %t", raw, got, want)
		}
	}
	t.Setenv("TARIFFS_FOR_ALL", "")
	if Load().TariffsAllowed("someone@example.com") {
		t.Error("an unlisted address is allowed with the switch off")
	}
}
