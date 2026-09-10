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
		securityHeaders(cfg, okHandler).ServeHTTP(w, r)

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
		securityHeaders(cfg, okHandler).ServeHTTP(w, r)
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
