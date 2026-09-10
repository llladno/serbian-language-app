package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
	"github.com/grisha/serbian-app/server/internal/ratelimit"
	"github.com/grisha/serbian-app/server/internal/store"
)

// mailToken pulls the raw action token out of a captured mail body. Both the
// verify link (…/api/auth/verify?token=) and the reset link (…/reset?token=)
// carry it after "token="; the token is base64url, so QueryEscape leaves it
// untouched and it appears verbatim up to the end of the line.
var mailTokenRe = regexp.MustCompile(`token=([A-Za-z0-9_-]+)`)

func mailToken(t *testing.T, m sentMail) string {
	t.Helper()
	sub := mailTokenRe.FindStringSubmatch(m.Text)
	if sub == nil {
		t.Fatalf("no token in mail body: %q", m.Text)
	}
	return sub[1]
}

// ---- test wiring ----

// sentMail is one message captured by mailSink.
type sentMail struct{ To, Subject, Text, HTML string }

// mailSink is a capturing SendMail: the auth tests assert on what would have
// been delivered without an SMTP server.
type mailSink struct {
	mu   sync.Mutex
	msgs []sentMail
}

func (s *mailSink) send(to, subject, text, html string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, sentMail{to, subject, text, html})
}

func (s *mailSink) all() []sentMail {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]sentMail(nil), s.msgs...)
}

// newAuthAPI builds the API with a capturing mailer and a synchronous Async,
// so a test can assert on the post-response work (mail, account rows) with no
// sleeps. tweak, if set, gets the last word on Deps (clock, limiters).
func newAuthAPI(t *testing.T, tweak func(*Deps)) (http.Handler, *store.Store, *mailSink) {
	t.Helper()
	sink := &mailSink{}
	h, st := newTestAPIWith(t, func(d *Deps) {
		d.SendMail = sink.send
		d.Async = func(f func()) { f() }
		if tweak != nil {
			tweak(d)
		}
	})
	return h, st, sink
}

// anon issues a request with no credentials (no X-User, no cookie); the Origin
// header is still stamped on writes so checkOrigin lets them through.
func anon(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	return doAs(h, "", method, path, body)
}

func registerBody(email, password, name string) string {
	return fmt.Sprintf(`{"email":%q,"password":%q,"name":%q}`, email, password, name)
}

func loginBody(email, password string) string {
	return fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
}

// registerAndVerify registers an account through the HTTP handler, then flips
// its identity to verified directly in the store. Returns the user id.
func registerAndVerify(t *testing.T, h http.Handler, st *store.Store, email, password, name string) string {
	t.Helper()
	if rr := anon(h, "POST", "/api/auth/register", registerBody(email, password, name)); rr.Code != http.StatusOK {
		t.Fatalf("register %s: %d %s", email, rr.Code, rr.Body)
	}
	id, err := st.IdentityByProviderUID("password", strings.ToLower(email))
	if err != nil {
		t.Fatalf("identity after register %s: %v", email, err)
	}
	if err := st.SetEmailVerified(id.ID, fixedNow); err != nil {
		t.Fatalf("SetEmailVerified: %v", err)
	}
	return id.UserID
}

func grabCookie(t *testing.T, rr *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, c := range rr.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no %q cookie in response (got %v)", name, rr.Result().Cookies())
	return nil
}

func grabSessionCookie(t *testing.T, rr *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	c := grabCookie(t, rr, "session")
	if c.Value == "" {
		t.Fatalf("session cookie has empty value: %+v", c)
	}
	return c
}

// ---- register ----

func TestRegisterCreatesUnverifiedAndSendsVerify(t *testing.T) {
	h, st, sink := newAuthAPI(t, nil)

	rr := anon(h, "POST", "/api/auth/register", registerBody("bob@example.com", "password123", "Bob"))
	if rr.Code != http.StatusOK {
		t.Fatalf("register = %d %s", rr.Code, rr.Body)
	}
	if got := decodeBody[map[string]string](t, rr); got["status"] != "ok" {
		t.Fatalf("body = %v, want {status: ok}", got)
	}

	msgs := sink.all()
	if len(msgs) != 1 {
		t.Fatalf("sent %d mails, want 1: %+v", len(msgs), msgs)
	}
	if msgs[0].To != "bob@example.com" || !strings.Contains(msgs[0].Text, "/api/auth/verify?token=") {
		t.Errorf("not a verification mail: %+v", msgs[0])
	}

	id, err := st.IdentityByProviderUID("password", "bob@example.com")
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	if id.EmailVerifiedAt != "" {
		t.Errorf("email_verified_at = %q, want empty", id.EmailVerifiedAt)
	}
	if id.Email != "bob@example.com" {
		t.Errorf("identity email = %q", id.Email)
	}
	if row, err := st.UserByID(id.UserID); err != nil || row.Name != "Bob" {
		t.Errorf("user row = %+v, err %v", row, err)
	}
}

func TestRegisterExistingEmailGenericAndAlreadyMail(t *testing.T) {
	h, st, sink := newAuthAPI(t, nil)
	body := registerBody("bob@example.com", "password123", "Bob")

	if rr := anon(h, "POST", "/api/auth/register", body); rr.Code != http.StatusOK {
		t.Fatalf("first register = %d", rr.Code)
	}
	rr := anon(h, "POST", "/api/auth/register", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("second register = %d %s", rr.Code, rr.Body)
	}
	if got := decodeBody[map[string]string](t, rr); got["status"] != "ok" {
		t.Fatalf("second register body = %v, want the same generic {status: ok}", got)
	}

	msgs := sink.all()
	if len(msgs) != 2 {
		t.Fatalf("sent %d mails, want 2 (verify, then already-registered): %+v", len(msgs), msgs)
	}
	last := msgs[1]
	if strings.Contains(last.Text, "verify?token=") {
		t.Errorf("already-registered mail leaked a verify token: %+v", last)
	}
	if !strings.Contains(last.Text, "/login") || !strings.Contains(last.Text, "/forgot") {
		t.Errorf("already-registered mail missing login/forgot links: %+v", last)
	}

	users, err := st.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 2 { // "tester" (fixture) + "Bob"
		t.Errorf("user count = %d, want 2 (no second account for the taken email)", len(users))
	}
}

func TestRegisterBadInput(t *testing.T) {
	h, _, sink := newAuthAPI(t, nil)
	for _, tc := range []struct {
		name, body string
	}{
		{"bad email", registerBody("not-an-email", "password123", "Bob")},
		{"short password", registerBody("bob@example.com", "short", "Bob")},
		{"blank name", registerBody("bob@example.com", "password123", "   ")},
		{"long name", registerBody("bob@example.com", "password123", strings.Repeat("x", 41))},
	} {
		if rr := anon(h, "POST", "/api/auth/register", tc.body); rr.Code != http.StatusBadRequest {
			t.Errorf("%s: code = %d, want 400", tc.name, rr.Code)
		}
	}
	if n := len(sink.all()); n != 0 {
		t.Errorf("bad input still sent %d mails", n)
	}
}

// ---- login ----

func TestLoginUnverified403(t *testing.T) {
	h, _, sink := newAuthAPI(t, nil)
	if rr := anon(h, "POST", "/api/auth/register", registerBody("bob@example.com", "password123", "Bob")); rr.Code != http.StatusOK {
		t.Fatalf("register = %d", rr.Code)
	}

	rr := anon(h, "POST", "/api/auth/login", loginBody("bob@example.com", "password123"))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("login = %d %s, want 403", rr.Code, rr.Body)
	}
	if got := decodeBody[map[string]string](t, rr); got["error"] != "email_unverified" {
		t.Errorf("error = %q, want email_unverified", got["error"])
	}

	msgs := sink.all()
	if len(msgs) != 2 {
		t.Fatalf("sent %d mails, want 2 (register verify + login resend)", len(msgs))
	}
	if !strings.Contains(msgs[1].Text, "/api/auth/verify?token=") {
		t.Errorf("resent mail is not a verification link: %+v", msgs[1])
	}
}

func TestLoginWrongPassword401Generic(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")

	rr := anon(h, "POST", "/api/auth/login", loginBody("bob@example.com", "WRONGpass1"))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("login = %d, want 401", rr.Code)
	}
	if got := decodeBody[map[string]string](t, rr); got["error"] != "неверная почта или пароль" {
		t.Errorf("error = %q", got["error"])
	}
}

func TestLoginSuccessSetsCookie(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	registerAndVerify(t, h, st, "alice@example.com", "password123", "Alice")

	rr := anon(h, "POST", "/api/auth/login", loginBody("alice@example.com", "password123"))
	if rr.Code != http.StatusOK {
		t.Fatalf("login = %d %s", rr.Code, rr.Body)
	}
	user := decodeBody[sessionUserDTO](t, rr)
	if user.Name != "Alice" || user.Email != "alice@example.com" || !user.EmailVerified {
		t.Errorf("login user = %+v", user)
	}
	c := grabSessionCookie(t, rr)

	sr := doCookie(h, c, "GET", "/api/auth/session", "")
	if sr.Code != http.StatusOK {
		t.Fatalf("GET /api/auth/session with the login cookie = %d %s", sr.Code, sr.Body)
	}
	if got := decodeBody[sessionUserDTO](t, sr); got.Name != "Alice" {
		t.Errorf("session user = %+v", got)
	}
}

// TestLoginDTOReflectsTelegramLink pins that the login-success body carries the
// real telegram linkage (it is built via summaryFor, not an inline literal that
// always reported linked=false).
func TestLoginDTOReflectsTelegramLink(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "linked@example.com", "password123", "Linked")

	if err := st.CreateIdentity(store.Identity{
		ID:          auth.NewIdentityID(),
		UserID:      uid,
		Provider:    "telegram",
		ProviderUID: "5550123",
		TgUsername:  "linked_tg",
	}); err != nil {
		t.Fatalf("CreateIdentity telegram: %v", err)
	}

	rr := anon(h, "POST", "/api/auth/login", loginBody("linked@example.com", "password123"))
	if rr.Code != http.StatusOK {
		t.Fatalf("login = %d %s", rr.Code, rr.Body)
	}
	user := decodeBody[sessionUserDTO](t, rr)
	if !user.Telegram.Linked || user.Telegram.Username != "linked_tg" {
		t.Errorf("login DTO telegram = %+v, want linked with username linked_tg", user.Telegram)
	}
	if user.Name != "Linked" || user.Email != "linked@example.com" || !user.EmailVerified {
		t.Errorf("login DTO = %+v", user)
	}
}

func TestLoginRateLimited(t *testing.T) {
	h, st, _ := newAuthAPI(t, func(d *Deps) {
		d.Login = ratelimit.NewLimiter(1, 0) // burst 0: never admits
	})
	registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")

	rr := anon(h, "POST", "/api/auth/login", loginBody("bob@example.com", "password123"))
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("login = %d, want 429", rr.Code)
	}
	if got := decodeBody[map[string]string](t, rr); got["error"] != "too many attempts" {
		t.Errorf("error = %q", got["error"])
	}
}

func TestLoginSoftLock(t *testing.T) {
	clock := fixedNow
	now := func() time.Time { return clock }
	fails := ratelimit.NewFailCounter(10, 15*time.Minute)
	fails.SetNow(now)

	h, st, _ := newAuthAPI(t, func(d *Deps) {
		d.Now = now
		d.Fails = fails
	})
	registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")

	for i := 0; i < 10; i++ {
		if rr := anon(h, "POST", "/api/auth/login", loginBody("bob@example.com", "nope-nope")); rr.Code != http.StatusUnauthorized {
			t.Fatalf("wrong attempt %d = %d, want 401", i, rr.Code)
		}
	}
	// Locked: the right password is now rejected too, with the same generic body.
	if rr := anon(h, "POST", "/api/auth/login", loginBody("bob@example.com", "password123")); rr.Code != http.StatusUnauthorized {
		t.Fatalf("locked attempt with correct password = %d, want 401", rr.Code)
	}

	// The failures age out of the window.
	clock = clock.Add(16 * time.Minute)
	rr := anon(h, "POST", "/api/auth/login", loginBody("bob@example.com", "password123"))
	if rr.Code != http.StatusOK {
		t.Fatalf("after the lock window = %d %s, want 200", rr.Code, rr.Body)
	}
	grabSessionCookie(t, rr)
}

// ---- verify email / resend verification ----

func TestVerifyEmailHappyRedirect(t *testing.T) {
	h, st, sink := newAuthAPI(t, nil)

	if rr := anon(h, "POST", "/api/auth/register", registerBody("bob@example.com", "password123", "Bob")); rr.Code != http.StatusOK {
		t.Fatalf("register = %d %s", rr.Code, rr.Body)
	}
	msgs := sink.all()
	if len(msgs) != 1 {
		t.Fatalf("register sent %d mails, want 1", len(msgs))
	}
	token := mailToken(t, msgs[0])

	rr := anon(h, "GET", "/api/auth/verify?token="+token, "")
	if rr.Code != http.StatusFound {
		t.Fatalf("verify = %d, want 302", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != testBaseURL+"/verify?ok=1" {
		t.Errorf("Location = %q, want %q", loc, testBaseURL+"/verify?ok=1")
	}

	id, err := st.IdentityByProviderUID("password", "bob@example.com")
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	if id.EmailVerifiedAt == "" {
		t.Errorf("email_verified_at still empty after verify")
	}

	// The same token a second time fails closed.
	rr = anon(h, "GET", "/api/auth/verify?token="+token, "")
	if rr.Code != http.StatusFound || rr.Header().Get("Location") != testBaseURL+"/verify?err=1" {
		t.Errorf("reused token = %d %q, want 302 %q", rr.Code, rr.Header().Get("Location"), testBaseURL+"/verify?err=1")
	}
}

func TestVerifyEmailBadToken(t *testing.T) {
	h, _, _ := newAuthAPI(t, nil)
	for _, name := range []string{"garbage-token-value", ""} {
		rr := anon(h, "GET", "/api/auth/verify?token="+name, "")
		if rr.Code != http.StatusFound || rr.Header().Get("Location") != testBaseURL+"/verify?err=1" {
			t.Errorf("token %q = %d %q, want 302 %q", name, rr.Code, rr.Header().Get("Location"), testBaseURL+"/verify?err=1")
		}
	}
}

func TestResendVerificationGeneric(t *testing.T) {
	h, st, sink := newAuthAPI(t, nil)
	if rr := anon(h, "POST", "/api/auth/register", registerBody("bob@example.com", "password123", "Bob")); rr.Code != http.StatusOK {
		t.Fatalf("register = %d", rr.Code)
	}
	if n := len(sink.all()); n != 1 {
		t.Fatalf("after register: %d mails, want 1", n)
	}

	// Existing + unverified: a fresh verify mail, generic body.
	rr := anon(h, "POST", "/api/auth/resend-verification", `{"email":"Bob@example.com"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("resend = %d %s", rr.Code, rr.Body)
	}
	if got := decodeBody[map[string]string](t, rr); got["status"] != "ok" {
		t.Errorf("body = %v, want {status: ok}", got)
	}
	msgs := sink.all()
	if len(msgs) != 2 {
		t.Fatalf("after resend: %d mails, want 2", len(msgs))
	}
	if msgs[1].To != "bob@example.com" || !strings.Contains(msgs[1].Text, "/api/auth/verify?token=") {
		t.Errorf("resend mail is not a verification link: %+v", msgs[1])
	}

	// Nonexistent email: still 200, no mail.
	rr = anon(h, "POST", "/api/auth/resend-verification", `{"email":"nobody@example.com"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("resend nonexistent = %d", rr.Code)
	}
	if n := len(sink.all()); n != 2 {
		t.Errorf("nonexistent email produced mail: %d total", n)
	}

	// Already verified: 200, no mail.
	id, _ := st.IdentityByProviderUID("password", "bob@example.com")
	if err := st.SetEmailVerified(id.ID, fixedNow); err != nil {
		t.Fatalf("SetEmailVerified: %v", err)
	}
	rr = anon(h, "POST", "/api/auth/resend-verification", `{"email":"bob@example.com"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("resend already-verified = %d", rr.Code)
	}
	if n := len(sink.all()); n != 2 {
		t.Errorf("already-verified produced mail: %d total", n)
	}
}

// ---- forgot password / reset password ----

func TestForgotGeneric(t *testing.T) {
	h, st, sink := newAuthAPI(t, nil)
	registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	base := len(sink.all()) // register verify mail

	rr := anon(h, "POST", "/api/auth/forgot", `{"email":"Bob@example.com"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("forgot = %d %s", rr.Code, rr.Body)
	}
	body := rr.Body.String()
	msgs := sink.all()
	if len(msgs) != base+1 {
		t.Fatalf("forgot sent %d new mails, want 1", len(msgs)-base)
	}
	reset := msgs[len(msgs)-1]
	if reset.To != "bob@example.com" || !strings.Contains(reset.Text, "/reset?token=") {
		t.Errorf("not a reset mail: %+v", reset)
	}

	// Nonexistent email: identical body, no mail.
	rr = anon(h, "POST", "/api/auth/forgot", `{"email":"ghost@example.com"}`)
	if rr.Code != http.StatusOK || rr.Body.String() != body {
		t.Errorf("nonexistent forgot = %d %q, want 200 %q", rr.Code, rr.Body.String(), body)
	}
	if len(sink.all()) != base+1 {
		t.Errorf("nonexistent forgot produced a mail: %d total", len(sink.all()))
	}
}

func TestResetChangesPasswordAndKillsSessions(t *testing.T) {
	h, st, sink := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")

	// A pre-existing session (device A), created straight in the store.
	cookieA := authed(t, st, uid)
	if rr := doCookie(h, cookieA, "GET", "/api/auth/session", ""); rr.Code != http.StatusOK {
		t.Fatalf("session A should be live before reset: %d", rr.Code)
	}

	if rr := anon(h, "POST", "/api/auth/forgot", `{"email":"bob@example.com"}`); rr.Code != http.StatusOK {
		t.Fatalf("forgot = %d", rr.Code)
	}
	msgs := sink.all()
	token := mailToken(t, msgs[len(msgs)-1])

	rr := anon(h, "POST", "/api/auth/reset", fmt.Sprintf(`{"token":%q,"password":"newpass456"}`, token))
	if rr.Code != http.StatusOK {
		t.Fatalf("reset = %d %s", rr.Code, rr.Body)
	}
	newCookie := grabSessionCookie(t, rr)
	if user := decodeBody[sessionUserDTO](t, rr); user.Name != "Bob" || user.Email != "bob@example.com" {
		t.Errorf("reset DTO = %+v", user)
	}

	// The autologin cookie is live; the pre-existing one is dead.
	if sr := doCookie(h, newCookie, "GET", "/api/auth/session", ""); sr.Code != http.StatusOK {
		t.Errorf("autologin cookie not accepted: %d", sr.Code)
	}
	if sr := doCookie(h, cookieA, "GET", "/api/auth/session", ""); sr.Code != http.StatusUnauthorized {
		t.Errorf("pre-existing session survived reset: %d", sr.Code)
	}

	// Old password no longer works; the new one does.
	if lr := anon(h, "POST", "/api/auth/login", loginBody("bob@example.com", "password123")); lr.Code != http.StatusUnauthorized {
		t.Errorf("old password still logs in: %d", lr.Code)
	}
	if lr := anon(h, "POST", "/api/auth/login", loginBody("bob@example.com", "newpass456")); lr.Code != http.StatusOK {
		t.Errorf("new password rejected: %d %s", lr.Code, lr.Body)
	}

	// A password-changed notice went out.
	last := sink.all()[len(sink.all())-1]
	if last.To != "bob@example.com" || !strings.Contains(last.Subject, "Пароль изменён") {
		t.Errorf("no password-changed mail: %+v", last)
	}

	// The reset token is single-use.
	rr = anon(h, "POST", "/api/auth/reset", fmt.Sprintf(`{"token":%q,"password":"another789"}`, token))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("reused reset token = %d, want 400", rr.Code)
	}
	if got := decodeBody[map[string]string](t, rr); got["error"] != "invalid_token" {
		t.Errorf("error = %q, want invalid_token", got["error"])
	}
}

func TestResetBadToken(t *testing.T) {
	h, _, _ := newAuthAPI(t, nil)
	rr := anon(h, "POST", "/api/auth/reset", `{"token":"nonsense","password":"password123"}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("bad token = %d, want 400", rr.Code)
	}
	if got := decodeBody[map[string]string](t, rr); got["error"] != "invalid_token" {
		t.Errorf("error = %q, want invalid_token", got["error"])
	}
}

func TestResetShortPassword(t *testing.T) {
	h, st, sink := newAuthAPI(t, nil)
	registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	if rr := anon(h, "POST", "/api/auth/forgot", `{"email":"bob@example.com"}`); rr.Code != http.StatusOK {
		t.Fatalf("forgot = %d", rr.Code)
	}
	token := mailToken(t, sink.all()[len(sink.all())-1])

	rr := anon(h, "POST", "/api/auth/reset", fmt.Sprintf(`{"token":%q,"password":"short"}`, token))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("short password = %d, want 400", rr.Code)
	}
	// The short-password rejection must not have burned the token.
	rr = anon(h, "POST", "/api/auth/reset", fmt.Sprintf(`{"token":%q,"password":"newpass456"}`, token))
	if rr.Code != http.StatusOK {
		t.Errorf("valid reset after a short-password attempt = %d %s", rr.Code, rr.Body)
	}
}

// TestResetMarksEmailVerified: a user who never verified, then completes an
// emailed reset link, is verified afterwards — finishing the link proves inbox
// control, so the autologin session must not be bounced to /verify.
func TestResetMarksEmailVerified(t *testing.T) {
	h, st, sink := newAuthAPI(t, nil)

	if rr := anon(h, "POST", "/api/auth/register", registerBody("bob@example.com", "password123", "Bob")); rr.Code != http.StatusOK {
		t.Fatalf("register = %d %s", rr.Code, rr.Body)
	}
	if id, err := st.IdentityByProviderUID("password", "bob@example.com"); err != nil || id.EmailVerifiedAt != "" {
		t.Fatalf("precondition: identity should be unverified, got %+v err %v", id, err)
	}

	if rr := anon(h, "POST", "/api/auth/forgot", `{"email":"bob@example.com"}`); rr.Code != http.StatusOK {
		t.Fatalf("forgot = %d", rr.Code)
	}
	token := mailToken(t, sink.all()[len(sink.all())-1])

	rr := anon(h, "POST", "/api/auth/reset", fmt.Sprintf(`{"token":%q,"password":"newpass456"}`, token))
	if rr.Code != http.StatusOK {
		t.Fatalf("reset = %d %s", rr.Code, rr.Body)
	}
	cookie := grabSessionCookie(t, rr)

	// The identity column is stamped...
	if id, err := st.IdentityByProviderUID("password", "bob@example.com"); err != nil || id.EmailVerifiedAt == "" {
		t.Errorf("email_verified_at still empty after reset: %+v err %v", id, err)
	}
	// ...and the autologin session reports it, so the SPA won't route to /verify.
	sr := doCookie(h, cookie, "GET", "/api/auth/session", "")
	if sr.Code != http.StatusOK {
		t.Fatalf("session after reset = %d %s", sr.Code, sr.Body)
	}
	if got := decodeBody[sessionUserDTO](t, sr); !got.EmailVerified {
		t.Errorf("session DTO email_verified = false after reset, want true; DTO = %+v", got)
	}
}

// ---- clientIP ----

// TestClientIP locks the proxy-header trust order: X-Real-Ip wins, else the
// RIGHT-most X-Forwarded-For entry (the hop Traefik appended), never a
// client-supplied left-most one, else RemoteAddr's host.
func TestClientIP(t *testing.T) {
	req := func(remote string, headers map[string]string) *http.Request {
		r := httptest.NewRequest("POST", "/api/auth/login", nil)
		r.RemoteAddr = remote
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		return r
	}

	// Right-most XFF entry wins; a spoofed left-most entry is ignored.
	if got := clientIP(req("10.0.0.1:9999", map[string]string{
		"X-Forwarded-For": "1.2.3.4, 5.6.7.8",
	})); got != "5.6.7.8" {
		t.Errorf("XFF right-most: got %q, want 5.6.7.8", got)
	}
	// X-Real-Ip beats any X-Forwarded-For.
	if got := clientIP(req("10.0.0.1:9999", map[string]string{
		"X-Real-Ip":       "9.9.9.9",
		"X-Forwarded-For": "1.2.3.4, 5.6.7.8",
	})); got != "9.9.9.9" {
		t.Errorf("X-Real-Ip precedence: got %q, want 9.9.9.9", got)
	}
	// Neither header: RemoteAddr host:port -> host.
	if got := clientIP(req("192.0.2.5:55000", nil)); got != "192.0.2.5" {
		t.Errorf("RemoteAddr host:port: got %q, want 192.0.2.5", got)
	}
	// Neither header, RemoteAddr already a bare host.
	if got := clientIP(req("192.0.2.9", nil)); got != "192.0.2.9" {
		t.Errorf("RemoteAddr bare host: got %q, want 192.0.2.9", got)
	}
}

// ---- logout ----

func TestLogoutClearsSession(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := grabSessionCookie(t, anon(h, "POST", "/api/auth/login", loginBody("bob@example.com", "password123")))
	hash := auth.HashToken(c.Value)
	if _, err := st.SessionByHash(hash, fixedNow); err != nil {
		t.Fatalf("session should exist before logout: %v", err)
	}

	rr := doCookie(h, c, "POST", "/api/auth/logout", "")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("logout = %d %s, want 204", rr.Code, rr.Body)
	}
	if cleared := grabCookie(t, rr, "session"); cleared.MaxAge >= 0 || cleared.Value != "" {
		t.Errorf("cookie not cleared: %+v", cleared)
	}
	if _, err := st.SessionByHash(hash, fixedNow); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("session row survived logout: err = %v", err)
	}
}

func TestLogoutAllKillsEverySession(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")

	c1 := grabSessionCookie(t, anon(h, "POST", "/api/auth/login", loginBody("bob@example.com", "password123")))
	grabSessionCookie(t, anon(h, "POST", "/api/auth/login", loginBody("bob@example.com", "password123")))
	if sess, _ := st.ListUserSessions(uid); len(sess) != 2 {
		t.Fatalf("want 2 sessions before logout-all, got %d", len(sess))
	}

	rr := doCookie(h, c1, "POST", "/api/auth/logout-all", "")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("logout-all = %d %s, want 204", rr.Code, rr.Body)
	}
	if sess, _ := st.ListUserSessions(uid); len(sess) != 0 {
		t.Errorf("sessions survived logout-all: %d left", len(sess))
	}
}

// ---- telegram login ----

// tgTestToken is the bot token newTelegramAPI injects; the signing helper and
// auth.VerifyInitData must agree on it.
const tgTestToken = "12345:TESTTOKEN"

// signTelegramInitData builds a signed Mini App initData query string with the
// same construction as auth.VerifyInitData: the secret is
// HMAC_SHA256(key="WebAppData", msg=botToken); the check string is the
// key-sorted "k=v" lines (every field except hash) joined by "\n"; and hash is
// the hex HMAC-SHA256 of that check string under the secret.
func signTelegramInitData(fields map[string]string, botToken string) string {
	sk := hmac.New(sha256.New, []byte("WebAppData"))
	sk.Write([]byte(botToken))
	secret := sk.Sum(nil)

	keys := make([]string, 0, len(fields))
	for k := range fields {
		if k == "hash" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, len(keys))
	for i, k := range keys {
		lines[i] = k + "=" + fields[k]
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(strings.Join(lines, "\n")))
	sig := hex.EncodeToString(mac.Sum(nil))

	q := url.Values{}
	for k, v := range fields {
		q.Set(k, v)
	}
	q.Set("hash", sig)
	return q.Encode()
}

// tgInitData signs a fresh Mini App payload for (tgID, username) against the
// fixed test clock.
func tgInitData(tgID int64, username string) string {
	return signTelegramInitData(map[string]string{
		"auth_date": strconv.FormatInt(fixedNow.Add(-time.Minute).Unix(), 10),
		"query_id":  "AAHtest",
		"user":      fmt.Sprintf(`{"id":%d,"username":%q,"first_name":"Neo"}`, tgID, username),
	}, tgTestToken)
}

// tgBody wraps an initData query string in the {init_data: "..."} envelope.
func tgBody(initData string) string {
	return fmt.Sprintf(`{"init_data":%q}`, initData)
}

// newTelegramAPI is newAuthAPI with a Telegram bot token configured.
func newTelegramAPI(t *testing.T) (http.Handler, *store.Store) {
	t.Helper()
	h, st, _ := newAuthAPI(t, func(d *Deps) {
		d.Config.TelegramBotToken = tgTestToken
	})
	return h, st
}

func TestTelegramDisabled503(t *testing.T) {
	h, _, _ := newAuthAPI(t, nil) // no bot token configured
	rr := anon(h, "POST", "/api/auth/telegram", tgBody("auth_date=1&hash=deadbeef"))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("telegram disabled = %d %s, want 503", rr.Code, rr.Body)
	}
	if got := decodeBody[map[string]string](t, rr); got["error"] != "telegram_disabled" {
		t.Errorf("error = %q, want telegram_disabled", got["error"])
	}
}

func TestTelegramLoginNewUser(t *testing.T) {
	h, st := newTelegramAPI(t)

	rr := anon(h, "POST", "/api/auth/telegram", tgBody(tgInitData(555, "neo")))
	if rr.Code != http.StatusOK {
		t.Fatalf("telegram login = %d %s, want 200", rr.Code, rr.Body)
	}
	grabSessionCookie(t, rr)

	user := decodeBody[sessionUserDTO](t, rr)
	if user.Name != "neo" || !user.Telegram.Linked || user.Telegram.Username != "neo" {
		t.Errorf("login DTO = %+v, want name neo + linked telegram", user)
	}

	id, err := st.IdentityByProviderUID("telegram", "555")
	if err != nil {
		t.Fatalf("identity after login: %v", err)
	}
	if id.UserID != user.ID || id.TgUsername != "neo" {
		t.Errorf("identity = %+v, want user_id %q tg_username neo", id, user.ID)
	}
}

func TestTelegramLoginExisting(t *testing.T) {
	h, st := newTelegramAPI(t)
	body := tgBody(tgInitData(777, "trinity"))

	rr := anon(h, "POST", "/api/auth/telegram", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("first login = %d %s", rr.Code, rr.Body)
	}
	first := decodeBody[sessionUserDTO](t, rr).ID

	rr = anon(h, "POST", "/api/auth/telegram", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("second login = %d %s", rr.Code, rr.Body)
	}
	second := decodeBody[sessionUserDTO](t, rr).ID

	if first != second {
		t.Errorf("user id changed across logins: %q then %q", first, second)
	}
	users, err := st.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 2 { // "tester" fixture + the one telegram account
		t.Errorf("user count = %d, want 2 (repeat login must not create a second user)", len(users))
	}
}

func TestTelegramClaimsPending(t *testing.T) {
	h, st := newTelegramAPI(t)

	uid, err := st.CreateUser("Гриша")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if err := st.CreateIdentity(store.Identity{
		ID:          auth.NewIdentityID(),
		UserID:      uid,
		Provider:    "telegram",
		ProviderUID: "pending:llladnooo",
		TgUsername:  "llladnooo",
	}); err != nil {
		t.Fatalf("CreateIdentity pending: %v", err)
	}

	rr := anon(h, "POST", "/api/auth/telegram", tgBody(tgInitData(999, "llladnooo")))
	if rr.Code != http.StatusOK {
		t.Fatalf("claim login = %d %s, want 200", rr.Code, rr.Body)
	}
	user := decodeBody[sessionUserDTO](t, rr)
	if user.ID != uid {
		t.Errorf("logged in as %q, want the pre-seeded %q", user.ID, uid)
	}
	if user.Name != "Гриша" {
		t.Errorf("name = %q, want Гриша", user.Name)
	}

	id, err := st.IdentityByProviderUID("telegram", "999")
	if err != nil {
		t.Fatalf("identity by real tg id: %v", err)
	}
	if id.UserID != uid || id.ProviderUID != "999" {
		t.Errorf("identity = %+v, want user_id %q provider_uid 999", id, uid)
	}
	if _, err := st.IdentityByProviderUID("telegram", "pending:llladnooo"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("pending row survived the claim: err = %v", err)
	}
	if users, _ := st.ListUsers(); len(users) != 2 { // "tester" + "Гриша"
		t.Errorf("user count = %d, want 2 (claim must not create a new user)", len(users))
	}
}

func TestTelegramBadHash401(t *testing.T) {
	h, st := newTelegramAPI(t)

	vals, err := url.ParseQuery(tgInitData(555, "neo"))
	if err != nil {
		t.Fatal(err)
	}
	hsh := []byte(vals.Get("hash"))
	if hsh[0] == '0' {
		hsh[0] = '1'
	} else {
		hsh[0] = '0'
	}
	vals.Set("hash", string(hsh))

	rr := anon(h, "POST", "/api/auth/telegram", tgBody(vals.Encode()))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("bad hash = %d %s, want 401", rr.Code, rr.Body)
	}
	if got := decodeBody[map[string]string](t, rr); got["error"] != "bad_telegram_auth" {
		t.Errorf("error = %q, want bad_telegram_auth", got["error"])
	}
	if n, _ := st.ListUsers(); len(n) != 1 {
		t.Errorf("a rejected login created rows: user count = %d, want 1", len(n))
	}
}
