# Tribute Donations (serbian-app) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let users open the Tribute "Šoljica kafe" donation tier from the profile page (direct Telegram deep link inside the Mini App, a modal with two links in a regular browser), and record every donation Tribute reports via webhook so it's queryable later.

**Architecture:** A `DonateCard`/`DonateModal` pair on the frontend, mirroring the existing `SupportCard`/`SupportModal` pair exactly (same Teleport/composable pattern). A new `POST /api/tribute/webhook` backend route, mirroring `POST /api/telegram/webhook`'s shape but with real HMAC-SHA256 signature verification instead of a static secret, storing one row per event in a new `donations` table.

**Tech Stack:** Vue 3 + Vite + Tailwind (frontend), Go 1.25 stdlib `net/http` + `database/sql` (backend), Postgres (prod) / SQLite (tests), Vitest + `@vue/test-utils` (frontend tests), Go `testing` (backend tests).

**Spec:** [docs/superpowers/specs/2026-09-23-donations-design.md](../specs/2026-09-23-donations-design.md)

## Global Constraints

- Go comments in English; every user-facing string in Russian (CLAUDE.md).
- No borders on cards — separation is background/shadow only (`.card` class), matching every existing card.
- The two Tribute links are fixed, hardcoded constants — no env var, no config-driven URL building (matches how `SUPPORT_TELEGRAM_URL` is defined today): `https://t.me/tribute/app?startapp=dQWl` and `https://web.tribute.tg/d/QWl`.
- `donations.amount_minor_units` is an integer (smallest currency unit) — never a float.
- The webhook must be idempotent: a retried Tribute delivery for an already-stored event must not create a second row.
- **Prerequisite, not part of this plan:** `web/src/assets/donate/heart.png` and `web/src/assets/donate/support-bird.png` must already exist on disk (the user is placing them manually) before Task 3 can run — Vite/Vitest will fail to resolve the `import` otherwise.

---

### Task 1: Donations store layer

**Files:**
- Create: `server/internal/store/migrations/011_donations.sql`
- Create: `server/internal/store/donations.go`
- Test: `server/internal/store/donations_test.go`

**Interfaces:**
- Consumes: `s.db` (`*database`, package-internal), `nullIf(s string) any` — both already in `store.go`. `newUser(t *testing.T) (*Store, *UserStore)` and `var day0 time.Time` — both already in `store_test.go`.
- Produces: `type Donation struct { ID int64; UserID, TelegramUserID, TelegramUsername string; AmountMinorUnits int64; Currency, EventType, TributeEventID, RawPayload string; CreatedAt time.Time }`, `func (s *Store) CreateDonation(d Donation, at time.Time) error`, `func (s *Store) ListDonations() ([]Donation, error)` — Task 2's `tribute.go` calls both.

- [ ] **Step 1: Write the failing test**

```go
// server/internal/store/donations_test.go
package store

import (
	"testing"
	"time"
)

func TestCreateAndListDonations(t *testing.T) {
	s, u := newUser(t)

	if err := s.CreateDonation(Donation{
		UserID: u.user, TelegramUserID: "555", TelegramUsername: "alice",
		AmountMinorUnits: 10000, Currency: "RUB", EventType: "newDonation",
		TributeEventID: "evt_1", RawPayload: `{"name":"newDonation"}`,
	}, day0); err != nil {
		t.Fatalf("CreateDonation: %v", err)
	}
	if err := s.CreateDonation(Donation{
		TelegramUserID: "999", AmountMinorUnits: 50000, Currency: "RUB", EventType: "newDonation",
		TributeEventID: "evt_2", RawPayload: `{"name":"newDonation"}`,
	}, day0.Add(time.Hour)); err != nil {
		t.Fatalf("CreateDonation (2nd): %v", err)
	}

	got, err := s.ListDonations()
	if err != nil {
		t.Fatalf("ListDonations: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListDonations returned %d rows, want 2: %+v", len(got), got)
	}
	// newest first
	if got[0].TelegramUserID != "999" || got[0].UserID != "" {
		t.Errorf("got[0] = %+v, want the unlinked donor's donation", got[0])
	}
	if got[1].UserID != u.user || got[1].AmountMinorUnits != 10000 {
		t.Errorf("got[1] = %+v, want %s's linked donation", got[1], u.user)
	}
}

func TestCreateDonationIdempotentOnEventID(t *testing.T) {
	s, _ := newUser(t)
	d := Donation{
		TelegramUserID: "555", AmountMinorUnits: 10000, Currency: "RUB",
		EventType: "newDonation", TributeEventID: "evt_dup", RawPayload: `{}`,
	}
	if err := s.CreateDonation(d, day0); err != nil {
		t.Fatalf("CreateDonation (1st): %v", err)
	}
	if err := s.CreateDonation(d, day0.Add(time.Minute)); err != nil {
		t.Fatalf("CreateDonation (retry): %v", err)
	}
	got, err := s.ListDonations()
	if err != nil {
		t.Fatalf("ListDonations: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListDonations = %d rows after a retried delivery, want 1 (idempotent)", len(got))
	}
}

func TestCreateDonationRejectsEmptyTelegramUserID(t *testing.T) {
	s, _ := newUser(t)
	err := s.CreateDonation(Donation{
		TributeEventID: "evt_x", AmountMinorUnits: 100, Currency: "RUB",
		EventType: "newDonation", RawPayload: `{}`,
	}, day0)
	if err == nil {
		t.Fatal("CreateDonation with empty telegram_user_id = nil error, want error")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd server && go test ./internal/store/ -run TestCreateAndListDonations -v`
Expected: FAIL to compile — `undefined: Donation` (the type and both methods don't exist yet).

- [ ] **Step 3: Write the migration and the minimal implementation**

```sql
-- server/internal/store/migrations/011_donations.sql
-- Migration 011: donations received via the Tribute webhook (see
-- docs/superpowers/specs/2026-09-23-donations-design.md). Read-only from the
-- admin panel's ProdDbService, same as support_messages — no workflow state.

CREATE TABLE donations (
	id                 {{.AutoID}},
	user_id            TEXT,
	telegram_user_id   TEXT NOT NULL,
	telegram_username  TEXT,
	amount_minor_units BIGINT NOT NULL,
	currency           TEXT NOT NULL,
	event_type         TEXT NOT NULL,
	tribute_event_id   TEXT NOT NULL,
	raw_payload        TEXT NOT NULL,
	created_at         TEXT NOT NULL
);
CREATE UNIQUE INDEX donations_tribute_event_id_idx ON donations(tribute_event_id);
```

```go
// server/internal/store/donations.go
package store

import (
	"fmt"
	"time"
)

// Donation is one payment event from the Tribute webhook — a new donation, a
// recurring one, or a cancellation (see internal/api/tribute.go). Read-only
// from here on; the admin panel lists them directly against this repo's
// Postgres, same as SupportMessage. ListDonations exists for tests — no
// handler in this repo calls it.
type Donation struct {
	ID                int64
	UserID            string // "" when the donor's telegram id matched no account
	TelegramUserID    string
	TelegramUsername  string // "" if Tribute didn't send one
	AmountMinorUnits  int64
	Currency          string
	EventType         string // "newDonation" | "recurrentDonation" | "cancelledDonation" | ...
	TributeEventID    string
	RawPayload        string
	CreatedAt         time.Time
}

// CreateDonation stores one Tribute webhook event. Idempotent on
// tributeEventID (via the unique index in migration 011): a retried webhook
// delivery for an event already stored is silently ignored rather than
// double-counted.
func (s *Store) CreateDonation(d Donation, at time.Time) error {
	if d.TelegramUserID == "" {
		return fmt.Errorf("empty telegram_user_id")
	}
	if d.TributeEventID == "" {
		return fmt.Errorf("empty tribute event id")
	}
	if _, err := s.db.Exec(`INSERT INTO donations
		(user_id, telegram_user_id, telegram_username, amount_minor_units, currency, event_type, tribute_event_id, raw_payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (tribute_event_id) DO NOTHING`,
		nullIf(d.UserID), d.TelegramUserID, nullIf(d.TelegramUsername), d.AmountMinorUnits,
		d.Currency, d.EventType, d.TributeEventID, d.RawPayload, at.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("create donation: %w", err)
	}
	return nil
}

// ListDonations returns every donation, newest first.
func (s *Store) ListDonations() ([]Donation, error) {
	rows, err := s.db.Query(`SELECT id, COALESCE(user_id,''), telegram_user_id, COALESCE(telegram_username,''),
		amount_minor_units, currency, event_type, tribute_event_id, raw_payload, created_at
		FROM donations ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list donations: %w", err)
	}
	defer rows.Close()
	var out []Donation
	for rows.Next() {
		var d Donation
		var createdAt string
		if err := rows.Scan(&d.ID, &d.UserID, &d.TelegramUserID, &d.TelegramUsername,
			&d.AmountMinorUnits, &d.Currency, &d.EventType, &d.TributeEventID, &d.RawPayload, &createdAt); err != nil {
			return nil, fmt.Errorf("list donations: %w", err)
		}
		t, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("list donations: parse created_at: %w", err)
		}
		d.CreatedAt = t
		out = append(out, d)
	}
	return out, rows.Err()
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd server && go test ./internal/store/ -run TestCreateDonation -v && go test ./internal/store/ -run TestCreateAndListDonations -v`
Expected: PASS (all three tests).

- [ ] **Step 5: Commit**

```bash
git add server/internal/store/migrations/011_donations.sql server/internal/store/donations.go server/internal/store/donations_test.go
git commit -m "Add donations store layer for the Tribute webhook"
```

---

### Task 2: Tribute webhook handler

**Files:**
- Modify: `server/internal/config/config.go` (add `TributeAPIKey`)
- Modify: `server/internal/api/api.go:142` (register the route, right after the Telegram webhook route)
- Modify: `server/internal/api/middleware.go:85` (exempt the new route from `checkOrigin`, alongside the Telegram webhook)
- Create: `server/internal/api/tribute.go`
- Test: `server/internal/api/tribute_test.go`

**Interfaces:**
- Consumes: `store.Donation`, `store.CreateDonation` (Task 1). `h.Store.IdentityByProviderUID("telegram", tgID)` (existing, `internal/store/identities.go`). `h.Config.TributeAPIKey` (this task). `h.Now()`, `fail(w, status, msg)`, `newTestAPIWith(t, tweak)` (existing test helper, `internal/api/api_test.go`).
- Produces: `func (h handlers) tributeWebhook(w http.ResponseWriter, r *http.Request)`, registered at `POST /api/tribute/webhook`.

- [ ] **Step 1: Write the failing test**

```go
// server/internal/api/tribute_test.go
package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grisha/serbian-app/server/internal/store"
)

const testTributeKey = "test-tribute-key"

func newTributeAPI(t *testing.T) (http.Handler, *store.Store) {
	return newTestAPIWith(t, func(d *Deps) { d.Config.TributeAPIKey = testTributeKey })
}

func signTribute(body string) string {
	mac := hmac.New(sha256.New, []byte(testTributeKey))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

// postTribute posts body to the webhook with the given signature header
// (empty = header omitted). The webhook is exempt from checkOrigin.
func postTribute(h http.Handler, sig, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/api/tribute/webhook", strings.NewReader(body))
	if sig != "" {
		r.Header.Set("trbt-signature", sig)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	return rr
}

const donationBody = `{"name":"newDonation","payload":{"id":"evt_abc","telegram_user_id":"555","telegram_username":"alice","amount":100,"currency":"RUB"}}`

func TestTributeWebhookWrongSignatureIs401(t *testing.T) {
	h, _ := newTributeAPI(t)
	rr := postTribute(h, "wrong", donationBody)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("webhook wrong signature = %d, want 401", rr.Code)
	}
}

func TestTributeWebhookMissingSignatureIs401(t *testing.T) {
	h, _ := newTributeAPI(t)
	rr := postTribute(h, "", donationBody)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("webhook missing signature = %d, want 401", rr.Code)
	}
}

func TestTributeWebhookStoresDonation(t *testing.T) {
	h, st := newTributeAPI(t)
	rr := postTribute(h, signTribute(donationBody), donationBody)
	if rr.Code != http.StatusOK {
		t.Fatalf("webhook = %d %s, want 200", rr.Code, rr.Body)
	}
	got, err := st.ListDonations()
	if err != nil {
		t.Fatalf("ListDonations: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListDonations = %d rows, want 1", len(got))
	}
	d := got[0]
	if d.TelegramUserID != "555" || d.TelegramUsername != "alice" || d.Currency != "RUB" ||
		d.EventType != "newDonation" || d.TributeEventID != "evt_abc" {
		t.Errorf("stored donation = %+v, want fields parsed from the webhook body", d)
	}
	if d.AmountMinorUnits != 10000 { // amount:100 (major units) * 100
		t.Errorf("AmountMinorUnits = %d, want 10000", d.AmountMinorUnits)
	}
}

func TestTributeWebhookLinksKnownDonor(t *testing.T) {
	h, st := newTributeAPI(t)
	uid, err := st.CreateUser("Alice")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if err := st.CreateIdentity(store.Identity{
		ID: "id_1", UserID: uid, Provider: "telegram", ProviderUID: "555", TgUsername: "alice",
	}); err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}

	rr := postTribute(h, signTribute(donationBody), donationBody)
	if rr.Code != http.StatusOK {
		t.Fatalf("webhook = %d %s, want 200", rr.Code, rr.Body)
	}
	got, err := st.ListDonations()
	if err != nil {
		t.Fatalf("ListDonations: %v", err)
	}
	if len(got) != 1 || got[0].UserID != uid {
		t.Fatalf("ListDonations = %+v, want one donation linked to %s", got, uid)
	}
}

func TestTributeWebhookRetryIsIdempotent(t *testing.T) {
	h, st := newTributeAPI(t)
	sig := signTribute(donationBody)
	postTribute(h, sig, donationBody)
	rr := postTribute(h, sig, donationBody) // Tribute retries on a slow/failed response
	if rr.Code != http.StatusOK {
		t.Fatalf("webhook retry = %d, want 200", rr.Code)
	}
	got, err := st.ListDonations()
	if err != nil {
		t.Fatalf("ListDonations: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListDonations = %d rows after a retried delivery, want 1", len(got))
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd server && go test ./internal/api/ -run TestTributeWebhook -v`
Expected: FAIL to compile — `d.Config.TributeAPIKey` and `h.tributeWebhook` don't exist yet, and `POST /api/tribute/webhook` isn't routed (404).

- [ ] **Step 3: Write the minimal implementation**

In `server/internal/config/config.go`, add a field to `Config` (after `TelegramWebhookURL string` — the last field before the struct's closing `}`):

```go
	// TributeAPIKey authenticates the Tribute donation webhook
	// (POST /api/tribute/webhook): Tribute signs every request with
	// HMAC-SHA256(key=TributeAPIKey, msg=body) in the "trbt-signature"
	// header. Empty disables the webhook (falls through as a no-op 200).
	// From TRIBUTE_API_KEY.
	TributeAPIKey string
```

and in `Load()`'s returned struct literal (after `TelegramWebhookURL: os.Getenv("TELEGRAM_WEBHOOK_URL"),`):

```go
		TributeAPIKey: os.Getenv("TRIBUTE_API_KEY"),
```

In `server/internal/api/api.go`, register the route right after the existing Telegram webhook line (`root.HandleFunc("POST /api/telegram/webhook", h.telegramWebhook)`):

```go
	root.HandleFunc("POST /api/tribute/webhook", h.tributeWebhook)
```

In `server/internal/api/middleware.go`, change the `checkOrigin` exemption (currently `if r.URL.Path == "/api/telegram/webhook" {`) to cover both paths, and broaden its doc comment:

```go
	// The Telegram and Tribute webhooks are exempt: both are server-to-server
	// POSTs from the provider's own infrastructure, which never send an
	// Origin/Referer header at all — CSRF is a browser-borne-credential
	// problem and doesn't apply here. Each has its own, unrelated
	// authentication (a shared secret / HMAC signature, checked in the
	// handler itself), so being unauthenticated with respect to origin
	// checking is not a gap.
	if r.URL.Path == "/api/telegram/webhook" || r.URL.Path == "/api/tribute/webhook" {
		next.ServeHTTP(w, r)
		return
	}
```

Create `server/internal/api/tribute.go`:

```go
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
		UserID:            userID,
		TelegramUserID:    tgID,
		TelegramUsername:  stringField(evt.Payload, "telegram_username"),
		AmountMinorUnits:  numberField(evt.Payload, "amount"),
		Currency:          stringField(evt.Payload, "currency"),
		EventType:         evt.Name,
		TributeEventID:    eventID,
		RawPayload:        string(body),
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
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd server && go test ./internal/api/ -run TestTribute -v`
Expected: PASS (all five tests).

- [ ] **Step 5: Run the full Go test suite**

Run: `cd server && go test ./...`
Expected: PASS — confirms the `middleware.go`/`api.go` edits didn't break any existing routing or CSRF test.

- [ ] **Step 6: Commit**

```bash
git add server/internal/config/config.go server/internal/api/api.go server/internal/api/middleware.go server/internal/api/tribute.go server/internal/api/tribute_test.go
git commit -m "Add Tribute donation webhook receiver"
```

---

### Task 3: DonateModal — shared state and component

**Files:**
- Create: `web/src/lib/donateModal.ts`
- Create: `web/src/components/DonateModal.vue`
- Test: `web/src/components/DonateModal.test.ts`

**Interfaces:**
- Consumes: `web/src/assets/donate/support-bird.png` (prerequisite asset, see Global Constraints). Tailwind classes `.card`, `.modal-panel`, `.btn`, `.btn-primary`, `.btn-ghost`, `.icon-btn` (already defined in `web/src/style.css`).
- Produces: `export const TRIBUTE_TELEGRAM_LINK: string`, `export const TRIBUTE_WEB_LINK: string`, `export function useDonateModal(): { open: Ref<boolean>; openModal: () => void; closeModal: () => void }` — Task 4's `DonateCard.vue` and Task 5's `AppNav.vue` both use these.

- [ ] **Step 1: Write the failing test**

```ts
// web/src/components/DonateModal.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import DonateModal from './DonateModal.vue'
import { useDonateModal, TRIBUTE_TELEGRAM_LINK, TRIBUTE_WEB_LINK } from '../lib/donateModal'

function mountModal() {
  // Teleport's real target (document.body) is outside the wrapper's DOM tree;
  // stubbing it keeps the modal inline so it can be found/asserted on.
  return mount(DonateModal, { global: { stubs: { teleport: true } } })
}

beforeEach(() => {
  const { open } = useDonateModal()
  open.value = false
})

describe('DonateModal', () => {
  it('renders nothing until the shared state opens it', () => {
    const w = mountModal()
    expect(w.find('[data-test="donate-telegram"]').exists()).toBe(false)
  })

  it('links both buttons to the Tribute donation tier', () => {
    const { openModal } = useDonateModal()
    openModal()
    const w = mountModal()

    expect(w.find('[data-test="donate-telegram"]').attributes('href')).toBe(TRIBUTE_TELEGRAM_LINK)
    expect(w.find('[data-test="donate-web"]').attributes('href')).toBe(TRIBUTE_WEB_LINK)
  })

  it('closes when the close button is clicked', async () => {
    const { open, openModal } = useDonateModal()
    openModal()
    const w = mountModal()

    await w.find('[title="Закрыть"]').trigger('click')
    expect(open.value).toBe(false)
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd web && npx vitest run src/components/DonateModal.test.ts`
Expected: FAIL — `Failed to resolve import "./DonateModal.vue"`.

- [ ] **Step 3: Write the minimal implementation**

```ts
// web/src/lib/donateModal.ts
// Shared state for the donate modal, mirroring supportModal.ts: opened from
// DonateCard at the bottom of ProfileView when the click happens outside
// Telegram (inside Telegram, DonateCard skips the modal and opens
// TRIBUTE_TELEGRAM_LINK directly — see isTelegram() in ../telegram).
// Module-level refs so the single <DonateModal> instance (mounted once, in
// AppNav) shares state with the card.
import { ref } from 'vue'

export const TRIBUTE_TELEGRAM_LINK = 'https://t.me/tribute/app?startapp=dQWl'
export const TRIBUTE_WEB_LINK = 'https://web.tribute.tg/d/QWl'

const open = ref(false)

function openModal() {
  open.value = true
}
function closeModal() {
  open.value = false
}

export function useDonateModal() {
  return { open, openModal, closeModal }
}
```

```vue
<!-- web/src/components/DonateModal.vue -->
<script setup lang="ts">
import { X } from 'lucide-vue-next'
import { useDonateModal, TRIBUTE_TELEGRAM_LINK, TRIBUTE_WEB_LINK } from '../lib/donateModal'
import supportBird from '../assets/donate/support-bird.png'

const { open, closeModal } = useDonateModal()
</script>

<template>
  <Teleport to="body">
    <Transition name="modal-overlay">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto p-4 sm:items-center"
      >
        <div class="fixed inset-0 bg-black/40" @click="closeModal" />
        <div class="card modal-panel relative z-10 w-full max-w-md p-5 text-center">
          <button class="icon-btn absolute right-3 top-3" title="Закрыть" aria-label="Закрыть" @click="closeModal">
            <X :size="19" :stroke-width="2.25" />
          </button>

          <div class="space-y-3">
            <img :src="supportBird" alt="" class="mx-auto h-24 w-24 object-contain" />
            <p class="text-lg font-extrabold">Šoljica kafe ☕ (Чашечка кофе)</p>
            <p class="text-sm text-[var(--muted)]">
              Hvala što si tu! 💛 Ваша поддержка помогает добавлять новые уроки, слова и улучшать приложение.
            </p>

            <div class="flex flex-col gap-2 sm:flex-row">
              <a
                :href="TRIBUTE_TELEGRAM_LINK"
                target="_blank"
                rel="noopener"
                class="btn btn-primary flex-1"
                data-test="donate-telegram"
              >
                Поддержать через телеграм
              </a>
              <a
                :href="TRIBUTE_WEB_LINK"
                target="_blank"
                rel="noopener"
                class="btn btn-ghost flex-1"
                data-test="donate-web"
              >
                Поддержать
              </a>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-overlay-enter-active,
.modal-overlay-leave-active {
  transition: opacity 0.18s ease;
}
.modal-overlay-enter-from,
.modal-overlay-leave-to {
  opacity: 0;
}
.modal-overlay-enter-active .modal-panel,
.modal-overlay-leave-active .modal-panel {
  transition: transform 0.18s ease, opacity 0.18s ease;
}
.modal-overlay-enter-from .modal-panel,
.modal-overlay-leave-to .modal-panel {
  opacity: 0;
  transform: scale(0.95) translateY(6px);
}
</style>
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd web && npx vitest run src/components/DonateModal.test.ts`
Expected: PASS (all three tests).

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/donateModal.ts web/src/components/DonateModal.vue web/src/components/DonateModal.test.ts
git commit -m "Add DonateModal and its shared open/close state"
```

---

### Task 4: DonateCard component

**Files:**
- Create: `web/src/components/DonateCard.vue`
- Test: `web/src/components/DonateCard.test.ts`

**Interfaces:**
- Consumes: `useDonateModal`, `TRIBUTE_TELEGRAM_LINK` (Task 3, `../lib/donateModal`). `isTelegram()` (existing, `../telegram`). `web/src/assets/donate/heart.png` (prerequisite asset).
- Produces: default-exported `DonateCard.vue` component — Task 5's `ProfileView.vue` renders it.

- [ ] **Step 1: Write the failing test**

```ts
// web/src/components/DonateCard.test.ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import DonateCard from './DonateCard.vue'
import { useDonateModal, TRIBUTE_TELEGRAM_LINK } from '../lib/donateModal'
import { isTelegram } from '../telegram'

vi.mock('../telegram', () => ({ isTelegram: vi.fn() }))

beforeEach(() => {
  const { open } = useDonateModal()
  open.value = false
  vi.mocked(isTelegram).mockReturnValue(false)
})
afterEach(() => vi.restoreAllMocks())

describe('DonateCard', () => {
  it('opens the shared donate modal when clicked outside Telegram', async () => {
    const { open } = useDonateModal()
    const w = mount(DonateCard)

    expect(open.value).toBe(false)
    await w.find('[data-test="open-donate"]').trigger('click')
    expect(open.value).toBe(true)
  })

  it('opens the Tribute Telegram link directly inside Telegram, without the modal', async () => {
    vi.mocked(isTelegram).mockReturnValue(true)
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    const { open } = useDonateModal()
    const w = mount(DonateCard)

    await w.find('[data-test="open-donate"]').trigger('click')

    expect(openSpy).toHaveBeenCalledWith(TRIBUTE_TELEGRAM_LINK, '_blank')
    expect(open.value).toBe(false)
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd web && npx vitest run src/components/DonateCard.test.ts`
Expected: FAIL — `Failed to resolve import "./DonateCard.vue"`.

- [ ] **Step 3: Write the minimal implementation**

```vue
<!-- web/src/components/DonateCard.vue -->
<script setup lang="ts">
import { Heart } from 'lucide-vue-next'
import { useDonateModal, TRIBUTE_TELEGRAM_LINK } from '../lib/donateModal'
import { isTelegram } from '../telegram'
import heartIcon from '../assets/donate/heart.png'

const { openModal } = useDonateModal()

function onClick() {
  if (isTelegram()) {
    window.open(TRIBUTE_TELEGRAM_LINK, '_blank')
  } else {
    openModal()
  }
}
</script>

<template>
  <div class="card flex flex-col items-center gap-3 p-5 text-center sm:flex-row sm:text-left">
    <img :src="heartIcon" alt="" class="h-16 w-16 shrink-0 object-contain" />
    <div class="flex-1">
      <p class="font-extrabold">Поддержать проект</p>
      <p class="text-sm text-[var(--muted)]">Помогите развитию курса — новые уроки, слова и улучшения.</p>
    </div>
    <button class="btn btn-primary flex shrink-0 items-center gap-1.5" data-test="open-donate" @click="onClick">
      <Heart :size="17" :stroke-width="2.25" />
      Podrži
    </button>
  </div>
</template>
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd web && npx vitest run src/components/DonateCard.test.ts`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add web/src/components/DonateCard.vue web/src/components/DonateCard.test.ts
git commit -m "Add DonateCard component"
```

---

### Task 5: Wire DonateCard/DonateModal into the app

**Files:**
- Modify: `web/src/views/ProfileView.vue:13` (import), `web/src/views/ProfileView.vue:191` (render)
- Modify: `web/src/components/AppNav.vue:17` (import), `web/src/components/AppNav.vue:137` (mount)

**Interfaces:**
- Consumes: `DonateCard.vue` (Task 4), `DonateModal.vue` (Task 3). No new interfaces produced — this task only wires existing components into the page tree.

- [ ] **Step 1: Add the import and render call to ProfileView.vue**

In `web/src/views/ProfileView.vue`, add an import next to the existing one (line 13, `import SupportCard from '../components/SupportCard.vue'`):

```ts
import DonateCard from '../components/DonateCard.vue'
```

and render it immediately after the existing support card (line 191, `<SupportCard v-if="me" />`):

```vue
    <SupportCard v-if="me" />
    <DonateCard v-if="me" />
```

- [ ] **Step 2: Mount DonateModal in AppNav.vue**

In `web/src/components/AppNav.vue`, add an import next to the existing one (line 17, `import SupportModal from './SupportModal.vue'`):

```ts
import DonateModal from './DonateModal.vue'
```

and mount it next to the existing modal (line 137, `<SupportModal />`):

```vue
  <SupportModal />
  <DonateModal />
```

- [ ] **Step 3: Run the full frontend test suite**

Run: `cd web && npx vitest run && npx vue-tsc -b`
Expected: PASS — confirms the wiring compiles and no existing `ProfileView`/`AppNav` test broke.

- [ ] **Step 4: Manual smoke test**

Run: `make dev`, open `http://localhost:5173`, log in, go to the profile page, confirm the new donate card renders below the support card, and clicking its button opens the modal with both Tribute links (this is a plain browser, so no Telegram deep link path to check here — that only applies inside an actual Telegram Mini App session, which needs a real device/Telegram client to verify and is out of scope for local `make dev`).

- [ ] **Step 5: Commit**

```bash
git add web/src/views/ProfileView.vue web/src/components/AppNav.vue
git commit -m "Wire DonateCard/DonateModal into the profile page"
```

---

## After this plan

`donations` rows are now being written, but nothing reads them yet — that's the companion plan in `ucimo-content-admin`: [docs/superpowers/plans/2026-09-23-donations-admin.md](../../../ucimo-content-admin/docs/superpowers/plans/2026-09-23-donations-admin.md).

Before relying on real data: trigger one real Tribute test donation (or wait for the first live one) and check the stored row's `telegram_user_id`/`amount_minor_units`/`currency` look right — `tribute.go`'s field-name and unit-scale assumptions are a best-effort read of Tribute's docs, not a confirmed payload (see that file's doc comment). `raw_payload` has the full body if something needs correcting.
