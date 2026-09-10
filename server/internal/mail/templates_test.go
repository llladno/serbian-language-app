package mail

import (
	"net/url"
	"strings"
	"testing"
)

func TestRenderTemplates(t *testing.T) {
	const (
		baseURL = "https://srpski.example"
		name    = "Гриша"
		token   = "abc123XYZ_-.tok"
	)
	verifyURL := baseURL + "/api/auth/verify?token=" + url.QueryEscape(token)
	resetURL := baseURL + "/reset?token=" + url.QueryEscape(token)

	type rendered struct {
		subject, text, html string
		wantURL             string // full action URL that must appear in text+html; "" to skip
	}

	vS, vT, vH := RenderVerify(baseURL, token, name)
	rS, rT, rH := RenderReset(baseURL, token, name)
	aS, aT, aH := RenderAlreadyRegistered(baseURL, name)
	pS, pT, pH := RenderPasswordChanged(baseURL, name)

	cases := map[string]rendered{
		"verify":             {vS, vT, vH, verifyURL},
		"reset":              {rS, rT, rH, resetURL},
		"already_registered": {aS, aT, aH, ""},
		"password_changed":   {pS, pT, pH, ""},
	}

	for tn, c := range cases {
		t.Run(tn, func(t *testing.T) {
			for field, v := range map[string]string{"subject": c.subject, "text": c.text, "html": c.html} {
				if strings.Contains(v, "{{") || strings.Contains(v, "}}") {
					t.Errorf("%s contains unrendered template markers: %q", field, v)
				}
				if strings.Contains(v, "<no value>") {
					t.Errorf("%s contains <no value>: %q", field, v)
				}
			}
			for field, v := range map[string]string{"text": c.text, "html": c.html} {
				if !strings.Contains(v, name) {
					t.Errorf("%s does not mention the recipient name: %q", field, v)
				}
			}
			if c.subject == "" {
				t.Error("subject is empty")
			}
			if strings.ContainsAny(c.subject, "\r\n") {
				t.Errorf("subject contains a newline: %q", c.subject)
			}
			if !strings.Contains(c.html, "<a ") {
				t.Errorf("html has no anchor button: %q", c.html)
			}
			if c.wantURL != "" {
				if !strings.Contains(c.text, c.wantURL) {
					t.Errorf("text is missing the action URL %q\n%s", c.wantURL, c.text)
				}
				if !strings.Contains(c.html, c.wantURL) {
					t.Errorf("html is missing the action URL %q\n%s", c.wantURL, c.html)
				}
			}
		})
	}
}

func TestRenderVerifyResetLinkShape(t *testing.T) {
	_, vT, _ := RenderVerify("https://x.example", "a b/c+d", "Имя")
	if !strings.Contains(vT, "https://x.example/api/auth/verify?token=a+b%2Fc%2Bd") {
		t.Errorf("verify link not query-escaped: %q", vT)
	}
	_, rT, _ := RenderReset("https://x.example", "a b/c+d", "Имя")
	if !strings.Contains(rT, "https://x.example/reset?token=a+b%2Fc%2Bd") {
		t.Errorf("reset link not query-escaped: %q", rT)
	}
}

func TestRenderAlreadyRegisteredPhrasing(t *testing.T) {
	_, text, html := RenderAlreadyRegistered("https://srpski.example", "Гриша")
	body := text + "\n" + html

	if !strings.Contains(strings.ToLower(body), "если это был") {
		t.Errorf("already_registered must use the \"если это был ты…\" framing:\n%s", text)
	}
	if strings.Contains(body, "у тебя есть аккаунт") {
		t.Errorf("already_registered must not directly state that an account exists:\n%s", text)
	}
	// Points the reader at login and password recovery.
	if !strings.Contains(body, "https://srpski.example/login") {
		t.Errorf("already_registered missing /login link:\n%s", text)
	}
	if !strings.Contains(body, "https://srpski.example/forgot") {
		t.Errorf("already_registered missing /forgot link:\n%s", text)
	}
}
