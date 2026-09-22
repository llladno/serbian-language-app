package store

import (
	"testing"

	"github.com/grisha/serbian-app/server/internal/telegram"
)

func TestBotMessageTextUnknownKeyIsNotFound(t *testing.T) {
	s := newStore(t)
	_, ok, err := s.BotMessageText("no_such_key")
	if err != nil {
		t.Fatalf("BotMessageText: %v", err)
	}
	if ok {
		t.Errorf("BotMessageText(unknown) ok = true, want false")
	}
}

func TestSetAndGetBotMessageText(t *testing.T) {
	s := newStore(t)
	if err := s.SetBotMessageText("start_greeting", "Custom text", day0); err != nil {
		t.Fatalf("SetBotMessageText: %v", err)
	}
	text, ok, err := s.BotMessageText("start_greeting")
	if err != nil {
		t.Fatalf("BotMessageText: %v", err)
	}
	if !ok || text != "Custom text" {
		t.Fatalf("BotMessageText = (%q, %v), want (\"Custom text\", true)", text, ok)
	}

	if err := s.SetBotMessageText("start_greeting", "Updated text", day0); err != nil {
		t.Fatalf("SetBotMessageText (update): %v", err)
	}
	text, _, _ = s.BotMessageText("start_greeting")
	if text != "Updated text" {
		t.Errorf("BotMessageText after update = %q, want %q", text, "Updated text")
	}
}

func TestMigration007SeedsAllDefaultMessages(t *testing.T) {
	s := newStore(t)
	for key, want := range telegram.DefaultMessages {
		text, ok, err := s.BotMessageText(string(key))
		if err != nil {
			t.Fatalf("BotMessageText(%s): %v", key, err)
		}
		if !ok {
			t.Errorf("key %s not seeded by migration 007", key)
			continue
		}
		if text != want {
			t.Errorf("seeded text for %s = %q, want %q", key, text, want)
		}
	}
}
