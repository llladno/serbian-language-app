package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
	"github.com/grisha/serbian-app/server/internal/config"
	"github.com/grisha/serbian-app/server/internal/store"
)

// okHandler is the terminal handler in a middleware chain under test.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// mwHandlers builds a real handlers value backed by an in-memory store.
func mwHandlers(t *testing.T, base string) (handlers, *store.Store) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	h := handlers{Deps{
		Store:  st,
		Now:    func() time.Time { return fixedNow },
		Config: config.Config{AppBaseURL: base},
	}}
	return h, st
}

// liveSession creates a user plus a non-expired session and returns the uid
// and the cookie carrying the raw token.
func liveSession(t *testing.T, st *store.Store, name string) (string, *http.Cookie) {
	t.Helper()
	uid, err := st.CreateUser(name)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	raw, hash, err := auth.NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if err := st.CreateSession(hash, uid, "ua", fixedNow, fixedNow.Add(720*time.Hour)); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	return uid, &http.Cookie{Name: "session", Value: raw}
}

func TestRequireAuthNoCredentials401(t *testing.T) {
	h, _ := mwHandlers(t, "http://localhost:8080")
	r := httptest.NewRequest(http.MethodGet, "/api/progress", nil)
	w := httptest.NewRecorder()
	h.requireAuth(okHandler).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestRequireAuthValidCookie(t *testing.T) {
	h, st := mwHandlers(t, "http://localhost:8080")
	uid, cookie := liveSession(t, st, "Гриша")

	// Seed a verified password identity and a linked telegram identity so the
	// assertions below pin summaryFor's field mapping.
	if err := st.CreateIdentity(store.Identity{
		ID:              auth.NewIdentityID(),
		UserID:          uid,
		Provider:        "password",
		ProviderUID:     "grisha@example.com",
		Email:           "grisha@example.com",
		EmailVerifiedAt: fixedNow.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("CreateIdentity password: %v", err)
	}
	if err := st.CreateIdentity(store.Identity{
		ID:          auth.NewIdentityID(),
		UserID:      uid,
		Provider:    "telegram",
		ProviderUID: "123456",
		TgUsername:  "grisha_tg",
	}); err != nil {
		t.Fatalf("CreateIdentity telegram: %v", err)
	}

	var gotAC authCtx
	var gotOK bool
	term := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAC, gotOK = authFrom(r)
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/api/progress", nil)
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	h.requireAuth(term).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !gotOK || gotAC.UserID != uid {
		t.Fatalf("authFrom = (%q, %v), want (%q, true)", gotAC.UserID, gotOK, uid)
	}
	s := gotAC.Summary
	if s.Name != "Гриша" {
		t.Errorf("Summary.Name = %q, want %q", s.Name, "Гриша")
	}
	if s.Email != "grisha@example.com" {
		t.Errorf("Summary.Email = %q, want %q", s.Email, "grisha@example.com")
	}
	if !s.EmailVerified {
		t.Errorf("Summary.EmailVerified = false, want true")
	}
	if !s.TelegramLinked {
		t.Errorf("Summary.TelegramLinked = false, want true")
	}
	if s.TelegramUsername != "grisha_tg" {
		t.Errorf("Summary.TelegramUsername = %q, want %q", s.TelegramUsername, "grisha_tg")
	}
}

func TestRequireAuthExpiredCookie401(t *testing.T) {
	h, st := mwHandlers(t, "http://localhost:8080")
	uid, err := st.CreateUser("Ана")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	raw, hash, err := auth.NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	// expires_at strictly before fixedNow.
	if err := st.CreateSession(hash, uid, "ua", fixedNow.Add(-1000*time.Hour), fixedNow.Add(-time.Hour)); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/progress", nil)
	r.AddCookie(&http.Cookie{Name: "session", Value: raw})
	w := httptest.NewRecorder()
	h.requireAuth(okHandler).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestRequireAuthBridgeXUser(t *testing.T) {
	h, st := mwHandlers(t, "http://localhost:8080")
	uid, err := st.CreateUser("Мира")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	var gotUID, gotHash string
	term := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ac, _ := authFrom(r)
		gotUID, gotHash = ac.UserID, ac.SessionHash
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/api/progress", nil)
	r.Header.Set("X-User", "Мира")
	w := httptest.NewRecorder()
	h.requireAuth(term).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if gotUID != uid || gotHash != "" {
		t.Fatalf("authFrom = (uid %q, hash %q), want (%q, \"\")", gotUID, gotHash, uid)
	}
}

// TestRequireSessionRejectsBridge pins the unit-level contract: requireSession
// lets a real cookie session through and 401s a bridge-resolved caller, even
// though requireAuth beneath it happily resolves one.
func TestRequireSessionRejectsBridge(t *testing.T) {
	h, st := mwHandlers(t, "http://localhost:8080")
	uid, cookie := liveSession(t, st, "Ковачевић")

	t.Run("cookie passes", func(t *testing.T) {
		var gotUID, gotHash string
		term := func(w http.ResponseWriter, r *http.Request) {
			ac, _ := authFrom(r)
			gotUID, gotHash = ac.UserID, ac.SessionHash
			w.WriteHeader(http.StatusOK)
		}
		r := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		h.requireSession(term).ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if gotUID != uid || gotHash == "" {
			t.Fatalf("authFrom = (uid %q, hash %q), want (%q, non-empty)", gotUID, gotHash, uid)
		}
	})

	t.Run("X-User bridge rejected", func(t *testing.T) {
		reached := false
		term := func(w http.ResponseWriter, r *http.Request) {
			reached = true
			w.WriteHeader(http.StatusOK)
		}
		r := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		r.Header.Set("X-User", "Ковачевић")
		w := httptest.NewRecorder()
		h.requireSession(term).ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401 (bridge must not authenticate here)", w.Code)
		}
		if reached {
			t.Fatal("handler ran for a bridge-resolved caller")
		}
	})

	t.Run("no credentials rejected", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		w := httptest.NewRecorder()
		h.requireSession(okHandler.ServeHTTP).ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})
}

// TestXUserBridgeCannotReachAccountRoutes is the end-to-end regression for the
// account-takeover hole: every /api/me* and account-scoped auth route must 401
// an X-User-only caller, while the legacy content routes still honour the
// bridge. The seeded accounts here are Telegram-only — exactly the shape that
// POST /api/me/password would have taken over with no credential at all.
func TestXUserBridgeCannotReachAccountRoutes(t *testing.T) {
	h, st := newTestAPI(t)
	uid, err := st.CreateUser("Гриша")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if err := st.CreateIdentity(store.Identity{
		ID:          auth.NewIdentityID(),
		UserID:      uid,
		Provider:    "telegram",
		ProviderUID: "555001",
		TgUsername:  "grisha_tg",
	}); err != nil {
		t.Fatalf("CreateIdentity telegram: %v", err)
	}

	sealed := []struct{ method, path, body string }{
		{"GET", "/api/me", ""},
		{"PATCH", "/api/me", `{"name":"Взломан"}`},
		{"POST", "/api/me/password", `{"new":"attacker-password","email":"attacker@example.com"}`},
		{"POST", "/api/me/link/telegram", `{"id":1}`},
		{"DELETE", "/api/me/telegram", ""},
		{"DELETE", "/api/me/sessions/abcdef012345", ""},
		{"DELETE", "/api/me", `{"password":"x"}`},
		{"POST", "/api/auth/logout-all", ""},
		{"GET", "/api/auth/session", ""},
	}
	for _, tc := range sealed {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rr := doAs(h, "Гриша", tc.method, tc.path, tc.body)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401 (X-User must not authenticate here); body %s", rr.Code, rr.Body)
			}
		})
	}

	// Nothing was mutated by any of the above.
	if ids, err := st.IdentitiesForUser(uid); err != nil {
		t.Fatalf("IdentitiesForUser: %v", err)
	} else if len(ids) != 1 || ids[0].Provider != "telegram" {
		t.Fatalf("identities = %+v, want the single telegram identity untouched", ids)
	}
	if row, err := st.UserByID(uid); err != nil {
		t.Fatalf("UserByID: %v", err)
	} else if row.Name != "Гриша" {
		t.Fatalf("name = %q, want %q (PATCH /api/me must not have landed)", row.Name, "Гриша")
	}

	// The bridge still works for the legacy content routes the old front-end
	// actually needs.
	for _, path := range []string{"/api/progress", "/api/course", "/api/leaderboard", "/api/vocab"} {
		if rr := doAs(h, "Гриша", "GET", path, ""); rr.Code != http.StatusOK {
			t.Fatalf("GET %s with X-User = %d, want 200 (bridge must keep working here); body %s",
				path, rr.Code, rr.Body)
		}
	}
}

// TestSessionCookieStillReachesAccountRoutes confirms moving the nine routes
// onto the root mux did not break ServeMux precedence: a real cookie still
// resolves them (not a 404 from the "/api/" catch-all).
func TestSessionCookieStillReachesAccountRoutes(t *testing.T) {
	h, st := newTestAPI(t)
	uid, err := st.CreateUser("Алина")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	c := authed(t, st, uid)

	for _, tc := range []struct {
		method, path, body string
		want               int
	}{
		{"GET", "/api/me", "", http.StatusOK},
		{"GET", "/api/auth/session", "", http.StatusOK},
		{"PATCH", "/api/me", `{"name":"Алина Н"}`, http.StatusOK},
		// Telegram-only account with no telegram identity: the handler's own
		// 404 proves routing reached it.
		{"DELETE", "/api/me/telegram", "", http.StatusNotFound},
		{"DELETE", "/api/me/sessions/000000000000", "", http.StatusNotFound},
		{"POST", "/api/auth/logout-all", "", http.StatusNoContent},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rr := doCookie(h, c, tc.method, tc.path, tc.body)
			if rr.Code != tc.want {
				t.Fatalf("status = %d, want %d; body %s", rr.Code, tc.want, rr.Body)
			}
		})
	}
}

func TestCheckOriginRejectsCrossOrigin(t *testing.T) {
	h, _ := mwHandlers(t, "http://localhost:8080")
	mw := checkOrigin(h.Config, okHandler)

	cases := []struct {
		name, method, origin string
		want                 int
	}{
		{"cross-origin POST", http.MethodPost, "https://evil.com", http.StatusForbidden},
		{"matching-origin POST", http.MethodPost, "http://localhost:8080", http.StatusOK},
		{"no-origin POST", http.MethodPost, "", http.StatusForbidden},
		{"GET without origin", http.MethodGet, "", http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, "/api/thing", nil)
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			w := httptest.NewRecorder()
			mw.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d", w.Code, tc.want)
			}
		})
	}
}

// TestSameOriginHostCaseInsensitive pins RFC 3986 §3.2.2: the host half of an
// origin is case-insensitive, the scheme comparison stays exact.
func TestSameOriginHostCaseInsensitive(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"https://App.Example.COM", "https://app.example.com", true},
		{"https://app.example.com", "https://APP.EXAMPLE.COM", true},
		{"http://app.example.com", "https://app.example.com", false},
		{"https://evil.com", "https://app.example.com", false},
		{"https://app.example.com:8443", "https://app.example.com", false},
		{"not a url", "https://app.example.com", false},
	}
	for _, tc := range cases {
		if got := sameOrigin(tc.a, tc.b); got != tc.want {
			t.Errorf("sameOrigin(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestCheckOriginRefererFallback(t *testing.T) {
	h, _ := mwHandlers(t, "http://localhost:8080")
	mw := checkOrigin(h.Config, okHandler)

	r := httptest.NewRequest(http.MethodPost, "/api/thing", nil)
	r.Header.Set("Referer", "http://localhost:8080/lesson/03")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (Referer same-origin)", w.Code)
	}
}

func TestSecurityHeadersPresent(t *testing.T) {
	t.Run("http base", func(t *testing.T) {
		cfg := config.Config{AppBaseURL: "http://localhost:8080"}
		r := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		w := httptest.NewRecorder()
		SecurityHeaders(cfg, okHandler).ServeHTTP(w, r)

		if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("X-Content-Type-Options = %q", got)
		}
		if got := w.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
			t.Errorf("Referrer-Policy = %q", got)
		}
		if policy := w.Header().Get("Content-Security-Policy"); policy != csp {
			t.Fatalf("CSP = %q, want exactly %q", policy, csp)
		}
		if got := w.Header().Get("Strict-Transport-Security"); got != "" {
			t.Errorf("HSTS should be absent on http base, got %q", got)
		}
		if got := w.Header().Get("X-Frame-Options"); got != "" {
			t.Errorf("X-Frame-Options must not be set, got %q", got)
		}
	})

	t.Run("https base", func(t *testing.T) {
		cfg := config.Config{AppBaseURL: "https://app.example.com"}
		r := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		w := httptest.NewRecorder()
		SecurityHeaders(cfg, okHandler).ServeHTTP(w, r)
		if got := w.Header().Get("Strict-Transport-Security"); got == "" {
			t.Errorf("HSTS should be present on https base")
		}
	})
}

func TestSessionCookieAttributes(t *testing.T) {
	t.Run("https base", func(t *testing.T) {
		h, _ := mwHandlers(t, "https://app.example.com")
		w := httptest.NewRecorder()
		h.setSessionCookie(w, "raw-token", sessionTTL)
		c := readOneCookie(t, w)
		if c.Name != "__Host-session" {
			t.Errorf("name = %q, want __Host-session", c.Name)
		}
		if !c.Secure {
			t.Errorf("Secure = false, want true on https base")
		}
		if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" {
			t.Errorf("flags = %+v", c)
		}
		if c.Value != "raw-token" {
			t.Errorf("value = %q", c.Value)
		}
	})

	t.Run("http base", func(t *testing.T) {
		h, _ := mwHandlers(t, "http://localhost:8080")
		w := httptest.NewRecorder()
		h.setSessionCookie(w, "raw-token", sessionTTL)
		c := readOneCookie(t, w)
		if c.Name != "session" {
			t.Errorf("name = %q, want session", c.Name)
		}
		if c.Secure {
			t.Errorf("Secure = true, want false on http base")
		}
		if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" {
			t.Errorf("flags = %+v", c)
		}
	})

	t.Run("clear", func(t *testing.T) {
		h, _ := mwHandlers(t, "http://localhost:8080")
		w := httptest.NewRecorder()
		h.clearSessionCookie(w)
		c := readOneCookie(t, w)
		if c.Name != "session" || c.MaxAge >= 0 || c.Value != "" {
			t.Errorf("clear cookie = %+v", c)
		}
	})
}

func readOneCookie(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	cs := w.Result().Cookies()
	if len(cs) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cs))
	}
	return cs[0]
}
