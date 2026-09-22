# Friendly Bot Messages + Broadcasts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the Telegram bot's hardcoded, unfriendly reply strings with 8 admin-editable templates (covering all `/start` login/link scenarios, including two currently-silent cases), route every bot-originated send (webhook replies, periodic reminders, and new admin broadcasts) through one rate-limited priority queue, and add two pages to ucimo-content-admin: a template editor and a broadcast composer.

**Architecture:** serbian-app owns two new Postgres tables — `bot_messages` (editable template text) and `bot_outbox` (a send queue, high priority for `/start` replies, normal for reminders/broadcasts) — drained by a new ticker-driven worker that is the only code path left that calls the Telegram Bot API. ucimo-content-admin gets write access to exactly these two tables (a first exception to its otherwise read-only `ucimo_admin_ro` role) via two new NestJS modules and two new Vue pages under a new "Бот" sidebar section.

**Tech Stack:** Go 1.25 (stdlib `net/http`, `modernc.org/sqlite`, `pgx/v5`) for serbian-app; NestJS + `pg` (via the existing `ProdDbService`, no Prisma) + Vue 3 + Tailwind for ucimo-content-admin.

**Spec:** [docs/superpowers/specs/2026-09-22-bot-messages-design.md](../specs/2026-09-22-bot-messages-design.md)

## Global Constraints

- All 8 message templates and their exact default copy are fixed by the spec (Zdravo greeting, `@ucimosupport` mentions) — do not paraphrase them.
- Telegram Bot API rate limits (core.telegram.org/bots/faq): max 1 msg/s per chat, max 30 msg/s bulk. The worker caps itself at **25 msg/s** (a ticker at `time.Second/25`), hardcoded — not admin-editable.
- `bot_outbox.priority`: `0` = high (every `/start`-flow reply), `1` = normal (reminders, admin broadcasts). The worker always drains priority 0 before priority 1.
- `ucimo_admin_ro` gets **only** `INSERT`/`UPDATE` on `bot_messages` and `INSERT` on `bot_outbox` — nothing else changes about its read-only access to serbian-app's DB.
- Broadcast messages from admin are plain text only, no button (v1) — see spec's "Вне скоупа".
- No history/audit view for edited templates or sent broadcasts (v1) — see spec's "Вне скоупа".
- Go doc-comments in English; user-facing/bot copy in Russian (per `serbian-app/CLAUDE.md`).

---

## Part A — serbian-app: message templates

### Task 1: `bot_messages` table, seed, and store accessors

**Files:**
- Create: `server/internal/store/migrations/007_bot_messages.sql`
- Modify: `server/internal/store/migration_hooks.go`
- Create: `server/internal/store/bot_messages.go`
- Create: `server/internal/store/bot_messages_test.go`

**Interfaces:**
- Consumes: `telegram.DefaultMessages map[telegram.MessageKey]string` (produced by Task 2 — this task's seed hook needs it, so do Task 2 first if working sequentially; if working out of order, stub is not allowed, so complete Task 2 before this task's hook step).
- Produces: `(*store.Store).BotMessageText(key string) (text string, ok bool, err error)`, `(*store.Store).SetBotMessageText(key, text string, at time.Time) error` — used by Task 7 (webhook) and this task's own tests.

> **Ordering note:** this task's migration hook imports `internal/telegram`, which Task 2 creates. Do Task 2 first, then this task.

- [ ] **Step 1: Write the migration file (schema only, no seed data — seeding needs Go's `time.Now()`, done via a hook)**

`server/internal/store/migrations/007_bot_messages.sql`:
```sql
-- Migration 007: bot_messages — editable Telegram bot reply templates,
-- edited from ucimo-content-admin's "Бот" → "Сообщения" page. Seeded with
-- default copy by migrate007 (migration_hooks.go) so there's always
-- something to show/edit. See
-- docs/superpowers/specs/2026-09-22-bot-messages-design.md.

CREATE TABLE IF NOT EXISTS bot_messages (
	key        TEXT PRIMARY KEY,
	text       TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
```

- [ ] **Step 2: Register the seed hook**

In `server/internal/store/migration_hooks.go`, add `"github.com/grisha/serbian-app/server/internal/telegram"` to the imports and add to the `init()` function's existing `registerHook` calls:
```go
registerHook(7, migrate007)
```
Then add the function (anywhere after `init()`):
```go
// migrate007 seeds bot_messages with telegram.DefaultMessages so
// ucimo-content-admin's editor always has something to show. Runs once,
// inside migration 007's transaction, right after 007_bot_messages.sql
// creates the table.
func migrate007(tx *dbtx, pg bool) error {
	now := time.Now().UTC().Format(time.RFC3339)
	for key, text := range telegram.DefaultMessages {
		if _, err := tx.Exec(`INSERT INTO bot_messages (key, text, updated_at) VALUES (?, ?, ?)`,
			string(key), text, now); err != nil {
			return fmt.Errorf("seed bot message %s: %w", key, err)
		}
	}
	return nil
}
```

- [ ] **Step 3: Write the failing tests**

`server/internal/store/bot_messages_test.go`:
```go
package store

import (
	"testing"

	"github.com/grisha/serbian-app/server/internal/telegram"
)

func TestBotMessageTextUnknownKeyIsNotFound(t *testing.T) {
	s := newStore(t)
	_, ok, err := s.BotMessageText("no_such_key")
	if err != nil {
		t.Fatalf("BotMessageText: %v", err)
	}
	if ok {
		t.Errorf("BotMessageText(unknown) ok = true, want false")
	}
}

func TestSetAndGetBotMessageText(t *testing.T) {
	s := newStore(t)
	if err := s.SetBotMessageText("start_greeting", "Custom text", day0); err != nil {
		t.Fatalf("SetBotMessageText: %v", err)
	}
	text, ok, err := s.BotMessageText("start_greeting")
	if err != nil {
		t.Fatalf("BotMessageText: %v", err)
	}
	if !ok || text != "Custom text" {
		t.Fatalf("BotMessageText = (%q, %v), want (\"Custom text\", true)", text, ok)
	}

	if err := s.SetBotMessageText("start_greeting", "Updated text", day0); err != nil {
		t.Fatalf("SetBotMessageText (update): %v", err)
	}
	text, _, _ = s.BotMessageText("start_greeting")
	if text != "Updated text" {
		t.Errorf("BotMessageText after update = %q, want %q", text, "Updated text")
	}
}

func TestMigration007SeedsAllDefaultMessages(t *testing.T) {
	s := newStore(t)
	for key, want := range telegram.DefaultMessages {
		text, ok, err := s.BotMessageText(string(key))
		if err != nil {
			t.Fatalf("BotMessageText(%s): %v", key, err)
		}
		if !ok {
			t.Errorf("key %s not seeded by migration 007", key)
			continue
		}
		if text != want {
			t.Errorf("seeded text for %s = %q, want %q", key, text, want)
		}
	}
}
```

- [ ] **Step 4: Run the tests to verify they fail**

Run: `cd server && go test ./internal/store/... -run 'BotMessage|Migration007' -v`
Expected: FAIL — `BotMessageText`/`SetBotMessageText` undefined.

- [ ] **Step 5: Implement the store accessors**

`server/internal/store/bot_messages.go`:
```go
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// BotMessageText returns the current text for key, ok=false if no row
// exists (fall back to telegram.DefaultMessages — only expected for a key
// added to that map after migration 007 last seeded the table).
func (s *Store) BotMessageText(key string) (text string, ok bool, err error) {
	row := s.db.QueryRow(`SELECT text FROM bot_messages WHERE key = ?`, key)
	if err := row.Scan(&text); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("bot message text: %w", err)
	}
	return text, true, nil
}

// SetBotMessageText overwrites key's text (inserts if the row is somehow
// missing). Production edits come from ucimo-content-admin writing
// directly against Postgres, not through this method — it exists for tests
// and any future in-process tooling.
func (s *Store) SetBotMessageText(key, text string, at time.Time) error {
	_, err := s.db.Exec(`INSERT INTO bot_messages (key, text, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (key) DO UPDATE SET text = excluded.text, updated_at = excluded.updated_at`,
		key, text, at.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("set bot message text: %w", err)
	}
	return nil
}
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd server && go test ./internal/store/... -run 'BotMessage|Migration007' -v`
Expected: PASS (all three tests)

- [ ] **Step 7: Run the full store suite (migration ordering can affect other tests) and commit**

Run: `cd server && go test ./internal/store/...`
Expected: PASS

```bash
git add server/internal/store/migrations/007_bot_messages.sql server/internal/store/migration_hooks.go server/internal/store/bot_messages.go server/internal/store/bot_messages_test.go
git commit -m "$(cat <<'EOF'
Add bot_messages table + store accessors, seeded from telegram.DefaultMessages

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: `telegram` package — message keys, default copy, name substitution

**Files:**
- Create: `server/internal/telegram/messages.go`
- Create: `server/internal/telegram/messages_test.go`

**Interfaces:**
- Consumes: nothing (leaf package, stdlib only).
- Produces: `type MessageKey string` + 8 constants (`MsgStartGreeting`, `MsgLoginSuccessNew`, `MsgLoginSuccessExisting`, `MsgLoginTokenExpired`, `MsgLoginError`, `MsgLinkSuccess`, `MsgLinkTaken`, `MsgLinkError`), `var DefaultMessages map[MessageKey]string`, `const SupportURL`, `func Substitute(text, name string) string`, `func DisplayName(firstName, username string) string`. Task 1's `migrate007` and Task 7's webhook both depend on these exact names.

> Do this task **before** Task 1 (Task 1's seed hook imports `telegram.DefaultMessages`).

- [ ] **Step 1: Write the failing tests**

`server/internal/telegram/messages_test.go`:
```go
package telegram

import "testing"

func TestSubstituteReplacesName(t *testing.T) {
	got := Substitute("Zdravo, {name}! Ты — {name}.", "Neo")
	want := "Zdravo, Neo! Ты — Neo."
	if got != want {
		t.Errorf("Substitute = %q, want %q", got, want)
	}
}

func TestDisplayNamePrefersFirstName(t *testing.T) {
	if got := DisplayName("Neo", "neo_bot"); got != "Neo" {
		t.Errorf("DisplayName = %q, want %q", got, "Neo")
	}
}

func TestDisplayNameFallsBackToUsername(t *testing.T) {
	if got := DisplayName("", "neo_bot"); got != "@neo_bot" {
		t.Errorf("DisplayName = %q, want %q", got, "@neo_bot")
	}
}

func TestDisplayNameFallsBackToGeneric(t *testing.T) {
	if got := DisplayName("", ""); got != "друг" {
		t.Errorf("DisplayName = %q, want %q", got, "друг")
	}
}

func TestDefaultMessagesHaveAllEightKeys(t *testing.T) {
	want := []MessageKey{
		MsgStartGreeting, MsgLoginSuccessNew, MsgLoginSuccessExisting,
		MsgLoginTokenExpired, MsgLoginError, MsgLinkSuccess, MsgLinkTaken, MsgLinkError,
	}
	if len(DefaultMessages) != len(want) {
		t.Fatalf("DefaultMessages has %d entries, want %d", len(DefaultMessages), len(want))
	}
	for _, k := range want {
		if DefaultMessages[k] == "" {
			t.Errorf("DefaultMessages[%s] is empty", k)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd server && go test ./internal/telegram/... -run 'Substitute|DisplayName|DefaultMessages' -v`
Expected: FAIL — package doesn't compile (`MessageKey`, `Substitute`, etc. undefined).

- [ ] **Step 3: Implement**

`server/internal/telegram/messages.go`:
```go
package telegram

import "strings"

// MessageKey identifies one editable bot reply template, stored in
// serbian-app's bot_messages table and edited from ucimo-content-admin's
// "Бот" → "Сообщения" page.
type MessageKey string

const (
	MsgStartGreeting        MessageKey = "start_greeting"
	MsgLoginSuccessNew      MessageKey = "login_success_new"
	MsgLoginSuccessExisting MessageKey = "login_success_existing"
	MsgLoginTokenExpired    MessageKey = "login_token_expired"
	MsgLoginError           MessageKey = "login_error"
	MsgLinkSuccess          MessageKey = "link_success"
	MsgLinkTaken            MessageKey = "link_taken"
	MsgLinkError            MessageKey = "link_error"
)

// SupportURL is where every "написать в поддержку" button and @-mention
// points — the same account the website's support card links to
// (web/src/lib/supportModal.ts's SUPPORT_TELEGRAM_URL).
const SupportURL = "https://t.me/ucimosupport"

// Outbox priorities — lower runs first in internal/outbox.ProcessNext.
// High is every direct reaction to something a user just did (the /start
// webhook flow); Normal is everything else (periodic reminders, admin
// broadcasts).
const (
	PriorityHigh   = 0
	PriorityNormal = 1
)

// DefaultMessages holds the fallback text for every MessageKey. Used by
// server/internal/api's sendBotMessage only when bot_messages has no row
// for a key (in normal operation migration 007 has already seeded all of
// them — this only matters for a key added to this map after that
// migration last ran). Also exactly what migrate007
// (server/internal/store/migration_hooks.go) seeds the table with.
var DefaultMessages = map[MessageKey]string{
	MsgStartGreeting: "Zdravo (привет), {name}! 👋 Это бот Учимо — сервиса для изучения сербского с нуля.\n\n" +
		"Жми на кнопку ниже, чтобы открыть приложение и начать учиться.",
	MsgLoginSuccessNew: "Готово, {name}! 🎉 Регистрация прошла успешно — добро пожаловать в Учимо.\n\n" +
		"Заходи на сайт ucimo.ru или сразу открывай приложение здесь, в Telegram, — и начинай первый урок!",
	MsgLoginSuccessExisting: "С возвращением, {name}! 👋 Ты уже с нами — продолжай изучать сербский.\n\n" +
		"Открывай приложение и вперёд: тебя ждут уроки и повторение слов.",
	MsgLoginTokenExpired: "Кажется, эта ссылка уже устарела 🙈\n\n" +
		"Зайди на ucimo.ru и попробуй войти через Telegram ещё раз.",
	MsgLoginError: "Упс, что-то пошло не так 😔\n\n" +
		"Попробуй войти ещё раз с сайта ucimo.ru. Если не получится — напиши нам: @ucimosupport",
	MsgLinkSuccess: "Готово! 🎉 Telegram привязан к твоему аккаунту, {name}.\n\n" +
		"Теперь можно входить в Учимо и через Telegram — возвращайся на сайт.",
	MsgLinkTaken: "Этот Telegram уже привязан к другому аккаунту Учимо.\n\n" +
		"Если это ошибка — напиши нам: @ucimosupport",
	MsgLinkError: "Упс, не получилось привязать Telegram 😔\n\n" +
		"Попробуй ещё раз с сайта, а если не поможет — напиши нам: @ucimosupport",
}

// Substitute replaces every "{name}" placeholder in text with name.
func Substitute(text, name string) string {
	return strings.ReplaceAll(text, "{name}", name)
}

// DisplayName picks what to greet a Telegram user by: their first name,
// else "@" + their username, else the generic fallback "друг" (Telegram
// guarantees at least one of first_name/username is non-empty for a real
// user, but never both are absent and no name at all is defensively
// handled anyway).
func DisplayName(firstName, username string) string {
	if firstName != "" {
		return firstName
	}
	if username != "" {
		return "@" + username
	}
	return "друг"
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd server && go test ./internal/telegram/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/telegram/messages.go server/internal/telegram/messages_test.go
git commit -m "$(cat <<'EOF'
Add telegram.MessageKey/DefaultMessages/Substitute/DisplayName

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: `telegram` package — inline buttons, rate-limit detection, bare-/start detection

**Files:**
- Modify: `server/internal/telegram/telegram.go`
- Modify: `server/internal/telegram/telegram_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `type InlineButton struct{Label, WebAppURL, URL string}`, `func SendMessageWithButton(botToken string, chatID int64, text string, button *InlineButton) error`, `func RateLimited(err error) (time.Duration, bool)`, `func IsBareStart(text string) bool`. Task 5 (`internal/outbox`) and Task 7 (webhook) depend on all four.

- [ ] **Step 1: Write the failing tests**

Add to `server/internal/telegram/telegram_test.go` (add `"strings"` and `"time"` to the existing import block):
```go
func TestSendMessageWithButtonWebApp(t *testing.T) {
	var gotBody string
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotBody = r.Form.Encode()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":true}`))
	})
	err := SendMessageWithButton("tok", 555, "hi", &InlineButton{Label: "Открыть", WebAppURL: "https://ucimo.ru/profile"})
	if err != nil {
		t.Fatalf("SendMessageWithButton: %v", err)
	}
	form, err := url.ParseQuery(gotBody)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(form.Get("reply_markup"), `"web_app":{"url":"https://ucimo.ru/profile"}`) {
		t.Errorf("reply_markup = %s, want a web_app button", form.Get("reply_markup"))
	}
}

func TestSendMessageWithButtonURL(t *testing.T) {
	var gotBody string
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotBody = r.Form.Encode()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":true}`))
	})
	err := SendMessageWithButton("tok", 555, "hi", &InlineButton{Label: "Поддержка", URL: SupportURL})
	if err != nil {
		t.Fatalf("SendMessageWithButton: %v", err)
	}
	form, err := url.ParseQuery(gotBody)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(form.Get("reply_markup"), `"url":"`+SupportURL+`"`) {
		t.Errorf("reply_markup = %s, want a url button to %s", form.Get("reply_markup"), SupportURL)
	}
}

func TestSendMessageWithButtonNilOmitsMarkup(t *testing.T) {
	var gotBody string
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotBody = r.Form.Encode()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":true}`))
	})
	if err := SendMessageWithButton("tok", 555, "hi", nil); err != nil {
		t.Fatalf("SendMessageWithButton: %v", err)
	}
	form, err := url.ParseQuery(gotBody)
	if err != nil {
		t.Fatal(err)
	}
	if form.Get("reply_markup") != "" {
		t.Errorf("reply_markup = %q, want empty for a nil button", form.Get("reply_markup"))
	}
}

func TestRateLimitedParsesRetryAfter(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"ok":false,"error_code":429,"description":"Too Many Requests","parameters":{"retry_after":7}}`))
	})
	err := SendMessageWithButton("tok", 1, "hi", nil)
	if err == nil {
		t.Fatal("SendMessageWithButton: want error for a 429 response")
	}
	wait, ok := RateLimited(err)
	if !ok || wait != 7*time.Second {
		t.Errorf("RateLimited = (%v, %v), want (7s, true)", wait, ok)
	}
}

func TestRateLimitedFalseForOtherErrors(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":false,"error_code":400,"description":"Bad Request"}`))
	})
	err := SendMessageWithButton("tok", 1, "hi", nil)
	if err == nil {
		t.Fatal("SendMessageWithButton: want error for a 400 response")
	}
	if _, ok := RateLimited(err); ok {
		t.Errorf("RateLimited(400 error) = true, want false")
	}
}

func TestIsBareStart(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"/start", true},
		{"/start@ucimoappbot", true},
		{"/start abc123", false},
		{"/start@ucimoappbot abc123", false},
		{"hello", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsBareStart(c.text); got != c.want {
			t.Errorf("IsBareStart(%q) = %v, want %v", c.text, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd server && go test ./internal/telegram/... -v`
Expected: FAIL — `InlineButton`, `SendMessageWithButton`, `RateLimited`, `IsBareStart` undefined.

- [ ] **Step 3: Implement**

In `server/internal/telegram/telegram.go`:

1. Add `"errors"` and `"time"` to the import block (`"time"` is likely already imported for `httpClient`'s timeout — check before adding a duplicate).

2. Extend `apiError` and `call`:
```go
// apiError is the shape Telegram's Bot API returns on ok:false.
type apiError struct {
	Description string `json:"description"`
	ErrorCode   int    `json:"error_code"`
	RetryAfter  int    `json:"-"` // seconds, from a 429's parameters.retry_after
}
```
```go
func call(botToken, method string, params url.Values, out any) error {
	endpoint := apiBase + "/bot" + botToken + "/" + method
	res, err := httpClient.PostForm(endpoint, params)
	if err != nil {
		return fmt.Errorf("telegram %s: %w", method, err)
	}
	defer res.Body.Close()

	var envelope struct {
		OK          bool            `json:"ok"`
		Result      json.RawMessage `json:"result"`
		Description string          `json:"description"`
		ErrorCode   int             `json:"error_code"`
		Parameters  *struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("telegram %s: decode response: %w", method, err)
	}
	if !envelope.OK {
		ae := apiError{Description: envelope.Description, ErrorCode: envelope.ErrorCode}
		if envelope.Parameters != nil {
			ae.RetryAfter = envelope.Parameters.RetryAfter
		}
		return ae
	}
	if out != nil && len(envelope.Result) > 0 {
		if err := json.Unmarshal(envelope.Result, out); err != nil {
			return fmt.Errorf("telegram %s: decode result: %w", method, err)
		}
	}
	return nil
}
```

3. Add after `apiError.Error()`:
```go
// RateLimited reports whether err is a Telegram 429 (Too Many Requests)
// response and, if so, how long to wait before retrying — the response's
// retry_after, or 1 second if Telegram didn't include one.
func RateLimited(err error) (time.Duration, bool) {
	var ae apiError
	if !errors.As(err, &ae) || ae.ErrorCode != http.StatusTooManyRequests {
		return 0, false
	}
	if ae.RetryAfter <= 0 {
		return time.Second, true
	}
	return time.Duration(ae.RetryAfter) * time.Second, true
}
```

4. Add near `SendMessage`:
```go
// InlineButton is a single-button inline keyboard row shown below a
// message. Exactly one of WebAppURL (opens a Telegram Mini App with
// initData auto-login) or URL (opens a plain link) should be set.
type InlineButton struct {
	Label     string
	WebAppURL string
	URL       string
}

// SendMessageWithButton sends text to chatID, with a single inline button
// below it if button is non-nil. The only caller in production is
// internal/outbox.ProcessNext — every bot reply is enqueued into
// bot_outbox first, never sent directly by a request handler.
func SendMessageWithButton(botToken string, chatID int64, text string, button *InlineButton) error {
	params := url.Values{
		"chat_id": {fmt.Sprintf("%d", chatID)},
		"text":    {text},
	}
	if button != nil {
		btn := map[string]any{"text": button.Label}
		if button.WebAppURL != "" {
			btn["web_app"] = map[string]string{"url": button.WebAppURL}
		} else {
			btn["url"] = button.URL
		}
		markup, err := json.Marshal(map[string]any{"inline_keyboard": [][]map[string]any{{btn}}})
		if err != nil {
			return fmt.Errorf("telegram sendMessage: marshal reply_markup: %w", err)
		}
		params.Set("reply_markup", string(markup))
	}
	return call(botToken, "sendMessage", params, nil)
}
```

5. Add near `ParseStartToken`:
```go
// IsBareStart reports whether text is exactly "/start" (optionally
// "/start@botname"), with no deep-link token — a user who opened the bot
// directly instead of following a t.me link from the site.
func IsBareStart(text string) bool {
	fields := strings.Fields(text)
	if len(fields) != 1 {
		return false
	}
	cmd := fields[0]
	return cmd == "/start" || strings.HasPrefix(cmd, "/start@")
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd server && go test ./internal/telegram/... -v`
Expected: PASS (all tests, old and new)

- [ ] **Step 5: Commit**

```bash
git add server/internal/telegram/telegram.go server/internal/telegram/telegram_test.go
git commit -m "$(cat <<'EOF'
Add telegram.InlineButton/SendMessageWithButton, 429 retry_after parsing, IsBareStart

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: `bot_outbox` table and store accessors

**Files:**
- Create: `server/internal/store/migrations/008_bot_outbox.sql`
- Create: `server/internal/store/bot_outbox.go`
- Create: `server/internal/store/bot_outbox_test.go`

**Interfaces:**
- Consumes: nothing (store package stays free of any `internal/telegram` import here — `OutboxButton` is a plain-string local type so this table's shape doesn't leak Telegram's wire format into the store layer).
- Produces: `type OutboxButton struct{Label, Type, Target string}`, `type OutboxMessage struct{ID, ChatID int64; Text string; Button OutboxButton}`, `(*Store).EnqueueBotMessage(chatID int64, text string, button OutboxButton, priority int, at time.Time) error`, `(*Store).NextPendingOutboxMessage() (OutboxMessage, bool, error)`, `(*Store).MarkOutboxSent(id int64, at time.Time) error`, `(*Store).MarkOutboxFailed(id int64, errText string) error`. Task 5 (`internal/outbox`) depends on all of these.

- [ ] **Step 1: Write the migration**

`server/internal/store/migrations/008_bot_outbox.sql`:
```sql
-- Migration 008: bot_outbox — priority send queue for every bot-originated
-- message (webhook replies, reminders, admin broadcasts from
-- ucimo-content-admin's "Бот" → "Рассылки" page), drained by
-- internal/outbox.ProcessNext at a rate-limited pace so Bot API limits are
-- never hit. See docs/superpowers/specs/2026-09-22-bot-messages-design.md.
--
-- chat_id is TEXT (not a numeric type), matching the existing convention
-- for Telegram ids (identities.provider_uid) elsewhere in this schema.

CREATE TABLE IF NOT EXISTS bot_outbox (
	id            {{.AutoID}},
	chat_id       TEXT NOT NULL,
	text          TEXT NOT NULL,
	button_label  TEXT NOT NULL DEFAULT '',
	button_type   TEXT NOT NULL DEFAULT '',
	button_target TEXT NOT NULL DEFAULT '',
	priority      SMALLINT NOT NULL,
	status        TEXT NOT NULL DEFAULT 'pending',
	error         TEXT NOT NULL DEFAULT '',
	created_at    TEXT NOT NULL,
	sent_at       TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS bot_outbox_pending_idx ON bot_outbox (status, priority, created_at);
```

- [ ] **Step 2: Write the failing tests**

`server/internal/store/bot_outbox_test.go`:
```go
package store

import "testing"

func TestEnqueueAndDequeueOutboxMessage(t *testing.T) {
	s := newStore(t)
	button := OutboxButton{Label: "Открыть", Type: "web_app", Target: "https://ucimo.ru/profile"}
	if err := s.EnqueueBotMessage(555, "hi", button, PriorityHigh, day0); err != nil {
		t.Fatalf("EnqueueBotMessage: %v", err)
	}

	msg, ok, err := s.NextPendingOutboxMessage()
	if err != nil {
		t.Fatalf("NextPendingOutboxMessage: %v", err)
	}
	if !ok {
		t.Fatal("NextPendingOutboxMessage: ok = false, want a queued row")
	}
	if msg.ChatID != 555 || msg.Text != "hi" || msg.Button != button {
		t.Errorf("dequeued = %+v, want ChatID=555 Text=hi Button=%+v", msg, button)
	}
}

func TestNextPendingOutboxMessageEmptyQueue(t *testing.T) {
	s := newStore(t)
	_, ok, err := s.NextPendingOutboxMessage()
	if err != nil {
		t.Fatalf("NextPendingOutboxMessage: %v", err)
	}
	if ok {
		t.Error("NextPendingOutboxMessage: ok = true on an empty queue, want false")
	}
}

func TestOutboxPriorityOrdersBeforeAge(t *testing.T) {
	s := newStore(t)
	// Enqueue normal first, then high — high must still come out first.
	if err := s.EnqueueBotMessage(1, "normal", OutboxButton{}, PriorityNormal, day0); err != nil {
		t.Fatal(err)
	}
	if err := s.EnqueueBotMessage(2, "high", OutboxButton{}, PriorityHigh, day0.Add(1000)); err != nil {
		t.Fatal(err)
	}
	msg, _, err := s.NextPendingOutboxMessage()
	if err != nil {
		t.Fatal(err)
	}
	if msg.Text != "high" {
		t.Errorf("first dequeued = %q, want the high-priority row even though it was enqueued later", msg.Text)
	}
}

func TestMarkOutboxSentRemovesFromPendingQueue(t *testing.T) {
	s := newStore(t)
	if err := s.EnqueueBotMessage(1, "hi", OutboxButton{}, PriorityHigh, day0); err != nil {
		t.Fatal(err)
	}
	msg, _, err := s.NextPendingOutboxMessage()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkOutboxSent(msg.ID, day0); err != nil {
		t.Fatalf("MarkOutboxSent: %v", err)
	}
	_, ok, err := s.NextPendingOutboxMessage()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("NextPendingOutboxMessage after MarkOutboxSent: ok = true, want the row gone from the pending queue")
	}
}

func TestMarkOutboxFailedRemovesFromPendingQueue(t *testing.T) {
	s := newStore(t)
	if err := s.EnqueueBotMessage(1, "hi", OutboxButton{}, PriorityHigh, day0); err != nil {
		t.Fatal(err)
	}
	msg, _, err := s.NextPendingOutboxMessage()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkOutboxFailed(msg.ID, "bot was blocked"); err != nil {
		t.Fatalf("MarkOutboxFailed: %v", err)
	}
	_, ok, err := s.NextPendingOutboxMessage()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("NextPendingOutboxMessage after MarkOutboxFailed: ok = true, want the row gone from the pending queue")
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `cd server && go test ./internal/store/... -run Outbox -v`
Expected: FAIL — types/methods undefined.

- [ ] **Step 4: Implement**

`server/internal/store/bot_outbox.go`:
```go
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// Outbox message priorities — lower runs first. See telegram.PriorityHigh/
// PriorityNormal, which this package intentionally does not import (this
// table's row shape stays plain strings/ints, not Telegram's wire types).
const (
	PriorityHigh   = 0
	PriorityNormal = 1
)

// OutboxButton is the inline button (if any) attached to a queued message.
// Type is "web_app", "url", or "" for no button.
type OutboxButton struct {
	Label  string
	Type   string
	Target string
}

// OutboxMessage is one row dequeued from bot_outbox.
type OutboxMessage struct {
	ID     int64
	ChatID int64
	Text   string
	Button OutboxButton
}

// EnqueueBotMessage queues text for chatID at priority (PriorityHigh or
// PriorityNormal). internal/outbox.ProcessNext is the only thing that ever
// actually calls the Bot API — everything else just enqueues.
func (s *Store) EnqueueBotMessage(chatID int64, text string, button OutboxButton, priority int, at time.Time) error {
	_, err := s.db.Exec(`INSERT INTO bot_outbox
		(chat_id, text, button_label, button_type, button_target, priority, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 'pending', ?)`,
		strconv.FormatInt(chatID, 10), text, button.Label, button.Type, button.Target, priority,
		at.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("enqueue bot message: %w", err)
	}
	return nil
}

// NextPendingOutboxMessage returns the oldest highest-priority pending
// message, ok=false if the queue is empty.
func (s *Store) NextPendingOutboxMessage() (OutboxMessage, bool, error) {
	row := s.db.QueryRow(`SELECT id, chat_id, text, button_label, button_type, button_target
		FROM bot_outbox WHERE status = 'pending' ORDER BY priority ASC, created_at ASC LIMIT 1`)
	var m OutboxMessage
	var chatID string
	if err := row.Scan(&m.ID, &chatID, &m.Text, &m.Button.Label, &m.Button.Type, &m.Button.Target); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return OutboxMessage{}, false, nil
		}
		return OutboxMessage{}, false, fmt.Errorf("next outbox message: %w", err)
	}
	parsed, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return OutboxMessage{}, false, fmt.Errorf("next outbox message: bad chat_id %q: %w", chatID, err)
	}
	m.ChatID = parsed
	return m, true, nil
}

// MarkOutboxSent records a successful send.
func (s *Store) MarkOutboxSent(id int64, at time.Time) error {
	_, err := s.db.Exec(`UPDATE bot_outbox SET status = 'sent', sent_at = ? WHERE id = ?`,
		at.UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("mark outbox sent: %w", err)
	}
	return nil
}

// MarkOutboxFailed records a permanent failure. Never called for a 429 —
// internal/outbox.ProcessNext leaves those rows pending and retries later.
func (s *Store) MarkOutboxFailed(id int64, errText string) error {
	_, err := s.db.Exec(`UPDATE bot_outbox SET status = 'failed', error = ? WHERE id = ?`,
		errText, id)
	if err != nil {
		return fmt.Errorf("mark outbox failed: %w", err)
	}
	return nil
}
```

- [ ] **Step 5: Run to verify it passes**

Run: `cd server && go test ./internal/store/... -run Outbox -v`
Expected: PASS

- [ ] **Step 6: Run the full store suite and commit**

Run: `cd server && go test ./internal/store/...`
Expected: PASS

```bash
git add server/internal/store/migrations/008_bot_outbox.sql server/internal/store/bot_outbox.go server/internal/store/bot_outbox_test.go
git commit -m "$(cat <<'EOF'
Add bot_outbox priority send queue table and store accessors

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: `internal/outbox` — the worker that actually calls the Bot API

**Files:**
- Create: `server/internal/outbox/outbox.go`
- Create: `server/internal/outbox/outbox_test.go`
- Modify: `server/internal/telegram/telegram.go` (adds the `SetAPIBaseForTesting` test seam)

**Interfaces:**
- Consumes: `store.Open`, `store.PriorityHigh/Normal`, `store.OutboxButton`, `store.EnqueueBotMessage`/`NextPendingOutboxMessage`/`MarkOutboxSent`/`MarkOutboxFailed` (Task 4); `telegram.InlineButton`, `telegram.SendMessageWithButton`, `telegram.RateLimited` (Task 3).
- Produces: `func ProcessNext(st *store.Store, botToken string, now time.Time) (ok bool, retryAfter time.Duration, err error)` — Task 6 wires this into a `main.go` ticker loop.

- [ ] **Step 1: Add a test-only seam to `internal/telegram`**

`internal/outbox`'s tests need to point Telegram calls at a local `httptest` server, the same way `internal/telegram`'s own tests do via the unexported `apiBase` var — but Go doesn't let one package's tests reach into another package's unexported vars, so export a small setter for it. In `server/internal/telegram/telegram.go`, add (in the main file, not a `_test.go` file):
```go
// SetAPIBaseForTesting points every subsequent Bot API call at base instead
// of https://api.telegram.org, and returns a func that restores the real
// value. For tests only (in this package and internal/outbox's).
func SetAPIBaseForTesting(base string) (restore func()) {
	prev := apiBase
	apiBase = base
	return func() { apiBase = prev }
}
```

- [ ] **Step 2: Write the failing tests**

`server/internal/outbox/outbox_test.go`:
```go
package outbox

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestProcessNextEmptyQueue(t *testing.T) {
	st := newTestStore(t)
	ok, wait, err := ProcessNext(st, "tok", time.Now())
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	if ok || wait != 0 {
		t.Errorf("ProcessNext on empty queue = (%v, %v), want (false, 0)", ok, wait)
	}
}

func TestProcessNextSendsAndMarksSent(t *testing.T) {
	var gotChatID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotChatID = r.Form.Get("chat_id")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer srv.Close()
	restore := telegram.SetAPIBaseForTesting(srv.URL)
	defer restore()

	st := newTestStore(t)
	if err := st.EnqueueBotMessage(555, "hi", store.OutboxButton{}, store.PriorityHigh, time.Now()); err != nil {
		t.Fatal(err)
	}

	ok, wait, err := ProcessNext(st, "tok", time.Now())
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	if !ok || wait != 0 {
		t.Fatalf("ProcessNext = (%v, %v), want (true, 0)", ok, wait)
	}
	if gotChatID != "555" {
		t.Errorf("chat_id sent to Telegram = %q, want 555", gotChatID)
	}
	if _, pending, _ := st.NextPendingOutboxMessage(); pending {
		t.Error("message still pending after a successful send, want it marked sent")
	}
}

func TestProcessNextLeavesRowPendingOn429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"ok":false,"error_code":429,"description":"Too Many Requests","parameters":{"retry_after":3}}`))
	}))
	defer srv.Close()
	restore := telegram.SetAPIBaseForTesting(srv.URL)
	defer restore()

	st := newTestStore(t)
	if err := st.EnqueueBotMessage(555, "hi", store.OutboxButton{}, store.PriorityHigh, time.Now()); err != nil {
		t.Fatal(err)
	}

	ok, wait, err := ProcessNext(st, "tok", time.Now())
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	if !ok || wait != 3*time.Second {
		t.Fatalf("ProcessNext = (%v, %v), want (true, 3s)", ok, wait)
	}
	if _, pending, _ := st.NextPendingOutboxMessage(); !pending {
		t.Error("message no longer pending after a 429, want it left in the queue for a retry")
	}
}

func TestProcessNextMarksFailedOnOtherErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`))
	}))
	defer srv.Close()
	restore := telegram.SetAPIBaseForTesting(srv.URL)
	defer restore()

	st := newTestStore(t)
	if err := st.EnqueueBotMessage(555, "hi", store.OutboxButton{}, store.PriorityHigh, time.Now()); err != nil {
		t.Fatal(err)
	}

	ok, wait, err := ProcessNext(st, "tok", time.Now())
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	if !ok || wait != 0 {
		t.Fatalf("ProcessNext = (%v, %v), want (true, 0)", ok, wait)
	}
	if _, pending, _ := st.NextPendingOutboxMessage(); pending {
		t.Error("message still pending after a permanent error, want it marked failed (removed from the pending queue)")
	}
}

func TestProcessNextSendsHighPriorityBeforeNormal(t *testing.T) {
	var order []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		order = append(order, r.Form.Get("text"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer srv.Close()
	restore := telegram.SetAPIBaseForTesting(srv.URL)
	defer restore()

	st := newTestStore(t)
	now := time.Now()
	if err := st.EnqueueBotMessage(1, "normal", store.OutboxButton{}, store.PriorityNormal, now); err != nil {
		t.Fatal(err)
	}
	if err := st.EnqueueBotMessage(2, "high", store.OutboxButton{}, store.PriorityHigh, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ProcessNext(st, "tok", now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ProcessNext(st, "tok", now); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "high" || order[1] != "normal" {
		t.Errorf("send order = %v, want [high, normal] regardless of enqueue time", order)
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `cd server && go test ./internal/outbox/... -v`
Expected: FAIL — `ProcessNext` doesn't exist yet, so the package doesn't compile.

- [ ] **Step 4: Implement `ProcessNext`**

`server/internal/outbox/outbox.go`:
```go
// Package outbox drains bot_outbox at a pace that stays under Telegram's
// documented Bot API limits (core.telegram.org/bots/faq: 1 msg/s per chat,
// 30 msg/s bulk) — server/main.go runs ProcessNext from a ticker at
// time.Second/25. This is the only code in serbian-app that actually calls
// the Bot API to send a message; everything else (the /start webhook,
// reminders, admin broadcasts) enqueues into bot_outbox instead.
package outbox

import (
	"time"

	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
)

// ProcessNext sends at most one queued message — the oldest, highest
// priority pending row. ok is false if the queue was empty (nothing to
// do). On a Telegram 429, the row is left pending and retryAfter tells the
// caller how long to pause before calling ProcessNext again; any other
// send error marks the row permanently failed (no retry — the usual cause
// is the user having blocked the bot).
func ProcessNext(st *store.Store, botToken string, now time.Time) (ok bool, retryAfter time.Duration, err error) {
	msg, found, err := st.NextPendingOutboxMessage()
	if err != nil {
		return false, 0, err
	}
	if !found {
		return false, 0, nil
	}

	var button *telegram.InlineButton
	if msg.Button.Type != "" {
		button = &telegram.InlineButton{Label: msg.Button.Label}
		switch msg.Button.Type {
		case "web_app":
			button.WebAppURL = msg.Button.Target
		case "url":
			button.URL = msg.Button.Target
		}
	}

	sendErr := telegram.SendMessageWithButton(botToken, msg.ChatID, msg.Text, button)
	if sendErr == nil {
		if err := st.MarkOutboxSent(msg.ID, now); err != nil {
			return true, 0, err
		}
		return true, 0, nil
	}
	if wait, limited := telegram.RateLimited(sendErr); limited {
		return true, wait, nil
	}
	if err := st.MarkOutboxFailed(msg.ID, sendErr.Error()); err != nil {
		return true, 0, err
	}
	return true, 0, nil
}
```

- [ ] **Step 5: Run to verify it passes**

Run: `cd server && go test ./internal/outbox/... ./internal/telegram/... -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add server/internal/outbox/outbox.go server/internal/outbox/outbox_test.go server/internal/telegram/telegram.go
git commit -m "$(cat <<'EOF'
Add internal/outbox.ProcessNext — the only code path that calls the Bot API

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

## Part B — serbian-app: wire it all together

### Task 6: Replace `Deps.SendTelegramMessage` with `Deps.EnqueueTelegramMessage`; start the outbox worker

**Files:**
- Modify: `server/internal/api/api.go`
- Modify: `server/internal/api/reminders.go`
- Modify: `server/internal/api/reminders_test.go`
- Modify: `server/internal/api/telegram_bot_test.go`
- Modify: `server/main.go`

**Interfaces:**
- Consumes: `store.OutboxButton`, `store.PriorityHigh/Normal`, `(*store.Store).EnqueueBotMessage` (Task 4); `telegram.InlineButton` (Task 3); `outbox.ProcessNext` (Task 5).
- Produces: `Deps.EnqueueTelegramMessage func(chatID int64, text string, button *telegram.InlineButton, priority int)` — Task 7's webhook rewrite depends on this exact signature.

This task is purely mechanical (rename + reroute), no new bot behavior yet — Task 7 changes what `telegramWebhook` actually sends. Existing tests keep passing with adjusted call sites.

- [ ] **Step 1: Rename the Deps field**

In `server/internal/api/api.go`, replace:
```go
	// SendTelegramMessage sends a chat message from the bot. Production
	// calls the real Bot API; tests capture it. A nil value is replaced by a
```
through its declaration and nil-default (the doc comment continues past what was shown earlier — keep whatever wording follows, just retarget it) with:
```go
	// EnqueueTelegramMessage queues a chat message for the outbox worker
	// (internal/outbox.ProcessNext) instead of calling the Bot API
	// directly — keeps every sender (webhook replies, reminders, admin
	// broadcasts) behind one rate limiter. Production writes a bot_outbox
	// row; tests capture the call. A nil value is replaced by a no-op in
	// Handler.
	EnqueueTelegramMessage func(chatID int64, text string, button *telegram.InlineButton, priority int)
```
And its nil-default:
```go
	if deps.EnqueueTelegramMessage == nil {
		deps.EnqueueTelegramMessage = func(chatID int64, text string, button *telegram.InlineButton, priority int) {}
	}
```
Add `"github.com/grisha/serbian-app/server/internal/telegram"` to `api.go`'s imports if not already present (check — `auth.go` in the same package already imports it, but each file needs its own import).

- [ ] **Step 2: Update `reminders.go`'s two call sites**

In `server/internal/api/reminders.go`, both:
```go
deps.SendTelegramMessage(tu.ChatID, pickMessage(allDoneMessages))
```
and
```go
deps.SendTelegramMessage(tu.ChatID, pickMessage(inactivityMessages))
```
become:
```go
deps.EnqueueTelegramMessage(tu.ChatID, pickMessage(allDoneMessages), nil, telegram.PriorityNormal)
```
and
```go
deps.EnqueueTelegramMessage(tu.ChatID, pickMessage(inactivityMessages), nil, telegram.PriorityNormal)
```
Add the `telegram` package to `reminders.go`'s imports if not already present.

- [ ] **Step 3: Update `reminders_test.go`'s `newReminderDeps` helper**

In `server/internal/api/reminders_test.go`, change:
```go
	sink := &tgSink{}
	d := Deps{
		Course:              func() *content.Course { return c },
		Store:               st,
		Now:                 func() time.Time { return fixedNow },
		Stale:               func() bool { return false },
		Config:              config.Config{AppBaseURL: testBaseURL},
		SendTelegramMessage: sink.send,
	}
```
to:
```go
	sink := &tgSink{}
	d := Deps{
		Course:                func() *content.Course { return c },
		Store:                 st,
		Now:                   func() time.Time { return fixedNow },
		Stale:                 func() bool { return false },
		Config:                config.Config{AppBaseURL: testBaseURL},
		EnqueueTelegramMessage: sink.enqueue,
	}
```
(gofmt will realign the struct literal's colons — don't hand-align, just run `gofmt -w` after, per Step 6.)

- [ ] **Step 4: Update `telegram_bot_test.go`'s `sentTgMessage`/`tgSink`**

In `server/internal/api/telegram_bot_test.go`, replace:
```go
// sentTgMessage is one message captured by tgSink.
type sentTgMessage struct {
	ChatID int64
	Text   string
}

// tgSink is a capturing SendTelegramMessage.
type tgSink struct {
	mu   sync.Mutex
	msgs []sentTgMessage
}

func (s *tgSink) send(chatID int64, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, sentTgMessage{chatID, text})
}
```
with:
```go
// sentTgMessage is one message captured by tgSink.
type sentTgMessage struct {
	ChatID   int64
	Text     string
	Button   *telegram.InlineButton
	Priority int
}

// tgSink is a capturing EnqueueTelegramMessage.
type tgSink struct {
	mu   sync.Mutex
	msgs []sentTgMessage
}

func (s *tgSink) enqueue(chatID int64, text string, button *telegram.InlineButton, priority int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, sentTgMessage{chatID, text, button, priority})
}
```
Add `"github.com/grisha/serbian-app/server/internal/telegram"` to this file's imports. Then in `newTelegramBotAPI`, change:
```go
		d.SendTelegramMessage = sink.send
```
to:
```go
		d.EnqueueTelegramMessage = sink.enqueue
```

- [ ] **Step 5: Wire `main.go`: the enqueue closure and the outbox worker goroutine**

In `server/main.go`, replace:
```go
	sendTelegramMessage := func(chatID int64, text string) {
		if err := telegram.SendMessage(cfg.TelegramBotToken, chatID, text); err != nil {
			log.Printf("telegram: send message: %v", err)
		}
	}
```
with:
```go
	enqueueTelegramMessage := func(chatID int64, text string, button *telegram.InlineButton, priority int) {
		ob := store.OutboxButton{}
		if button != nil {
			ob.Label = button.Label
			if button.WebAppURL != "" {
				ob.Type, ob.Target = "web_app", button.WebAppURL
			} else {
				ob.Type, ob.Target = "url", button.URL
			}
		}
		if err := st.EnqueueBotMessage(chatID, text, ob, priority, time.Now()); err != nil {
			log.Printf("telegram: enqueue message: %v", err)
		}
	}
```
Then update the `apiDeps` literal:
```go
		SendTelegramMessage:   sendTelegramMessage,
```
becomes:
```go
		EnqueueTelegramMessage: enqueueTelegramMessage,
```
Add `"github.com/grisha/serbian-app/server/internal/outbox"` to `main.go`'s imports, and add the worker goroutine near the existing reminder-sweep goroutine (after the `if cfg.TelegramEnabled() { go func() { ... reminder sweep ... }() }` block):
```go
	// Outbox worker: the only thing that actually calls the Bot API. A
	// tight tick (25/s — Telegram's documented bulk cap is 30/s, see
	// docs/superpowers/specs/2026-09-22-bot-messages-design.md) keeps
	// high-priority /start replies near-instant while still respecting the
	// limit when a broadcast or reminder sweep queues many rows at once.
	if cfg.TelegramEnabled() {
		go func() {
			t := time.NewTicker(time.Second / 25)
			defer t.Stop()
			for range t.C {
				_, retryAfter, err := outbox.ProcessNext(st, cfg.TelegramBotToken, time.Now())
				if err != nil {
					log.Printf("outbox: process: %v", err)
				}
				if retryAfter > 0 {
					time.Sleep(retryAfter)
				}
			}
		}()
	}
```

- [ ] **Step 6: Remove now-dead `telegram.SendMessage` and repoint its test**

`main.go` was the only production caller of `telegram.SendMessage` (confirm with `grep -rn "telegram.SendMessage(" server/ --include='*.go'` — it should only match `SendMessageWithButton` now and the test file). Remove `SendMessage` from `server/internal/telegram/telegram.go` (`SendMessageWithButton(botToken, chatID, text, nil)` fully replaces it). In `server/internal/telegram/telegram_test.go`, change `TestSendMessageSendsChatIDAndText` to call `SendMessageWithButton(..., nil)` instead of the removed `SendMessage`, keeping the same assertions (chat_id/text form-encoding).

- [ ] **Step 7: Format, build, and run the full suite**

```bash
cd server
gofmt -w internal/api/api.go internal/api/reminders.go internal/api/reminders_test.go internal/api/telegram_bot_test.go internal/telegram/telegram.go internal/telegram/telegram_test.go main.go
go build ./...
go test ./...
```
Expected: builds clean, all tests PASS (webhook tests still pass — Task 7 changes *what* gets enqueued, not the enqueue mechanism itself, so `TestTelegramWebhookUnknownTokenSendsNoMessage` still passes exactly as before since that behavior change is Task 7's job).

- [ ] **Step 8: Commit**

```bash
git add server/internal/api/api.go server/internal/api/reminders.go server/internal/api/reminders_test.go server/internal/api/telegram_bot_test.go server/internal/telegram/telegram.go server/internal/telegram/telegram_test.go server/main.go
git commit -m "$(cat <<'EOF'
Route every bot send through the outbox queue instead of calling the Bot API directly

Deps.SendTelegramMessage -> Deps.EnqueueTelegramMessage; starts the
outbox worker goroutine in main.go. No behavior change yet — webhook
reply text/scenarios are unchanged until the next commit.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: Rewrite `telegramWebhook` — all 8 scenarios, templates, buttons, priority

**Files:**
- Modify: `server/internal/api/auth.go`
- Modify: `server/internal/api/telegram_bot_test.go`

**Interfaces:**
- Consumes: `telegram.MessageKey` consts, `telegram.DefaultMessages`, `telegram.Substitute`, `telegram.DisplayName`, `telegram.SupportURL`, `telegram.IsBareStart`, `telegram.InlineButton`, `telegram.PriorityHigh` (Tasks 2, 3); `(*store.Store).BotMessageText` (Task 1); `Deps.EnqueueTelegramMessage` (Task 6).
- Produces: the new `telegramWebhook` behavior — nothing downstream depends on new symbols beyond this task's own tests.

- [ ] **Step 1: Write the failing tests**

Add `"github.com/grisha/serbian-app/server/internal/telegram"` to `telegram_bot_test.go`'s imports if Task 6 didn't already (it did, in Step 4 above — skip if present). Replace `TestTelegramWebhookUnknownTokenSendsNoMessage` and add new tests:

```go
func TestTelegramWebhookUnknownTokenSendsExpiredNotice(t *testing.T) {
	h, _, sink := newTelegramBotAPI(t)
	rr := postWebhook(h, tgWebhookSecret, webhookUpdate(1, 42, "neo", "Neo", "/start never-issued-token"))
	if rr.Code != http.StatusOK {
		t.Fatalf("webhook unknown token = %d, want 200", rr.Code)
	}
	msgs := sink.all()
	if len(msgs) != 1 {
		t.Fatalf("sent %d messages for an unknown token, want 1 (the expired notice)", len(msgs))
	}
	want := telegram.Substitute(telegram.DefaultMessages[telegram.MsgLoginTokenExpired], "Neo")
	if msgs[0].Text != want {
		t.Errorf("text = %q, want %q", msgs[0].Text, want)
	}
	if msgs[0].Priority != telegram.PriorityHigh {
		t.Errorf("priority = %d, want PriorityHigh", msgs[0].Priority)
	}
	if msgs[0].Button != nil {
		t.Errorf("button = %+v, want nil for the expired notice", msgs[0].Button)
	}
}

func TestTelegramWebhookBareStartSendsGreeting(t *testing.T) {
	h, _, sink := newTelegramBotAPI(t)
	rr := postWebhook(h, tgWebhookSecret, webhookUpdate(1, 42, "neo_bot", "Neo", "/start"))
	if rr.Code != http.StatusOK {
		t.Fatalf("webhook bare start = %d, want 200", rr.Code)
	}
	msgs := sink.all()
	if len(msgs) != 1 {
		t.Fatalf("sent %d messages for a bare /start, want 1 (the greeting)", len(msgs))
	}
	want := telegram.Substitute(telegram.DefaultMessages[telegram.MsgStartGreeting], "Neo")
	if msgs[0].Text != want {
		t.Errorf("text = %q, want %q", msgs[0].Text, want)
	}
	if msgs[0].Button == nil || msgs[0].Button.WebAppURL != testBaseURL+"/profile" {
		t.Errorf("button = %+v, want a Mini App button to %s/profile", msgs[0].Button, testBaseURL)
	}
}

func TestTelegramWebhookBareStartAtMentionVariant(t *testing.T) {
	h, _, sink := newTelegramBotAPI(t)
	rr := postWebhook(h, tgWebhookSecret, webhookUpdate(1, 42, "neo_bot", "Neo", "/start@"+tgBotUsername))
	if rr.Code != http.StatusOK {
		t.Fatalf("webhook bare start@bot = %d, want 200", rr.Code)
	}
	if len(sink.all()) != 1 {
		t.Errorf("sent %d messages for /start@%s, want 1", len(sink.all()), tgBotUsername)
	}
}

func TestTelegramLoginDistinguishesNewVsExistingAccount(t *testing.T) {
	h, _, sink := newTelegramBotAPI(t)

	firstStart := anon(h, "POST", "/api/auth/telegram/start", "")
	firstToken := startToken(t, firstStart)
	postWebhook(h, tgWebhookSecret, webhookUpdate(555, 424242, "neo_bot", "Neo", "/start "+firstToken))

	secondStart := anon(h, "POST", "/api/auth/telegram/start", "")
	secondToken := startToken(t, secondStart)
	postWebhook(h, tgWebhookSecret, webhookUpdate(555, 424242, "neo_bot", "Neo", "/start "+secondToken))

	msgs := sink.all()
	if len(msgs) != 2 {
		t.Fatalf("sent %d messages, want 2 (one per login)", len(msgs))
	}
	wantNew := telegram.Substitute(telegram.DefaultMessages[telegram.MsgLoginSuccessNew], "Neo")
	wantExisting := telegram.Substitute(telegram.DefaultMessages[telegram.MsgLoginSuccessExisting], "Neo")
	if msgs[0].Text != wantNew {
		t.Errorf("first login text = %q, want the new-account greeting %q", msgs[0].Text, wantNew)
	}
	if msgs[1].Text != wantExisting {
		t.Errorf("second login text = %q, want the returning-account greeting %q", msgs[1].Text, wantExisting)
	}
	if msgs[0].Button == nil || msgs[0].Button.WebAppURL != testBaseURL+"/profile" {
		t.Errorf("first login button = %+v, want a Mini App button to %s/profile", msgs[0].Button, testBaseURL)
	}
}

func TestTelegramWebhookUsesBotMessageOverride(t *testing.T) {
	h, st, sink := newTelegramBotAPI(t)
	if err := st.SetBotMessageText(string(telegram.MsgStartGreeting), "Custom override {name}!", fixedNow); err != nil {
		t.Fatal(err)
	}
	postWebhook(h, tgWebhookSecret, webhookUpdate(1, 42, "neo", "Neo", "/start"))
	msgs := sink.all()
	if len(msgs) != 1 || msgs[0].Text != "Custom override Neo!" {
		t.Fatalf("messages = %+v, want the DB override substituted", msgs)
	}
}
```

`fixedNow` must already be defined in the `api` test package (used by `reminders_test.go`) — if it isn't visible from this file (it's package-level, so it is), no change needed.

Also strengthen the existing `TestTelegramLinkTakenReportsError` and `TestTelegramLinkFullRoundTrip` with one assertion each. In `TestTelegramLinkTakenReportsError`, after the existing `sink.all()` length check, add:
```go
	wantTaken := telegram.Substitute(telegram.DefaultMessages[telegram.MsgLinkTaken], "Taken")
	if msgs := sink.all(); msgs[0].Text != wantTaken || msgs[0].Button == nil || msgs[0].Button.URL != telegram.SupportURL {
		t.Errorf("taken message = %+v, want text %q with a support button", msgs[0], wantTaken)
	}
```
In `TestTelegramLinkFullRoundTrip`, no button/text assertion is added (that test doesn't capture `sink` today — leave it as is; `TestTelegramWebhookUsesBotMessageOverride` and the two new tests above already cover template substitution thoroughly).

- [ ] **Step 2: Run to verify the new/changed tests fail**

Run: `cd server && go test ./internal/api/... -run Telegram -v`
Expected: FAIL — old behavior still in place (no greeting for bare start, silent on expired token, no new/existing distinction, `SetBotMessageText` unused-override not applied).

- [ ] **Step 3: Rewrite `telegramWebhook` and add its helper methods**

In `server/internal/api/auth.go`, replace the entire `telegramWebhook` function body with:
```go
func (h handlers) telegramWebhook(w http.ResponseWriter, r *http.Request) {
	if !h.Config.TelegramEnabled() || h.TelegramWebhookSecret == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != h.TelegramWebhookSecret {
		fail(w, http.StatusUnauthorized, "bad secret")
		return
	}
	w.WriteHeader(http.StatusOK)

	var update struct {
		Message *struct {
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			Text string `json:"text"`
			From struct {
				ID        int64  `json:"id"`
				Username  string `json:"username"`
				FirstName string `json:"first_name"`
			} `json:"from"`
		} `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil || update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	name := telegram.DisplayName(update.Message.From.FirstName, update.Message.From.Username)

	token, ok := telegram.ParseStartToken(update.Message.Text)
	if !ok {
		if telegram.IsBareStart(update.Message.Text) {
			h.sendBotMessage(chatID, telegram.MsgStartGreeting, name, h.miniAppButton())
		}
		return
	}
	callerUserID, ok := h.TelegramPending.Lookup(token)
	if !ok {
		h.sendBotMessage(chatID, telegram.MsgLoginTokenExpired, name, nil)
		return
	}

	u := auth.TelegramUser{
		ID:        update.Message.From.ID,
		Username:  update.Message.From.Username,
		FirstName: update.Message.From.FirstName,
		AuthDate:  h.Now(),
	}

	if callerUserID == "" {
		userID, created, err := h.resolveTelegramLogin(u)
		if err != nil {
			log.Printf("telegram webhook: resolve login: %v", err)
			h.TelegramPending.Fail(token, "internal error")
			h.sendBotMessage(chatID, telegram.MsgLoginError, name, h.supportButton())
			return
		}
		h.TelegramPending.Resolve(token, userID)
		key := telegram.MsgLoginSuccessExisting
		if created {
			key = telegram.MsgLoginSuccessNew
		}
		h.sendBotMessage(chatID, key, name, h.miniAppButton())
		return
	}

	if err := h.resolveTelegramLink(callerUserID, u); err != nil {
		if errors.Is(err, errTelegramTaken) {
			h.TelegramPending.Fail(token, "telegram_taken")
			h.sendBotMessage(chatID, telegram.MsgLinkTaken, name, h.supportButton())
			return
		}
		log.Printf("telegram webhook: resolve link: %v", err)
		h.TelegramPending.Fail(token, "internal error")
		h.sendBotMessage(chatID, telegram.MsgLinkError, name, h.supportButton())
		return
	}
	h.TelegramPending.Resolve(token, callerUserID)
	h.sendBotMessage(chatID, telegram.MsgLinkSuccess, name, h.miniAppButton())
}

// sendBotMessage resolves key to its current text (an admin override in
// bot_messages, falling back to telegram.DefaultMessages), substitutes
// {name}, and enqueues it at high priority — every /start-flow reply is a
// direct reaction to something the user just did, so it always jumps ahead
// of reminders/broadcasts in the outbox.
func (h handlers) sendBotMessage(chatID int64, key telegram.MessageKey, name string, button *telegram.InlineButton) {
	text, ok, err := h.Store.BotMessageText(string(key))
	if err != nil {
		log.Printf("telegram webhook: bot message %s: %v", key, err)
	}
	if !ok {
		text = telegram.DefaultMessages[key]
	}
	h.EnqueueTelegramMessage(chatID, telegram.Substitute(text, name), button, telegram.PriorityHigh)
}

// miniAppButton opens the Mini App (auto-login via Telegram initData) — the
// same target telegram.SetChatMenuButton already points at.
func (h handlers) miniAppButton() *telegram.InlineButton {
	return &telegram.InlineButton{Label: "Открыть Учимо", WebAppURL: h.Config.AppBaseURL + "/profile"}
}

// supportButton points at the same account the website's support card
// links to (web/src/lib/supportModal.ts).
func (h handlers) supportButton() *telegram.InlineButton {
	return &telegram.InlineButton{Label: "Написать в поддержку", URL: telegram.SupportURL}
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd server && go test ./internal/api/... -run Telegram -v`
Expected: PASS (all telegram tests, old and new)

- [ ] **Step 5: Run the full suite (SQLite and, if available, Postgres)**

```bash
cd server
go test ./...
make test-pg   # only if a local test Postgres is set up; skip otherwise, CI will run it
```
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add server/internal/api/auth.go server/internal/api/telegram_bot_test.go
git commit -m "$(cat <<'EOF'
Give telegramWebhook all 8 friendly, admin-editable scenarios

Bare /start now greets instead of staying silent; an expired/unknown
token now gets a friendly notice instead of staying silent; a login
now distinguishes a brand-new account from a returning one
(resolveTelegramLogin's `created` was previously discarded).

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

## Part C — ucimo-content-admin: editor + broadcasts

Before starting this part, confirm the local dev tunnel is up (`lsof -nP -iTCP:5433 -sTCP:LISTEN` and `:5436`) or start it (`cd /Users/grisha/plans/ucimo-content-admin && make tunnel &`) — `make dev`/tests that touch `ProdDbService` need it, though per Task 8/10 below the actual automated tests deliberately avoid hitting the tunnel (see their "Interfaces" notes) so this is only needed for manual verification (Task 14) and `make dev`.

### Task 8: `bot-messages` backend module

**Files:**
- Create: `server/src/bot-messages/bot-messages.service.ts`
- Create: `server/src/bot-messages/bot-messages.service.spec.ts`
- Create: `server/src/bot-messages/bot-messages.controller.ts`
- Create: `server/src/bot-messages/bot-messages.controller.spec.ts`
- Create: `server/src/bot-messages/dto/update-bot-message.dto.ts`
- Create: `server/src/bot-messages/bot-messages.module.ts`
- Modify: `server/src/app.module.ts`

**Interfaces:**
- Consumes: `ProdDbService` (existing, `../prod-db/prod-db.service`).
- Produces: `GET /api/bot-messages` → `BotMessage[]`, `PUT /api/bot-messages/:key` `{text}` → `BotMessage`. Task 9's frontend depends on this exact shape.

**Testing note (deviates from the `links` module's "real Postgres, no mocks" precedent):** `bot_messages` lives in **serbian-app's live production database**, not admin's own local content DB — unlike `MarketingLink` (which `links.service.spec.ts` safely tests for real against `TEST_DATABASE_URL`, admin's own DB). Automated tests here follow the *actual* existing convention for `ProdDbService`-backed modules instead (see `insights.controller.dashboard.spec.ts`): a controller-level "503 when the tunnel/prod DB is unreachable" test, plus service-level tests for logic that runs *before* any DB call (validation guards), using a stub `ProdDbService` whose `query()` throws if ever invoked — so these tests never need the tunnel and never touch real data.

- [ ] **Step 1: Write the failing tests**

`server/src/bot-messages/bot-messages.service.spec.ts`:
```typescript
import { BadRequestException, NotFoundException } from '@nestjs/common';
import { BotMessagesService, BOT_MESSAGE_KEYS } from './bot-messages.service';

// A ProdDbService stand-in whose query() fails the test if it's ever
// called — proves the validation guards below run before any DB access,
// so these tests need no tunnel and touch no real data.
const explodingDb = {
  query: () => {
    throw new Error('query() should not have been called');
  },
};

describe('BotMessagesService.update validation', () => {
  const service = new BotMessagesService(explodingDb as any);

  it('rejects an unknown key with 404', async () => {
    await expect(service.update('not-a-real-key', 'hello')).rejects.toBeInstanceOf(NotFoundException);
  });

  it('rejects blank text with 400', async () => {
    await expect(service.update(BOT_MESSAGE_KEYS[0].key, '   ')).rejects.toBeInstanceOf(BadRequestException);
  });
});

describe('BOT_MESSAGE_KEYS', () => {
  it('has exactly the 8 keys serbian-app defines in telegram.MessageKey', () => {
    const keys = BOT_MESSAGE_KEYS.map((m) => m.key).sort();
    expect(keys).toEqual(
      [
        'link_error',
        'link_success',
        'link_taken',
        'login_error',
        'login_success_existing',
        'login_success_new',
        'login_token_expired',
        'start_greeting',
      ].sort(),
    );
  });
});
```

`server/src/bot-messages/bot-messages.controller.spec.ts`:
```typescript
import { Test, TestingModule } from '@nestjs/testing';
import { INestApplication } from '@nestjs/common';
import request from 'supertest';
import { BotMessagesModule } from './bot-messages.module';

describe('GET /bot-messages', () => {
  let app: INestApplication;
  const originalUrl = process.env.PROD_DATABASE_URL;

  beforeAll(async () => {
    // No tunnel in CI/local-without-tunnel: PROD_DATABASE_URL unset means
    // ProdDbService.query rejects immediately with ServiceUnavailableException.
    delete process.env.PROD_DATABASE_URL;
    const moduleRef: TestingModule = await Test.createTestingModule({
      imports: [BotMessagesModule],
    }).compile();
    app = moduleRef.createNestApplication();
    await app.init();
  });

  afterAll(async () => {
    process.env.PROD_DATABASE_URL = originalUrl;
    await app.close();
  });

  it('returns 503 when the prod DB is unreachable', async () => {
    await request(app.getHttpServer()).get('/bot-messages').expect(503);
  });
});
```

- [ ] **Step 2: Run to verify they fail**

Run: `cd server && npx jest bot-messages`
Expected: FAIL — module doesn't exist yet.

- [ ] **Step 3: Implement**

`server/src/bot-messages/dto/update-bot-message.dto.ts`:
```typescript
import { IsNotEmpty, IsString } from 'class-validator';

export class UpdateBotMessageDto {
  @IsString()
  @IsNotEmpty()
  text!: string;
}
```

`server/src/bot-messages/bot-messages.service.ts`:
```typescript
import { BadRequestException, Injectable, NotFoundException } from '@nestjs/common';
import { ProdDbService } from '../prod-db/prod-db.service';

export interface BotMessageMeta {
  key: string;
  label: string;
  hint: string;
}

export interface BotMessage extends BotMessageMeta {
  text: string;
  updatedAt: string;
}

// The 8 editable templates. `key` must match
// server/internal/telegram/messages.go's MessageKey constants on the
// serbian-app side exactly — that string is what ties a bot_messages row
// to the scenario that reads it. See
// docs/superpowers/specs/2026-09-22-bot-messages-design.md.
export const BOT_MESSAGE_KEYS: BotMessageMeta[] = [
  { key: 'start_greeting', label: 'Приветствие (голый /start)', hint: 'Открыли бота напрямую, без ссылки с сайта.' },
  { key: 'login_success_new', label: 'Успешная регистрация', hint: 'Впервые вошли через Telegram с сайта.' },
  { key: 'login_success_existing', label: 'Повторный вход', hint: 'Уже был аккаунт, вошли через Telegram снова.' },
  { key: 'login_token_expired', label: 'Ссылка устарела', hint: 'Токен диплинка неизвестен или просрочен.' },
  { key: 'login_error', label: 'Ошибка входа', hint: 'Внутренняя ошибка при входе через Telegram.' },
  {
    key: 'link_success',
    label: 'Telegram привязан',
    hint: 'Привязка Telegram к уже созданному аккаунту (из профиля).',
  },
  { key: 'link_taken', label: 'Telegram уже привязан', hint: 'Этот Telegram уже привязан к другому аккаунту.' },
  { key: 'link_error', label: 'Ошибка привязки', hint: 'Внутренняя ошибка при привязке Telegram.' },
];

interface MessageRow {
  key: string;
  text: string;
  updated_at: string;
}

function rfc3339(d: Date): string {
  return d.toISOString().replace(/\.\d{3}Z$/, 'Z');
}

@Injectable()
export class BotMessagesService {
  constructor(private readonly db: ProdDbService) {}

  async findAll(): Promise<BotMessage[]> {
    const rows = await this.db.query<MessageRow>(`SELECT key, text, updated_at FROM bot_messages`);
    const byKey = new Map(rows.map((r) => [r.key, r]));
    return BOT_MESSAGE_KEYS.map((meta) => {
      const row = byKey.get(meta.key);
      return { ...meta, text: row?.text ?? '', updatedAt: row?.updated_at ?? '' };
    });
  }

  async update(key: string, text: string): Promise<BotMessage> {
    const meta = BOT_MESSAGE_KEYS.find((m) => m.key === key);
    if (!meta) throw new NotFoundException(`Unknown bot message key "${key}"`);
    if (text.trim() === '') throw new BadRequestException('text must not be blank');

    const updatedAt = rfc3339(new Date());
    await this.db.query(`UPDATE bot_messages SET text = $1, updated_at = $2 WHERE key = $3`, [
      text,
      updatedAt,
      key,
    ]);
    return { ...meta, text, updatedAt };
  }
}
```

`server/src/bot-messages/bot-messages.controller.ts`:
```typescript
import { Body, Controller, Get, Param, Put } from '@nestjs/common';
import { BotMessagesService } from './bot-messages.service';
import { UpdateBotMessageDto } from './dto/update-bot-message.dto';

@Controller('bot-messages')
export class BotMessagesController {
  constructor(private readonly botMessages: BotMessagesService) {}

  @Get()
  findAll() {
    return this.botMessages.findAll();
  }

  @Put(':key')
  update(@Param('key') key: string, @Body() dto: UpdateBotMessageDto) {
    return this.botMessages.update(key, dto.text);
  }
}
```

`server/src/bot-messages/bot-messages.module.ts`:
```typescript
import { Module } from '@nestjs/common';
import { ProdDbModule } from '../prod-db/prod-db.module';
import { BotMessagesController } from './bot-messages.controller';
import { BotMessagesService } from './bot-messages.service';

@Module({
  imports: [ProdDbModule],
  controllers: [BotMessagesController],
  providers: [BotMessagesService],
})
export class BotMessagesModule {}
```

In `server/src/app.module.ts`, add the import and register it:
```typescript
import { BotMessagesModule } from './bot-messages/bot-messages.module';
```
```typescript
@Module({
  imports: [PrismaModule, ItemsModule, ProdDbModule, InsightsModule, LinksModule, BotMessagesModule],
  controllers: [AppController],
})
export class AppModule {}
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd server && npx jest bot-messages`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/src/bot-messages server/src/app.module.ts
git commit -m "$(cat <<'EOF'
Add bot-messages backend module (GET/PUT against serbian-app's bot_messages)

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 9: `bot-messages` frontend — new "Бот" sidebar section, editor page

**Files:**
- Modify: `web/src/types.ts`
- Create: `web/src/api/bot-messages.ts`
- Create: `web/src/views/BotMessagesView.vue`
- Modify: `web/src/router.ts`
- Modify: `web/src/components/AdminSidebar.vue`
- Modify: `web/src/components/AdminSidebar.spec.ts`

**Interfaces:**
- Consumes: `GET /api/bot-messages`, `PUT /api/bot-messages/:key` (Task 8).
- Produces: route `/bot/messages`; a "Бот" sidebar section (extended by Task 11 with a second link).

- [ ] **Step 1: Add types**

Append to `web/src/types.ts`:
```typescript
export interface BotMessage {
  key: string;
  label: string;
  hint: string;
  text: string;
  updatedAt: string;
}
```

- [ ] **Step 2: Add the API client**

`web/src/api/bot-messages.ts`:
```typescript
import type { BotMessage } from '../types';

const BASE = '/api/bot-messages';

async function handle<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.message ?? `Request failed: ${res.status}`);
  }
  return res.json();
}

export function listBotMessages(): Promise<BotMessage[]> {
  return fetch(BASE).then((r) => handle(r));
}

export function updateBotMessage(key: string, text: string): Promise<BotMessage> {
  return fetch(`${BASE}/${key}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text }),
  }).then((r) => handle(r));
}
```

- [ ] **Step 3: Build the view**

`web/src/views/BotMessagesView.vue`:
```vue
<template>
  <div>
    <h1 class="mb-6 text-2xl font-semibold text-primary-700">Сообщения бота</h1>
    <p v-if="loading" class="text-sm text-text-500">Загрузка…</p>
    <p v-else-if="error" class="text-sm text-error">{{ error }}</p>
    <div v-else class="flex flex-col gap-4">
      <div v-for="m in messages" :key="m.key" class="rounded bg-white p-4 shadow-sm">
        <h2 class="text-sm font-semibold text-text-800">{{ m.label }}</h2>
        <p class="mb-2 text-xs text-text-500">{{ m.hint }}</p>
        <textarea v-model="drafts[m.key]" rows="4" class="w-full rounded border border-border p-2 text-sm"></textarea>
        <div class="mt-2 flex items-center gap-3">
          <button
            type="button"
            class="rounded bg-primary-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
            :disabled="saving === m.key || drafts[m.key] === m.text"
            @click="save(m.key)"
          >
            Сохранить
          </button>
          <span v-if="savedKey === m.key" class="text-sm text-success">Сохранено</span>
          <span v-if="saveError && saving === m.key" class="text-sm text-error">{{ saveError }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import type { BotMessage } from '../types';
import { listBotMessages, updateBotMessage } from '../api/bot-messages';

const messages = ref<BotMessage[]>([]);
const drafts = reactive<Record<string, string>>({});
const loading = ref(true);
const error = ref('');
const saving = ref<string | null>(null);
const savedKey = ref<string | null>(null);
const saveError = ref('');

onMounted(async () => {
  try {
    messages.value = await listBotMessages();
    for (const m of messages.value) drafts[m.key] = m.text;
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
});

async function save(key: string) {
  saving.value = key;
  saveError.value = '';
  savedKey.value = null;
  try {
    const updated = await updateBotMessage(key, drafts[key]);
    const idx = messages.value.findIndex((m) => m.key === key);
    if (idx !== -1) messages.value[idx] = updated;
    savedKey.value = key;
  } catch (e) {
    saveError.value = e instanceof Error ? e.message : String(e);
  } finally {
    saving.value = null;
  }
}
</script>
```

- [ ] **Step 4: Wire the route**

In `web/src/router.ts`, add the import:
```typescript
import BotMessagesView from './views/BotMessagesView.vue';
```
and the route:
```typescript
    { path: '/bot/messages', name: 'bot-messages', component: BotMessagesView },
```

- [ ] **Step 5: Add the "Бот" sidebar section**

In `web/src/components/AdminSidebar.vue`, add to the `sections` array (after the `links` entry):
```typescript
  { key: 'bot', title: 'Бот', links: [{ label: 'Сообщения', to: '/bot/messages' }] },
```
and add `bot: true` to the `openState` reactive object:
```typescript
const openState = reactive<Record<string, boolean>>({ content: true, data: true, links: true, bot: true });
```

- [ ] **Step 6: Update the sidebar test's router fixture**

In `web/src/components/AdminSidebar.spec.ts`'s `makeRouter()`, add the new route to the routes array:
```typescript
      { path: '/bot/messages', component: { template: '<div />' } },
```
Add one new test mirroring the existing "shows the links section expanded by default" test:
```typescript
  it('shows the bot section expanded by default', async () => {
    const router = makeRouter();
    await router.push('/');
    const wrapper = mount(AdminSidebar, { global: { plugins: [router] } });
    expect(wrapper.text()).toContain('Сообщения');
  });
```

- [ ] **Step 7: Run the frontend tests**

Run: `cd web && npx vitest run AdminSidebar`
Expected: PASS

Run: `cd web && npx vue-tsc -b`
Expected: no type errors

- [ ] **Step 8: Manual check (dev server)**

```bash
cd /Users/grisha/plans/ucimo-content-admin && make dev
```
Open `http://localhost:4401/bot/messages` — with the tunnel up (see Part C's preamble), confirm all 8 templates load with their seeded default text, edit one, save, reload, confirm it persisted.

- [ ] **Step 9: Commit**

```bash
git add web/src/types.ts web/src/api/bot-messages.ts web/src/views/BotMessagesView.vue web/src/router.ts web/src/components/AdminSidebar.vue web/src/components/AdminSidebar.spec.ts
git commit -m "$(cat <<'EOF'
Add "Бот" → "Сообщения" admin page to edit bot_messages templates

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 10: `broadcasts` backend module

**Files:**
- Create: `server/src/broadcasts/broadcasts.service.ts`
- Create: `server/src/broadcasts/broadcasts.service.spec.ts`
- Create: `server/src/broadcasts/broadcasts.controller.ts`
- Create: `server/src/broadcasts/broadcasts.controller.spec.ts`
- Create: `server/src/broadcasts/dto/send-broadcast.dto.ts`
- Create: `server/src/broadcasts/broadcasts.module.ts`
- Modify: `server/src/app.module.ts`

**Interfaces:**
- Consumes: `ProdDbService`.
- Produces: `GET /api/broadcasts/candidates` → `BroadcastCandidate[]`, `POST /api/broadcasts` `{text, recipients, dryRun?}` → `{count: number}`. Task 11's frontend depends on this exact shape.

**Testing note — read this before writing any test that inserts into `bot_outbox`:** `bot_outbox` is live production infrastructure — serbian-app's outbox worker (Task 5/6) polls it roughly 25 times a second and **will actually call the Telegram Bot API** for any row it finds, sending a real message to a real chat_id. An integration test that runs `POST /broadcasts` for real against the tunneled prod DB would queue rows that the live bot then genuinely sends. **Never write a test (or run one manually) that calls `send()` with `dryRun` unset/false against `PROD_DATABASE_URL` pointed at the real tunnel.** All automated tests below either stay pure (no `db` call at all) or use the standard "unreachable DB → 503" pattern (Task 8's testing note) — neither ever performs a live write.

- [ ] **Step 1: Write the failing tests**

`server/src/broadcasts/broadcasts.service.spec.ts`:
```typescript
import { BroadcastCandidate, BroadcastsService } from './broadcasts.service';

describe('BroadcastsService.resolveTargets (pure — no DB involved)', () => {
  const service = new BroadcastsService({} as any); // db is never touched by resolveTargets
  const all: BroadcastCandidate[] = [
    { id: 'u1', name: 'Аня', chatId: '111' },
    { id: 'u2', name: 'Боря', chatId: '222' },
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

`server/src/broadcasts/broadcasts.controller.spec.ts`:
```typescript
import { Test, TestingModule } from '@nestjs/testing';
import { INestApplication, ValidationPipe } from '@nestjs/common';
import request from 'supertest';
import { BroadcastsModule } from './broadcasts.module';

describe('POST /broadcasts', () => {
  let app: INestApplication;
  const originalUrl = process.env.PROD_DATABASE_URL;

  beforeAll(async () => {
    delete process.env.PROD_DATABASE_URL;
    const moduleRef: TestingModule = await Test.createTestingModule({
      imports: [BroadcastsModule],
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
      .post('/broadcasts')
      .send({ text: '', recipients: { mode: 'all' } })
      .expect(400);
  });

  it('rejects an invalid recipients.mode with 400 before ever touching the DB', async () => {
    await request(app.getHttpServer())
      .post('/broadcasts')
      .send({ text: 'hi', recipients: { mode: 'bogus' } })
      .expect(400);
  });

  it('returns 503 for a valid dry-run request when the prod DB is unreachable', async () => {
    await request(app.getHttpServer())
      .post('/broadcasts')
      .send({ text: 'hi', recipients: { mode: 'all' }, dryRun: true })
      .expect(503);
  });
});
```

- [ ] **Step 2: Run to verify they fail**

Run: `cd server && npx jest broadcasts`
Expected: FAIL — module doesn't exist yet.

- [ ] **Step 3: Implement**

`server/src/broadcasts/dto/send-broadcast.dto.ts`:
```typescript
import { Type } from 'class-transformer';
import { IsArray, IsBoolean, IsIn, IsNotEmpty, IsOptional, IsString, ValidateNested } from 'class-validator';

export class BroadcastRecipientsDto {
  @IsIn(['all', 'users'])
  mode!: 'all' | 'users';

  @IsOptional()
  @IsArray()
  @IsString({ each: true })
  userIds?: string[];
}

export class SendBroadcastDto {
  @IsString()
  @IsNotEmpty()
  text!: string;

  @ValidateNested()
  @Type(() => BroadcastRecipientsDto)
  recipients!: BroadcastRecipientsDto;

  @IsOptional()
  @IsBoolean()
  dryRun?: boolean;
}
```

`server/src/broadcasts/broadcasts.service.ts`:
```typescript
import { Injectable } from '@nestjs/common';
import { ProdDbService } from '../prod-db/prod-db.service';
import { SendBroadcastDto, BroadcastRecipientsDto } from './dto/send-broadcast.dto';

export interface BroadcastCandidate {
  id: string;
  name: string;
  chatId: string;
}

interface CandidateRow {
  id: string;
  name: string;
  chat_id: string;
}

// Must match server/internal/store/bot_outbox.go's PriorityNormal.
const PRIORITY_NORMAL = 1;

function rfc3339(d: Date): string {
  return d.toISOString().replace(/\.\d{3}Z$/, 'Z');
}

@Injectable()
export class BroadcastsService {
  constructor(private readonly db: ProdDbService) {}

  async candidates(): Promise<BroadcastCandidate[]> {
    const rows = await this.db.query<CandidateRow>(`
      SELECT u.id AS id, u.name AS name, i.provider_uid AS chat_id
      FROM users u
      JOIN identities i ON i.user_id = u.id
      WHERE i.provider = 'telegram' AND i.provider_uid NOT LIKE 'pending:%'
      ORDER BY u.name
    `);
    return rows.map((r) => ({ id: r.id, name: r.name, chatId: r.chat_id }));
  }

  // The pure part of send() — no DB write — so it's unit-testable without
  // touching Postgres (see the "Testing note" in Task 10 of the plan: a
  // real INSERT into bot_outbox is a live send, never do that from a test).
  resolveTargets(all: BroadcastCandidate[], recipients: BroadcastRecipientsDto): BroadcastCandidate[] {
    if (recipients.mode === 'all') return all;
    const wanted = new Set(recipients.userIds ?? []);
    return all.filter((c) => wanted.has(c.id));
  }

  async send(dto: SendBroadcastDto): Promise<{ count: number }> {
    const all = await this.candidates();
    const targets = this.resolveTargets(all, dto.recipients);

    if (dto.dryRun || targets.length === 0) {
      return { count: targets.length };
    }

    const createdAt = rfc3339(new Date());
    const values: string[] = [];
    const params: unknown[] = [];
    targets.forEach((t, i) => {
      const base = i * 3;
      values.push(`($${base + 1}, $${base + 2}, ${PRIORITY_NORMAL}, 'pending', $${base + 3})`);
      params.push(t.chatId, dto.text, createdAt);
    });
    await this.db.query(
      `INSERT INTO bot_outbox (chat_id, text, priority, status, created_at) VALUES ${values.join(', ')}`,
      params,
    );
    return { count: targets.length };
  }
}
```

`server/src/broadcasts/broadcasts.controller.ts`:
```typescript
import { Body, Controller, Get, Post } from '@nestjs/common';
import { BroadcastsService } from './broadcasts.service';
import { SendBroadcastDto } from './dto/send-broadcast.dto';

@Controller('broadcasts')
export class BroadcastsController {
  constructor(private readonly broadcasts: BroadcastsService) {}

  @Get('candidates')
  candidates() {
    return this.broadcasts.candidates();
  }

  @Post()
  send(@Body() dto: SendBroadcastDto) {
    return this.broadcasts.send(dto);
  }
}
```

`server/src/broadcasts/broadcasts.module.ts`:
```typescript
import { Module } from '@nestjs/common';
import { ProdDbModule } from '../prod-db/prod-db.module';
import { BroadcastsController } from './broadcasts.controller';
import { BroadcastsService } from './broadcasts.service';

@Module({
  imports: [ProdDbModule],
  controllers: [BroadcastsController],
  providers: [BroadcastsService],
})
export class BroadcastsModule {}
```

In `server/src/app.module.ts`:
```typescript
import { BroadcastsModule } from './broadcasts/broadcasts.module';
```
```typescript
@Module({
  imports: [PrismaModule, ItemsModule, ProdDbModule, InsightsModule, LinksModule, BotMessagesModule, BroadcastsModule],
  controllers: [AppController],
})
export class AppModule {}
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd server && npx jest broadcasts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/src/broadcasts server/src/app.module.ts
git commit -m "$(cat <<'EOF'
Add broadcasts backend module (candidates + queueing into bot_outbox)

resolveTargets is pure and unit-tested directly; no test ever performs
a real (non-dry-run) send against the tunnel — see the module's
testing note (a live send would message real users).

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 11: `broadcasts` frontend — composer with dry-run confirmation

**Files:**
- Modify: `web/src/types.ts`
- Create: `web/src/api/broadcasts.ts`
- Create: `web/src/views/BroadcastsView.vue`
- Modify: `web/src/router.ts`
- Modify: `web/src/components/AdminSidebar.vue`
- Modify: `web/src/components/AdminSidebar.spec.ts`

**Interfaces:**
- Consumes: `GET /api/broadcasts/candidates`, `POST /api/broadcasts` (Task 10).
- Produces: route `/bot/broadcasts`; extends the "Бот" sidebar section (added by Task 9) with a second link.

- [ ] **Step 1: Add types**

Append to `web/src/types.ts`:
```typescript
export interface BroadcastCandidate {
  id: string;
  name: string;
  chatId: string;
}
```

- [ ] **Step 2: Add the API client**

`web/src/api/broadcasts.ts`:
```typescript
import type { BroadcastCandidate } from '../types';

const BASE = '/api/broadcasts';

async function handle<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.message ?? `Request failed: ${res.status}`);
  }
  return res.json();
}

export function listCandidates(): Promise<BroadcastCandidate[]> {
  return fetch(`${BASE}/candidates`).then((r) => handle(r));
}

export type BroadcastRecipients = { mode: 'all' } | { mode: 'users'; userIds: string[] };

export function sendBroadcast(
  text: string,
  recipients: BroadcastRecipients,
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

`web/src/views/BroadcastsView.vue`:
```vue
<template>
  <div>
    <h1 class="mb-6 text-2xl font-semibold text-primary-700">Рассылки</h1>
    <p v-if="loading" class="text-sm text-text-500">Загрузка…</p>
    <p v-else-if="error" class="text-sm text-error">{{ error }}</p>
    <div v-else class="flex max-w-xl flex-col gap-4">
      <label class="flex items-center gap-2 text-sm text-text-700">
        <input type="checkbox" v-model="sendToAll" />
        Всем пользователям с Telegram ({{ candidates.length }})
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

      <textarea
        v-model="text"
        rows="5"
        placeholder="Текст сообщения"
        class="w-full rounded border border-border p-2 text-sm"
      ></textarea>

      <button
        type="button"
        class="self-start rounded bg-primary-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
        :disabled="!canSend"
        @click="openConfirm"
      >
        Отправить
      </button>
      <p v-if="sendError" class="text-sm text-error">{{ sendError }}</p>
      <p v-if="sentCount !== null" class="text-sm text-success">Поставлено в очередь: {{ sentCount }}</p>

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
import type { BroadcastCandidate } from '../types';
import { listCandidates, sendBroadcast } from '../api/broadcasts';

const candidates = ref<BroadcastCandidate[]>([]);
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
    const { count } = await sendBroadcast(text.value, recipients(), true);
    pendingCount.value = count;
    confirming.value = true;
  } catch (e) {
    sendError.value = e instanceof Error ? e.message : String(e);
  }
}

async function confirmSend() {
  try {
    const { count } = await sendBroadcast(text.value, recipients(), false);
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

- [ ] **Step 4: Wire the route**

In `web/src/router.ts`, add the import:
```typescript
import BroadcastsView from './views/BroadcastsView.vue';
```
and the route:
```typescript
    { path: '/bot/broadcasts', name: 'bot-broadcasts', component: BroadcastsView },
```

- [ ] **Step 5: Extend the "Бот" sidebar section**

In `web/src/components/AdminSidebar.vue`, change the `bot` section (added in Task 9) from:
```typescript
  { key: 'bot', title: 'Бот', links: [{ label: 'Сообщения', to: '/bot/messages' }] },
```
to:
```typescript
  {
    key: 'bot',
    title: 'Бот',
    links: [
      { label: 'Сообщения', to: '/bot/messages' },
      { label: 'Рассылки', to: '/bot/broadcasts' },
    ],
  },
```

- [ ] **Step 6: Update the sidebar test's router fixture**

In `web/src/components/AdminSidebar.spec.ts`'s `makeRouter()`, add:
```typescript
      { path: '/bot/broadcasts', component: { template: '<div />' } },
```

- [ ] **Step 7: Run the frontend tests**

Run: `cd web && npx vitest run AdminSidebar`
Expected: PASS

Run: `cd web && npx vue-tsc -b`
Expected: no type errors

- [ ] **Step 8: Manual check (dev server)**

With `make dev` still running and the tunnel up, open `http://localhost:4401/bot/broadcasts`. Confirm the candidate list loads, select one test user (not "all"), type a short test message, click "Отправить", confirm the modal shows the right count, click "Да, отправить N" — **only do this against a test/your-own Telegram-linked account**, since a real send does message the real chat. Confirm the toast shows "Поставлено в очередь: 1" and that the message actually arrives in that Telegram chat (proves the whole pipeline — admin write → outbox worker → Bot API — end to end).

- [ ] **Step 9: Commit**

```bash
git add web/src/types.ts web/src/api/broadcasts.ts web/src/views/BroadcastsView.vue web/src/router.ts web/src/components/AdminSidebar.vue web/src/components/AdminSidebar.spec.ts
git commit -m "$(cat <<'EOF'
Add "Бот" → "Рассылки" admin page with dry-run recipient-count confirmation

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

## Part D — Deploy

### Task 12: Deploy serbian-app and grant `ucimo_admin_ro` its two new, narrowly-scoped write permissions

**Files:** none (operational task)

- [ ] **Step 1: Push and deploy serbian-app**

```bash
cd /Users/grisha/plans/serbian-app
git push origin main
```
Deploy via the Dokploy API (`POST /api/application.deploy` with serbian-app's `applicationId` — see your deploy notes/memory for the base URL, API key, and id; do not paste them into this plan file). This runs migrations 007 and 008 automatically on boot (`store.Open` applies pending migrations before the server starts serving).

- [ ] **Step 2: Verify migrations applied and the seed landed**

Using the same SSH access documented in your deploy notes, connect to the prod Postgres (`srpski` database) and confirm:
```sql
SELECT count(*) FROM bot_messages;   -- want 8
SELECT count(*) FROM bot_outbox;     -- want 0 (freshly created, empty)
```

- [ ] **Step 3: Grant `ucimo_admin_ro` the two new permissions**

Still connected to the prod Postgres as an admin role, run:
```sql
GRANT SELECT, INSERT, UPDATE ON bot_messages TO ucimo_admin_ro;
GRANT INSERT ON bot_outbox TO ucimo_admin_ro;
```
This is the only change to that role's access — everything else it can touch stays read-only, per the spec's "Права на проде" section.

- [ ] **Step 4: Verify the deployed binary is actually live**

Per your deploy notes' "Verify after deploy" section: confirm the `deployments[]` entry for this commit shows `status:"done"`, then poll `https://ucimo.ru/` once more and diff the hashed JS asset filename against a fresh local `npm run build` (there's a documented ~1-2 min lag between "done" and the new container actually serving).

No commit for this task (nothing in the repo changes).

---

### Task 13: Deploy ucimo-content-admin

**Files:** none (operational task)

- [ ] **Step 1: Build and push the image, redeploy**

Follow the existing no-git-remote build-on-host flow (rsync the repo to the Dokploy host, `docker build`, tag/push to the local registry, `POST /api/application.deploy` with ucimo-content-admin's `applicationId`) — see your deploy notes for the exact commands and credentials; do not paste them into this plan file.

- [ ] **Step 2: Verify**

```bash
curl -s -o /dev/null -w '%{http_code}\n' https://admin.ucimo.ru/api/bot-messages           # 401, no creds
curl -s -o /dev/null -w '%{http_code}\n' -u "admin:<password>" https://admin.ucimo.ru/api/bot-messages  # 200
```
The second call's body should list all 8 keys with non-empty `text`.

No commit for this task (nothing in the repo changes).

---

### Task 14: End-to-end manual smoke test

**Files:** none (verification task)

- [ ] **Step 1: Bare `/start`**

In Telegram, open `@ucimoappbot` directly (search for it, not via a t.me deep link) and tap **Start**. Expect the Zdravo greeting with an "Открыть Учимо" button that opens the Mini App.

- [ ] **Step 2: New registration via the site**

On ucimo.ru, log out (or use a fresh Telegram account never linked before), click "Войти через Telegram", follow the link, tap Start in the resulting chat. Expect the "Регистрация прошла успешно" message.

- [ ] **Step 3: Returning login**

Repeat step 2 with the same Telegram account (log out on the site first). Expect the "С возвращением" message this time, not the new-account one.

- [ ] **Step 4: Edit a template live**

In admin (`https://admin.ucimo.ru/bot/messages`), change `login_token_expired`'s text to something distinctive, save. On ucimo.ru, generate a `/start` link, wait for it to expire (or use an already-consumed one), send it to the bot. Expect the edited text, not the original default — confirms live editing needs no redeploy.

- [ ] **Step 5: Broadcast to one real (your own) account**

In admin (`/bot/broadcasts`), uncheck "всем", select only your own test account, send a short distinctive message, confirm the dialog, confirm it arrives in Telegram within a couple of seconds.

- [ ] **Step 6: Rate-limit safety spot check**

Trigger the reminder sweep's normal-priority path indirectly (or just observe): while the queue has both a high-priority row (do step 1 again right as you send a broadcast to several accounts) and normal-priority rows pending, confirm the bare-`/start` greeting still arrives promptly rather than waiting behind the broadcast — this is the priority ordering working as designed.

No commit for this task (verification only). If any step fails, file it as a bug against the relevant task above rather than patching ad hoc — reopen that task's checklist.

---

## Self-Review Notes

**Spec coverage:** all 8 message keys + copy (Task 2), Zdravo + `@ucimosupport` edits (Task 2), bare-`/start` and expired-token cases that were previously silent (Task 7), new-vs-existing distinction using the previously-discarded `created` (Task 7), `bot_messages` table + admin-editable via `ucimo_admin_ro`'s new grant (Tasks 1, 8, 9, 12), outbox queue + priority + 25 msg/s cap + 429/`retry_after` handling (Tasks 3, 4, 5, 6, 12), reminders routed through the same queue (Task 6), admin broadcasts with specific-users-or-all targeting and dry-run confirmation (Tasks 10, 11), text-only broadcasts / no history / no admin-editable rate limit (respected by design — see "Вне скоупа" cross-references in Tasks 10-11).

**Deviation from the spec's testing section, and why:** the spec says the `bot-messages`/`broadcasts` NestJS modules should be tested "like `links` — real Postgres via `ProdDbService`, no mocks." Investigating the actual codebase during planning showed this was based on a mis-paraphrase: `links.service.spec.ts` tests for real against **Prisma** (admin's own local content DB), not `ProdDbService` — the one existing `ProdDbService`-backed spec (`insights.controller.dashboard.spec.ts`) only automates the "unreachable → 503" path, and it does so specifically because a real integration test against tunneled prod data is out of scope for CI (no tunnel there) and, for `bot_messages`/`bot_outbox` specifically, would be actively dangerous — a live INSERT into `bot_outbox` is a live message send to a real user. Tasks 8 and 10 follow the *actual* established pattern (503 test + pure-logic unit tests) instead of the spec's imprecise description, and call this out explicitly in each module's testing note so a future reader isn't confused by the mismatch.
