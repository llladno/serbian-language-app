package api

import (
	"net/http"

	"github.com/grisha/serbian-app/server/internal/store"
)

// trackVisit handles POST /api/track-visit — a public, unauthenticated
// beacon the frontend fires once per browser session when it sees UTM
// parameters in the landing URL. Always answers 204, even when throttled or
// given a malformed body: this is best-effort analytics, never something
// that should surface an error to a visitor. See
// docs/superpowers/specs/2026-09-22-utm-link-tracking-design.md.
func (h handlers) trackVisit(w http.ResponseWriter, r *http.Request) {
	defer w.WriteHeader(http.StatusNoContent)

	if !allow(h.Visits, "ip:"+clientIP(r)) {
		return
	}
	var req struct {
		UtmSource   string `json:"utm_source"`
		UtmMedium   string `json:"utm_medium"`
		UtmCampaign string `json:"utm_campaign"`
		UtmContent  string `json:"utm_content"`
	}
	if err := decode(r, &req); err != nil {
		return
	}
	_ = h.Store.RecordLinkVisit(store.Attribution{
		UtmSource:   req.UtmSource,
		UtmMedium:   req.UtmMedium,
		UtmCampaign: req.UtmCampaign,
		UtmContent:  req.UtmContent,
	}, h.Now())
}
