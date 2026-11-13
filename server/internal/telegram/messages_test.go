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

func TestDefaultMessagesHaveEveryKey(t *testing.T) {
	want := []MessageKey{
		MsgStartGreeting, MsgLoginSuccessNew, MsgLoginSuccessExisting,
		MsgLoginTokenExpired, MsgLoginError, MsgLinkSuccess, MsgLinkTaken, MsgLinkError,
		MsgChannelPrompt, MsgChannelOK, MsgChannelNotYet,
		MsgPrefsStopped, MsgPrefsMenu, MsgPrefsNoAccount,
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

func TestGreetingName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Гриша", "Гриша"},
		{"гриша", "Гриша"},
		{"ГРИША", "Гриша"},
		{"  Анна  Петрова ", "Анна"},
		{"Анна-Мария", "Анна-Мария"},
		{"McDonald", "McDonald"},
		{"Ivan | дизайн", "Ivan"},
		{"gri_92", ""},
		{"gri92", ""},
		{"@gri", ""},
		{"tg123456", ""},
		{"Гриша 🔥", "Гриша"},
		{"🔥Гриша", ""},
		{"Г", ""},
		{"", ""},
		{"   ", ""},
		{"Александрооооооооооооовна", ""},
		{"-Гриша", ""},
	}
	for _, c := range cases {
		if got := GreetingName(c.in); got != c.want {
			t.Errorf("GreetingName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
