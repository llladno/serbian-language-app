package api

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	netmail "net/mail"
	"strings"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
	"github.com/grisha/serbian-app/server/internal/mail"
	"github.com/grisha/serbian-app/server/internal/store"
)

// This file holds the /api/auth/* handlers. register / login / logout /
// logout-all / session are implemented here (Task 12); the remaining bodies
// are 501 stubs replaced by Tasks 13-16.

// verifyTTL is how long an email-verification token stays usable.
const verifyTTL = 24 * time.Hour

// resetTTL is how long a password-reset token stays usable. It is short on
// purpose: a reset link is high-value and a legitimate user acts on it at once.
const resetTTL = time.Hour

// badCredentials is the deliberately vague message returned for every login
// failure (unknown email, wrong password, soft lock) so the response never
// tells an attacker which accounts exist.
const badCredentials = "неверная почта или пароль"

// issueSession mints a fresh session for userID: a new opaque token, a
// sessions row keyed by its hash, and the session cookie carrying the raw
// token. The row and the cookie share sessionTTL so they expire together.
func (h handlers) issueSession(w http.ResponseWriter, r *http.Request, userID string) error {
	raw, hash, err := auth.NewToken()
	if err != nil {
		return fmt.Errorf("new session token: %w", err)
	}
	now := h.Now()
	if err := h.Store.CreateSession(hash, userID, r.UserAgent(), now, now.Add(sessionTTL)); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	h.setSessionCookie(w, raw, sessionTTL)
	return nil
}

// clientIP is the best available caller address for per-IP rate-limit keys.
// Behind Traefik/Dokploy the proxy sets X-Real-Ip to the real peer and
// appends that peer to the RIGHT of any inbound X-Forwarded-For, so a client
// cannot pin the key by sending its own X-Forwarded-For: the right-most entry
// is the hop Traefik actually saw. Without either header (tests, direct
// connections) it falls back to RemoteAddr's host part.
func clientIP(r *http.Request) string {
	if xrip := strings.TrimSpace(r.Header.Get("X-Real-Ip")); xrip != "" {
		return xrip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if last := strings.TrimSpace(parts[len(parts)-1]); last != "" {
			return last
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// summaryToDTO maps the denormalized request summary onto the wire shape
// shared by GET /api/auth/session and GET /api/me.
func summaryToDTO(userID string, s userSummary) sessionUserDTO {
	dto := sessionUserDTO{
		ID:            userID,
		Name:          s.Name,
		Email:         s.Email,
		EmailVerified: s.EmailVerified,
	}
	dto.Telegram.Linked = s.TelegramLinked
	dto.Telegram.Username = s.TelegramUsername
	return dto
}

// register accepts {email, password, name}. It validates the shape
// synchronously and answers 200 {"status":"ok"} before doing any account
// lookup, so a caller cannot tell a taken address from a free one by timing
// the response. The real work — send "already registered" mail, or create the
// account and send a verification link — runs in h.Async after the reply.
func (h handlers) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}

	addr, err := netmail.ParseAddress(req.Email)
	if err != nil {
		fail(w, http.StatusBadRequest, "invalid email")
		return
	}
	if n := len([]rune(req.Password)); n < 8 || n > 128 {
		fail(w, http.StatusBadRequest, "password must be 8 to 128 characters")
		return
	}
	name := store.NormalizeName(req.Name)
	if name == "" || len([]rune(name)) > 40 {
		fail(w, http.StatusBadRequest, "name must be 1 to 40 characters")
		return
	}

	email := addr.Address
	uid := strings.ToLower(email)
	ip := clientIP(r)
	// Evaluate the throttle now (it consumes a token); act on it in the
	// goroutine. A blocked caller still gets the same 200.
	slowOK := allow(h.Slow, "ip:"+ip) && allow(h.Slow, "email:"+uid)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	h.Async(func() {
		if !slowOK {
			return
		}
		switch _, err := h.Store.IdentityByProviderUID("password", uid); {
		case err == nil:
			subject, text, html := mail.RenderAlreadyRegistered(h.Config.AppBaseURL, name)
			h.SendMail(email, subject, text, html)
			return
		case !errors.Is(err, sql.ErrNoRows):
			log.Printf("register: lookup identity: %v", err)
			return
		}

		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			log.Printf("register: hash password: %v", err)
			return
		}
		userID, err := h.Store.CreateUser(name)
		if err != nil {
			log.Printf("register: create user: %v", err)
			return
		}
		identityID := auth.NewIdentityID()
		if err := h.Store.CreateIdentity(store.Identity{
			ID:           identityID,
			UserID:       userID,
			Provider:     "password",
			ProviderUID:  uid,
			Email:        email,
			PasswordHash: hash,
		}); err != nil {
			// A racing double-submit trips UNIQUE(provider, provider_uid)
			// here; the earlier CreateUser row is then orphaned. Rare, and
			// the Slow limiter caps the blast radius — log, do not panic.
			log.Printf("register: create identity: %v", err)
			return
		}
		if err := h.sendVerifyEmail(identityID, email, name); err != nil {
			log.Printf("register: send verify mail: %v", err)
		}
	})
}

// sendVerifyEmail mints a verify token for the identity and mails the link.
func (h handlers) sendVerifyEmail(identityID, email, name string) error {
	raw, tokenHash, err := auth.NewToken()
	if err != nil {
		return fmt.Errorf("new verify token: %w", err)
	}
	if err := h.Store.CreateEmailToken(tokenHash, identityID, "verify", h.Now(), h.Now().Add(verifyTTL)); err != nil {
		return fmt.Errorf("create verify token: %w", err)
	}
	subject, text, html := mail.RenderVerify(h.Config.AppBaseURL, raw, name)
	h.SendMail(email, subject, text, html)
	return nil
}

// login accepts {email, password}. Every failure returns the same generic
// 401 body; a soft lock (too many recent failures) short-circuits before the
// password is even checked. An unverified account gets 403 email_unverified
// plus a fresh verification mail. Success sets the session cookie and returns
// the account.
func (h handlers) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	uid := strings.ToLower(strings.TrimSpace(req.Email))

	if locked(h.Fails, uid) {
		fail(w, http.StatusUnauthorized, badCredentials)
		return
	}
	ip := clientIP(r)
	if !allow(h.Login, "ip:"+ip) || !allow(h.LoginEmail, uid) {
		fail(w, http.StatusTooManyRequests, "too many attempts")
		return
	}

	id, err := h.Store.IdentityByProviderUID("password", uid)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		noteFail(h.Fails, uid)
		fail(w, http.StatusUnauthorized, badCredentials)
		return
	case err != nil:
		log.Printf("login: lookup identity: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}

	if !auth.VerifyPassword(id.PasswordHash, req.Password) {
		noteFail(h.Fails, uid)
		fail(w, http.StatusUnauthorized, badCredentials)
		return
	}

	if id.EmailVerifiedAt == "" {
		// Correct password, but the address was never confirmed. The
		// credentials are good, so clear the fail count, resend the link,
		// and tell the client to route to the "check your email" screen.
		clearFails(h.Fails, uid)
		name := id.Email
		if row, err := h.Store.UserByID(id.UserID); err == nil && row.Name != "" {
			name = row.Name
		}
		identityID, to := id.ID, id.Email
		h.Async(func() {
			if err := h.sendVerifyEmail(identityID, to, name); err != nil {
				log.Printf("login: resend verify mail: %v", err)
			}
		})
		fail(w, http.StatusForbidden, "email_unverified")
		return
	}

	clearFails(h.Fails, uid)
	if err := h.issueSession(w, r, id.UserID); err != nil {
		log.Printf("login: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	sum, err := h.summaryFor(id.UserID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, summaryToDTO(id.UserID, sum))
}

// logout drops the current session and clears the cookie. It is mounted on
// the public mux (no requireAuth), so it reads the cookie directly rather
// than through authFrom and is a no-op when there is nothing to clear.
func (h handlers) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(h.Config.CookieName()); err == nil && c.Value != "" {
		if err := h.Store.DeleteSession(auth.HashToken(c.Value)); err != nil {
			log.Printf("logout: delete session: %v", err)
		}
	}
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// logoutAll deletes every session of the caller and clears this device's
// cookie, forcing a fresh login everywhere. It sits behind requireAuth.
func (h handlers) logoutAll(w http.ResponseWriter, r *http.Request) {
	ac, _ := authFrom(r)
	if err := h.Store.DeleteUserSessions(ac.UserID); err != nil {
		// Best-effort, like logout: a failed delete still clears this
		// device's cookie and answers 204 rather than a false 500.
		log.Printf("logout-all: delete sessions: %v", err)
	}
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// currentSession returns the authenticated account. It sits behind
// requireAuth, so a missing or dead session is already a 401 by the time this
// runs. The legacy X-User bridge resolves a user id but no Summary, so fall
// back to a name lookup in that case.
func (h handlers) currentSession(w http.ResponseWriter, r *http.Request) {
	ac, _ := authFrom(r)
	sum := ac.Summary
	if sum.Name == "" {
		if row, err := h.Store.UserByID(ac.UserID); err == nil {
			sum.Name = row.Name
		}
	}
	writeJSON(w, http.StatusOK, summaryToDTO(ac.UserID, sum))
}

// verifyEmail handles GET /api/auth/verify?token=. It consumes the token, marks
// the address confirmed, and redirects to the SPA either way — a browser opened
// this link, so it must land on a page, never a JSON error. Every failure path
// (missing token, unknown/used/expired token, or a real store error) redirects
// to /verify?err=1 with an identical 302: the endpoint never renders a body and
// never logs the token, so a leaked link in a referrer or a log stays inert.
func (h handlers) verifyEmail(w http.ResponseWriter, r *http.Request) {
	redirect := func(q string) {
		http.Redirect(w, r, h.Config.AppBaseURL+"/verify?"+q, http.StatusFound)
	}
	token := r.URL.Query().Get("token")
	if token == "" {
		redirect("err=1")
		return
	}
	identityID, err := h.Store.UseEmailToken(auth.HashToken(token), "verify", h.Now())
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("verify: use token: %v", err)
		}
		redirect("err=1")
		return
	}
	if err := h.Store.SetEmailVerified(identityID, h.Now()); err != nil {
		// The token is spent; failing to stamp the column is a server fault,
		// but the user cannot retry with this link. Log and still land them on
		// the success page — the next login resends a verify mail anyway.
		log.Printf("verify: set verified: %v", err)
	}
	redirect("ok=1")
}

// resendVerification handles POST {email}. It answers 200 {"status":"ok"}
// unconditionally and before any lookup, so the response cannot be used to
// probe which addresses have an account. The real work — only for an existing,
// still-unverified password identity — mints a fresh verify token and mails it,
// after the response, in h.Async.
func (h handlers) resendVerification(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	uid := strings.ToLower(strings.TrimSpace(req.Email))
	slowOK := allow(h.Slow, "ip:"+clientIP(r)) && allow(h.Slow, "email:"+uid)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	h.Async(func() {
		if !slowOK {
			return
		}
		id, err := h.Store.IdentityByProviderUID("password", uid)
		if err != nil {
			return // unknown address (or a store blip) — stay silent
		}
		if id.EmailVerifiedAt != "" {
			return // already confirmed, nothing to resend
		}
		name := ""
		if row, err := h.Store.UserByID(id.UserID); err == nil {
			name = row.Name
		}
		if err := h.sendVerifyEmail(id.ID, id.Email, name); err != nil {
			log.Printf("resend-verification: send verify mail: %v", err)
		}
	})
}

// forgotPassword handles POST {email}. Like register/resend it answers 200
// {"status":"ok"} immediately and uniformly — the sync path never branches on
// whether the account exists — then, in h.Async, mints a short-lived "reset"
// token for an existing password identity and mails the link.
func (h handlers) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	uid := strings.ToLower(strings.TrimSpace(req.Email))
	slowOK := allow(h.Slow, "ip:"+clientIP(r)) && allow(h.Slow, "email:"+uid)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	h.Async(func() {
		if !slowOK {
			return
		}
		id, err := h.Store.IdentityByProviderUID("password", uid)
		if err != nil {
			return
		}
		name := ""
		if row, err := h.Store.UserByID(id.UserID); err == nil {
			name = row.Name
		}
		raw, tokenHash, err := auth.NewToken()
		if err != nil {
			log.Printf("forgot-password: new token: %v", err)
			return
		}
		if err := h.Store.CreateEmailToken(tokenHash, id.ID, "reset", h.Now(), h.Now().Add(resetTTL)); err != nil {
			log.Printf("forgot-password: create token: %v", err)
			return
		}
		subject, text, html := mail.RenderReset(h.Config.AppBaseURL, raw, name)
		h.SendMail(id.Email, subject, text, html)
	})
}

// resetPassword handles POST {token, password}. It validates the new password
// shape first (cheap, and so a bad password never burns the token), consumes
// the "reset" token, writes the new hash, kills every existing session for the
// account, then autologins this device and returns the account. A security
// notice is mailed after the response. The token is never logged and never
// echoed in an error body.
func (h handlers) resetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	if n := len([]rune(req.Password)); n < 8 || n > 128 {
		fail(w, http.StatusBadRequest, "password must be 8-128 characters")
		return
	}
	if req.Token == "" {
		fail(w, http.StatusBadRequest, "invalid_token")
		return
	}

	identityID, err := h.Store.UseEmailToken(auth.HashToken(req.Token), "reset", h.Now())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fail(w, http.StatusBadRequest, "invalid_token")
			return
		}
		log.Printf("reset-password: use token: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("reset-password: hash password: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := h.Store.SetPasswordHash(identityID, hash); err != nil {
		log.Printf("reset-password: set password hash: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := h.Store.DeleteIdentityTokens(identityID, "reset"); err != nil {
		// Best-effort cleanup of any sibling reset tokens; the one just used is
		// already spent, so a failure here is not fatal.
		log.Printf("reset-password: delete reset tokens: %v", err)
	}

	idn, err := h.Store.IdentityByID(identityID)
	if err != nil {
		log.Printf("reset-password: identity by id: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := h.Store.DeleteUserSessions(idn.UserID); err != nil {
		// A stale session outliving a password reset is the thing to avoid, but
		// the caller's request already succeeded; log and carry on.
		log.Printf("reset-password: delete sessions: %v", err)
	}
	if err := h.issueSession(w, r, idn.UserID); err != nil {
		log.Printf("reset-password: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}

	name := ""
	if row, err := h.Store.UserByID(idn.UserID); err == nil {
		name = row.Name
	}
	to := idn.Email
	h.Async(func() {
		subject, text, html := mail.RenderPasswordChanged(h.Config.AppBaseURL, name)
		h.SendMail(to, subject, text, html)
	})

	sum, err := h.summaryFor(idn.UserID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, summaryToDTO(idn.UserID, sum))
}

func (h handlers) telegramLogin(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}
