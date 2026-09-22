package telegram

import "testing"

func TestSubstituteReplacesName(t *testing.T) {
	got := Substitute("Zdravo, {name}! Ты — {name}.", "Neo")
	want := "Zdravo, Neo! Ты — Neo."
	if got != want {
		t.Errorf("Substitute = %q, want %q", got, want)
	}
}

func TestDisplayNamePrefersFirstName(t *testing.T) {
	if got := DisplayName("Neo", "neo_bot"); got != "Neo" {
		t.Errorf("DisplayName = %q, want %q", got, "Neo")
	}
}

func TestDisplayNameFallsBackToUsername(t *testing.T) {
	if got := DisplayName("", "neo_bot"); got != "@neo_bot" {
		t.Errorf("DisplayName = %q, want %q", got, "@neo_bot")
	}
}

func TestDisplayNameFallsBackToGeneric(t *testing.T) {
	if got := DisplayName("", ""); got != "друг" {
		t.Errorf("DisplayName = %q, want %q", got, "друг")
	}
}

func TestDefaultMessagesHaveAllEightKeys(t *testing.T) {
	want := []MessageKey{
		MsgStartGreeting, MsgLoginSuccessNew, MsgLoginSuccessExisting,
		MsgLoginTokenExpired, MsgLoginError, MsgLinkSuccess, MsgLinkTaken, MsgLinkError,
	}
	if len(DefaultMessages) != len(want) {
		t.Fatalf("DefaultMessages has %d entries, want %d", len(DefaultMessages), len(want))
	}
	for _, k := range want {
		if DefaultMessages[k] == "" {
			t.Errorf("DefaultMessages[%s] is empty", k)
		}
	}
}
