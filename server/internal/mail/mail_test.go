package mail

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// LogMailer must satisfy the Mailer interface.
var _ Mailer = LogMailer{}

func TestLogMailerCapturesToLogf(t *testing.T) {
	var got string
	m := LogMailer{Logf: func(format string, args ...any) {
		got = fmt.Sprintf(format, args...)
	}}

	err := m.Send(context.Background(),
		"grisha@example.com",
		"Подтверди почту",
		"Открой ссылку https://srpski.example/verify чтобы продолжить.",
		"<a href=\"https://srpski.example/verify\">верификация</a>")
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	for _, want := range []string{
		"grisha@example.com",
		"Подтверди почту",
		"Открой ссылку https://srpski.example/verify",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("captured log %q does not contain %q", got, want)
		}
	}
	if strings.Contains(got, "<a href") {
		t.Errorf("captured log must not contain the html body: %q", got)
	}
}

func TestLogMailerNilLogf(t *testing.T) {
	// A zero-value LogMailer must fall back to log.Printf: no panic, nil error.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Send panicked with nil Logf: %v", r)
		}
	}()
	if err := (LogMailer{}).Send(context.Background(), "a@b.c", "s", "t", "h"); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
}

func TestLogMailerSkipsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	m := LogMailer{Logf: func(string, ...any) { called = true }}
	if err := m.Send(ctx, "a@b.c", "s", "t", "h"); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if called {
		t.Error("Logf was called even though ctx was already cancelled")
	}
}
