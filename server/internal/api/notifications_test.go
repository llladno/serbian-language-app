package api

import (
	"net/http"
	"testing"
)

func TestListNotificationsRequiresSession(t *testing.T) {
	h, _, _ := newAuthAPI(t, nil)
	rr := anon(h, "GET", "/api/me/notifications", "")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("list notifications (no cookie) = %d, want 401", rr.Code)
	}
}

func TestListNotificationsReturnsUnreadCount(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	if _, err := st.SeedNotificationForTest(uid, "hello", fixedNow); err != nil {
		t.Fatal(err)
	}
	c := authed(t, st, uid)

	rr := doCookie(h, c, "GET", "/api/me/notifications", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list notifications = %d %s", rr.Code, rr.Body)
	}
	body := decodeBody[struct {
		Items []struct {
			ID   int64  `json:"id"`
			Text string `json:"text"`
			Read bool   `json:"read"`
		} `json:"items"`
		UnreadCount int `json:"unread_count"`
	}](t, rr)
	if len(body.Items) != 1 || body.Items[0].Text != "hello" || body.Items[0].Read {
		t.Fatalf("items = %+v, want one unread \"hello\"", body.Items)
	}
	if body.UnreadCount != 1 {
		t.Fatalf("unread_count = %d, want 1", body.UnreadCount)
	}
}

func TestListNotificationsScopedToCaller(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	otherUID := registerAndVerify(t, h, st, "other@example.com", "password123", "Other")
	if _, err := st.SeedNotificationForTest(otherUID, "not for bob", fixedNow); err != nil {
		t.Fatal(err)
	}
	c := authed(t, st, uid)

	rr := doCookie(h, c, "GET", "/api/me/notifications", "")
	body := decodeBody[struct {
		Items       []struct{} `json:"items"`
		UnreadCount int        `json:"unread_count"`
	}](t, rr)
	if len(body.Items) != 0 || body.UnreadCount != 0 {
		t.Fatalf("bob's notifications = %+v, want none (the seeded one belongs to another user)", body)
	}
}

func TestMarkNotificationsReadEndpoint(t *testing.T) {
	h, st, _ := newAuthAPI(t, nil)
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	if _, err := st.SeedNotificationForTest(uid, "hello", fixedNow); err != nil {
		t.Fatal(err)
	}
	c := authed(t, st, uid)

	rr := doCookie(h, c, "POST", "/api/me/notifications/mark-read", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("mark-read = %d %s", rr.Code, rr.Body)
	}

	rr = doCookie(h, c, "GET", "/api/me/notifications", "")
	body := decodeBody[struct {
		UnreadCount int `json:"unread_count"`
	}](t, rr)
	if body.UnreadCount != 0 {
		t.Fatalf("unread_count after mark-read = %d, want 0", body.UnreadCount)
	}
}

func TestMarkNotificationsReadRequiresSession(t *testing.T) {
	h, _, _ := newAuthAPI(t, nil)
	rr := anon(h, "POST", "/api/me/notifications/mark-read", "")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("mark-read (no cookie) = %d, want 401", rr.Code)
	}
}
