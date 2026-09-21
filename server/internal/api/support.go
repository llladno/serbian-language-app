package api

import (
	"net/http"
)

// createSupportMessage handles POST /api/me/support, {message}. Called from
// the profile page's support card when the user writes directly instead of
// going to Telegram — stored for the admin panel to list, nothing more.
func (h handlers) createSupportMessage(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	var req struct {
		Message string `json:"message"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	if err := h.Store.CreateSupportMessage(ac.UserID, req.Message, h.Now()); err != nil {
		fail(w, http.StatusBadRequest, "message must not be empty")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
