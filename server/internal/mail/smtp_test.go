package mail

import (
	"strings"
	"testing"
)

// SMTPMailer must satisfy the Mailer interface.
var _ Mailer = (*SMTPMailer)(nil)

func TestNewSMTPMailerNilOnEmptyHost(t *testing.T) {
	if NewSMTPMailer(SMTPConfig{}) != nil {
		t.Error("NewSMTPMailer(SMTPConfig{}) must be nil when Host is empty")
	}
	if NewSMTPMailer(SMTPConfig{Host: "smtp.example.com"}) == nil {
		t.Error("NewSMTPMailer must return a mailer when Host is set")
	}
}

func TestBuildMessageStructure(t *testing.T) {
	raw := buildMessage(
		"noreply@srpski.example",
		"Сербский с нуля", // non-ASCII: must be RFC 2047 encoded
		"grisha@example.com",
		"Подтверди почту", // Cyrillic subject: must be Q-encoded
		"Открой ссылку.\nСпасибо!",
		"<p>Открой ссылку</p>",
	)
	msg := string(raw)

	for _, want := range []string{
		"From: ",
		"To: grisha@example.com\r\n",
		"Subject: ",
		"Date: ",
		"MIME-Version: 1.0\r\n",
		"Message-ID: <",
		"Content-Type: multipart/alternative; boundary=",
		"Content-Type: text/plain; charset=utf-8",
		"Content-Type: text/html; charset=utf-8",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("message missing %q\n----\n%s", want, msg)
		}
	}

	// Cyrillic subject is RFC 2047 Q-encoded, not raw UTF-8.
	if strings.Contains(msg, "Subject: Подтверди") {
		t.Errorf("Subject not encoded:\n%s", msg)
	}
	if !strings.Contains(strings.ToLower(msg), "=?utf-8?q?") {
		t.Errorf("Subject not Q-encoded:\n%s", msg)
	}
	// Non-ASCII display name is RFC 2047 encoded too.
	if strings.Contains(msg, "From: Сербский") {
		t.Errorf("From display name not encoded:\n%s", msg)
	}

	// Header block ends with a blank CRLF line.
	if !strings.Contains(msg, "\r\n\r\n") {
		t.Errorf("no CRLF header/body separator:\n%s", msg)
	}
	// CRLF only: stripping every CRLF must leave no bare LF.
	if strings.Contains(strings.ReplaceAll(msg, "\r\n", ""), "\n") {
		t.Errorf("message contains a bare LF (want CRLF only):\n%q", msg)
	}

	// Both alternative parts carry their bodies.
	if !strings.Contains(msg, "Открой ссылку") {
		t.Errorf("plain-text part body missing:\n%s", msg)
	}
	if !strings.Contains(msg, "<p>Открой ссылку</p>") {
		t.Errorf("html part body missing:\n%s", msg)
	}
}

func TestBuildMessageASCIISubjectNotEncoded(t *testing.T) {
	msg := string(buildMessage("a@b.example", "Team", "c@d.example", "Welcome", "hi", "<p>hi</p>"))
	if !strings.Contains(msg, "Subject: Welcome\r\n") {
		t.Errorf("plain ASCII subject should pass through unencoded:\n%s", msg)
	}
	if !strings.Contains(msg, "From: Team <a@b.example>\r\n") {
		t.Errorf("ASCII From should be 'Name <addr>':\n%s", msg)
	}
}
