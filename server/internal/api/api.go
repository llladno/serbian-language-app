package api

import (
	"encoding/json"
	"net/http"
)

// Deps holds the API's collaborators. Fields are added as the API grows.
type Deps struct{}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Handler builds the /api router.
func Handler(deps Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "content_stale": false})
	})
	return mux
}
