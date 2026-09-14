package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	netmail "net/mail"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
	"github.com/grisha/serbian-app/server/internal/mail"
	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
)

// This file holds the /api/auth/* handlers: register / login / logout /
// logout-all / session (Task 12), verify / resend / forgot / reset (Tasks
// 13-14) and the Telegram sign-in (Task 15). verifyTelegramPayload is also
// reused by linkTelegram in me.go (Task 16).

// verifyTTL is how long an email-verification token stays usable.
const verifyTTL = 24 * time.Hour

// resetTTL is how long a password-reset token stays usable. It is short on
// purpose: a reset link is high-value and a legitimate user acts on it at once.
const resetTTL = time.Hour

// badCredentials is the deliberately vague message returned for every login
// failure (unknown email, wrong password, soft lock) so the response never
// tells an attacker which accounts exist.
const badCredentials = "неверная почта или пароль"

// dummyPasswordHash backs the timing-equalization verify in login's
// unknown-email branch. A bcrypt verify costs ~150-200ms; returning early when
// the identity does not exist made "unknown address" answer ~400x faster than
// "known address, wrong password", which is a trivially measurable enumeration
// oracle over the network — and it undermines the uniform 200s that register /
// resend / forgot go out of their way to produce. Built lazily and once, so
// process start pays nothing and every caller shares the same cost.
var (
	dummyHashOnce sync.Once
	dummyHash     string
)

// equalizeLoginTiming burns roughly one bcrypt verify. It is called on the
// branches that have no stored hash to check, so every login failure costs the
// same wall-clock time regardless of whether the address exists.
func equalizeLoginTiming(password string) {
	dummyHashOnce.Do(func() {
		// The only way this fails is bcrypt rejecting the cost, which the
		// package's own constant cannot do. Log it if the impossible happens:
		// an empty hash makes CompareHashAndPassword bail early and the
		// equalization silently stops working, which is worth a line.
		h, err := auth.HashPassword("dummy-password-for-timing-equalization")
		if err != nil {
			log.Printf("login: build timing-equalization hash: %v", err)
			return
		}
		dummyHash = h
	})
	_ = auth.VerifyPassword(dummyHash, password)
}

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
		switch existing, err := h.Store.IdentityByProviderUID("password", uid); {
		case err == nil:
			// This mail goes to a THIRD PARTY's inbox — the address already has
			// an account, and whoever posted this form is not necessarily its
			// owner. Greet them by their own stored name, never by the name from
			// the request body: that would let anyone inject ~40 characters of
			// chosen text into someone else's mailbox.
			holder := ""
			if row, err := h.Store.UserByID(existing.UserID); err == nil {
				holder = row.Name
			}
			subject, text, html := mail.RenderAlreadyRegistered(h.Config.AppBaseURL, holder)
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
		// Spend a bcrypt verify we will throw away, so an unknown address costs
		// the same wall-clock time as a known one with a wrong password.
		equalizeLoginTiming(req.Password)
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
			// Unknown address is the common case (sql.ErrNoRows) — stay
			// silent. A real store fault is worth a server-side line; it
			// carries no email and costs nothing in enumeration terms.
			if !errors.Is(err, sql.ErrNoRows) {
				log.Printf("resend: lookup identity: %v", err)
			}
			return
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
			// sql.ErrNoRows (no account for this address) is the silent,
			// expected path; a genuine store fault is logged server-side
			// only — no email, no enumeration signal.
			if !errors.Is(err, sql.ErrNoRows) {
				log.Printf("forgot: lookup identity: %v", err)
			}
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
// the "reset" token, kills every existing session for the account, writes the
// new hash, marks the address verified (finishing an emailed link proves inbox
// control), then autologins this device and returns the account. Sessions are
// purged before the password is touched so a failed purge fails the whole
// request instead of silently leaving a stale session alive. A security notice
// is mailed after the response. The token is never logged and never echoed in
// an error body.
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

	idn, err := h.Store.IdentityByID(identityID)
	if err != nil {
		log.Printf("reset-password: identity by id: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Kill every existing session before touching the password. Nothing is
	// mutated yet, so this is safe to retry; and a failed purge must not be
	// swallowed — a stale session outliving a reset is exactly the guarantee
	// this endpoint owes the user, so a failure here is a 500, not a 200.
	if err := h.Store.DeleteUserSessions(idn.UserID); err != nil {
		log.Printf("reset-password: delete sessions: %v", err)
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
	if err := h.Store.SetEmailVerified(identityID, h.Now()); err != nil {
		// Completing an emailed reset link proves control of the inbox, so the
		// address is verified now — a user who reset without ever verifying
		// shouldn't then be bounced to /verify. Non-fatal, like verifyEmail:
		// log and carry on.
		log.Printf("reset: set verified: %v", err)
	}
	if err := h.Store.DeleteIdentityTokens(identityID, "reset"); err != nil {
		// Best-effort cleanup of any sibling reset tokens; the one just used is
		// already spent, so a failure here is not fatal.
		log.Printf("reset-password: delete reset tokens: %v", err)
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

// telegramMaxAge is how old a Telegram auth_date may be before a sign-in is
// rejected as stale. Telegram's own guidance for Mini Apps is 24h.
const telegramMaxAge = 24 * time.Hour

// verifyTelegramPayload verifies a raw Telegram Mini App envelope
// ({"init_data": "<query string>"}) and returns the authenticated Telegram
// user. Shared by telegramLogin (sign-in) and linkTelegram (linking an
// existing account to a Telegram id); the bot token and the raw initData are
// never logged.
//
// The Login Widget payload shape this used to also accept is gone: the
// widget button itself was replaced by the /start deep-link flow (see
// telegramLoginStart/telegramWebhook), so nothing sends that shape anymore.
func verifyTelegramPayload(body []byte, botToken string, now time.Time) (auth.TelegramUser, error) {
	var envelope struct {
		InitData string `json:"init_data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || strings.TrimSpace(envelope.InitData) == "" {
		return auth.TelegramUser{}, auth.ErrMalformed
	}
	return auth.VerifyInitData(envelope.InitData, botToken, now, telegramMaxAge)
}

// resolveTelegramLogin finds or creates the account a verified Telegram
// identity belongs to, in order: an existing telegram identity for this tg id
// logs straight in; a "pending:<username>" row seeded by a migration is
// rewritten to the real id and claimed; otherwise a brand-new account is
// created. Shared by telegramLogin (Mini App, HTTP-synchronous) and the
// /start webhook flow (telegramWebhook, resolved out-of-band and picked up by
// a poll) — this function only resolves the account, it never touches the
// response writer or issues a session.
func (h handlers) resolveTelegramLogin(u auth.TelegramUser) (userID string, err error) {
	tgID := strconv.FormatInt(u.ID, 10)

	// 1. Known telegram identity — straight login.
	id, err := h.Store.IdentityByProviderUID("telegram", tgID)
	switch {
	case err == nil:
		return id.UserID, nil
	case !errors.Is(err, sql.ErrNoRows):
		return "", fmt.Errorf("lookup identity: %w", err)
	}

	// 2. A pending row seeded for this username — rewrite it to the real id.
	// Telegram usernames are case-insensitive and the migration-003 seed keys
	// (pending:llladnooo, pending:alinsssk) are lowercase, but a verified
	// payload may carry any case — lowercase the lookup key so the claim hits.
	// An account with no username at all must skip this entirely: the lookup key
	// would be a bare "pending:" prefix, which matches nothing today but is a
	// claim attempt that should never be made.
	matched := false
	if u.Username != "" {
		matched, err = h.Store.AttachPendingTelegram(strings.ToLower(u.Username), tgID)
		if err != nil {
			return "", fmt.Errorf("attach pending: %w", err)
		}
	}
	if matched {
		id, err := h.Store.IdentityByProviderUID("telegram", tgID)
		if err != nil {
			return "", fmt.Errorf("identity after claim: %w", err)
		}
		return id.UserID, nil
	}

	// 3. First contact — create the account and its telegram identity.
	name := u.Username
	if name == "" {
		name = u.FirstName
	}
	if name == "" {
		name = "tg" + tgID
	}
	name = store.NormalizeName(name)
	// NormalizeName can still leave a name store.CreateUser rejects (empty after
	// trimming a whitespace-only first_name, or >40 runes) — that would be an
	// unrecoverable failure on every future login. Fall back to the stable tg id.
	if name == "" || len([]rune(name)) > 40 {
		name = "tg" + tgID
	}

	uid, err := h.Store.CreateUser(name)
	if err != nil {
		return "", fmt.Errorf("create user: %w", err)
	}
	if err := h.Store.CreateIdentity(store.Identity{
		ID:          auth.NewIdentityID(),
		UserID:      uid,
		Provider:    "telegram",
		ProviderUID: tgID,
		TgUsername:  u.Username,
	}); err != nil {
		return "", fmt.Errorf("create identity: %w", err)
	}
	return uid, nil
}

// errTelegramTaken is resolveTelegramLink's sentinel for "this Telegram
// account already belongs to a different user" — distinct from a plain store
// error so callers can map it to 409 instead of 500.
var errTelegramTaken = errors.New("telegram_taken")

// resolveTelegramLink attaches u's telegram identity to callerUserID: a no-op
// if already linked to that same caller, errTelegramTaken if linked to
// someone else, otherwise creates the identity. Shared by linkTelegram (Mini
// App, HTTP-synchronous) and the /start webhook flow.
func (h handlers) resolveTelegramLink(callerUserID string, u auth.TelegramUser) error {
	tgID := strconv.FormatInt(u.ID, 10)
	existing, err := h.Store.IdentityByProviderUID("telegram", tgID)
	switch {
	case err == nil && existing.UserID == callerUserID:
		return nil
	case err == nil:
		return errTelegramTaken
	case errors.Is(err, sql.ErrNoRows):
		return h.Store.CreateIdentity(store.Identity{
			ID:          auth.NewIdentityID(),
			UserID:      callerUserID,
			Provider:    "telegram",
			ProviderUID: tgID,
			TgUsername:  u.Username,
		})
	default:
		return fmt.Errorf("lookup identity: %w", err)
	}
}

// telegramLogin handles POST /api/auth/telegram (Mini App initData). Every
// success mints a session and returns the account view.
func (h handlers) telegramLogin(w http.ResponseWriter, r *http.Request) {
	if !h.Config.TelegramEnabled() {
		fail(w, http.StatusServiceUnavailable, "telegram_disabled")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}

	u, err := verifyTelegramPayload(body, h.Config.TelegramBotToken, h.Now())
	if err != nil {
		// verifyTelegramPayload only ever returns auth's sentinel errors; every
		// one of them means the same thing to the caller — fail closed.
		fail(w, http.StatusUnauthorized, "bad_telegram_auth")
		return
	}

	userID, err := h.resolveTelegramLogin(u)
	if err != nil {
		log.Printf("telegram login: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.finishTelegramLogin(w, r, userID)
}

// finishTelegramLogin mints a session for userID and writes the 200 account
// view. Shared by the existing-identity, claimed-pending and new-account
// branches of telegramLogin.
func (h handlers) finishTelegramLogin(w http.ResponseWriter, r *http.Request, userID string) {
	if err := h.issueSession(w, r, userID); err != nil {
		log.Printf("telegram login: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	sum, err := h.summaryFor(userID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, summaryToDTO(userID, sum))
}

// telegramStartFor generates a one-time /start deep-link token bound to
// callerUserID ("" for a login attempt from telegramLoginStart, an
// authenticated user's own id for a link attempt from telegramLinkStart in
// me.go) and returns the t.me URL the frontend opens in a new tab.
func (h handlers) telegramStartFor(w http.ResponseWriter, callerUserID string) {
	if !h.Config.TelegramEnabled() || h.TelegramBotUsername == "" {
		fail(w, http.StatusServiceUnavailable, "telegram_disabled")
		return
	}
	raw, err := h.TelegramPending.Create(callerUserID)
	if err != nil {
		log.Printf("telegram start: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"url":   "https://t.me/" + h.TelegramBotUsername + "?start=" + raw,
		"token": raw,
	})
}

// telegramLoginStart handles POST /api/auth/telegram/start (public — this is
// how a caller gets a login token in the first place, so it cannot itself
// require a session).
func (h handlers) telegramLoginStart(w http.ResponseWriter, r *http.Request) {
	h.telegramStartFor(w, "")
}

// telegramPoll handles GET /api/auth/telegram/poll?token=... (public — the
// token itself is the credential, the same trust model as an email verify
// link). A PendingDone result for a *login* token issues the session right
// here, on this response: this is an ordinary request from the waiting
// browser tab, not the webhook, so it is the one place in the whole flow that
// can actually set a cookie for that tab. A PendingDone result for a *link*
// token does not touch the session — the caller already has one, and this
// just confirms the link went through.
func (h handlers) telegramPoll(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		fail(w, http.StatusBadRequest, "missing token")
		return
	}
	res := h.TelegramPending.Poll(token)
	switch res.Status {
	case auth.PendingWaiting:
		writeJSON(w, http.StatusOK, map[string]string{"status": "pending"})
	case auth.PendingError:
		writeJSON(w, http.StatusOK, map[string]string{"status": "error", "error": res.Error})
	case auth.PendingDone:
		if res.CallerUserID != "" {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
		if err := h.issueSession(w, r, res.ResolvedUserID); err != nil {
			log.Printf("telegram poll: %v", err)
			fail(w, http.StatusInternalServerError, "internal error")
			return
		}
		sum, err := h.summaryFor(res.ResolvedUserID)
		if err != nil {
			fail(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"user":   summaryToDTO(res.ResolvedUserID, sum),
		})
	}
}

// telegramWebhook handles POST /api/telegram/webhook — Telegram's own call to
// us, never a browser, so it sits outside requireAuth/requireSession
// entirely. Verifies the shared secret Telegram echoes back on every request
// (set once at startup via telegram.SetWebhook) before parsing anything, so a
// forged POST to this public URL cannot resolve someone else's pending token.
//
// Always answers 200 once the secret checks out: Telegram retries on
// anything else, and a message that isn't "/start <token>", or a token that's
// unknown/expired, is not an error — just nothing to act on. message.from is
// Telegram's own attestation of who sent it (this is a server-to-server call
// from Telegram itself), so it needs no additional signature check the way
// browser-supplied initData does.
func (h handlers) telegramWebhook(w http.ResponseWriter, r *http.Request) {
	if !h.Config.TelegramEnabled() || h.TelegramWebhookSecret == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != h.TelegramWebhookSecret {
		fail(w, http.StatusUnauthorized, "bad secret")
		return
	}
	w.WriteHeader(http.StatusOK)

	var update struct {
		Message *struct {
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			Text string `json:"text"`
			From struct {
				ID        int64  `json:"id"`
				Username  string `json:"username"`
				FirstName string `json:"first_name"`
			} `json:"from"`
		} `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil || update.Message == nil {
		return
	}
	token, ok := telegram.ParseStartToken(update.Message.Text)
	if !ok {
		return
	}
	callerUserID, ok := h.TelegramPending.Lookup(token)
	if !ok {
		return // expired or unknown — nothing sensible to say back in-chat
	}

	u := auth.TelegramUser{
		ID:        update.Message.From.ID,
		Username:  update.Message.From.Username,
		FirstName: update.Message.From.FirstName,
		AuthDate:  h.Now(),
	}
	chatID := update.Message.Chat.ID

	if callerUserID == "" {
		userID, err := h.resolveTelegramLogin(u)
		if err != nil {
			log.Printf("telegram webhook: resolve login: %v", err)
			h.TelegramPending.Fail(token, "internal error")
			h.SendTelegramMessage(chatID, "Что-то пошло не так, попробуйте войти ещё раз с сайта.")
			return
		}
		h.TelegramPending.Resolve(token, userID)
		h.SendTelegramMessage(chatID, "Готово! Вернитесь на сайт.")
		return
	}

	if err := h.resolveTelegramLink(callerUserID, u); err != nil {
		if errors.Is(err, errTelegramTaken) {
			h.TelegramPending.Fail(token, "telegram_taken")
			h.SendTelegramMessage(chatID, "Этот Telegram уже привязан к другому аккаунту.")
			return
		}
		log.Printf("telegram webhook: resolve link: %v", err)
		h.TelegramPending.Fail(token, "internal error")
		h.SendTelegramMessage(chatID, "Что-то пошло не так, попробуйте ещё раз с сайта.")
		return
	}
	h.TelegramPending.Resolve(token, callerUserID)
	h.SendTelegramMessage(chatID, "Готово! Telegram привязан, вернитесь на сайт.")
}
