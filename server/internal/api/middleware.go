package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
	"github.com/grisha/serbian-app/server/internal/config"
	"github.com/grisha/serbian-app/server/internal/store"
)

// lastSeenThrottle is the minimum gap between two expiry-sliding writes for
// one session: store.TouchSession has no throttle of its own, so requireAuth
// only calls it once per window.
const lastSeenThrottle = time.Hour

// sessionTTL is how long a fresh session cookie lives. It MUST match
// store.sessionSlide (part 1, sessions.go) so the cookie and the row expire
// together.
const sessionTTL = 30 * 24 * time.Hour

// csp is the Content-Security-Policy sent on every API response. It is the
// minimum that lets the Vite build run (inline styles for Tailwind, data:
// fonts and images) while allowing the Telegram Mini App to frame the page.
const csp = "default-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"font-src 'self' data:; " +
	"img-src 'self' data:; " +
	"connect-src 'self'; " +
	"base-uri 'self'; " +
	"form-action 'self'; " +
	"frame-ancestors 'self' https://web.telegram.org https://*.telegram.org"

// SecurityHeaders stamps the static security headers on every response. HSTS
// is only meaningful (and only sent) when the app is served over HTTPS. There
// is deliberately no X-Frame-Options: the Telegram Mini App needs the iframe,
// and the CSP frame-ancestors directive already scopes who may embed us.
//
// It is exported so main.go can wrap the WHOLE server with it — the SPA HTML
// and the /img//audio static trees need the CSP (frame-ancestors is the only
// clickjacking guard here) just as much as the JSON API does.
func SecurityHeaders(cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		if cfg.Secure() {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy", csp)
		next.ServeHTTP(w, r)
	})
}

// checkOrigin is a CSRF guard for state-changing requests. For any method
// other than GET/HEAD/OPTIONS the Origin header (or Referer, when Origin is
// absent) must resolve to the same scheme+host as cfg.AppBaseURL. A non-GET
// request with neither header, or with a mismatched one, is rejected 403.
func checkOrigin(cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		got := r.Header.Get("Origin")
		if got == "" {
			got = r.Header.Get("Referer")
		}
		if got == "" || !sameOrigin(got, cfg.AppBaseURL) {
			fail(w, http.StatusForbidden, "bad origin")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// sameOrigin reports whether two URLs share a scheme and host. A URL that
// fails to parse, or lacks either component, never matches. Hosts are compared
// case-insensitively (RFC 3986 §3.2.2 — a host is case-insensitive), the
// scheme exactly.
func sameOrigin(a, b string) bool {
	ua, err := url.Parse(a)
	if err != nil {
		return false
	}
	ub, err := url.Parse(b)
	if err != nil {
		return false
	}
	if ua.Scheme == "" || ua.Host == "" || ub.Scheme == "" || ub.Host == "" {
		return false
	}
	return ua.Scheme == ub.Scheme && strings.EqualFold(ua.Host, ub.Host)
}

// userSummary is the denormalized view of an account attached to an
// authenticated request, so downstream handlers (notably /me) need no extra
// store round-trips. It is only populated for cookie-session requests.
type userSummary struct {
	ID, Name, Email  string
	EmailVerified    bool
	TelegramLinked   bool
	TelegramUsername string
}

// authCtx is what requireAuth puts on the request context. SessionHash is the
// hex token hash for a real cookie session and "" for the legacy X-User
// bridge; Summary is the zero value unless a cookie session resolved.
type authCtx struct {
	UserID      string
	SessionHash string
	Summary     userSummary
}

// ctxKey is the unexported type for this package's context keys.
type ctxKey int

const authCtxKey ctxKey = iota

// authFrom returns the authCtx stored by requireAuth, if any.
func authFrom(r *http.Request) (authCtx, bool) {
	ac, ok := r.Context().Value(authCtxKey).(authCtx)
	return ac, ok
}

// requireAuth resolves the caller and rejects the request 401 when it cannot.
// A valid session cookie is preferred; failing that, the legacy X-User header
// (pre-session front-end) is honored as a bridge. On success the authCtx is
// placed on the request context for authFrom.
func (h handlers) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Session cookie.
		if c, err := r.Cookie(h.Config.CookieName()); err == nil && c.Value != "" {
			hash := auth.HashToken(c.Value)
			sess, err := h.Store.SessionByHash(hash, h.Now())
			switch {
			case err == nil:
				h.slideSession(sess, hash)
				sum, err := h.summaryFor(sess.UserID)
				if err != nil {
					// The session was valid; a summary lookup failing is a
					// real server fault, not a reason to try the bridge.
					fail(w, http.StatusInternalServerError, "internal error")
					return
				}
				ac := authCtx{
					UserID:      sess.UserID,
					SessionHash: hash,
					Summary:     sum,
				}
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authCtxKey, ac)))
				return
			case errors.Is(err, sql.ErrNoRows):
				// Unknown or expired session — fall through to the bridge.
			default:
				fail(w, http.StatusInternalServerError, "session lookup failed")
				return
			}
		}

		// 2. Legacy X-User bridge.
		raw := r.Header.Get("X-User")
		if dec, err := url.PathUnescape(raw); err == nil {
			raw = dec
		}
		if name := store.NormalizeName(raw); name != "" {
			row, err := h.Store.UserByName(name)
			switch {
			case err == nil:
				ac := authCtx{UserID: row.ID}
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authCtxKey, ac)))
				return
			case errors.Is(err, sql.ErrNoRows):
				// Unknown account — fall through to 401.
			default:
				fail(w, http.StatusInternalServerError, "account lookup failed")
				return
			}
		}

		// 3. Nothing resolved.
		fail(w, http.StatusUnauthorized, "no session")
	})
}

// requireSession is requireAuth minus the legacy X-User bridge: only a real
// session cookie gets through. It guards /api/me*, POST /api/auth/logout-all
// and GET /api/auth/session — endpoints that read or mutate the account itself,
// where the bridge would let anyone who merely knows a display name act as that
// account (display names are public via /api/leaderboard). A bridge-resolved
// authCtx has an empty SessionHash, which is exactly the discriminator here.
func (h handlers) requireSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ac, ok := authFrom(r)
			if !ok || ac.SessionHash == "" {
				fail(w, http.StatusUnauthorized, "session required")
				return
			}
			next(w, r)
		})).ServeHTTP(w, r)
	}
}

// slideSession pushes the session's expiry forward via store.TouchSession, but
// at most once per lastSeenThrottle window. Best-effort: a write failure here
// never blocks the request (liveness was already gated by SessionByHash).
func (h handlers) slideSession(sess store.Session, hash string) {
	last, err := time.Parse(time.RFC3339, sess.LastSeenAt)
	if err != nil || h.Now().Sub(last) > lastSeenThrottle {
		_ = h.Store.TouchSession(hash, h.Now())
	}
}

// summaryFor loads the denormalized account view for an authenticated request.
// A missing row (sql.ErrNoRows) leaves that part of the summary zero and the
// load continues; any other store error is a real fault (timeout, connection
// blip) and is returned so requireAuth can 500 rather than silently serve a
// half-populated summary that downstream security checks would trust.
func (h handlers) summaryFor(userID string) (userSummary, error) {
	sum := userSummary{ID: userID}
	// UserByID returning ErrNoRows for a live session's user_id is anomalous
	// (the FK target is gone), but per the brief we leave Name zero and carry
	// on rather than fail the request.
	if u, err := h.Store.UserByID(userID); err == nil {
		sum.Name = u.Name
	} else if !errors.Is(err, sql.ErrNoRows) {
		return userSummary{}, fmt.Errorf("load summary: %w", err)
	}
	if id, err := h.Store.IdentityForUser(userID, "password"); err == nil {
		sum.Email = id.Email
		sum.EmailVerified = id.EmailVerifiedAt != ""
	} else if !errors.Is(err, sql.ErrNoRows) {
		return userSummary{}, fmt.Errorf("load summary: %w", err)
	}
	if id, err := h.Store.IdentityForUser(userID, "telegram"); err == nil {
		sum.TelegramLinked = true
		sum.TelegramUsername = id.TgUsername
	} else if !errors.Is(err, sql.ErrNoRows) {
		return userSummary{}, fmt.Errorf("load summary: %w", err)
	}
	return sum, nil
}

// setSessionCookie writes the session cookie holding the raw token. The name
// and Secure flag follow cfg (locked-down __Host-session over HTTPS). maxAge
// is truncated to whole seconds.
func (h handlers) setSessionCookie(w http.ResponseWriter, rawToken string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.Config.CookieName(),
		Value:    rawToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.Config.Secure(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(maxAge.Seconds()),
	})
}

// clearSessionCookie expires the session cookie (logout). It repeats the same
// name/path/flags so the browser matches and drops the stored cookie.
func (h handlers) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.Config.CookieName(),
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.Config.Secure(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
