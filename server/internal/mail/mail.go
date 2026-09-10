// Package mail defines the Mailer interface used to send transactional
// email (verification, password reset, security notices) and ships two
// implementations: LogMailer for development and SMTPMailer for production.
package mail

import (
	"context"
	"log"
)

// Mailer delivers transactional email.
type Mailer interface {
	// Send delivers one message. Implementations must be safe for concurrent use.
	Send(ctx context.Context, to, subject, text, html string) error
}

// LogMailer writes a one-line summary plus the plaintext body to a logger
// instead of delivering anything. It is used in dev and tests, and as the
// first rollout step before SMTP credentials exist.
type LogMailer struct {
	// Logf receives the formatted line. A nil Logf falls back to log.Printf.
	Logf func(format string, args ...any)
}

// Send logs the recipient, subject and plaintext body (never the HTML part)
// and returns nil. It is a no-op when ctx is already cancelled.
func (m LogMailer) Send(ctx context.Context, to, subject, text, html string) error {
	if ctx.Err() != nil {
		return nil
	}
	logf := m.Logf
	if logf == nil {
		logf = log.Printf
	}
	logf("mail: to=%s subject=%q\n%s", to, subject, text)
	return nil
}
