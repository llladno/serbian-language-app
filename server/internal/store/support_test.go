package store

import (
	"testing"
	"time"
)

func TestCreateAndListSupportMessages(t *testing.T) {
	s, u := newUser(t)

	if err := s.CreateSupportMessage(u.user, "Не работает кнопка на уроке 5", day0); err != nil {
		t.Fatalf("CreateSupportMessage: %v", err)
	}
	sentAt := day0.Add(time.Hour)
	if err := s.CreateSupportMessage(u.user, "Было бы круто добавить тёмную тему", sentAt); err != nil {
		t.Fatalf("CreateSupportMessage (2nd): %v", err)
	}

	got, err := s.ListSupportMessages()
	if err != nil {
		t.Fatalf("ListSupportMessages: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListSupportMessages returned %d rows, want 2: %+v", len(got), got)
	}
	// newest first
	if got[0].Message != "Было бы круто добавить тёмную тему" || got[0].UserID != u.user {
		t.Errorf("got[0] = %+v, want the later message for %s", got[0], u.user)
	}
	if !got[0].CreatedAt.Equal(sentAt.UTC()) {
		t.Errorf("got[0].CreatedAt = %v, want %v", got[0].CreatedAt, sentAt.UTC())
	}
	if got[1].Message != "Не работает кнопка на уроке 5" {
		t.Errorf("got[1].Message = %q, want the first message", got[1].Message)
	}
}

func TestCreateSupportMessageRejectsBlank(t *testing.T) {
	s, u := newUser(t)
	if err := s.CreateSupportMessage(u.user, "   ", day0); err == nil {
		t.Fatal("CreateSupportMessage(blank) = nil error, want error")
	}
	got, err := s.ListSupportMessages()
	if err != nil {
		t.Fatalf("ListSupportMessages: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListSupportMessages = %d rows, want 0 (blank message must not be stored)", len(got))
	}
}
