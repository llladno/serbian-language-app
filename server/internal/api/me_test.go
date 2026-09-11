package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
)

// ---- GET /me ----

func TestGetMeShape(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	cA := authed(t, st, uid)
	authed(t, st, uid) // a second, unnamed session (B)

	rr := doCookie(h, cA, "GET", "/api/me", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("get me = %d %s", rr.Code, rr.Body)
	}
	got := decodeBody[meDTO](t, rr)
	if got.Name != "Bob" || got.Email != "bob@example.com" || !got.EmailVerified {
		t.Errorf("account fields = %+v", got.sessionUserDTO)
	}
	if got.Telegram.Linked {
		t.Errorf("telegram.linked = true, want false")
	}
	if len(got.Sessions) != 2 {
		t.Fatalf("sessions = %+v, want 2", got.Sessions)
	}
	currentCount := 0
	for _, s := range got.Sessions {
		if s.Current {
			currentCount++
		}
		if len(s.ID) != 12 {
			t.Errorf("device id = %q, want a 12-char handle", s.ID)
		}
	}
	if currentCount != 1 {
		t.Errorf("current=true count = %d, want exactly 1 (cookie A's session)", currentCount)
	}
}

// ---- PATCH /me ----

func TestPatchMeRenames(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)

	rr := doCookie(h, c, "PATCH", "/api/me", `{"name":"Роберт"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("patch me = %d %s", rr.Code, rr.Body)
	}
	got := decodeBody[sessionUserDTO](t, rr)
	if got.Name != "Роберт" {
		t.Errorf("name = %q, want Роберт", got.Name)
	}
	row, err := st.UserByID(uid)
	if err != nil || row.Name != "Роберт" {
		t.Errorf("store row = %+v, err %v", row, err)
	}

	// Bad input: blank / too long -> 400, name unchanged.
	for _, bad := range []string{`{"name":"   "}`, `{"name":"` + strings.Repeat("x", 41) + `"}`} {
		if rr := doCookie(h, c, "PATCH", "/api/me", bad); rr.Code != http.StatusBadRequest {
			t.Errorf("patch me %q = %d, want 400", bad, rr.Code)
		}
	}
	row, _ = st.UserByID(uid)
	if row.Name != "Роберт" {
		t.Errorf("bad patch changed the name: %q", row.Name)
	}
}

// ---- POST /me/password ----

func TestChangePasswordWrongCurrent403(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)

	rr := doCookie(h, c, "POST", "/api/me/password", `{"current":"nope-nope","new":"newpass456"}`)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("wrong current = %d %s, want 403", rr.Code, rr.Body)
	}
	if got := decodeBody[map[string]string](t, rr); got["error"] != "wrong_password" {
		t.Errorf("error = %q, want wrong_password", got["error"])
	}
	// Missing current is treated the same as wrong.
	rr = doCookie(h, c, "POST", "/api/me/password", `{"new":"newpass456"}`)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("missing current = %d, want 403", rr.Code)
	}
}

func TestChangePasswordShortNew400(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)

	rr := doCookie(h, c, "POST", "/api/me/password", `{"current":"password123","new":"short"}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("short new password = %d, want 400", rr.Code)
	}
}

func TestChangePasswordHappyKillsOtherSessions(t *testing.T) {
	h, st, sink := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	cA := authed(t, st, uid)
	cB := authed(t, st, uid)

	if rr := doCookie(h, cB, "GET", "/api/auth/session", ""); rr.Code != http.StatusOK {
		t.Fatalf("session B should be live before change: %d", rr.Code)
	}

	rr := doCookie(h, cA, "POST", "/api/me/password", `{"current":"password123","new":"newpass456"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("change password = %d %s", rr.Code, rr.Body)
	}
	if got := decodeBody[map[string]string](t, rr); got["status"] != "ok" {
		t.Errorf("status = %v, want ok", got)
	}

	// Session B is dead; session A (the one that made the change) still works.
	if rr := doCookie(h, cB, "GET", "/api/auth/session", ""); rr.Code != http.StatusUnauthorized {
		t.Errorf("session B survived password change: %d", rr.Code)
	}
	if rr := doCookie(h, cA, "GET", "/api/auth/session", ""); rr.Code != http.StatusOK {
		t.Errorf("session A (the actor) was killed: %d", rr.Code)
	}

	// A password_changed notice went out.
	msgs := sink.all()
	last := msgs[len(msgs)-1]
	if last.To != "bob@example.com" || !strings.Contains(last.Subject, "Пароль изменён") {
		t.Errorf("no password-changed mail: %+v", last)
	}

	// Login with the new password works; the old one no longer does.
	if lr := anon(h, "POST", "/api/auth/login", loginBody("bob@example.com", "password123")); lr.Code != http.StatusUnauthorized {
		t.Errorf("old password still logs in: %d", lr.Code)
	}
	if lr := anon(h, "POST", "/api/auth/login", loginBody("bob@example.com", "newpass456")); lr.Code != http.StatusOK {
		t.Errorf("new password rejected: %d %s", lr.Code, lr.Body)
	}
}

func TestChangePasswordTgOnlyRequiresEmail(t *testing.T) {
	h, st, sink := newAuthAPI(t, func(d *Deps) {
		d.Config.TelegramBotToken = tgTestToken
	})

	rr := anon(h, "POST", "/api/auth/telegram", tgBody(tgInitData(4242, "tgonly")))
	if rr.Code != http.StatusOK {
		t.Fatalf("telegram login = %d %s", rr.Code, rr.Body)
	}
	uid := decodeBody[sessionUserDTO](t, rr).ID
	c := authed(t, st, uid)

	// No email: 400 email_required, no mail, no identity created.
	rr = doCookie(h, c, "POST", "/api/me/password", `{"new":"newpass456"}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("no email = %d %s, want 400", rr.Code, rr.Body)
	}
	if got := decodeBody[map[string]string](t, rr); got["error"] != "email_required" {
		t.Errorf("error = %q, want email_required", got["error"])
	}
	if _, err := st.IdentityForUser(uid, "password"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("a password identity was created despite the 400: err = %v", err)
	}
	if n := len(sink.all()); n != 0 {
		t.Errorf("mail sent despite missing email: %d", n)
	}

	// With email: 200 verify_sent, a verify mail, identity created unverified.
	rr = doCookie(h, c, "POST", "/api/me/password", `{"new":"newpass456","email":"tgonly@example.com"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("with email = %d %s, want 200", rr.Code, rr.Body)
	}
	if got := decodeBody[map[string]string](t, rr); got["status"] != "verify_sent" {
		t.Errorf("status = %v, want verify_sent", got)
	}
	msgs := sink.all()
	if len(msgs) != 1 {
		t.Fatalf("mails sent = %d, want 1", len(msgs))
	}
	if msgs[0].To != "tgonly@example.com" || !strings.Contains(msgs[0].Text, "/api/auth/verify?token=") {
		t.Errorf("not a verify mail: %+v", msgs[0])
	}
	id, err := st.IdentityForUser(uid, "password")
	if err != nil {
		t.Fatalf("password identity: %v", err)
	}
	if id.Email != "tgonly@example.com" || id.EmailVerifiedAt != "" {
		t.Errorf("identity = %+v, want unverified tgonly@example.com", id)
	}
}

// ---- POST /me/link/telegram ----

func TestLinkTelegramConflict409(t *testing.T) {
	h, st := newTelegramAPI(t)
	uidA := registerAndVerify(t, h, st, "a@example.com", "password123", "A")
	uidB := registerAndVerify(t, h, st, "b@example.com", "password123", "B")

	// B links the tg id first.
	cB := authed(t, st, uidB)
	rr := doCookie(h, cB, "POST", "/api/me/link/telegram", tgBody(tgInitData(9001, "taken")))
	if rr.Code != http.StatusOK {
		t.Fatalf("B link = %d %s", rr.Code, rr.Body)
	}

	// A tries to link the same tg id.
	cA := authed(t, st, uidA)
	rr = doCookie(h, cA, "POST", "/api/me/link/telegram", tgBody(tgInitData(9001, "taken")))
	if rr.Code != http.StatusConflict {
		t.Fatalf("A link = %d %s, want 409", rr.Code, rr.Body)
	}
	if got := decodeBody[map[string]string](t, rr); got["error"] != "telegram_taken" {
		t.Errorf("error = %q, want telegram_taken", got["error"])
	}
}

func TestLinkTelegramHappy(t *testing.T) {
	h, st := newTelegramAPI(t)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)

	rr := doCookie(h, c, "POST", "/api/me/link/telegram", tgBody(tgInitData(1234, "bobby")))
	if rr.Code != http.StatusOK {
		t.Fatalf("link = %d %s", rr.Code, rr.Body)
	}
	got := decodeBody[sessionUserDTO](t, rr)
	if !got.Telegram.Linked || got.Telegram.Username != "bobby" {
		t.Errorf("link response = %+v", got.Telegram)
	}

	// Reflected on the next /me too.
	me := decodeBody[meDTO](t, doCookie(h, c, "GET", "/api/me", ""))
	if !me.Telegram.Linked || me.Telegram.Username != "bobby" {
		t.Errorf("me after link = %+v", me.Telegram)
	}

	id, err := st.IdentityByProviderUID("telegram", "1234")
	if err != nil || id.UserID != uid {
		t.Errorf("identity = %+v, err %v", id, err)
	}

	// Linking again (same account, same tg id) is idempotent, not a conflict.
	rr = doCookie(h, c, "POST", "/api/me/link/telegram", tgBody(tgInitData(1234, "bobby")))
	if rr.Code != http.StatusOK {
		t.Errorf("re-link = %d, want 200 (idempotent)", rr.Code)
	}
}

func TestLinkTelegramDisabled503(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil) // no bot token configured
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)

	rr := doCookie(h, c, "POST", "/api/me/link/telegram", tgBody("auth_date=1&hash=deadbeef"))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("link disabled = %d, want 503", rr.Code)
	}
}

// ---- DELETE /me/telegram ----

func TestUnlinkTelegramOnlyMethod409(t *testing.T) {
	h, st := newTelegramAPI(t)

	rr := anon(h, "POST", "/api/auth/telegram", tgBody(tgInitData(5551, "onlytg")))
	if rr.Code != http.StatusOK {
		t.Fatalf("telegram login = %d %s", rr.Code, rr.Body)
	}
	uid := decodeBody[sessionUserDTO](t, rr).ID
	c := authed(t, st, uid)

	rr = doCookie(h, c, "DELETE", "/api/me/telegram", "")
	if rr.Code != http.StatusConflict {
		t.Fatalf("unlink only method = %d %s, want 409", rr.Code, rr.Body)
	}
	if got := decodeBody[map[string]string](t, rr); got["error"] != "only_login_method" {
		t.Errorf("error = %q, want only_login_method", got["error"])
	}
	if _, err := st.IdentityForUser(uid, "telegram"); err != nil {
		t.Errorf("telegram identity removed despite the 409: %v", err)
	}
}

func TestUnlinkTelegramHappy(t *testing.T) {
	h, st := newTelegramAPI(t)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)
	if rr := doCookie(h, c, "POST", "/api/me/link/telegram", tgBody(tgInitData(2222, "bobby"))); rr.Code != http.StatusOK {
		t.Fatalf("link = %d %s", rr.Code, rr.Body)
	}

	rr := doCookie(h, c, "DELETE", "/api/me/telegram", "")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("unlink = %d %s, want 204", rr.Code, rr.Body)
	}
	if _, err := st.IdentityForUser(uid, "telegram"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("telegram identity survived unlink: err = %v", err)
	}
	// Password identity is untouched.
	if _, err := st.IdentityForUser(uid, "password"); err != nil {
		t.Errorf("password identity gone after unlink: %v", err)
	}
}

func TestUnlinkTelegramNotLinked404(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)

	rr := doCookie(h, c, "DELETE", "/api/me/telegram", "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("unlink when not linked = %d, want 404", rr.Code)
	}
}

// ---- DELETE /me/sessions/{id} ----

func TestDeleteSessionByHandle(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	cA := authed(t, st, uid)
	cB := authed(t, st, uid)

	me := decodeBody[meDTO](t, doCookie(h, cA, "GET", "/api/me", ""))
	var otherID string
	for _, s := range me.Sessions {
		if !s.Current {
			otherID = s.ID
		}
	}
	if otherID == "" {
		t.Fatalf("no non-current session in %+v", me.Sessions)
	}

	rr := doCookie(h, cA, "DELETE", "/api/me/sessions/"+otherID, "")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete session = %d %s, want 204", rr.Code, rr.Body)
	}
	// The deleted session (B) is dead; the actor's own (A) still works.
	if rr := doCookie(h, cB, "GET", "/api/auth/session", ""); rr.Code != http.StatusUnauthorized {
		t.Errorf("deleted session still live: %d", rr.Code)
	}
	if rr := doCookie(h, cA, "GET", "/api/auth/session", ""); rr.Code != http.StatusOK {
		t.Errorf("actor's own session was killed: %d", rr.Code)
	}
}

// TestDeleteSessionSelfClearsCookie covers deleting your OWN current session
// by handle: it must succeed and clear this device's cookie too.
func TestDeleteSessionSelfClearsCookie(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)

	me := decodeBody[meDTO](t, doCookie(h, c, "GET", "/api/me", ""))
	var selfID string
	for _, s := range me.Sessions {
		if s.Current {
			selfID = s.ID
		}
	}
	if selfID == "" {
		t.Fatalf("no current session in %+v", me.Sessions)
	}

	rr := doCookie(h, c, "DELETE", "/api/me/sessions/"+selfID, "")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete own session = %d %s, want 204", rr.Code, rr.Body)
	}
	if cleared := grabCookie(t, rr, "session"); cleared.MaxAge >= 0 {
		t.Errorf("cookie not cleared: %+v", cleared)
	}
	if rr := doCookie(h, c, "GET", "/api/auth/session", ""); rr.Code != http.StatusUnauthorized {
		t.Errorf("own session survived self-delete: %d", rr.Code)
	}
}

func TestDeleteSessionUnknown404(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)

	rr := doCookie(h, c, "DELETE", "/api/me/sessions/ffffffffffff", "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("unknown session = %d, want 404", rr.Code)
	}
}

// ---- DELETE /me ----

func TestDeleteMeWrongPassword403(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)

	rr := doCookie(h, c, "DELETE", "/api/me", `{"password":"nope-nope"}`)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("wrong password = %d %s, want 403", rr.Code, rr.Body)
	}
	if got := decodeBody[map[string]string](t, rr); got["error"] != "wrong_password" {
		t.Errorf("error = %q, want wrong_password", got["error"])
	}
	if _, err := st.UserByID(uid); err != nil {
		t.Errorf("user deleted despite wrong password: %v", err)
	}

	// Empty body (no password) is treated the same as wrong.
	rr = doCookie(h, c, "DELETE", "/api/me", "")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("empty body = %d, want 403", rr.Code)
	}
}

func TestDeleteMeHappy(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)
	// Seed some state so we can confirm it is gone afterward.
	if rr := doCookie(h, c, "POST", "/api/lessons/01/complete", ""); rr.Code != http.StatusOK {
		t.Fatalf("seed complete = %d", rr.Code)
	}

	rr := doCookie(h, c, "DELETE", "/api/me", `{"password":"password123"}`)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete me = %d %s, want 204", rr.Code, rr.Body)
	}
	if cleared := grabCookie(t, rr, "session"); cleared.MaxAge >= 0 || cleared.Value != "" {
		t.Errorf("cookie not cleared: %+v", cleared)
	}
	if _, err := st.UserByID(uid); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("user row survived deletion: err = %v", err)
	}
	if _, err := st.IdentityForUser(uid, "password"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("identity survived deletion: err = %v", err)
	}
	if sess, _ := st.ListUserSessions(uid); len(sess) != 0 {
		t.Errorf("sessions survived deletion: %d left", len(sess))
	}
	if rr := doCookie(h, c, "GET", "/api/auth/session", ""); rr.Code != http.StatusUnauthorized {
		t.Errorf("deleted account's session still resolves: %d", rr.Code)
	}
}

// TestDeleteMeTgOnlyReauthWindow covers the Telegram-only self-delete path: no
// password to check, so the session's own age stands in for reauth.
func TestDeleteMeTgOnlyReauthWindow(t *testing.T) {
	h, st := newTelegramAPI(t)

	rr := anon(h, "POST", "/api/auth/telegram", tgBody(tgInitData(7777, "oldone")))
	if rr.Code != http.StatusOK {
		t.Fatalf("telegram login = %d %s", rr.Code, rr.Body)
	}
	uid := decodeBody[sessionUserDTO](t, rr).ID

	// A session seeded with an old created_at -> stale, reauth required.
	raw, hash, err := auth.NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	old := fixedNow.Add(-10 * time.Minute)
	if err := st.CreateSession(hash, uid, "test-agent", old, fixedNow.Add(720*time.Hour)); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	staleCookie := &http.Cookie{Name: "session", Value: raw}

	rr = doCookie(h, staleCookie, "DELETE", "/api/me", "")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("stale session delete = %d %s, want 403", rr.Code, rr.Body)
	}
	if got := decodeBody[map[string]string](t, rr); got["error"] != "reauth_required" {
		t.Errorf("error = %q, want reauth_required", got["error"])
	}
	if _, err := st.UserByID(uid); err != nil {
		t.Errorf("user deleted despite stale session: %v", err)
	}

	// A fresh session (created "now") succeeds with no password at all.
	freshCookie := authed(t, st, uid)
	rr = doCookie(h, freshCookie, "DELETE", "/api/me", "")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("fresh session delete = %d %s, want 204", rr.Code, rr.Body)
	}
	if _, err := st.UserByID(uid); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("user survived fresh-session delete: err = %v", err)
	}
}
