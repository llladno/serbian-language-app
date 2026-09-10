package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

// SMTPConfig holds the connection and envelope settings for an SMTPMailer.
type SMTPConfig struct {
	Host, Port, User, Pass, From, FromName string
}

// SMTPMailer delivers mail over SMTP, upgrading to TLS with STARTTLS when the
// server advertises it. It is safe for concurrent use.
type SMTPMailer struct {
	cfg SMTPConfig
}

// NewSMTPMailer returns an SMTPMailer for c, or nil when c.Host is empty so
// callers can fall back to a LogMailer. An empty Port defaults to 587.
func NewSMTPMailer(c SMTPConfig) *SMTPMailer {
	if c.Host == "" {
		return nil
	}
	if c.Port == "" {
		c.Port = "587"
	}
	return &SMTPMailer{cfg: c}
}

// Send delivers one multipart/alternative message (plain text + HTML). ctx
// governs the initial dial; net/smtp has no context support past that point,
// so when ctx carries a deadline it is also pushed onto the raw connection to
// bound the greeting/STARTTLS/AUTH/DATA dialog and stop a stalled server from
// leaking the connection forever.
func (m *SMTPMailer) Send(ctx context.Context, to, subject, text, html string) error {
	addr := net.JoinHostPort(m.cfg.Host, m.cfg.Port)

	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("mail: dial %s: %w", addr, err)
	}

	// net/smtp ignores ctx once the socket is open. Bound the rest of the
	// dialog with the context deadline; it rides the same fd through StartTLS.
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}

	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("mail: smtp handshake with %s: %w", addr, err)
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: m.cfg.Host}); err != nil {
			return fmt.Errorf("mail: starttls: %w", err)
		}
	}

	if m.cfg.User != "" {
		if ok, _ := c.Extension("AUTH"); ok {
			auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Pass, m.cfg.Host)
			if err := c.Auth(auth); err != nil {
				return fmt.Errorf("mail: auth: %w", err)
			}
		}
	}

	if err := c.Mail(m.cfg.From); err != nil {
		return fmt.Errorf("mail: MAIL FROM %s: %w", m.cfg.From, err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("mail: RCPT TO %s: %w", to, err)
	}

	msg, err := buildMessage(m.cfg.From, m.cfg.FromName, to, subject, text, html)
	if err != nil {
		return err
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("mail: DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("mail: write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mail: finish message: %w", err)
	}
	return c.Quit()
}

// buildMessage renders one RFC 5322 message as MIME multipart/alternative
// (a plain-text part and an HTML part) with CRLF line endings, ready to hand
// to the SMTP DATA command. Part bodies are quoted-printable encoded so a
// submission host that does not advertise 8BITMIME cannot silently corrupt a
// UTF-8 body. Non-ASCII Subject and display name are RFC 2047 encoded. It
// returns an error if to or subject carries a CR/LF (header-injection guard;
// Send also calls c.Rcpt first, so this is defence in depth). This is the
// unit-tested seam; the SMTP wire dialog in Send is not exercised by tests.
func buildMessage(from, fromName, to, subject, text, html string) ([]byte, error) {
	if strings.ContainsAny(to, "\r\n") {
		return nil, fmt.Errorf("mail: recipient %q contains a newline", to)
	}
	if strings.ContainsAny(subject, "\r\n") {
		return nil, fmt.Errorf("mail: subject %q contains a newline", subject)
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	writePart := func(contentType, content string) {
		p, err := mw.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {contentType},
			"Content-Transfer-Encoding": {"quoted-printable"},
		})
		if err != nil { // bytes.Buffer writes never fail
			return
		}
		qw := quotedprintable.NewWriter(p)
		_, _ = qw.Write([]byte(toCRLF(content)))
		_ = qw.Close()
	}
	writePart("text/plain; charset=utf-8", text)
	writePart("text/html; charset=utf-8", html)
	_ = mw.Close()

	var msg bytes.Buffer
	header := func(k, v string) {
		msg.WriteString(k)
		msg.WriteString(": ")
		msg.WriteString(v)
		msg.WriteString("\r\n")
	}
	header("From", formatAddress(fromName, from))
	header("To", to)
	header("Subject", mime.QEncoding.Encode("utf-8", subject))
	header("Date", time.Now().Format(time.RFC1123Z))
	header("MIME-Version", "1.0")
	header("Message-ID", messageID(from, mw.Boundary()))
	header("Content-Type", `multipart/alternative; boundary="`+mw.Boundary()+`"`)
	msg.WriteString("\r\n")
	msg.Write(body.Bytes())
	return msg.Bytes(), nil
}

// addrSpecials are the RFC 5322 characters that force a display name to be
// wrapped in a quoted-string rather than emitted as a bare atom sequence.
const addrSpecials = "()<>[]:;@\\,.\""

// formatAddress renders a "Name <addr>" header value. An empty name emits the
// bare address. An ASCII name containing an RFC 5322 special is wrapped in a
// quoted-string (with \ and " escaped); a clean ASCII name passes through. A
// non-ASCII name becomes an RFC 2047 encoded-word, which needs no quoting.
func formatAddress(name, addr string) string {
	if name == "" {
		return addr
	}
	if isASCII(name) {
		if strings.ContainsAny(name, addrSpecials) {
			esc := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(name)
			return `"` + esc + `" <` + addr + ">"
		}
		return name + " <" + addr + ">"
	}
	return mime.QEncoding.Encode("utf-8", name) + " <" + addr + ">"
}

// messageID builds a unique <local@domain> identifier. The domain is taken
// from the From address; nonce is the already-random multipart boundary.
func messageID(from, nonce string) string {
	domain := "localhost"
	if i := strings.LastIndex(from, "@"); i >= 0 && i < len(from)-1 {
		domain = from[i+1:]
	}
	return fmt.Sprintf("<%s.%d@%s>", nonce, time.Now().UnixNano(), domain)
}

// toCRLF normalises any line ending to CRLF, as required by RFC 5322.
func toCRLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}

// isASCII reports whether s is pure 7-bit ASCII.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}
