package api

import (
	"net/http"
	"testing"
)

// The learner's choice of Serbian script is an account setting, so it follows
// them between the browser and the Telegram Mini App, which keep separate
// local storage.

func TestAccountScriptDefaultsToLatin(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)

	rr := doCookie(h, c, "GET", "/api/auth/session", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("session = %d %s", rr.Code, rr.Body)
	}
	if got := decodeBody[sessionUserDTO](t, rr); got.Script != "latin" {
		t.Errorf("default script = %q, want latin", got.Script)
	}
}

func TestPatchMeSetsScriptAndLeavesTheRestAlone(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)

	rr := doCookie(h, c, "PATCH", "/api/me", `{"script":"cyrillic"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("script-only patch = %d %s, want 200", rr.Code, rr.Body)
	}
	if got := decodeBody[sessionUserDTO](t, rr); got.Script != "cyrillic" || got.Name != "Bob" {
		t.Errorf("patch response = %+v, want script cyrillic and the name untouched", got)
	}

	// It sticks: a fresh read of the session sees it.
	sess := decodeBody[sessionUserDTO](t, doCookie(h, c, "GET", "/api/auth/session", ""))
	if sess.Script != "cyrillic" {
		t.Errorf("session script after patch = %q, want cyrillic", sess.Script)
	}
	if loc := st.UserLocation(uid); loc.String() != "Europe/Belgrade" {
		t.Errorf("a script patch changed the timezone: %q", loc)
	}

	// And back.
	rr = doCookie(h, c, "PATCH", "/api/me", `{"script":"latin"}`)
	if got := decodeBody[sessionUserDTO](t, rr); got.Script != "latin" {
		t.Errorf("script after switching back = %q, want latin", got.Script)
	}
}

func TestPatchMeRejectsAnUnknownScript(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)
	doCookie(h, c, "PATCH", "/api/me", `{"script":"cyrillic"}`)

	for _, bad := range []string{`{"script":"greek"}`, `{"script":""}`, `{"script":"Cyrillic"}`} {
		if rr := doCookie(h, c, "PATCH", "/api/me", bad); rr.Code != http.StatusBadRequest {
			t.Errorf("patch %s = %d, want 400", bad, rr.Code)
		}
	}
	sess := decodeBody[sessionUserDTO](t, doCookie(h, c, "GET", "/api/auth/session", ""))
	if sess.Script != "cyrillic" {
		t.Errorf("a rejected patch changed the script to %q", sess.Script)
	}
}
