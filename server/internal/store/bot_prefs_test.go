package store

import (
	"testing"
	"time"
)

func TestBotPrefsDefaultsAndRoundTrip(t *testing.T) {
	s := newStore(t)
	uid, err := s.CreateUser("Гриша")
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.BotPrefs(uid)
	if err != nil {
		t.Fatalf("BotPrefs: %v", err)
	}
	// Study and tariff reminders are what the bot has always sent; news is
	// advertising and must be switched on by the person.
	if got != (BotPrefs{Learning: true, Billing: true, News: false}) {
		t.Fatalf("defaults = %+v", got)
	}

	if err := s.SetBotPrefs(uid, BotPrefs{Learning: false, Billing: true, News: true}, day0); err != nil {
		t.Fatalf("SetBotPrefs: %v", err)
	}
	got, _ = s.BotPrefs(uid)
	if got != (BotPrefs{Learning: false, Billing: true, News: true}) {
		t.Fatalf("after set = %+v", got)
	}

	// A second write replaces the first, it does not add a row.
	if err := s.SetBotPrefs(uid, BotPrefs{}, day0.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, _ = s.BotPrefs(uid)
	if got != (BotPrefs{}) {
		t.Fatalf("after second set = %+v, want all off", got)
	}
}

func TestBotPrefsAreIndependentPerAccount(t *testing.T) {
	s := newStore(t)
	a, _ := s.CreateUser("А")
	b, _ := s.CreateUser("Б")
	if err := s.SetBotPrefs(a, BotPrefs{}, day0); err != nil {
		t.Fatal(err)
	}
	got, _ := s.BotPrefs(b)
	if got != DefaultBotPrefs {
		t.Fatalf("b = %+v, want defaults: a's opt-out must not leak", got)
	}
}
