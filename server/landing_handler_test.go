package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestLandingHandlerRoutes(t *testing.T) {
	files := fstest.MapFS{
		"index.html":           {Data: []byte("<h1>home</h1>")},
		"robots.txt":           {Data: []byte("User-agent: *")},
		"blog/index.html":      {Data: []byte("<h1>blog</h1>")},
		"blog/post/index.html": {Data: []byte("<h1>post</h1>")},
		"_nuxt/entry.js":       {Data: []byte("js")},
	}
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot) // marker: the request fell through to the SPA
	})
	h := landingHandler(files, app)

	cases := []struct {
		name     string
		target   string
		status   int
		location string
		body     string
	}{
		{"root", "/", 200, "", "home"},
		{"static file", "/robots.txt", 200, "", "User-agent"},
		{"page without slash is served directly", "/blog/post", 200, "", "post"},
		{"section index without slash", "/blog", 200, "", "blog"},
		{"slashed duplicate redirects to canonical", "/blog/post/", 301, "/blog/post", ""},
		{"redirect keeps the query", "/blog/post/?utm_source=x", 301, "/blog/post?utm_source=x", ""},
		{"app route falls through", "/login", http.StatusTeapot, "", ""},
		{"unknown path falls through", "/blog/missing", http.StatusTeapot, "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.target, nil))
			if rec.Code != c.status {
				t.Fatalf("GET %s: status %d, want %d", c.target, rec.Code, c.status)
			}
			if got := rec.Header().Get("Location"); got != c.location {
				t.Errorf("GET %s: Location %q, want %q", c.target, got, c.location)
			}
			if c.body != "" && !strings.Contains(rec.Body.String(), c.body) {
				t.Errorf("GET %s: body %q lacks %q", c.target, rec.Body.String(), c.body)
			}
		})
	}
}
