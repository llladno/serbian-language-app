package mail

import (
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	netmail "net/mail"
	"strings"
	"testing"
)

// qpEncode mirrors buildMessage's part-body encoding so tests can assert on
// what actually lands on the wire.
func qpEncode(s string) string {
	var b strings.Builder
	w := quotedprintable.NewWriter(&b)
	_, _ = w.Write([]byte(s))
	_ = w.Close()
	return b.String()
}

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
	raw, err := buildMessage(
		"noreply@srpski.example",
		"Сербский с нуля", // non-ASCII: must be RFC 2047 encoded
		"grisha@example.com",
		"Подтверди почту", // Cyrillic subject: must be Q-encoded
		"Открой ссылку.\nСпасибо!",
		"<p>Открой ссылку</p>",
	)
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}
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
		"Content-Transfer-Encoding: quoted-printable",
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

	// Both alternative parts carry their bodies, quoted-printable encoded.
	if !strings.Contains(msg, qpEncode("Открой ссылку")) {
		t.Errorf("plain-text part body missing (quoted-printable):\n%s", msg)
	}
	if strings.Contains(msg, "Открой ссылку") {
		t.Errorf("part body is raw UTF-8, not quoted-printable:\n%s", msg)
	}
	if !strings.Contains(msg, qpEncode("<p>Открой ссылку</p>")) {
		t.Errorf("html part body missing (quoted-printable):\n%s", msg)
	}

	// Parts round-trip through a real MIME parser: quoted-printable decoding
	// yields the original UTF-8 bodies.
	plain, html := parseAlternative(t, raw)
	if !strings.Contains(plain, "Открой ссылку.") || !strings.Contains(plain, "Спасибо!") {
		t.Errorf("decoded text/plain = %q", plain)
	}
	if strings.TrimSpace(html) != "<p>Открой ссылку</p>" {
		t.Errorf("decoded text/html = %q", html)
	}
}

// parseAlternative parses a multipart/alternative message and returns the
// decoded text/plain and text/html part bodies.
func parseAlternative(t *testing.T, raw []byte) (plain, html string) {
	t.Helper()
	m, err := netmail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	_, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("ParseMediaType: %v", err)
	}
	mr := multipart.NewReader(m.Body, params["boundary"])
	for {
		p, err := mr.NextPart()
		if err != nil {
			break
		}
		// multipart.Part transparently decodes a quoted-printable body and
		// hides the CTE header; the literal header is asserted on the raw
		// message in the caller.
		b, err := io.ReadAll(p)
		if err != nil {
			t.Fatalf("decode part: %v", err)
		}
		switch ct := p.Header.Get("Content-Type"); {
		case strings.HasPrefix(ct, "text/plain"):
			plain = string(b)
		case strings.HasPrefix(ct, "text/html"):
			html = string(b)
		}
	}
	return plain, html
}

func TestBuildMessageASCIISubjectNotEncoded(t *testing.T) {
	raw, err := buildMessage("a@b.example", "Team", "c@d.example", "Welcome", "hi", "<p>hi</p>")
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}
	msg := string(raw)
	if !strings.Contains(msg, "Subject: Welcome\r\n") {
		t.Errorf("plain ASCII subject should pass through unencoded:\n%s", msg)
	}
	if !strings.Contains(msg, "From: Team <a@b.example>\r\n") {
		t.Errorf("ASCII From should be 'Name <addr>':\n%s", msg)
	}
}

func TestBuildMessageRejectsHeaderInjection(t *testing.T) {
	if _, err := buildMessage("a@b.example", "Team", "victim@x.io\r\nBcc: evil@x.io", "Hi", "t", "h"); err == nil {
		t.Error("buildMessage accepted a CRLF in the recipient")
	}
	if _, err := buildMessage("a@b.example", "Team", "c@d.example", "Hi\r\nBcc: evil@x.io", "t", "h"); err == nil {
		t.Error("buildMessage accepted a CRLF in the subject")
	}
	if _, err := buildMessage("a@b.example", "Team", "c@d.example", "Hi\nplain-lf", "t", "h"); err == nil {
		t.Error("buildMessage accepted a bare LF in the subject")
	}
}

func TestFormatAddress(t *testing.T) {
	cases := []struct {
		name, addr, want string
	}{
		{"", "from@x.io", "from@x.io"},
		{"Team", "a@b.example", "Team <a@b.example>"},
		{"Srpski, App", "a@b.io", `"Srpski, App" <a@b.io>`},
		{`Weird "Quote" \Slash`, "a@b.io", `"Weird \"Quote\" \\Slash" <a@b.io>`},
	}
	for _, c := range cases {
		got := formatAddress(c.name, c.addr)
		if got != c.want {
			t.Errorf("formatAddress(%q, %q) = %q, want %q", c.name, c.addr, got, c.want)
		}
		if c.name == "" {
			continue
		}
		if _, err := netmail.ParseAddress(got); err != nil {
			t.Errorf("formatAddress(%q, %q) = %q: unparseable: %v", c.name, c.addr, got, err)
		}
	}

	// Non-ASCII name stays an RFC 2047 encoded-word (no quoting).
	got := formatAddress("Сербский, с нуля", "a@b.io")
	if !strings.HasPrefix(got, "=?utf-8?q?") && !strings.HasPrefix(got, "=?UTF-8?q?") {
		t.Errorf("non-ASCII name should be an encoded-word: %q", got)
	}
	if strings.HasPrefix(got, `"`) {
		t.Errorf("encoded-word must not be wrapped in a quoted-string: %q", got)
	}
}
