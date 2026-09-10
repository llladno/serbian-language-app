package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
	"github.com/grisha/serbian-app/server/internal/ratelimit"
	"github.com/grisha/serbian-app/server/internal/store"
)

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
