package api

import (
	"net/http"
	"testing"
)

func TestCreateSupportMessage(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)

	rr := doCookie(h, c, "POST", "/api/me/support", `{"message":"Не открывается урок 3"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("create support message = %d %s", rr.Code, rr.Body)
	}

	msgs, err := st.ListSupportMessages()
	if err != nil {
		t.Fatalf("ListSupportMessages: %v", err)
	}
	if len(msgs) != 1 || msgs[0].UserID != uid || msgs[0].Message != "Не открывается урок 3" {
		t.Fatalf("ListSupportMessages = %+v, want one message for %s", msgs, uid)
	}
}

func TestCreateSupportMessageRejectsBlank(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)

	rr := doCookie(h, c, "POST", "/api/me/support", `{"message":"   "}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("create support message (blank) = %d %s, want 400", rr.Code, rr.Body)
	}
	msgs, err := st.ListSupportMessages()
	if err != nil {
		t.Fatalf("ListSupportMessages: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("ListSupportMessages = %d rows, want 0", len(msgs))
	}
}

func TestCreateSupportMessageRequiresSession(t *testing.T) {
	h, _, _ := newAuthAPI(t, nil)
	rr := anon(h, "POST", "/api/me/support", `{"message":"hi"}`)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("create support message (no cookie) = %d, want 401", rr.Code)
	}
}
