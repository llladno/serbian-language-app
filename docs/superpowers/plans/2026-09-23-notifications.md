# In-App Notifications Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let Грiша compose a short message in ucimo-content-admin, target all/some/one users, and have it show up as an unread badge + dropdown on a bell icon in serbian-app — no websocket yet, polling only.

**Architecture:** Two new Postgres tables live in serbian-app (`notifications`, `notification_recipients`), written directly by ucimo-content-admin via `ProdDbService` (the same point-INSERT-grant pattern already used for Telegram broadcasts), and read/mark-read by two new serbian-app Go endpoints behind `requireSession`. The Vue frontend polls those endpoints from a bell icon in `AppNav.vue`.

**Tech Stack:** Go 1.25 (stdlib `net/http`, `modernc.org/sqlite`, `pgx/v5`) + Vue 3/Tailwind for serbian-app; NestJS + `pg` (via `ProdDbService`) + Vue 3/Tailwind for ucimo-content-admin.

**Spec:** [docs/superpowers/specs/2026-09-23-notifications-design.md](../specs/2026-09-23-notifications-design.md)

## Global Constraints

- Retention: an unread notification never expires; a read one keeps showing for **30 days** after `read_at`, then a query-time filter (not a delete job) hides it.
- Poll interval: **45 seconds** (`POLL_INTERVAL_MS = 45000` in `web/src/lib/notifications.ts`).
- "All users" targeting materializes a `notification_recipients` row per user **at send time** — future registrations are never retroactively included.
- Admin's user picker (`GET /notifications/candidates`) lists **every** user, not just Telegram-linked ones (unlike `broadcasts/candidates`).
- Notification text supports exactly one piece of markup: `[label](url)`, `url` restricted to `http://`, `https://`, or a `/`-relative path. No bold/lists/headings — rendered by a small regex helper, not `markdown-it`.
- Opening the bell dropdown marks the **whole visible list** read at once — no per-item mark-read.
- `ucimo_admin_ro` gets **only** `INSERT` on the two new tables — it never reads them back (no read-receipt/history UI, per spec's "Вне скоупа").
- JSON field names are `snake_case` end to end (`created_at`, `unread_count`), matching every existing serbian-app endpoint.
- Go doc-comments in English; user-facing/admin copy in Russian (per `serbian-app/CLAUDE.md`).

---

## Part A — serbian-app: storage + Go API

### Task 1: `notifications`/`notification_recipients` tables and store accessors

**Files:**
- Create: `server/internal/store/migrations/010_notifications.sql`
- Create: `server/internal/store/notifications.go`
- Create: `server/internal/store/notifications_test.go`

**Interfaces:**
- Consumes: nothing new (leaf addition to the store package).
- Produces: `type Notification struct{ID int64; Text string; CreatedAt time.Time; Read bool}`, `(*Store).ListNotificationsForUser(userID string, now time.Time) ([]Notification, error)`, `(*Store).MarkNotificationsRead(userID string, now time.Time) error` — Task 2's API handlers depend on both. Also `(*Store).SeedNotificationForTest(userID, text string, at time.Time) (int64, error)` and `(*Store).SeedNotificationReadAtForTest(notificationID int64, userID string, readAt time.Time) error` — test-only seams (same pattern as `telegram.SetAPIBaseForTesting`), needed because production never creates a notification from Go (only ucimo-content-admin does, via a direct Postgres write) so there is no ordinary writer to seed fixtures with. Task 2's tests depend on both seam functions.

- [ ] **Step 1: Write the migration**

`server/internal/store/migrations/010_notifications.sql`:
```sql
-- Migration 010: notifications — in-app notifications composed in
-- ucimo-content-admin's "Уведомления" page, read by the bell dropdown in
-- serbian-app's AppNav. notification_recipients materializes one row per
-- targeted user at send time (including "all" — future registrations are
-- not retroactively included). See
-- docs/superpowers/specs/2026-09-23-notifications-design.md.
--
-- Only ucimo-content-admin ever INSERTs into these two tables (a direct
-- Postgres write via ProdDbService) — serbian-app's own Go code only reads
-- and updates read_at.

CREATE TABLE IF NOT EXISTS notifications (
	id         {{.AutoID}},
	text       TEXT NOT NULL,
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS notification_recipients (
	id              {{.AutoID}},
	notification_id INTEGER NOT NULL,
	user_id         TEXT NOT NULL,
	read_at         TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS notification_recipients_user_idx
	ON notification_recipients (user_id, read_at);
```

- [ ] **Step 2: Write the failing tests**

`server/internal/store/notifications_test.go`:
```go
package store

import (
	"testing"
	"time"
)

func TestListNotificationsForUserOnlyOwnRows(t *testing.T) {
	s := newStore(t)
	if _, err := s.SeedNotificationForTest("user-a", "for A", day0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SeedNotificationForTest("user-b", "for B", day0); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListNotificationsForUser("user-a", day0)
	if err != nil {
		t.Fatalf("ListNotificationsForUser: %v", err)
	}
	if len(got) != 1 || got[0].Text != "for A" {
		t.Fatalf("ListNotificationsForUser(user-a) = %+v, want just \"for A\"", got)
	}
}

func TestListNotificationsForUserOrdersNewestFirst(t *testing.T) {
	s := newStore(t)
	if _, err := s.SeedNotificationForTest("user-a", "older", day0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SeedNotificationForTest("user-a", "newer", day0.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListNotificationsForUser("user-a", day0.Add(time.Hour))
	if err != nil {
		t.Fatalf("ListNotificationsForUser: %v", err)
	}
	if len(got) != 2 || got[0].Text != "newer" || got[1].Text != "older" {
		t.Fatalf("ListNotificationsForUser order = %+v, want [newer, older]", got)
	}
}

func TestListNotificationsForUserFiltersReadPastRetention(t *testing.T) {
	s := newStore(t)
	readLongAgo, err := s.SeedNotificationForTest("user-a", "read long ago", day0)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SeedNotificationReadAtForTest(readLongAgo, "user-a", day0); err != nil {
		t.Fatal(err)
	}
	readRecently, err := s.SeedNotificationForTest("user-a", "read recently", day0)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SeedNotificationReadAtForTest(readRecently, "user-a", day0.Add(29*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SeedNotificationForTest("user-a", "still unread", day0); err != nil {
		t.Fatal(err)
	}

	now := day0.Add(35 * 24 * time.Hour) // read_at cutoff = now - 30d = day0 + 5d
	got, err := s.ListNotificationsForUser("user-a", now)
	if err != nil {
		t.Fatalf("ListNotificationsForUser: %v", err)
	}
	texts := map[string]bool{}
	for _, n := range got {
		texts[n.Text] = true
	}
	if texts["read long ago"] {
		t.Errorf("got %+v, \"read long ago\" (read 35 days before now) should have expired", got)
	}
	if !texts["read recently"] {
		t.Errorf("got %+v, \"read recently\" (read 6 days before now) should still be present", got)
	}
	if !texts["still unread"] {
		t.Errorf("got %+v, unread notifications should never expire", got)
	}
}

func TestMarkNotificationsReadIsIdempotentAndScopedToUser(t *testing.T) {
	s := newStore(t)
	if _, err := s.SeedNotificationForTest("user-a", "for A", day0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SeedNotificationForTest("user-b", "for B", day0); err != nil {
		t.Fatal(err)
	}

	if err := s.MarkNotificationsRead("user-a", day0.Add(time.Hour)); err != nil {
		t.Fatalf("MarkNotificationsRead: %v", err)
	}
	gotA, err := s.ListNotificationsForUser("user-a", day0.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(gotA) != 1 || !gotA[0].Read {
		t.Fatalf("user-a's notification = %+v, want Read=true", gotA)
	}
	gotB, err := s.ListNotificationsForUser("user-b", day0.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(gotB) != 1 || gotB[0].Read {
		t.Fatalf("user-b's notification = %+v, want Read=false (mark-read must not leak across users)", gotB)
	}

	// Idempotent: calling again must not error or panic on the zero-row update.
	if err := s.MarkNotificationsRead("user-a", day0.Add(2*time.Hour)); err != nil {
		t.Fatalf("MarkNotificationsRead (second call): %v", err)
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `cd server && go test ./internal/store/... -run Notification -v`
Expected: FAIL — `Notification`, `ListNotificationsForUser`, `MarkNotificationsRead`, `SeedNotificationForTest`, `SeedNotificationReadAtForTest` undefined.

- [ ] **Step 4: Implement**

`server/internal/store/notifications.go`:
```go
package store

import (
	"fmt"
	"time"
)

// Notification is one row a user sees in the bell dropdown — the
// notifications/notification_recipients join for a single recipient.
type Notification struct {
	ID        int64
	Text      string
	CreatedAt time.Time
	Read      bool
}

// notificationRetention is how long a *read* notification keeps showing up
// in ListNotificationsForUser after it was read. Unread notifications have
// no expiry.
const notificationRetention = 30 * 24 * time.Hour

// ListNotificationsForUser returns userID's notifications, newest first,
// capped at 50: every unread one, plus read ones from the last 30 days.
func (s *Store) ListNotificationsForUser(userID string, now time.Time) ([]Notification, error) {
	cutoff := now.Add(-notificationRetention).UTC().Format(time.RFC3339)
	rows, err := s.db.Query(`
		SELECT n.id, n.text, n.created_at, nr.read_at
		FROM notifications n
		JOIN notification_recipients nr ON nr.notification_id = n.id
		WHERE nr.user_id = ? AND (nr.read_at = '' OR nr.read_at > ?)
		ORDER BY n.created_at DESC
		LIMIT 50`, userID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()
	var out []Notification
	for rows.Next() {
		var n Notification
		var createdAt, readAt string
		if err := rows.Scan(&n.ID, &n.Text, &createdAt, &readAt); err != nil {
			return nil, fmt.Errorf("list notifications: %w", err)
		}
		t, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("list notifications: parse created_at: %w", err)
		}
		n.CreatedAt = t
		n.Read = readAt != ""
		out = append(out, n)
	}
	return out, rows.Err()
}

// MarkNotificationsRead marks every currently-unread notification of userID
// as read at now. Idempotent — a second call touches zero rows and returns
// no error.
func (s *Store) MarkNotificationsRead(userID string, now time.Time) error {
	_, err := s.db.Exec(`UPDATE notification_recipients SET read_at = ? WHERE user_id = ? AND read_at = ''`,
		now.UTC().Format(time.RFC3339), userID)
	if err != nil {
		return fmt.Errorf("mark notifications read: %w", err)
	}
	return nil
}

// SeedNotificationForTest creates one notification with a single recipient,
// bypassing the normal write path — production notifications are always
// created by ucimo-content-admin writing directly into Postgres (see
// docs/superpowers/specs/2026-09-23-notifications-design.md), so there is no
// ordinary Go writer to seed fixtures with. Exported only so
// internal/api's tests can set up fixtures too; mirrors
// telegram.SetAPIBaseForTesting's "exported test seam" pattern.
func (s *Store) SeedNotificationForTest(userID, text string, at time.Time) (int64, error) {
	createdAt := at.UTC().Format(time.RFC3339)
	if _, err := s.db.Exec(`INSERT INTO notifications (text, created_at) VALUES (?, ?)`, text, createdAt); err != nil {
		return 0, fmt.Errorf("seed notification: %w", err)
	}
	var id int64
	if err := s.db.QueryRow(`SELECT id FROM notifications WHERE text = ? AND created_at = ? ORDER BY id DESC LIMIT 1`,
		text, createdAt).Scan(&id); err != nil {
		return 0, fmt.Errorf("seed notification: find id: %w", err)
	}
	if _, err := s.db.Exec(`INSERT INTO notification_recipients (notification_id, user_id, read_at) VALUES (?, ?, '')`,
		id, userID); err != nil {
		return 0, fmt.Errorf("seed notification: recipient: %w", err)
	}
	return id, nil
}

// SeedNotificationReadAtForTest backdates one recipient row's read_at. Test
// seam only, like SeedNotificationForTest.
func (s *Store) SeedNotificationReadAtForTest(notificationID int64, userID string, readAt time.Time) error {
	_, err := s.db.Exec(`UPDATE notification_recipients SET read_at = ? WHERE notification_id = ? AND user_id = ?`,
		readAt.UTC().Format(time.RFC3339), notificationID, userID)
	if err != nil {
		return fmt.Errorf("seed notification read_at: %w", err)
	}
	return nil
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd server && go test ./internal/store/... -run Notification -v`
Expected: PASS (all four tests)

- [ ] **Step 6: Run the full store suite and commit**

Run: `cd server && go test ./internal/store/...`
Expected: PASS

```bash
git add server/internal/store/migrations/010_notifications.sql server/internal/store/notifications.go server/internal/store/notifications_test.go
git commit -m "$(cat <<'EOF'
Add notifications/notification_recipients tables and store accessors

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: `GET /api/me/notifications` and `POST /api/me/notifications/mark-read`

**Files:**
- Create: `server/internal/api/notifications.go`
- Create: `server/internal/api/notifications_test.go`
- Modify: `server/internal/api/api.go`

**Interfaces:**
- Consumes: `store.Notification`, `(*store.Store).ListNotificationsForUser`, `(*store.Store).MarkNotificationsRead`, `(*store.Store).SeedNotificationForTest` (Task 1).
- Produces: two routes wired behind `requireSession` — `GET /api/me/notifications` → `{items: [{id, text, created_at, read}], unread_count}`, `POST /api/me/notifications/mark-read` → `{status: "ok"}`. Task 3's frontend `api.ts` calls both by exact path.

- [ ] **Step 1: Write the failing tests**

`server/internal/api/notifications_test.go`:
```go
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
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd server && go test ./internal/api/... -run Notification -v`
Expected: FAIL — routes 404 (handlers not wired yet, `notifications.go` doesn't exist).

- [ ] **Step 3: Implement the handlers**

`server/internal/api/notifications.go`:
```go
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
```

- [ ] **Step 4: Register the routes**

In `server/internal/api/api.go`, add right after the existing `POST /api/me/support` line (inside the account-scoped `requireSession` block):
```go
	root.HandleFunc("POST /api/me/support", h.requireSession(h.createSupportMessage))
	root.HandleFunc("GET /api/me/notifications", h.requireSession(h.listNotifications))
	root.HandleFunc("POST /api/me/notifications/mark-read", h.requireSession(h.markNotificationsRead))
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd server && go test ./internal/api/... -run Notification -v`
Expected: PASS (all five tests)

- [ ] **Step 6: Run the full API suite and commit**

Run: `cd server && go test ./internal/api/...`
Expected: PASS

```bash
git add server/internal/api/notifications.go server/internal/api/notifications_test.go server/internal/api/api.go
git commit -m "$(cat <<'EOF'
Add GET /api/me/notifications and POST /api/me/notifications/mark-read

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

## Part B — serbian-app: frontend bell

### Task 3: `lib/notifications.ts` — polling composable and link-only text renderer

**Files:**
- Modify: `web/src/types.ts`
- Modify: `web/src/api.ts`
- Create: `web/src/lib/notifications.ts`
- Create: `web/src/lib/notifications.test.ts`

**Interfaces:**
- Consumes: `api` object / `request()` (`web/src/api.ts`'s existing pattern).
- Produces: `interface Notification{id, text, created_at, read}`, `interface NotificationsResponse{items, unread_count}` (`types.ts`); `api.getNotifications(): Promise<NotificationsResponse>`, `api.markNotificationsRead(): Promise<{status: string}>` (`api.ts`); `useNotifications()` returning `{items, unreadCount, refresh, markRead, start, stop}` and `renderNotificationText(text: string): string` (`lib/notifications.ts`) — Task 4's `NotificationBell.vue` depends on all of these exact names.

- [ ] **Step 1: Add the types**

In `web/src/types.ts`, add near `Health`/`TelegramStart`:
```ts
export interface Notification {
  id: number
  text: string
  created_at: string
  read: boolean
}

export interface NotificationsResponse {
  items: Notification[]
  unread_count: number
}
```

- [ ] **Step 2: Add the API calls**

In `web/src/api.ts`, add `NotificationsResponse` to the type-only import block at the top (alongside `Health`, `TelegramStart`, etc.), then add near `sendSupportMessage` inside the `api` object:
```ts
  getNotifications: () => request<NotificationsResponse>('/me/notifications'),
  markNotificationsRead: () => request<{ status: string }>('/me/notifications/mark-read', { method: 'POST' }),
```

- [ ] **Step 3: Write the failing tests**

`web/src/lib/notifications.test.ts`:
```ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { renderNotificationText, useNotifications } from './notifications'
import { api } from '../api'

// useNotifications calls onUnmounted, so it must run inside a component
// setup context — same pattern as useTelegramStart.test.ts.
function mountHarness() {
  let exposed!: ReturnType<typeof useNotifications>
  const Harness = defineComponent({
    setup() {
      exposed = useNotifications()
      return () => h('div')
    },
  })
  mount(Harness)
  return exposed
}

afterEach(() => vi.restoreAllMocks())

describe('renderNotificationText', () => {
  it('escapes raw HTML', () => {
    expect(renderNotificationText('<b>hi</b> & bye')).toBe('&lt;b&gt;hi&lt;/b&gt; &amp; bye')
  })

  it('turns [label](url) into a link for http(s) and relative urls', () => {
    expect(renderNotificationText('Смотри [тут](https://ucimo.ru/course)')).toBe(
      'Смотри <a href="https://ucimo.ru/course" target="_blank" rel="noopener">тут</a>',
    )
    expect(renderNotificationText('[Курс](/course)')).toBe(
      '<a href="/course" target="_blank" rel="noopener">Курс</a>',
    )
  })

  it('does not turn a javascript: url into a link', () => {
    const input = '[кликни](javascript:alert(1))'
    expect(renderNotificationText(input)).toBe(
      '[кликни](javascript:alert(1))'.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;'),
    )
  })

  it('converts newlines to <br>', () => {
    expect(renderNotificationText('строка1\nстрока2')).toBe('строка1<br>строка2')
  })
})

describe('useNotifications', () => {
  beforeEach(() => vi.restoreAllMocks())

  it('refresh() populates items and unreadCount from the API', async () => {
    vi.spyOn(api, 'getNotifications').mockResolvedValue({
      items: [{ id: 1, text: 'hi', created_at: '2026-09-23T00:00:00Z', read: false }],
      unread_count: 1,
    })
    const { items, unreadCount, refresh } = mountHarness()
    await refresh()
    expect(items.value).toHaveLength(1)
    expect(unreadCount.value).toBe(1)
  })

  it('markRead() optimistically clears unreadCount and calls the API', async () => {
    vi.spyOn(api, 'getNotifications').mockResolvedValue({
      items: [{ id: 1, text: 'hi', created_at: '2026-09-23T00:00:00Z', read: false }],
      unread_count: 1,
    })
    const markSpy = vi.spyOn(api, 'markNotificationsRead').mockResolvedValue({ status: 'ok' })
    const { unreadCount, refresh, markRead } = mountHarness()
    await refresh()
    await markRead()
    expect(unreadCount.value).toBe(0)
    expect(markSpy).toHaveBeenCalled()
  })

  it('markRead() is a no-op when nothing is unread', async () => {
    const markSpy = vi.spyOn(api, 'markNotificationsRead').mockResolvedValue({ status: 'ok' })
    const { markRead } = mountHarness()
    await markRead()
    expect(markSpy).not.toHaveBeenCalled()
  })
})
```

- [ ] **Step 4: Run the tests to verify they fail**

Run: `cd web && npx vitest run src/lib/notifications.test.ts`
Expected: FAIL — `./notifications` module doesn't exist yet.

- [ ] **Step 5: Implement**

`web/src/lib/notifications.ts`:
```ts
// Polling-based unread-notifications state for the bell dropdown
// (NotificationBell.vue). No WebSocket yet — see
// docs/superpowers/specs/2026-09-23-notifications-design.md.
import { onUnmounted, ref } from 'vue'
import { api } from '../api'
import type { Notification } from '../types'

const POLL_INTERVAL_MS = 45000

export function useNotifications() {
  const items = ref<Notification[]>([])
  const unreadCount = ref(0)
  let timer: ReturnType<typeof setInterval> | undefined

  async function refresh() {
    try {
      const res = await api.getNotifications()
      items.value = res.items
      unreadCount.value = res.unread_count
    } catch {
      // Best-effort polling — a transient failure just retries next tick.
    }
  }

  async function markRead() {
    if (unreadCount.value === 0) return
    items.value = items.value.map((n) => ({ ...n, read: true }))
    unreadCount.value = 0
    try {
      await api.markNotificationsRead()
    } catch {
      // Best-effort: the next refresh() restores the true count if this failed.
    }
  }

  function start() {
    refresh()
    timer = setInterval(refresh, POLL_INTERVAL_MS)
  }
  function stop() {
    if (timer) clearInterval(timer)
    timer = undefined
  }
  onUnmounted(stop)

  return { items, unreadCount, refresh, markRead, start, stop }
}

// renderNotificationText escapes text as HTML, then turns [label](url) into
// a link — url must start with "http://", "https://", or "/" (blocks
// javascript: and other unsafe schemes) — and remaining newlines into <br>.
// Deliberately not full markdown (see the design doc): only this one
// pattern is recognized, so admin copy can never render an unexpected
// heading/list/table.
const LINK_PATTERN = /\[([^\]]+)\]\((https?:\/\/[^\s)]+|\/[^\s)]*)\)/g

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

export function renderNotificationText(text: string): string {
  const escaped = escapeHtml(text)
  const withLinks = escaped.replace(
    LINK_PATTERN,
    (_match, label: string, url: string) => `<a href="${url}" target="_blank" rel="noopener">${label}</a>`,
  )
  return withLinks.replace(/\n/g, '<br>')
}
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd web && npx vitest run src/lib/notifications.test.ts`
Expected: PASS (all 7 tests)

- [ ] **Step 7: Type-check and commit**

Run: `cd web && npx vue-tsc -b`
Expected: no errors

```bash
git add web/src/types.ts web/src/api.ts web/src/lib/notifications.ts web/src/lib/notifications.test.ts
git commit -m "$(cat <<'EOF'
Add useNotifications polling composable and link-only text renderer

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: `NotificationBell.vue` — icon, unread dot, dropdown

**Files:**
- Create: `web/src/components/NotificationBell.vue`
- Create: `web/src/components/NotificationBell.test.ts`
- Modify: `web/src/components/AppNav.vue`

**Interfaces:**
- Consumes: `useNotifications`, `renderNotificationText` (Task 3).
- Produces: `<NotificationBell />` — a self-contained component (owns its own open/closed + polling state), mounted once from `AppNav.vue`. No exported interface beyond the component itself.

- [ ] **Step 1: Write the failing tests**

`web/src/components/NotificationBell.test.ts`:
```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import NotificationBell from './NotificationBell.vue'
import { api } from '../api'

beforeEach(() => vi.restoreAllMocks())

describe('NotificationBell', () => {
  it('shows the unread dot when there are unread notifications', async () => {
    vi.spyOn(api, 'getNotifications').mockResolvedValue({
      items: [{ id: 1, text: 'hi', created_at: '2026-09-23T00:00:00Z', read: false }],
      unread_count: 1,
    })
    const w = mount(NotificationBell)
    await flushPromises()
    expect(w.find('[data-test="notification-dot"]').exists()).toBe(true)
  })

  it('hides the dot when there is nothing unread', async () => {
    vi.spyOn(api, 'getNotifications').mockResolvedValue({ items: [], unread_count: 0 })
    const w = mount(NotificationBell)
    await flushPromises()
    expect(w.find('[data-test="notification-dot"]').exists()).toBe(false)
  })

  it('marks everything read and clears the dot when the dropdown opens', async () => {
    vi.spyOn(api, 'getNotifications').mockResolvedValue({
      items: [{ id: 1, text: 'hi', created_at: '2026-09-23T00:00:00Z', read: false }],
      unread_count: 1,
    })
    const markSpy = vi.spyOn(api, 'markNotificationsRead').mockResolvedValue({ status: 'ok' })
    const w = mount(NotificationBell)
    await flushPromises()
    expect(w.find('[data-test="notification-dot"]').exists()).toBe(true)

    await w.find('[data-test="open-notifications"]').trigger('click')
    await flushPromises()

    expect(markSpy).toHaveBeenCalled()
    expect(w.find('[data-test="notification-dot"]').exists()).toBe(false)
  })

  it('renders notification text with links via renderNotificationText', async () => {
    vi.spyOn(api, 'getNotifications').mockResolvedValue({
      items: [{ id: 1, text: 'Зайди [сюда](https://ucimo.ru)', created_at: '2026-09-23T00:00:00Z', read: false }],
      unread_count: 1,
    })
    vi.spyOn(api, 'markNotificationsRead').mockResolvedValue({ status: 'ok' })
    const w = mount(NotificationBell)
    await flushPromises()
    await w.find('[data-test="open-notifications"]').trigger('click')
    await flushPromises()
    const link = w.find('a[href="https://ucimo.ru"]')
    expect(link.exists()).toBe(true)
    expect(link.text()).toBe('сюда')
  })
})
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd web && npx vitest run src/components/NotificationBell.test.ts`
Expected: FAIL — `./NotificationBell.vue` doesn't exist.

- [ ] **Step 3: Implement**

`web/src/components/NotificationBell.vue`:
```vue
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Bell } from 'lucide-vue-next'
import { useNotifications, renderNotificationText } from '../lib/notifications'

const { items, unreadCount, markRead, start } = useNotifications()
const open = ref(false)

function toggle() {
  open.value = !open.value
  if (open.value) markRead()
}
function close() {
  open.value = false
}

onMounted(start)
</script>

<template>
  <div class="relative">
    <button
      class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-[var(--muted)] transition hover:bg-[var(--bg-soft)] hover:text-[var(--fg)]"
      title="Уведомления"
      aria-label="Уведомления"
      data-test="open-notifications"
      @click="toggle"
    >
      <Bell :size="17" :stroke-width="2.25" />
      <span
        v-if="unreadCount > 0"
        class="absolute right-1.5 top-1.5 h-2 w-2 rounded-full"
        style="background: var(--accent)"
        data-test="notification-dot"
      />
    </button>

    <div v-if="open" class="fixed inset-0 z-10" @click="close" />

    <div v-if="open" class="card absolute right-0 top-full z-20 mt-2 max-h-96 w-80 max-w-[90vw] overflow-y-auto p-2">
      <p v-if="items.length === 0" class="p-3 text-center text-sm text-[var(--muted)]">Пока пусто</p>
      <div
        v-for="n in items"
        :key="n.id"
        class="rounded-2xl p-3 text-sm"
        :class="n.read ? '' : 'bg-[var(--accent-soft)]'"
        v-html="renderNotificationText(n.text)"
      />
    </div>
  </div>
</template>
```

- [ ] **Step 4: Wire it into `AppNav.vue`**

In `web/src/components/AppNav.vue`, add the import (alphabetically with the other local imports):
```ts
import NotificationBell from './NotificationBell.vue'
```
Then insert `<NotificationBell />` right after the theme-toggle `</button>` and before the Headphones support `<button>`:
```html
      <button
        class="ml-auto flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-[var(--muted)] transition hover:bg-[var(--bg-soft)] hover:text-[var(--fg)]"
        :title="`Тема: ${themeMeta.label}`"
        @click="cycleTheme()"
      >
        <component :is="themeIcon" :size="17" :stroke-width="2.25" />
      </button>

      <NotificationBell />

      <button
        class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-[var(--muted)] transition hover:bg-[var(--bg-soft)] hover:text-[var(--fg)]"
        title="Поддержка"
```
(only the theme button's closing tag and the `NotificationBell` line are new — the Headphones button below is unchanged, shown for anchoring.)

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd web && npx vitest run src/components/NotificationBell.test.ts`
Expected: PASS (all 4 tests)

- [ ] **Step 6: Run the full frontend suite (confirms `AppNav.test.ts` still passes with the bell mounted) and type-check**

Run: `cd web && npx vitest run && npx vue-tsc -b`
Expected: PASS, no type errors. (`AppNav.test.ts`'s existing test doesn't mock `api.getNotifications` — its unmocked `fetch` will reject in jsdom and `useNotifications`'s `refresh()` silently swallows that, same as `LoginView.test.ts` already relies on for `api.health()`. If this assumption is wrong and the suite fails here, mock `api.getNotifications` in `AppNav.test.ts`'s `beforeEach` the same way `notifications.test.ts` does before re-running.)

- [ ] **Step 7: Commit**

```bash
git add web/src/components/NotificationBell.vue web/src/components/NotificationBell.test.ts web/src/components/AppNav.vue
git commit -m "$(cat <<'EOF'
Add NotificationBell — unread dot + dropdown, wired into AppNav

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

## Part C — ucimo-content-admin: composer backend

### Task 5: `notifications` NestJS module

**Files:**
- Create: `server/src/notifications/notifications.service.ts`
- Create: `server/src/notifications/notifications.controller.ts`
- Create: `server/src/notifications/notifications.module.ts`
- Create: `server/src/notifications/dto/send-notification.dto.ts`
- Create: `server/src/notifications/notifications.service.spec.ts`
- Create: `server/src/notifications/notifications.controller.spec.ts`
- Modify: `server/src/app.module.ts`

**Interfaces:**
- Consumes: `ProdDbService` (`server/src/prod-db/prod-db.service.ts`, existing).
- Produces: `GET /notifications/candidates` → `NotificationCandidate[]`, `POST /notifications` (`SendNotificationDto`) → `{count: number}`. Task 6's `web/src/api/notifications.ts` calls both by exact path/shape.

- [ ] **Step 1: Write the DTO**

`server/src/notifications/dto/send-notification.dto.ts`:
```ts
import { Type } from 'class-transformer';
import { IsArray, IsBoolean, IsIn, IsNotEmpty, IsOptional, IsString, ValidateNested } from 'class-validator';

export class NotificationRecipientsDto {
  @IsIn(['all', 'users'])
  mode!: 'all' | 'users';

  @IsOptional()
  @IsArray()
  @IsString({ each: true })
  userIds?: string[];
}

export class SendNotificationDto {
  @IsString()
  @IsNotEmpty()
  text!: string;

  @ValidateNested()
  @Type(() => NotificationRecipientsDto)
  recipients!: NotificationRecipientsDto;

  @IsOptional()
  @IsBoolean()
  dryRun?: boolean;
}
```

- [ ] **Step 2: Write the failing service test (pure logic, no DB)**

`server/src/notifications/notifications.service.spec.ts`:
```ts
import { NotificationCandidate, NotificationsService } from './notifications.service';

describe('NotificationsService.resolveTargets (pure — no DB involved)', () => {
  const service = new NotificationsService({} as any); // db is never touched by resolveTargets
  const all: NotificationCandidate[] = [
    { id: 'u1', name: 'Аня' },
    { id: 'u2', name: 'Боря' },
  ];

  it('returns everyone for mode "all"', () => {
    expect(service.resolveTargets(all, { mode: 'all' })).toEqual(all);
  });

  it('filters to the selected ids for mode "users"', () => {
    expect(service.resolveTargets(all, { mode: 'users', userIds: ['u2'] })).toEqual([all[1]]);
  });

  it('returns nothing for mode "users" with no matching ids', () => {
    expect(service.resolveTargets(all, { mode: 'users', userIds: ['does-not-exist'] })).toEqual([]);
  });

  it('returns nothing for mode "users" with userIds omitted', () => {
    expect(service.resolveTargets(all, { mode: 'users' })).toEqual([]);
  });
});
```

- [ ] **Step 3: Write the failing controller test (validation + unreachable-DB path)**

`server/src/notifications/notifications.controller.spec.ts`:
```ts
import { Test, TestingModule } from '@nestjs/testing';
import { INestApplication, ValidationPipe } from '@nestjs/common';
import request from 'supertest';
import { NotificationsModule } from './notifications.module';

describe('POST /notifications', () => {
  let app: INestApplication;
  const originalUrl = process.env.PROD_DATABASE_URL;

  beforeAll(async () => {
    delete process.env.PROD_DATABASE_URL;
    const moduleRef: TestingModule = await Test.createTestingModule({
      imports: [NotificationsModule],
    }).compile();
    app = moduleRef.createNestApplication();
    app.useGlobalPipes(new ValidationPipe({ whitelist: true, transform: true }));
    await app.init();
  });

  afterAll(async () => {
    process.env.PROD_DATABASE_URL = originalUrl;
    await app.close();
  });

  it('rejects empty text with 400 before ever touching the DB', async () => {
    await request(app.getHttpServer())
      .post('/notifications')
      .send({ text: '', recipients: { mode: 'all' } })
      .expect(400);
  });

  it('rejects an invalid recipients.mode with 400 before ever touching the DB', async () => {
    await request(app.getHttpServer())
      .post('/notifications')
      .send({ text: 'hi', recipients: { mode: 'bogus' } })
      .expect(400);
  });

  it('returns 503 for a valid dry-run request when the prod DB is unreachable', async () => {
    await request(app.getHttpServer())
      .post('/notifications')
      .send({ text: 'hi', recipients: { mode: 'all' }, dryRun: true })
      .expect(503);
  });
});
```

- [ ] **Step 4: Run both to verify they fail**

Run: `cd server && npx jest notifications --runInBand`
Expected: FAIL — `NotificationsService`, `NotificationsModule` etc. don't exist yet.

- [ ] **Step 5: Implement the service**

`server/src/notifications/notifications.service.ts`:
```ts
import { Injectable } from '@nestjs/common';
import { ProdDbService } from '../prod-db/prod-db.service';
import { SendNotificationDto, NotificationRecipientsDto } from './dto/send-notification.dto';

export interface NotificationCandidate {
  id: string;
  name: string;
}

interface CandidateRow {
  id: string;
  name: string;
}

function rfc3339(d: Date): string {
  return d.toISOString().replace(/\.\d{3}Z$/, 'Z');
}

@Injectable()
export class NotificationsService {
  constructor(private readonly db: ProdDbService) {}

  // Unlike broadcasts/candidates (Telegram-linked users only), every
  // registered user is a valid in-app notification recipient.
  async candidates(): Promise<NotificationCandidate[]> {
    const rows = await this.db.query<CandidateRow>(`SELECT id, name FROM users ORDER BY name`);
    return rows.map((r) => ({ id: r.id, name: r.name }));
  }

  // The pure part of send() — no DB write — so it's unit-testable without
  // touching Postgres.
  resolveTargets(all: NotificationCandidate[], recipients: NotificationRecipientsDto): NotificationCandidate[] {
    if (recipients.mode === 'all') return all;
    const wanted = new Set(recipients.userIds ?? []);
    return all.filter((c) => wanted.has(c.id));
  }

  async send(dto: SendNotificationDto): Promise<{ count: number }> {
    const all = await this.candidates();
    const targets = this.resolveTargets(all, dto.recipients);

    if (dto.dryRun || targets.length === 0) {
      return { count: targets.length };
    }

    const createdAt = rfc3339(new Date());
    const inserted = await this.db.query<{ id: string }>(
      `INSERT INTO notifications (text, created_at) VALUES ($1, $2) RETURNING id`,
      [dto.text, createdAt],
    );
    const notificationId = inserted[0].id;

    const values: string[] = [];
    const params: unknown[] = [notificationId];
    targets.forEach((t, i) => {
      values.push(`($1, $${i + 2}, '')`);
      params.push(t.id);
    });
    await this.db.query(
      `INSERT INTO notification_recipients (notification_id, user_id, read_at) VALUES ${values.join(', ')}`,
      params,
    );
    return { count: targets.length };
  }
}
```

- [ ] **Step 6: Implement the controller and module**

`server/src/notifications/notifications.controller.ts`:
```ts
import { Body, Controller, Get, Post } from '@nestjs/common';
import { NotificationsService } from './notifications.service';
import { SendNotificationDto } from './dto/send-notification.dto';

@Controller('notifications')
export class NotificationsController {
  constructor(private readonly notifications: NotificationsService) {}

  @Get('candidates')
  candidates() {
    return this.notifications.candidates();
  }

  @Post()
  send(@Body() dto: SendNotificationDto) {
    return this.notifications.send(dto);
  }
}
```

`server/src/notifications/notifications.module.ts`:
```ts
import { Module } from '@nestjs/common';
import { ProdDbModule } from '../prod-db/prod-db.module';
import { NotificationsController } from './notifications.controller';
import { NotificationsService } from './notifications.service';

@Module({
  imports: [ProdDbModule],
  controllers: [NotificationsController],
  providers: [NotificationsService],
})
export class NotificationsModule {}
```

- [ ] **Step 7: Wire the module into the app**

In `server/src/app.module.ts`, add the import and list it alongside the other feature modules:
```ts
import { NotificationsModule } from './notifications/notifications.module';
```
```ts
@Module({
  imports: [
    PrismaModule,
    ItemsModule,
    ProdDbModule,
    InsightsModule,
    LinksModule,
    BotMessagesModule,
    BroadcastsModule,
    NotificationsModule,
  ],
```

- [ ] **Step 8: Run the tests to verify they pass**

Run: `cd server && npx jest notifications --runInBand`
Expected: PASS (4 service tests + 3 controller tests)

- [ ] **Step 9: Run the full server suite and commit**

Run: `cd server && npm test`
Expected: PASS

```bash
git add server/src/notifications server/src/app.module.ts
git commit -m "$(cat <<'EOF'
Add notifications module — candidates + send (all/users, dry-run)

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

## Part D — ucimo-content-admin: composer frontend

### Task 6: «Уведомления» sidebar page

**Files:**
- Modify: `web/src/types.ts`
- Create: `web/src/api/notifications.ts`
- Create: `web/src/views/NotificationsView.vue`
- Modify: `web/src/components/AdminSidebar.vue`
- Modify: `web/src/router.ts`

**Interfaces:**
- Consumes: `GET /notifications/candidates`, `POST /notifications` (Task 5).
- Produces: a reachable `/notifications` page and sidebar entry — nothing else in the codebase depends on this task's exports (it's the last piece of the feature).

- [ ] **Step 1: Add the type**

In `web/src/types.ts`, add near `BroadcastCandidate`:
```ts
export interface NotificationCandidate {
  id: string;
  name: string;
}
```

- [ ] **Step 2: Add the API client**

`web/src/api/notifications.ts`:
```ts
import type { NotificationCandidate } from '../types';

const BASE = '/api/notifications';

async function handle<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.message ?? `Request failed: ${res.status}`);
  }
  return res.json();
}

export function listCandidates(): Promise<NotificationCandidate[]> {
  return fetch(`${BASE}/candidates`).then((r) => handle(r));
}

export type NotificationRecipients = { mode: 'all' } | { mode: 'users'; userIds: string[] };

export function sendNotification(
  text: string,
  recipients: NotificationRecipients,
  dryRun: boolean,
): Promise<{ count: number }> {
  return fetch(BASE, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text, recipients, dryRun }),
  }).then((r) => handle(r));
}
```

- [ ] **Step 3: Build the view**

`web/src/views/NotificationsView.vue`:
```vue
<template>
  <div>
    <h1 class="mb-6 text-2xl font-semibold text-primary-700">Уведомления</h1>
    <p v-if="loading" class="text-sm text-text-500">Загрузка…</p>
    <p v-else-if="error" class="text-sm text-error">{{ error }}</p>
    <div v-else class="flex max-w-xl flex-col gap-4">
      <label class="flex items-center gap-2 text-sm text-text-700">
        <input type="checkbox" v-model="sendToAll" />
        Всем пользователям ({{ candidates.length }})
      </label>

      <div v-if="!sendToAll">
        <input
          v-model="search"
          type="text"
          placeholder="Поиск по имени"
          class="mb-2 w-full rounded border border-border px-2 py-1 text-sm"
        />
        <div class="max-h-48 overflow-y-auto rounded border border-border">
          <label
            v-for="c in filteredCandidates"
            :key="c.id"
            class="flex items-center gap-2 border-b border-border px-2 py-1 text-sm last:border-b-0"
          >
            <input type="checkbox" :value="c.id" v-model="selectedIds" />
            {{ c.name }}
          </label>
        </div>
      </div>

      <div>
        <textarea
          v-model="text"
          rows="5"
          placeholder="Текст уведомления"
          class="w-full rounded border border-border p-2 text-sm"
        ></textarea>
        <p class="mt-1 text-xs text-text-500">Чтобы добавить ссылку: [текст ссылки](https://...)</p>
      </div>

      <button
        type="button"
        class="self-start rounded bg-primary-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
        :disabled="!canSend"
        @click="openConfirm"
      >
        Отправить
      </button>
      <p v-if="sendError" class="text-sm text-error">{{ sendError }}</p>
      <p v-if="sentCount !== null" class="text-sm text-success">Отправлено: {{ sentCount }}</p>

      <div v-if="confirming" class="rounded border border-border bg-white p-4 shadow-sm">
        <p class="mb-3 text-sm text-text-800">
          Отправить {{ pendingCount }} получател{{ pendingCount === 1 ? 'ю' : 'ям' }}?
        </p>
        <div class="flex gap-3">
          <button
            type="button"
            class="rounded bg-primary-600 px-3 py-1.5 text-sm font-medium text-white"
            @click="confirmSend"
          >
            Да, отправить {{ sendToAll ? `всем ${pendingCount}` : pendingCount }}
          </button>
          <button type="button" class="rounded px-3 py-1.5 text-sm text-text-600" @click="confirming = false">
            Отмена
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import type { NotificationCandidate } from '../types';
import { listCandidates, sendNotification } from '../api/notifications';

const candidates = ref<NotificationCandidate[]>([]);
const loading = ref(true);
const error = ref('');

const sendToAll = ref(true);
const search = ref('');
const selectedIds = ref<string[]>([]);
const text = ref('');

const confirming = ref(false);
const pendingCount = ref(0);
const sendError = ref('');
const sentCount = ref<number | null>(null);

onMounted(async () => {
  try {
    candidates.value = await listCandidates();
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
});

const filteredCandidates = computed(() =>
  candidates.value.filter((c) => c.name.toLowerCase().includes(search.value.toLowerCase())),
);

const canSend = computed(() => text.value.trim() !== '' && (sendToAll.value || selectedIds.value.length > 0));

function recipients() {
  return sendToAll.value ? { mode: 'all' as const } : { mode: 'users' as const, userIds: selectedIds.value };
}

async function openConfirm() {
  sendError.value = '';
  sentCount.value = null;
  try {
    const { count } = await sendNotification(text.value, recipients(), true);
    pendingCount.value = count;
    confirming.value = true;
  } catch (e) {
    sendError.value = e instanceof Error ? e.message : String(e);
  }
}

async function confirmSend() {
  try {
    const { count } = await sendNotification(text.value, recipients(), false);
    sentCount.value = count;
    confirming.value = false;
    text.value = '';
    selectedIds.value = [];
  } catch (e) {
    sendError.value = e instanceof Error ? e.message : String(e);
    confirming.value = false;
  }
}
</script>
```

- [ ] **Step 4: Add the sidebar section**

In `web/src/components/AdminSidebar.vue`, add a new top-level section (not nested under «Бот» — this isn't Telegram-specific), right after the «Бот» section:
```ts
  {
    key: 'bot',
    title: 'Бот',
    links: [
      { label: 'Сообщения', to: '/bot/messages' },
      { label: 'Рассылки', to: '/bot/broadcasts' },
    ],
  },
  { key: 'notifications', title: 'Уведомления', links: [{ label: 'Уведомления', to: '/notifications' }] },
];
```
And add `notifications: true` to the `openState` reactive object, alongside `content`/`data`/`links`/`bot`.

- [ ] **Step 5: Register the route**

In `web/src/router.ts`, add the import:
```ts
import NotificationsView from './views/NotificationsView.vue';
```
and the route, alongside the other top-level pages:
```ts
    { path: '/notifications', name: 'notifications', component: NotificationsView },
```

- [ ] **Step 6: Type-check and manually verify**

Run: `cd web && npx vue-tsc --noEmit`
Expected: no errors.

Then, with `make tunnel` running and `make dev` up, open `http://localhost:5173/notifications` (or whatever `dev-web`'s printed port is) and confirm: the candidate list loads, the "Всем пользователям (N)" checkbox toggles the search/list, typing text enables Send, clicking Send shows the confirm dialog with the right count, and confirming shows "Отправлено: N".

- [ ] **Step 7: Commit**

```bash
git add web/src/types.ts web/src/api/notifications.ts web/src/views/NotificationsView.vue web/src/components/AdminSidebar.vue web/src/router.ts
git commit -m "$(cat <<'EOF'
Add "Уведомления" admin page — recipient picker + composer

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

## Part E — Deploy

### Task 7: Deploy serbian-app and grant `ucimo_admin_ro` its new write permissions

**Files:** none (operational task)

- [ ] **Step 1: Push and deploy serbian-app**

```bash
cd /Users/grisha/plans/serbian-app
git push origin main
```
Deploy via the Dokploy API (`POST /api/application.deploy` with serbian-app's `applicationId` — see your deploy notes/memory for the base URL, API key, and id; do not paste them into this plan file). This runs migration 010 automatically on boot (`store.Open` applies pending migrations before the server starts serving).

- [ ] **Step 2: Verify the migration applied**

Using the same SSH access documented in your deploy notes, connect to the prod Postgres (`srpski` database) and confirm:
```sql
SELECT count(*) FROM notifications;             -- want 0 (freshly created, empty)
SELECT count(*) FROM notification_recipients;    -- want 0
```

- [ ] **Step 3: Grant `ucimo_admin_ro` `INSERT` on the two new tables**

Still connected to the prod Postgres as an admin role, run:
```sql
GRANT INSERT ON notifications TO ucimo_admin_ro;
GRANT INSERT ON notification_recipients TO ucimo_admin_ro;
```
No `SELECT`/`UPDATE` — the admin never reads these back, per the spec.

- [ ] **Step 4: Verify the deployed binary is actually live**

Per your deploy notes' "Verify after deploy" section: confirm the `deployments[]` entry for this commit shows `status:"done"`, then poll `https://ucimo.ru/` once more and diff the hashed JS asset filename against a fresh local `npm run build` (there's a documented ~1-2 min lag between "done" and the new container actually serving).

No commit for this task (nothing in the repo changes).

---

### Task 8: Deploy ucimo-content-admin

**Files:** none (operational task)

- [ ] **Step 1: Build and push the image, redeploy**

Follow the existing no-git-remote build-on-host flow (rsync the repo to the Dokploy host, `docker build`, tag/push to the local registry, `POST /api/application.deploy` with ucimo-content-admin's `applicationId`) — see your deploy notes for the exact commands and credentials; do not paste them into this plan file.

- [ ] **Step 2: Verify**

```bash
curl -s -o /dev/null -w '%{http_code}\n' https://admin.ucimo.ru/api/notifications/candidates           # 401, no creds
curl -s -o /dev/null -w '%{http_code}\n' -u "admin:<password>" https://admin.ucimo.ru/api/notifications/candidates  # 200
```
The second call's body should be a JSON array of `{id, name}` covering every registered user (not just Telegram-linked ones — spot check the count against `data/users` in the admin if unsure).

No commit for this task (nothing in the repo changes).

---

### Task 9: End-to-end manual smoke test

**Files:** none (verification task)

- [ ] **Step 1: Send to one specific (your own) account**

In admin (`/notifications`), uncheck "Всем пользователям", select only your own account, write a short message that includes a link (e.g. `Проверка [тут](https://ucimo.ru)`), send, confirm the dialog.

- [ ] **Step 2: Confirm it shows up in serbian-app**

Log into ucimo.ru as that account (or refresh if already logged in — the bell polls every 45s, or reload the page to see it immediately). Confirm the bell shows an accent-colored dot.

- [ ] **Step 3: Confirm the dropdown and link render correctly**

Click the bell. Confirm the message appears with a highlighted (unread) background, the `[тут](...)` renders as an actual clickable link opening `https://ucimo.ru` in a new tab, and the dot disappears immediately after opening.

- [ ] **Step 4: Confirm read state persists across reloads**

Reload the page. Reopen the dropdown. Confirm the message is still there but no longer highlighted, and the bell has no dot.

- [ ] **Step 5: Send to "all"**

In admin, check "Всем пользователям", send a second distinctive message, confirm the count in the confirm dialog matches your total registered user count.

- [ ] **Step 6: Confirm targeting didn't leak**

Log into (or check, if you have one) a *different* test account that was **not** individually selected in Step 1 — confirm it does **not** see the Step-1 message but **does** see the Step-5 "all" message.

No commit for this task (verification only). If any step fails, file it as a bug against the relevant task above rather than patching ad hoc — reopen that task's checklist.

---

## Self-Review Notes

**Spec coverage:** admin section with all/some/one targeting and dry-run confirm (Tasks 5, 6), textarea + link-syntax hint (Task 6), theme-colored unread dot + dropdown + mark-read-on-open (Task 4), `notifications`/`notification_recipients` schema + 30-day post-read retention + "all" snapshot semantics (Task 1), point `INSERT`-only grant (Task 7), polling at 45s with an architecture that doesn't block a future websocket swap (Task 3 — `useNotifications` is the only place that would change), link-only markdown via a restricted regex renderer, not `markdown-it` (Task 3).

**Deviation from a literal reading of the spec's testing section, and why:** the spec's "Тестирование" section doesn't specify whether the NestJS `notifications` module should hit a real Postgres tunnel or stay mocked. Following the actual, current convention in this codebase (confirmed by reading `broadcasts.service.spec.ts`/`broadcasts.controller.spec.ts` during planning, and the explicit self-review note in `2026-09-22-bot-messages-and-broadcasts.md` about why no `ProdDbService`-backed test runs in CI — no tunnel there), Task 5 uses the same two-layer pattern: pure-logic unit tests for `resolveTargets`, and a 503-when-unreachable check for the DB-backed paths. Real end-to-end coverage is Task 9's manual smoke test, same as broadcasts.
