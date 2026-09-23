package api

import (
	"net/http"
	"time"
)

// notificationDTO is one item in GET /api/me/notifications' response.
type notificationDTO struct {
	ID        int64  `json:"id"`
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
	Read      bool   `json:"read"`
}

// listNotifications handles GET /api/me/notifications — the bell
// dropdown's data source. Notifications are written directly into Postgres
// by ucimo-content-admin; this only ever reads and marks-read.
func (h handlers) listNotifications(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	notifs, err := h.Store.ListNotificationsForUser(ac.UserID, h.Now())
	if err != nil {
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	items := make([]notificationDTO, 0, len(notifs))
	unread := 0
	for _, n := range notifs {
		items = append(items, notificationDTO{
			ID:        n.ID,
			Text:      n.Text,
			CreatedAt: n.CreatedAt.UTC().Format(time.RFC3339),
			Read:      n.Read,
		})
		if !n.Read {
			unread++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "unread_count": unread})
}

// markNotificationsRead handles POST /api/me/notifications/mark-read —
// marks every currently-unread notification of the caller as read. Called
// once when the bell dropdown opens: the whole visible batch flips together,
// not per-item.
func (h handlers) markNotificationsRead(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	if err := h.Store.MarkNotificationsRead(ac.UserID, h.Now()); err != nil {
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
