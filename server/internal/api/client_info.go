package api

import (
	"log"
	"net/http"
)

// postClientInfo handles POST /api/me/client {version, platform}: the Mini App tells us which Telegram app it runs in
// (Telegram.WebApp.version and .platform), so the admin can see who can open newer kinds of message. Values that are not
// a plain version number are ignored, never an error: this is a courtesy call and must not get in the person's way.
func (h handlers) postClientInfo(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	var req struct {
		Version  string `json:"version"`
		Platform string `json:"platform"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	if err := h.Store.NoteClientInfo(ac.UserID, req.Version, req.Platform); err != nil {
		log.Printf("client info: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
