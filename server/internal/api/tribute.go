package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/grisha/serbian-app/server/internal/store"
)

// tributeWebhook receives payment events from Tribute (https://tribute.tg) —
// donations made through the "Šoljica kafe" tier linked from the profile
// page's DonateCard/DonateModal. Authenticated by an HMAC-SHA256 signature
// (the "trbt-signature" header, keyed with TRIBUTE_API_KEY) rather than a
// session or a static shared secret — unlike telegramWebhook, a failure here
// is answered with a non-200 status on purpose, so Tribute's own retry
// policy re-delivers on a transient store error instead of losing the event.
//
// See docs/superpowers/specs/2026-09-23-donations-design.md — Tribute's
// public docs list the webhook event names and the signature scheme but not
// a full example payload, so the exact field names below (particularly
// whether amount arrives in major or minor units) are a best-effort read of
// the docs, not a confirmed spec. stringField/numberField are deliberately
// tolerant of either a JSON string or number for the same key. raw_payload
// is always stored in full regardless, so a wrong guess here costs nothing
// but needs fixing in this file once a real webhook has been observed.
func (h handlers) tributeWebhook(w http.ResponseWriter, r *http.Request) {
	if h.Config.TributeAPIKey == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fail(w, http.StatusBadRequest, "bad body")
		return
	}
	if !verifyTributeSignature(body, r.Header.Get("trbt-signature"), h.Config.TributeAPIKey) {
		fail(w, http.StatusUnauthorized, "bad signature")
		return
	}

	var evt struct {
		Name    string         `json:"name"`
		Payload map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(body, &evt); err != nil || evt.Payload == nil {
		// Malformed body from an authenticated caller is a caller error, not
		// grounds for a retry — 200 so Tribute doesn't keep resending it.
		w.WriteHeader(http.StatusOK)
		return
	}

	tgID := stringField(evt.Payload, "telegram_user_id")
	if tgID == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	eventID := stringField(evt.Payload, "id")
	if eventID == "" {
		sum := sha256.Sum256(body)
		eventID = hex.EncodeToString(sum[:])
	}

	userID := ""
	if id, err := h.Store.IdentityByProviderUID("telegram", tgID); err == nil {
		userID = id.UserID
	}

	if err := h.Store.CreateDonation(store.Donation{
		UserID:           userID,
		TelegramUserID:   tgID,
		TelegramUsername: stringField(evt.Payload, "telegram_username"),
		AmountMinorUnits: numberField(evt.Payload, "amount"),
		Currency:         stringField(evt.Payload, "currency"),
		EventType:        evt.Name,
		TributeEventID:   eventID,
		RawPayload:       string(body),
	}, h.Now()); err != nil {
		fail(w, http.StatusInternalServerError, "store donation")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// verifyTributeSignature reports whether sig is the hex HMAC-SHA256 of body
// keyed with apiKey — Tribute's "trbt-signature" header.
func verifyTributeSignature(body []byte, sig, apiKey string) bool {
	if sig == "" {
		return false
	}
	want, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write(body)
	return hmac.Equal(mac.Sum(nil), want)
}

// stringField reads a string-valued key from a decoded JSON object, coercing
// a JSON number to its decimal string form.
func stringField(m map[string]any, key string) string {
	switch v := m[key].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

// numberField reads key as an amount in major currency units (a JSON number
// or numeric string) and converts it to minor units (×100).
func numberField(m map[string]any, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v * 100)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return int64(f * 100)
	default:
		return 0
	}
}
