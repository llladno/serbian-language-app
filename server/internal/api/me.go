package api

import (
	"database/sql"
	"errors"
	"io"
	"log"
	"net/http"
	netmail "net/mail"
	"strings"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
	"github.com/grisha/serbian-app/server/internal/mail"
	"github.com/grisha/serbian-app/server/internal/store"
)

// This file holds the /api/me* handlers (Task 16). All of them sit behind
// requireAuth, so authFrom always resolves; the ", ok" is still checked so a
// future routing mistake fails safe (401) instead of panicking on a zero
// authCtx.

// getMe returns the full profile: the account view plus every active session
// ("devices"), the current one flagged so the UI can label it.
func (h handlers) getMe(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	sum, err := h.summaryFor(ac.UserID)
	if err != nil {
		log.Printf("get me: summary: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	sessions, err := h.Store.ListUserSessions(ac.UserID)
	if err != nil {
		log.Printf("get me: list sessions: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	dto := meDTO{sessionUserDTO: summaryToDTO(ac.UserID, sum), Sessions: []deviceDTO{}}
	for _, s := range sessions {
		dto.Sessions = append(dto.Sessions, deviceDTO{
			ID:         s.TokenHash[:12],
			UserAgent:  s.UserAgent,
			LastSeenAt: s.LastSeenAt,
			Current:    s.TokenHash == ac.SessionHash,
		})
	}
	writeJSON(w, http.StatusOK, dto)
}

// patchMe renames the account. {name}, normalized and capped like registration.
func (h handlers) patchMe(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	name := store.NormalizeName(req.Name)
	if name == "" || len([]rune(name)) > 40 {
		fail(w, http.StatusBadRequest, "name must be 1 to 40 characters")
		return
	}
	if err := h.Store.RenameUser(ac.UserID, name); err != nil {
		log.Printf("patch me: rename: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	sum, err := h.summaryFor(ac.UserID)
	if err != nil {
		log.Printf("patch me: summary: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, summaryToDTO(ac.UserID, sum))
}

// changePassword handles POST /api/me/password, {current, new, email}. Two
// shapes, and in both every other session is purged FIRST — before any
// mutation — and fails closed: a failed purge is a 500 with nothing changed
// yet, since a surviving (possibly hijacked) session would defeat the point
// of a password change (mirrors resetPassword's DeleteUserSessions).
//   - the caller already has a password identity: current must verify, then
//     sessions are purged, then new replaces the hash, and a
//     password-changed notice is mailed.
//   - a Telegram-only account (no password identity yet): email is required
//     (and validated like register's), then sessions are purged, then the
//     identity and its verify token are created, then a verify link is
//     mailed rather than the account being usable by password right away.
func (h handlers) changePassword(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	var req struct {
		Current string `json:"current"`
		New     string `json:"new"`
		Email   string `json:"email"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	if n := len([]rune(req.New)); n < 8 || n > 128 {
		fail(w, http.StatusBadRequest, "password must be 8-128 characters")
		return
	}

	id, err := h.Store.IdentityForUser(ac.UserID, "password")
	switch {
	case err == nil:
		if req.Current == "" || !auth.VerifyPassword(id.PasswordHash, req.Current) {
			fail(w, http.StatusForbidden, "wrong_password")
			return
		}
		// Kill every other session before touching the password: a failed
		// purge must not be swallowed — a surviving (possibly hijacked)
		// session defeats the point of a password change, so a failure here
		// is a 500, not a 200, and nothing is mutated yet.
		if err := h.Store.DeleteUserSessionsExcept(ac.UserID, ac.SessionHash); err != nil {
			log.Printf("change password: purge sessions: %v", err)
			fail(w, http.StatusInternalServerError, "internal error")
			return
		}
		hash, err := auth.HashPassword(req.New)
		if err != nil {
			log.Printf("change password: hash: %v", err)
			fail(w, http.StatusInternalServerError, "internal error")
			return
		}
		if err := h.Store.SetPasswordHash(id.ID, hash); err != nil {
			log.Printf("change password: set hash: %v", err)
			fail(w, http.StatusInternalServerError, "internal error")
			return
		}
		name := ""
		if row, err := h.Store.UserByID(ac.UserID); err == nil {
			name = row.Name
		}
		to := id.Email
		h.Async(func() {
			subject, text, html := mail.RenderPasswordChanged(h.Config.AppBaseURL, name)
			h.SendMail(to, subject, text, html)
		})
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return

	case errors.Is(err, sql.ErrNoRows):
		if req.Email == "" {
			fail(w, http.StatusBadRequest, "email_required")
			return
		}
		if _, err := netmail.ParseAddress(req.Email); err != nil {
			fail(w, http.StatusBadRequest, "invalid email")
			return
		}
		// Same fail-closed guarantee as the has-password branch above: purge
		// other sessions before creating anything, since setting a password
		// here changes how this account logs in and a surviving (possibly
		// hijacked) session would defeat the point of it. Email has only been
		// validated so far, not persisted, so nothing is mutated yet if this
		// fails.
		if err := h.Store.DeleteUserSessionsExcept(ac.UserID, ac.SessionHash); err != nil {
			log.Printf("change password: purge sessions: %v", err)
			fail(w, http.StatusInternalServerError, "internal error")
			return
		}
		hash, err := auth.HashPassword(req.New)
		if err != nil {
			log.Printf("change password: hash: %v", err)
			fail(w, http.StatusInternalServerError, "internal error")
			return
		}
		identityID := auth.NewIdentityID()
		if err := h.Store.CreateIdentity(store.Identity{
			ID:           identityID,
			UserID:       ac.UserID,
			Provider:     "password",
			ProviderUID:  strings.ToLower(req.Email),
			Email:        req.Email,
			PasswordHash: hash,
		}); err != nil {
			// A racing double-submit (or a taken address reused as provider_uid)
			// trips the UNIQUE constraint here; rare enough that a plain 500 plus
			// a log line is the right amount of ceremony.
			log.Printf("change password: create identity: %v", err)
			fail(w, http.StatusInternalServerError, "internal error")
			return
		}
		raw, tokenHash, err := auth.NewToken()
		if err != nil {
			log.Printf("change password: new verify token: %v", err)
			fail(w, http.StatusInternalServerError, "internal error")
			return
		}
		if err := h.Store.CreateEmailToken(tokenHash, identityID, "verify", h.Now(), h.Now().Add(verifyTTL)); err != nil {
			log.Printf("change password: create verify token: %v", err)
			fail(w, http.StatusInternalServerError, "internal error")
			return
		}
		name := ""
		if row, err := h.Store.UserByID(ac.UserID); err == nil {
			name = row.Name
		}
		to := req.Email
		h.Async(func() {
			subject, text, html := mail.RenderVerify(h.Config.AppBaseURL, raw, name)
			h.SendMail(to, subject, text, html)
		})
		writeJSON(w, http.StatusOK, map[string]string{"status": "verify_sent"})
		return

	default:
		log.Printf("change password: lookup identity: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
}

// linkTelegram handles POST /api/me/link/telegram (Mini App initData). It
// verifies the payload the same way telegramLogin does, then resolves via the
// shared resolveTelegramLink (already-linked-to-me is a no-op, linked to
// someone else is a 409, unlinked creates the identity).
func (h handlers) linkTelegram(w http.ResponseWriter, r *http.Request) {
	if !h.Config.TelegramEnabled() {
		fail(w, http.StatusServiceUnavailable, "telegram_disabled")
		return
	}
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	u, err := verifyTelegramPayload(body, h.Config.TelegramBotToken, h.Now())
	if err != nil {
		fail(w, http.StatusUnauthorized, "bad_telegram_auth")
		return
	}

	if err := h.resolveTelegramLink(ac.UserID, u); err != nil {
		if errors.Is(err, errTelegramTaken) {
			fail(w, http.StatusConflict, "telegram_taken")
			return
		}
		log.Printf("link telegram: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}

	sum, err := h.summaryFor(ac.UserID)
	if err != nil {
		log.Printf("link telegram: summary: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, summaryToDTO(ac.UserID, sum))
}

// telegramLinkStart handles POST /api/me/telegram/start (requireSession). It
// generates a /start deep-link token bound to the caller's own account, so
// the webhook resolves it via resolveTelegramLink rather than logging in as a
// new/different account.
func (h handlers) telegramLinkStart(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	h.telegramStartFor(w, ac.UserID)
}

// unlinkTelegram handles DELETE /api/me/telegram. Refuses to remove the
// account's only login method: if there is no password identity, telegram
// stays until one is added (via changePassword's TG-only branch).
func (h handlers) unlinkTelegram(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	ids, err := h.Store.IdentitiesForUser(ac.UserID)
	if err != nil {
		log.Printf("unlink telegram: list identities: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	hasTelegram, hasPassword := false, false
	for _, id := range ids {
		switch id.Provider {
		case "telegram":
			hasTelegram = true
		case "password":
			hasPassword = true
		}
	}
	if !hasTelegram {
		fail(w, http.StatusNotFound, "not_linked")
		return
	}
	if !hasPassword {
		fail(w, http.StatusConflict, "only_login_method")
		return
	}
	if err := h.Store.DeleteUserIdentity(ac.UserID, "telegram"); err != nil {
		log.Printf("unlink telegram: delete identity: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deleteSession handles DELETE /api/me/sessions/{id}, where id is the
// 12-character token-hash prefix getMe hands out as a device handle. Deleting
// your own current session this way is allowed — it just logs this device
// out, so the cookie is cleared too.
func (h handlers) deleteSession(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	id := r.PathValue("id")
	sessions, err := h.Store.ListUserSessions(ac.UserID)
	if err != nil {
		log.Printf("delete session: list sessions: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	var found *store.Session
	for i := range sessions {
		if len(sessions[i].TokenHash) >= 12 && sessions[i].TokenHash[:12] == id {
			found = &sessions[i]
			break
		}
	}
	if found == nil {
		fail(w, http.StatusNotFound, "unknown session")
		return
	}
	if err := h.Store.DeleteSession(found.TokenHash); err != nil {
		log.Printf("delete session: delete: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	if found.TokenHash == ac.SessionHash {
		h.clearSessionCookie(w)
	}
	w.WriteHeader(http.StatusNoContent)
}

// reauthWindow is how long after a session was created a Telegram-only
// account (no password identity, so nothing to re-verify against) may still
// self-delete without extra proof — a fresh login stands in for a password.
const reauthWindow = 5 * time.Minute

// deleteMe handles DELETE /api/me, {password?}. A password account must
// confirm with the correct password; a Telegram-only account may only do this
// from a session created within reauthWindow, since there is no password to
// check. Either way, success wipes every row belonging to the account in one
// store transaction and clears this device's cookie.
func (h handlers) deleteMe(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}

	id, err := h.Store.IdentityForUser(ac.UserID, "password")
	switch {
	case err == nil:
		if req.Password == "" || !auth.VerifyPassword(id.PasswordHash, req.Password) {
			fail(w, http.StatusForbidden, "wrong_password")
			return
		}
	case errors.Is(err, sql.ErrNoRows):
		sessions, err := h.Store.ListUserSessions(ac.UserID)
		if err != nil {
			log.Printf("delete me: list sessions: %v", err)
			fail(w, http.StatusInternalServerError, "internal error")
			return
		}
		fresh := false
		for _, s := range sessions {
			if s.TokenHash != ac.SessionHash {
				continue
			}
			if created, perr := time.Parse(time.RFC3339, s.CreatedAt); perr == nil {
				fresh = h.Now().Sub(created) <= reauthWindow
			}
			break
		}
		if !fresh {
			fail(w, http.StatusForbidden, "reauth_required")
			return
		}
	default:
		log.Printf("delete me: lookup identity: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.Store.DeleteUser(ac.UserID); err != nil {
		log.Printf("delete me: delete user: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
