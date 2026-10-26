package config

import "testing"

// PAYMENTS_DEV swaps the real payment services for a stand-in cash desk on the developer's machine.
// It must never be possible to switch it on where real money moves.
func TestPaymentsDevReplacesTheServicesOnAHTTPBase(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_BASE_URL", "http://localhost:5173")
	t.Setenv("PAYMENTS_DEV", "1")
	t.Setenv("TARIFFS_EMAILS", "admin@ucimo.local")
	t.Setenv("TBANK_EMAILS", "bank@example.com")
	c := Load()
	if !c.PaymentsDev {
		t.Fatal("PaymentsDev = false, want true on an http base with the flag set")
	}
	for email, want := range map[string]string{
		"admin@ucimo.local": ProviderDev, // would have paid through Т-Касса
		"bank@example.com":  ProviderDev, // the same, listed in the other list
		"stranger@x.com":    "",          // an account the tariffs are closed to stays closed
	} {
		if got := c.PaymentProviderFor(email); got != want {
			t.Errorf("PaymentProviderFor(%q) = %q, want %q", email, got, want)
		}
	}
	if !c.TariffsAllowed("admin@ucimo.local") || c.TariffsAllowed("stranger@x.com") {
		t.Error("the stand-in must not open the tariffs to anybody new")
	}
}

func TestPaymentsDevServesEveryoneWhenTheTariffsAreOpenToAll(t *testing.T) {
	clearEnv(t)
	t.Setenv("PAYMENTS_DEV", "yes")
	t.Setenv("TARIFFS_FOR_ALL", "1")
	c := Load()
	if got := c.PaymentProviderFor("anyone@example.com"); got != ProviderDev {
		t.Errorf("PaymentProviderFor = %q, want %q", got, ProviderDev)
	}
}

func TestPaymentsDevIsOffUnlessAsked(t *testing.T) {
	clearEnv(t)
	t.Setenv("TARIFFS_EMAILS", "admin@ucimo.local")
	for _, raw := range []string{"", "0", "no", "false", "dev"} {
		t.Setenv("PAYMENTS_DEV", raw)
		c := Load()
		if c.PaymentsDev {
			t.Errorf("PAYMENTS_DEV=%q switched the stand-in on", raw)
		}
		if got := c.PaymentProviderFor("admin@ucimo.local"); got != ProviderTBank {
			t.Errorf("PAYMENTS_DEV=%q: provider = %q, want the real one", raw, got)
		}
	}
}

// The production site is served over https. A copied .env with the flag in it must not turn the
// stand-in on there: it would let anyone pay with a button.
func TestPaymentsDevRefusesAnHTTPSBase(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_BASE_URL", "https://ucimo.ru")
	t.Setenv("PAYMENTS_DEV", "1")
	t.Setenv("TARIFFS_EMAILS", "admin@ucimo.local")
	c := Load()
	if c.PaymentsDev {
		t.Fatal("PaymentsDev = true on an https base")
	}
	if got := c.PaymentProviderFor("admin@ucimo.local"); got != ProviderTBank {
		t.Errorf("provider = %q, want %q", got, ProviderTBank)
	}
}
