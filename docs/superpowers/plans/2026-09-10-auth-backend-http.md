# Авторизация, часть 2 из 4 — бэкенд: пакеты + эндпойнты + middleware

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Собрать серверную авторизацию поверх готового слоя БД (часть 1): пакеты `auth` (пароль/токены/Telegram), `mail`, `ratelimit`, `config`; эндпойнты `/api/auth/*` и `/api/me*`; middleware `securityHeaders` / `checkOrigin` / `requireAuth` (сессия в `HttpOnly`-куке, с мостом `X-User` на переходный релиз). Всё под тестами, `go test ./server/...` + `make test-pg` зелёные.

**Architecture:** Две фазы. **A — листовые пакеты** без HTTP: `internal/auth` (bcrypt+sha256 пароль, случайные токены + их sha256, валидация Telegram-подписей), `internal/mail` (интерфейс `Mailer` + `LogMailer` + `SMTPMailer` + 4 шаблона), `internal/ratelimit` (токен-бакет по ключу + счётчик неудачных логинов), `internal/config` (env → структура). **B — HTTP**: `api/middleware.go`, `api/auth.go`, `api/me.go`, переезд `api.Handler`/`Deps`, провязка в `main.go` (+ фоновая отправка писем, housekeeping-свип сессий). Сессия — непрозрачный токен в куке, в БД только `sha256`; `requireAuth` при отсутствии куки пробует старый `X-User` (мост, часть 3 снимает).

**Tech Stack:** Go 1.25 stdlib (`net/http`, `crypto/*`, `net/smtp`, `html/template`, `text/template`, `context`), `golang.org/x/crypto/bcrypt`, `golang.org/x/time/rate`. Тесты `go test`, `make test-pg`.

**Spec:** [docs/superpowers/specs/2026-09-09-auth-postgres-profile-design.md](../specs/2026-09-09-auth-postgres-profile-design.md) — секции 2 и 3.

**Часть 1 (готова):** [2026-09-09-auth-db.md](2026-09-09-auth-db.md) — миграции, `identities`/`sessions`/`email_tokens` таблицы + CRUD, состояние по `user_id`, мостовые `store.UserByName`/`EnsureUserByName` + хендлеры `/api/users` (эта часть их снимает). См. её раздел «Хвост для части 2».

**Части 3–4:** 3 — фронт (экраны входа, `ProfileView`, навбар); 4 — SMTP-доступы, Telegram-бот, бэкапы БД.

## Global Constraints

- Модуль Go `github.com/grisha/serbian-app`; **`go.mod` `go` = `1.25.0`, не поднимать, `toolchain`-директиву не добавлять** — Dockerfile пинит `golang:1.25-alpine`, `go 1.26` там не соберётся. При `go get` любой зависимости пинить версию, чей `go`-директив ≤ 1.25 (`go list -m -f '{{.GoVersion}}' <mod>@<ver>` для проверки). Конкретно: `golang.org/x/crypto@v0.41.0` (нужен go 1.23) — **не** `@latest` (v0.57 требует go 1.26). После любого `go mod tidy` проверить `head -3 go.mod`.
- **Пароль:** `bcrypt(base64.StdEncoding(sha256(password)), cost)` где `cost = bcryptCost = 12` (константа в `auth`, не env). Длина пароля на приёме: 8–128 символов.
- **Токены** (сессия, verify, reset): 32 байта `crypto/rand`, значение для клиента — `base64.RawURLEncoding`; в БД только `hex(sha256(raw))`. Сравнение хешей — `subtle.ConstantTimeCompare` / `hmac.Equal`.
- **Кука:** имя `__Host-session` когда `APP_BASE_URL` начинается с `https://`, иначе `session`; всегда `HttpOnly`, `SameSite=Lax`, `Path=/`; `Secure` только для `__Host-`. Значение — `base64.RawURLEncoding` от 32 байт. `MaxAge` = TTL сессии.
- **TTL / лимиты** — константы с дефолтами из спеки: сессия скользящая 30 дней, жёсткий максимум 90 дней; `verify`-токен 24 ч, `reset`-токен 1 ч; `last_seen_at` троттлится 1 ч; login rate-limit 5/мин на IP + 10/час на email; register/resend/forgot 3/час на IP + 3/час на email; мягкий лок аккаунта после 10 неудач подряд на 15 минут.
- **Защита от энумерации:** `register` и `forgot` — всегда `200` generic, ветка письма выбирается в фоне, ответ отдаётся сразу и одинаково по времени. `login` неверный — единый `401 {"error":"неверная почта или пароль"}`.
- **Письма — фоновой горутиной** (`context.WithTimeout` 15 с + один ретрай). HTTP-ответ мгновенный. Ошибку логируем без адреса.
- **`email` и токены — не в логи** ни на каком уровне.
- SQL: `?`-плейсхолдеры (rebind в `store`). Время — RFC3339 TEXT.
- **Константы TTL в `package api`** (не тянуть неэкспортируемые из `store`): `sessionTTL = 30 * 24 * time.Hour` — **должна совпадать с `store.sessionSlide`** (часть 1, `sessions.go`); `verifyTTL = 24 * time.Hour`; `resetTTL = time.Hour`; `reauthWindow = 5 * time.Minute`; `lastSeenThrottle = time.Hour`.
- **Лимитеры через хелпер:** `func allow(l *ratelimit.Limiter, key string) bool { return l == nil || l.Allow(key) }` и `func locked(f *ratelimit.FailCounter, key string) bool { return f != nil && f.Locked(key) }` — все хендлеры зовут через них, чтобы `nil`-лимитер в тестах = «пропускать».
- Комментарии в коде — по-английски, короткие доки на экспортных символах, ошибки `fmt.Errorf("...: %w", err)`. Тексты писем и user-facing строки — по-русски.
- Тесты рядом с кодом; фейковый `Mailer` (сборщик) и фейковые часы (`Now func() time.Time`, уже в `api.Deps`) в API-тестах. Инъекция `ratelimit` — интерфейс, чтобы тест мог форсить лимит.
- Коммиты подписывать `Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>`.
- Не трогать: `content/`, `internal/srs`, `internal/checker`, `internal/content`, фронт `web/`, миграции `internal/store/migrations/`.

---

## Карта файлов

### Фаза A — создаются

| Файл (+ `_test.go`) | Ответственность |
|---|---|
| `server/internal/auth/password.go` | `HashPassword` / `VerifyPassword` (bcrypt+sha256). |
| `server/internal/auth/token.go` | `NewToken() (raw, hash)`, `HashToken(raw)`. |
| `server/internal/auth/telegram.go` | `VerifyInitData` / `VerifyWidget` (HMAC по спеке Telegram). |
| `server/internal/mail/mail.go` | Интерфейс `Mailer`, `LogMailer`. |
| `server/internal/mail/templates.go` | 4 шаблона (`verify`/`reset`/`already_registered`/`password_changed`) + `Render*`. |
| `server/internal/mail/smtp.go` | `SMTPMailer` + сборка MIME. |
| `server/internal/ratelimit/ratelimit.go` | `Limiter` (токен-бакет по ключу) + `FailCounter` (мягкий лок). |
| `server/internal/config/config.go` | `Config` + `Load()` из env. |

### Фаза B — создаются

| Файл (+ `_test.go`) | Ответственность |
|---|---|
| `server/internal/api/middleware.go` | `securityHeaders`, `checkOrigin`, `requireAuth` (+ мост `X-User`), контекст-хелперы, `session` cookie helpers. |
| `server/internal/api/auth.go` | Хендлеры `/api/auth/*`. |
| `server/internal/api/me.go` | Хендлеры `/api/me*`. |

### Модифицируются

| Файл | Что |
|---|---|
| `go.mod` / `go.sum` | + `golang.org/x/crypto`, `golang.org/x/time`. |
| `server/internal/api/api.go` | `Deps` + `Mailer`/`Limiter`/`Fails`/`Config`; `Handler` вешает middleware-цепочку и регистрирует auth/me-роуты; снять `GET/POST /api/users`; `handlers.user()` → чтение из контекста (заполняет `requireAuth`). |
| `server/internal/api/dto.go` | `sessionUserDTO`, `meDTO`, `deviceDTO` + request-структуры. |
| `server/internal/api/api_test.go` | Хелпер `authed(t, st, userID)` (создаёт сессию, возвращает `*http.Cookie`); фейковый `Mailer`; существующие тесты — под куку вместо `X-User`. |
| `server/main.go` | Сборка `Config` / `Mailer` / `Limiter` / `FailCounter`; фоновый воркер писем; housekeeping-свип `DeleteExpiredSessions`; передача в `api.Handler`. |
| `server/internal/store/store.go` | (мелочь, Task 9) `IdentityForUser(userID, provider) (Identity, error)`; убрать мостовые `UserByName`/`EnsureUserByName`? — **нет**, их снимает часть 3 вместе с фронтом; здесь `requireAuth` их ещё использует для моста. |
| `server/internal/store/store_test.go` | Гвард `_test`-суффикса на PG-`TRUNCATE` в `newStore` (хвост части 1). |

### Снимаются (в этой части)

- `GET /api/users`, `POST /api/users` (роуты + хендлеры `listUsers`/`createUser`).
- **НЕ** снимаются: `store.UserByName`/`EnsureUserByName`, `handlers.user()` X-User-мост — их убирает часть 3 (последним шагом, вместе с фронтом на сессии).

---

# Фаза A — листовые пакеты

## Task 1: `auth/password.go` + зависимости

**Files:**
- Modify: `go.mod`, `go.sum`
- Create: `server/internal/auth/password.go`, `server/internal/auth/password_test.go`

**Interfaces — Produces:**
- `auth.HashPassword(plain string) (string, error)` — `bcrypt.GenerateFromPassword(pre(plain), bcryptCost)`, где `pre(p) = []byte(base64.StdEncoding.EncodeToString(sha256.Sum256([]byte(p))[:]))`. Возвращает PHC-строку bcrypt.
- `auth.VerifyPassword(hash, plain string) bool` — `bcrypt.CompareHashAndPassword([]byte(hash), pre(plain)) == nil`.
- `const bcryptCost = 12` (не экспортируется).

- [ ] **Step 1: зависимость `x/crypto` (пиновая версия!)**
```bash
cd /Users/grisha/plans/serbian-app
go get golang.org/x/crypto@v0.41.0
go mod tidy
head -3 go.mod    # MUST still say: go 1.25.0  (no toolchain line)
```
`@v0.41.0` — потому что `@latest` (v0.57) тянет `go 1.26` в `go.mod`, а Dockerfile на `golang:1.25-alpine`. Если `go mod tidy` всё равно поднял `go`-строку или добавил `toolchain` — откатить: `go mod edit -go=1.25.0 -toolchain=none && go mod tidy`.
`x/time/rate` здесь НЕ ставим (Task 7 поставит, тоже пиново).

- [ ] **Step 2: падающий тест** `password_test.go`:
```go
package auth

import "testing"

func TestHashVerifyRoundTrip(t *testing.T) {
	h, err := HashPassword("correct horse battery staple")
	if err != nil { t.Fatal(err) }
	if !VerifyPassword(h, "correct horse battery staple") { t.Fatal("valid password rejected") }
	if VerifyPassword(h, "wrong") { t.Fatal("wrong password accepted") }
}

func TestHashLongPasswordNotTruncated(t *testing.T) {
	// >72 bytes: bcrypt would truncate without the sha256 pre-hash.
	a := "A_" + string(make([]byte, 100)) + "_tail_1"
	b := "A_" + string(make([]byte, 100)) + "_tail_2"
	h, _ := HashPassword(a)
	if VerifyPassword(h, b) { t.Fatal("distinct >72-byte passwords collide") }
	if !VerifyPassword(h, a) { t.Fatal("exact >72-byte password rejected") }
}

func TestHashIsSalted(t *testing.T) {
	h1, _ := HashPassword("same")
	h2, _ := HashPassword("same")
	if h1 == h2 { t.Fatal("hash not salted") }
}
```

- [ ] **Step 3: прогнать — падает.** `go test ./server/internal/auth/ -run 'TestHash|TestVerify' -v` → FAIL (`undefined: HashPassword`).

- [ ] **Step 4: реализация** `password.go` — пакетный doc не трогать (часть 1 уже описала пакет в `id.go`); просто добавить функции + `pre` helper + `import ("crypto/sha256"; "encoding/base64"; "golang.org/x/crypto/bcrypt")`.

- [ ] **Step 5: прогнать — проходит.** `go test ./server/internal/auth/ -v` → PASS.

- [ ] **Step 6: коммит** — `feat(auth): password hashing (bcrypt over sha256)`.

---

## Task 2: `auth/token.go`

**Files:** Create `server/internal/auth/token.go`, `token_test.go`

**Interfaces — Produces:**
- `auth.NewToken() (raw string, hash string, err error)` — 32 байта `crypto/rand`; `raw = base64.RawURLEncoding.EncodeToString(b)`; `hash = HashToken(raw)`.
- `auth.HashToken(raw string) string` — `hex.EncodeToString(sha256.Sum256([]byte(raw))[:])`.

- [ ] **Step 1: тест** `token_test.go`:
```go
func TestNewTokenShape(t *testing.T) {
	raw, hash, err := NewToken()
	if err != nil { t.Fatal(err) }
	if len(raw) < 40 { t.Fatalf("raw too short: %q", raw) }               // 32 bytes -> 43 base64url chars
	if len(hash) != 64 { t.Fatalf("hash not hex-sha256: %q", hash) }
	if HashToken(raw) != hash { t.Fatal("HashToken(raw) != returned hash") }
	raw2, _, _ := NewToken()
	if raw2 == raw { t.Fatal("collision") }
}
func TestHashTokenStable(t *testing.T) {
	if HashToken("abc") != HashToken("abc") { t.Fatal("not deterministic") }
	if HashToken("abc") == HashToken("abd") { t.Fatal("no diffusion") }
}
```

- [ ] **Step 2: прогнать — падает.**
- [ ] **Step 3: реализация** `token.go` (`crypto/rand`, `crypto/sha256`, `encoding/base64`, `encoding/hex`).
- [ ] **Step 4: прогнать — проходит.**
- [ ] **Step 5: коммит** — `feat(auth): opaque token generation + hashing`.

---

## Task 3: `auth/telegram.go`

Валидация подписи по спеке Telegram. Две поверхности: Mini App `initData` (query-string) и Login Widget (набор полей).

**Files:** Create `server/internal/auth/telegram.go`, `telegram_test.go`

**Interfaces — Produces:**
```go
type TelegramUser struct {
	ID        int64  // tg user id
	Username  string
	FirstName string
	AuthDate  time.Time
}

// VerifyInitData parses & verifies a Mini App initData query string.
// secret = HMAC_SHA256(key=[]byte("WebAppData"), msg=botToken)
// checkString = "\n"-joined sorted "k=v" of all fields except hash
// valid iff hex(HMAC_SHA256(secret, checkString)) == hash AND now-auth_date < maxAge
func VerifyInitData(initData, botToken string, now time.Time, maxAge time.Duration) (TelegramUser, error)

// VerifyWidget verifies Login Widget params (map[string]string).
// secret = SHA256(botToken); checkString same shape; same freshness rule.
func VerifyWidget(params map[string]string, botToken string, now time.Time, maxAge time.Duration) (TelegramUser, error)
```
- Ошибки: `ErrBadHash`, `ErrStale`, `ErrMalformed` (экспортируемые sentinel).
- `user` поле в `initData` — JSON, парсить в `TelegramUser` (`id`, `username`, `first_name`).

- [ ] **Step 1: тест** `telegram_test.go` — построить валидные векторы прямо в тесте (тестовый `botToken = "test:ABC"`), затем:
  - `TestVerifyInitDataValid` — собрать `initData` из `{auth_date, query_id, user:{"id":42,"username":"grisha"}}`, посчитать правильный `hash`, `VerifyInitData` → `TelegramUser{ID:42, Username:"grisha"}`.
  - `TestVerifyInitDataBadHash` — испортить один байт `hash` → `ErrBadHash`.
  - `TestVerifyInitDataStale` — `auth_date` = now-25h, maxAge 24h → `ErrStale`.
  - `TestVerifyInitDataMalformed` — нет поля `hash` / битый `user` JSON → `ErrMalformed`.
  - `TestVerifyWidgetValid` / `TestVerifyWidgetBadHash` — то же для map-варианта (secret = `sha256(botToken)`).
  - Хелпер в тесте: `signInitData(fields map[string]string, botToken string) string` и `signWidget(...)`.

- [ ] **Step 2: прогнать — падает.**
- [ ] **Step 3: реализация** `telegram.go` — `crypto/hmac`, `crypto/sha256`, `encoding/hex`, `encoding/json`, `net/url`, `sort`, `strconv`, `strings`, `time`. Общий `checkHash(fields map[string]string, secret []byte) (string, bool)` для обоих путей.
- [ ] **Step 4: прогнать — проходит.**
- [ ] **Step 5: коммит** — `feat(auth): Telegram initData + Login Widget signature validation`.

---

## Task 4: `mail/mail.go` — интерфейс + `LogMailer`

**Files:** Create `server/internal/mail/mail.go`, `mail_test.go`

**Interfaces — Produces:**
```go
package mail

type Mailer interface {
	// Send delivers one message. Implementations must be safe for concurrent use.
	Send(ctx context.Context, to, subject, text, html string) error
}

// LogMailer writes a one-line summary + the message body to a logger.
// Used in dev/tests and the first rollout step (before SMTP creds exist).
type LogMailer struct{ Logf func(format string, args ...any) } // nil Logf -> log.Printf

func (m LogMailer) Send(ctx context.Context, to, subject, text, html string) error
```
- `LogMailer.Send` печатает `to`, `subject` и `text` (не `html`); НЕ логирует, если `ctx` уже отменён.

- [ ] **Step 1: тест** `mail_test.go`:
  - `TestLogMailerCapturesToLogf` — `LogMailer{Logf: capture}` → `Send` → перехваченная строка содержит `to`, `subject`, фрагмент `text`.
  - `TestLogMailerNilLogf` — `LogMailer{}.Send(...)` не паникует, возвращает `nil`.
- [ ] **Step 2: прогнать — падает.**
- [ ] **Step 3: реализация** `mail.go`.
- [ ] **Step 4: прогнать — проходит.**
- [ ] **Step 5: коммит** — `feat(mail): Mailer interface + LogMailer`.

---

## Task 5: `mail/templates.go` — 4 письма

**Files:** Create `server/internal/mail/templates.go`, `templates_test.go`

**Interfaces — Produces:**
```go
// Each returns (subject, text, html). baseURL has no trailing slash.
func RenderVerify(baseURL, token, name string) (subject, text, html string)
func RenderReset(baseURL, token, name string) (subject, text, html string)
func RenderAlreadyRegistered(baseURL, name string) (subject, text, html string)     // link -> /login and /forgot
func RenderPasswordChanged(baseURL, name string) (subject, text, html string)
```
- Ссылки: verify → `baseURL + "/api/auth/verify?token=" + url.QueryEscape(token)`; reset → `baseURL + "/reset?token=" + url.QueryEscape(token)` (фронт-роут, часть 3).
- Тексты по-русски, дружелюбные, короткие. `already_registered` НЕ раскрывает, что аккаунт есть, прямо — формулировка «если это был ты…». Формулировки из спеки §2.4.
- Рендер через `text/template` + `html/template`, шаблоны — пакетные строковые константы. HTML — минимальный инлайн-стиль, одна кнопка-ссылка.

- [ ] **Step 1: тест** `templates_test.go` (table-driven по 4 функциям):
  - нет неотрендеренных `{{` / `<no value>` в `subject`/`text`/`html`;
  - `text` и `html` для verify/reset содержат полный URL с токеном;
  - `html` парсится `html/template.Must`-эквивалентом без ошибок (или просто проверить `strings.Contains(html, "<a ")`);
  - `subject` непустой, без переводов строки.
- [ ] **Step 2: прогнать — падает.**
- [ ] **Step 3: реализация.**
- [ ] **Step 4: прогнать — проходит.**
- [ ] **Step 5: коммит** — `feat(mail): Russian templates for verify/reset/already-registered/password-changed`.

---

## Task 6: `mail/smtp.go` — `SMTPMailer`

**Files:** Create `server/internal/mail/smtp.go`, `smtp_test.go`

**Interfaces — Produces:**
```go
type SMTPConfig struct{ Host, Port, User, Pass, From, FromName string }

func NewSMTPMailer(c SMTPConfig) *SMTPMailer   // returns nil if c.Host == ""
func (m *SMTPMailer) Send(ctx context.Context, to, subject, text, html string) error
```
- Реализация: `net/smtp` — `smtp.Dial(host:port)`, `StartTLS(&tls.Config{ServerName: host})` если сервер объявляет STARTTLS, `Auth` через `smtp.PlainAuth("", user, pass, host)`, `Mail`/`Rcpt`/`Data`.
- Сообщение — `multipart/alternative` (text + html), заголовки `From: FromName <From>`, `To`, `Subject` (RFC 2047 encode для кириллицы — `mime.QEncoding.Encode("utf-8", subject)`), `Date`, `MIME-Version`, `Message-ID`.
- **Вынести сборку сообщения в `buildMessage(from, fromName, to, subject, text, html string) []byte`** — её и тестировать (SMTP-диалог юнит-тестом не гоняем).
- `ctx` — уважать отмену/дедлайн: `net.Dialer{}.DialContext`.

- [ ] **Step 1: тест** `smtp_test.go`:
  - `TestBuildMessageStructure` — `buildMessage(...)` содержит `From: `, `To: `, `Subject: ` (Q-encoded если кириллица), `Content-Type: multipart/alternative; boundary=`, обе части (`text/plain`, `text/html`), CRLF-переводы строк.
  - `TestNewSMTPMailerNilOnEmptyHost` — `NewSMTPMailer(SMTPConfig{}) == nil`.
  - (диалог с сервером — не тестируем; отметить в отчёте.)
- [ ] **Step 2: прогнать — падает.**
- [ ] **Step 3: реализация.**
- [ ] **Step 4: прогнать — проходит.** `go vet` чистый.
- [ ] **Step 5: коммит** — `feat(mail): SMTPMailer (net/smtp, STARTTLS, MIME multipart)`.

---

## Task 7: `ratelimit/ratelimit.go`

**Files:** Create `server/internal/ratelimit/ratelimit.go`, `ratelimit_test.go`

**Interfaces — Produces:**
```go
package ratelimit

// Limiter is a keyed token-bucket set with lazy eviction of idle keys.
type Limiter struct{ /* mu, map[string]*entry, rate, burst, now func */ }
func NewLimiter(perSecond float64, burst int) *Limiter
func (l *Limiter) Allow(key string) bool           // false when the bucket is empty
func (l *Limiter) SetNow(fn func() time.Time)       // test seam

// FailCounter implements the soft account lock (spec §2.4):
// >= threshold failures within window -> Locked returns true until the window passes;
// a success (Reset) clears the count.
type FailCounter struct{ /* mu, map[string]*rec, threshold, window, now */ }
func NewFailCounter(threshold int, window time.Duration) *FailCounter
func (f *FailCounter) Locked(key string) bool
func (f *FailCounter) Fail(key string)
func (f *FailCounter) Reset(key string)
func (f *FailCounter) SetNow(fn func() time.Time)
```
- **Зависимость:** `go get golang.org/x/time@<пиновая>` — проверить `go list -m -f '{{.GoVersion}}' golang.org/x/time@<ver>` ≤ 1.25 (напр. `@v0.9.0`), НЕ `@latest`. После — `head -3 go.mod` = `go 1.25.0`, без `toolchain`.
- `Limiter` внутри — `golang.org/x/time/rate.Limiter` на ключ (или свой bucket). Чистка: при каждом `Allow` с вероятностью/счётчиком, либо отдельный `func (l *Limiter) reap()` вызываемый по таймеру из `main` — проще: ленивая чистка записей, не тронутых > 10 мин, во время `Allow` (проход по N записям).
- Всё потокобезопасно (`sync.Mutex`).

- [ ] **Step 1: тест** `ratelimit_test.go` (детерминированные часы через `SetNow`):
  - `TestLimiterAllowsBurstThenBlocks` — `NewLimiter(1.0/60, 5)` (5/мин): 5 подряд `Allow("ip")` = true, 6-й = false; ключ `"ip2"` не затронут.
  - `TestLimiterRefills` — сдвинуть часы на 60 с → снова `Allow` = true.
  - `TestLimiterEvictsIdle` — после «протухания» запись удаляется (проверить размер мапы или что старый ключ снова с полным бакетом).
  - `TestFailCounterLocksAfterThreshold` — `NewFailCounter(3, 15*time.Minute)`: `Fail x3` → `Locked` = true; сдвиг часов на 16 мин → `Locked` = false.
  - `TestFailCounterResetClears` — `Fail x3`, `Reset`, `Locked` = false.
- [ ] **Step 2: прогнать — падает.**
- [ ] **Step 3: реализация.**
- [ ] **Step 4: прогнать — проходит.** `go test -race ./server/internal/ratelimit/` (есть конкурентный доступ) → PASS.
- [ ] **Step 5: коммит** — `feat(ratelimit): keyed token-bucket + soft-lock fail counter`.

---

## Task 8: `config/config.go`

**Files:** Create `server/internal/config/config.go`, `config_test.go`

**Interfaces — Produces:**
```go
type Config struct {
	AppBaseURL       string   // APP_BASE_URL, default "http://localhost:8080", trailing slash trimmed
	SMTP             mail.SMTPConfig // from SMTP_HOST/PORT/USER/PASS/FROM/FROM_NAME
	TelegramBotToken string   // TELEGRAM_BOT_TOKEN, "" -> telegram disabled
}
func Load() Config                       // reads os.Getenv
func (c Config) Secure() bool            // strings.HasPrefix(c.AppBaseURL, "https://")
func (c Config) CookieName() string      // "__Host-session" if Secure() else "session"
func (c Config) TelegramEnabled() bool   // TelegramBotToken != ""
func (c Config) SMTPEnabled() bool       // SMTP.Host != ""
```
- `config` импортирует `mail` (для `SMTPConfig`) — ок, `mail` не импортирует `config`.

- [ ] **Step 1: тест** `config_test.go` (через `t.Setenv`):
  - дефолт `AppBaseURL` при пустом env; трим хвостового `/`.
  - `Secure()` / `CookieName()` для `http://` и `https://` баз.
  - `SMTPEnabled()` false при пустом `SMTP_HOST`, true при заданном; поля прокидываются.
  - `TelegramEnabled()`.
- [ ] **Step 2: прогнать — падает.**
- [ ] **Step 3: реализация.**
- [ ] **Step 4: прогнать — проходит.**
- [ ] **Step 5: коммит** — `feat(config): env-driven Config (base URL, SMTP, Telegram)`.

### Проверка фазы A

- [ ] `go test ./server/internal/auth/ ./server/internal/mail/ ./server/internal/ratelimit/ ./server/internal/config/ -v` — зелёное.
- [ ] `go build ./... && go vet ./...` — чисто. `internal/api` ещё не трогали, компилируется как есть.

---

# Фаза B — HTTP

## Task 9: `store` — мелкие дополнения

**Files:**
- Modify: `server/internal/store/identities.go`, `server/internal/store/store_test.go`
- Test: `identities_test.go` (+кейсы)

**Interfaces — Produces:**
- `Store.IdentityForUser(userID, provider string) (Identity, error)` — одна identity юзера по провайдеру; `sql.ErrNoRows` если нет. (нужно `/me`, смене пароля, `/me/link/telegram`.)
- `Store.IdentityByID(id string) (Identity, error)` — по `identities.id`; `sql.ErrNoRows` если нет. (нужно `reset` — из `identity_id` токена достать `user_id`.)
- `Store.DeleteUserIdentity(userID, provider string) error` — `DELETE FROM identities WHERE user_id = ? AND provider = ?`. (нужно `DELETE /me/telegram`.)

- [ ] **Step 1: тесты** в `identities_test.go`:
  - `TestIdentityForUser` — у юзера `password` + `telegram` → `IdentityForUser(id,"telegram")` возвращает telegram-строку; `IdentityForUser(id,"password")` — password-строку; несуществующий провайдер → `sql.ErrNoRows`.
  - `TestIdentityByID` — round-trip по id; неизвестный id → `sql.ErrNoRows`.
  - `TestDeleteUserIdentity` — удаляет только строку нужного провайдера этого юзера, чужие/другого провайдера не трогает; `email_tokens` этой identity уходят каскадом (FK `ON DELETE CASCADE` из части 1) — проверить.
- [ ] **Step 2: падает.**
- [ ] **Step 3: реализация** — `SELECT <identityCols> FROM identities WHERE ...` через `scanIdentity` (из части 1). `IdentityForUser`: `WHERE user_id = ? AND provider = ?` (одна на провайдера по инварианту части 1). `IdentityByID`: `WHERE id = ?`. `DeleteUserIdentity`: `Exec` + обёрнутая ошибка.
- [ ] **Step 4: гвард `_test` в `newStore`** (хвост части 1): в `server/internal/store/store_test.go` `newStore`, перед PG-`TRUNCATE`, — `if !strings.HasSuffix(dbNameFromDSN(dsn), "_test") { t.Fatalf("refusing to TRUNCATE non-test db %q", dsn) }` (переиспользовать/вынести helper `dbNameFromDSN` из `migrate_test_helper_test.go`, если он там приватный к файлу — сделать пакетным в `store_test.go` или отдельном `testutil_test.go`).
- [ ] **Step 5: прогнать** `go test ./server/internal/store/ -v` (+ `make test-pg` если есть) → PASS.
- [ ] **Step 6: коммит** — `feat(store): IdentityForUser; guard test-only DB in newStore`.

---

## Task 10: `api/middleware.go`

**Files:** Create `server/internal/api/middleware.go`, `middleware_test.go`

**Interfaces:**
- Consumes: `config.Config` (Task 8), `store.SessionByHash`/`TouchSession`/`UserByID`/`UserByName`/`User` (part 1), `auth.HashToken` (Task 2).
- Produces (все в `package api`):
```go
// mw chain, applied by Handler (Task 11)
func securityHeaders(cfg config.Config, next http.Handler) http.Handler
func checkOrigin(cfg config.Config, next http.Handler) http.Handler   // 403 on non-GET/HEAD/OPTIONS with mismatched Origin/Referer
func (h handlers) requireAuth(next http.Handler) http.Handler          // 401 unless a valid session cookie OR (bridge) X-User resolves

// context
type authCtx struct { UserID string; SessionHash string; Summary userSummary }
func authFrom(r *http.Request) (authCtx, bool)
type userSummary struct { ID, Name, Email string; EmailVerified bool; TelegramLinked bool; TelegramUsername string }

// cookie helpers
func (h handlers) setSessionCookie(w http.ResponseWriter, rawToken string, maxAge time.Duration)
func (h handlers) clearSessionCookie(w http.ResponseWriter)
```

**`requireAuth` логика:**
1. Кука `cfg.CookieName()` есть → `hash := auth.HashToken(raw)` → `sess, err := Store.SessionByHash(hash, now)` → нет/протухла → падаем в мост (не сразу 401 — вдруг кука мусорная, но `X-User` валиден... на практике редко, но не мешает). Есть → `Store.TouchSession(hash, now)` (троттлинг внутри стора — из части 1 `TouchSession` уже слайдит; троттлинг `last_seen_at` 1 ч добавить в `TouchSession`? — **см. примечание**), грузим `userSummary` (`UserByID` + `IdentityForUser`), кладём `authCtx` в контекст, `next`.
2. Мост (кука не сработала): `X-User` header (percent-decoded) → `store.NormalizeName` → `Store.UserByName` → есть → `authCtx{UserID: row.ID, SessionHash: ""}` (без summary email — мост только для старого фронта, `/me` он не зовёт), `next`.
3. Ни то ни другое → `401 {"error":"no session"}`.

> **Примечание про троттлинг `last_seen_at`.** Часть 1 `TouchSession` слайдит `expires_at` и пишет `last_seen_at` каждый раз. Спека просит «не чаще раза в час». **Решение:** в `requireAuth` звать `TouchSession` только если `now.Sub(sess.LastSeenAt) > lastSeenThrottle` (1 ч). `SessionByHash` возвращает `LastSeenAt` (часть 1, `Session` struct). Задокументировать в коде.

**`checkOrigin`:** для методов кроме GET/HEAD/OPTIONS — `Origin` (или `Referer` если пусто) → распарсить → `scheme://host` должно совпасть с `cfg.AppBaseURL`'s `scheme://host`. Иначе `403 {"error":"bad origin"}`. Пустые оба на не-GET → `403` (нормальный браузер всегда шлёт `Origin` на не-GET same-origin fetch).

**`securityHeaders`:** заголовки из спеки §2.7. HSTS только если `cfg.Secure()`. CSP: `default-src 'self'; frame-ancestors 'self' https://web.telegram.org https://*.telegram.org; ...` — минимальный рабочий для Vite-сборки (разрешить `style-src 'self' 'unsafe-inline'` для Tailwind-инлайна, `font-src 'self' data:`, `img-src 'self' data:`, `connect-src 'self'`). Точный CSP — доработать в отчёте по факту.

- [ ] **Step 1: тесты** `middleware_test.go` (голый `handlers{}` + fake store через реальный `store.Open(":memory:")`):
  - `TestRequireAuthNoCredentials401` — GET `/api/progress` без куки/хедера → 401.
  - `TestRequireAuthValidCookie` — создать юзера + сессию в сторе, поставить куку → проходит, `authFrom` даёт `UserID`.
  - `TestRequireAuthExpiredCookie401` — сессия с `expires_at` в прошлом → 401.
  - `TestRequireAuthBridgeXUser` — нет куки, `X-User: Гриша` (существует) → проходит (мост).
  - `TestCheckOriginRejectsCrossOrigin` — POST с `Origin: https://evil.com` → 403; с правильным `Origin` → проходит; GET без `Origin` → проходит.
  - `TestSecurityHeadersPresent` — ответ несёт `X-Content-Type-Options`, `Referrer-Policy`, `Content-Security-Policy` с `frame-ancestors ... web.telegram.org`; HSTS только при `https` base.
  - `TestSetSessionCookieAttributes` — `__Host-session` + `Secure` при `https` base; `session` без `Secure` при `http`; всегда `HttpOnly`, `SameSite=Lax`, `Path=/`.
- [ ] **Step 2: падает.**
- [ ] **Step 3: реализация.**
- [ ] **Step 4: проходит.** `go test ./server/internal/api/ -run 'Middleware|RequireAuth|CheckOrigin|SecurityHeaders|SessionCookie' -v`.
- [ ] **Step 5: коммит** — `feat(api): security/origin/auth middleware + session cookie helpers`.

---

## Task 11: `api.go` / `dto.go` / `api_test.go` — интеграция middleware

**Files:** Modify `server/internal/api/api.go`, `dto.go`, `api_test.go`

**Interfaces:**
- `Deps` расширяется:
```go
type Deps struct {
	Course     func() *content.Course
	Store      *store.Store
	Now        func() time.Time
	Stale      func() bool
	Config     config.Config
	SendMail   func(to, subject, text, html string)  // delivers a message (prod: enqueue; tests: capture)
	Async      func(func())                          // runs f; prod: `go f()`, tests: `f()` (deterministic)
	Login      *ratelimit.Limiter                    // 5/min IP
	LoginEmail *ratelimit.Limiter                    // 10/hour email
	Slow       *ratelimit.Limiter                    // register/resend/forgot: 3/hour (keyed "ip:"+ip / "email:"+addr)
	Fails      *ratelimit.FailCounter                // soft lock, keyed email
}
```
  Дефолты в `Handler`: `SendMail == nil` → no-op; `Async == nil` → `func(f func()){ go f() }`; лимитеры `nil` → пропускать (через хелперы `allow`/`locked` из Global Constraints).
  Хендлеры зовут фон через `h.Async(func(){ ... })` и письма через `h.SendMail(...)`.
- `handlers.user(w, r)` — **переписать**: читать `authFrom(r)`; нет → `fail(401)`; есть → `Store.User(ctx.UserID)`. (Заменяет разбор `X-User` — теперь это делает `requireAuth`.)

**`Handler` (переписать сборку):**
```go
func Handler(deps Deps) http.Handler {
	// ...defaults...
	h := handlers{deps}
	api := http.NewServeMux()          // routes under /api
	// public:
	api.HandleFunc("GET /api/health", h.health)
	// auth (public, no requireAuth):
	api.HandleFunc("POST /api/auth/register", h.register)
	api.HandleFunc("POST /api/auth/login", h.login)
	api.HandleFunc("POST /api/auth/logout", h.logout)
	api.HandleFunc("GET  /api/auth/verify", h.verifyEmail)
	api.HandleFunc("POST /api/auth/resend-verification", h.resendVerification)
	api.HandleFunc("POST /api/auth/forgot", h.forgotPassword)
	api.HandleFunc("POST /api/auth/reset", h.resetPassword)
	api.HandleFunc("GET  /api/auth/session", h.currentSession)   // 401 if none — but NOT behind requireAuth (returns {user:null} vs 401? spec: 401)
	api.HandleFunc("POST /api/auth/telegram", h.telegramLogin)
	// protected:
	protected := http.NewServeMux()
	protected.HandleFunc("POST /api/auth/logout-all", h.logoutAll)
	protected.HandleFunc("GET /api/me", h.getMe)
	// ...all /api/me* ...
	// ...ALL existing content/lesson/review/progress/leaderboard routes moved here verbatim...
	api.Handle("/api/", h.requireAuth(protected))     // catch-all -> requireAuth
	// chain: securityHeaders -> checkOrigin -> api
	return securityHeaders(deps.Config, checkOrigin(deps.Config, api))
}
```
  Порядок регистрации: точные auth-пути на `api`; всё остальное `/api/` → `protected` за `requireAuth`. Убрать `GET/POST /api/users`.
  `GET /api/auth/session` — **не** за `requireAuth`; хендлер сам зовёт `authFrom`, отдаёт `200 {user}` или `401`.

**`dto.go` — добавить:**
```go
type sessionUserDTO struct {
	ID string `json:"id"`; Name string `json:"name"`; Email string `json:"email"`
	EmailVerified bool `json:"email_verified"`
	Telegram struct{ Linked bool `json:"linked"`; Username string `json:"username"` } `json:"telegram"`
}
type deviceDTO struct {
	ID string `json:"id"`            // first 12 chars of token_hash — opaque handle for DELETE /me/sessions/{id}
	UserAgent string `json:"user_agent"`; LastSeenAt string `json:"last_seen_at"`; Current bool `json:"current"`
}
type meDTO struct { sessionUserDTO; Sessions []deviceDTO `json:"sessions"` }
```

**`api_test.go` — минимальные правки** (мост `X-User` держит старые тесты зелёными):
- `newTestAPI` заполняет новые `Deps`: `Config: config.Config{AppBaseURL:"http://localhost:8080"}`, `SendMail`: синхронный сборщик в срез `*[]sentMail`, лимитеры `nil` (пропускать).
- Добавить хелпер `authed(t, st, userID) *http.Cookie` — `raw, hash, _ := auth.NewToken(); st.CreateSession(hash, userID, "test-agent", now, now.Add(720h)); return &http.Cookie{Name:"session", Value:raw}`. И `doCookie(h, cookie, method, path, body)`.
- Удалить `TestUsersEndpoints` (эндпойнты сняты). Тесты, что слали `X-User` — оставить как есть (мост). `TestStateEndpointsRequireAccount` — теперь ожидает 401 без куки И без `X-User` (обновить).
- `checkOrigin`: тестовые запросы `httptest.NewRequest` не ставят `Origin` — для не-GET это станет 403! **Правка хелперов:** `doAs`/`doCookie` для не-GET ставят `r.Header.Set("Origin", "http://localhost:8080")`.

- [ ] **Step 1: обновить `api_test.go` хелперы + `newTestAPI` + удалить users-тесты (падающий шаг — компиляция).**
- [ ] **Step 2: `go test ./server/internal/api/` — падает** (компиляция: новые поля Deps, отсутствующие хендлеры `h.register` и т.д.). Ожидаемо — временно закомментировать роуты несуществующих хендлеров ИЛИ сделать хендлеры-заглушки `func (h handlers) register(w,r){ fail(w,501,"todo") }` в `auth.go`/`me.go`, чтобы компилировалось. **Решение:** создать `auth.go` и `me.go` с заглушками всех хендлеров (`501`), затем Tasks 12–16 наполняют.
- [ ] **Step 3: реализация** — `Handler` переписан, `handlers.user()` через контекст, `dto.go` дополнен, заглушки хендлеров.
- [ ] **Step 4: прогнать** — `go test ./server/internal/api/ -v`: старые тесты (через мост / куку) зелёные; заглушечные auth-эндпойнты возвращают 501 (пока без тестов на них). `go build ./... && go vet ./...` чисто.
- [ ] **Step 5: коммит** — `feat(api): wire middleware chain, /api/auth + /api/me routes (stubs), drop /api/users`.

---

## Task 12: `auth.go` — register / login / logout / logout-all / session

**Files:** Modify `server/internal/api/auth.go` (наполнить заглушки), `server/internal/api/auth_test.go` (создать)

Общий хелпер в `auth.go`: `func (h handlers) issueSession(w http.ResponseWriter, r *http.Request, userID string) error` — `raw, hash, _ := auth.NewToken()`; `Store.CreateSession(hash, userID, r.UserAgent(), now, now.Add(sessionTTL))`; `h.setSessionCookie(w, raw, sessionTTL)`.

**`register`** `{email, password, name}`:
- **Синхронно:** валидация формата — email через `net/mail.ParseAddress`, пароль 8–128, name после `NormalizeName` непустой ≤40. Не прошло → `400 {"error":"..."}` (формат, не энумерация). Прошло → `allow(h.Slow, "ip:"+ip)` && `allow(h.Slow, "email:"+lower(email))` (результат запомнить) → **сразу `200 {"status":"ok"}`**.
- **Всё остальное — в фоновой горутине после ответа** (поэтому ветвление «есть/нет email» не даёт тайминг-оракула; ответ уже ушёл). Если rate-limit не пропустил — горутина просто ничего не делает. Иначе `IdentityByProviderUID("password", lower(email))`:
  - есть → `h.SendMail(RenderAlreadyRegistered(...))`;
  - нет → `hash := auth.HashPassword(password)` → `CreateUser(name)` → `CreateIdentity(Identity{ID: auth.NewIdentityID(), UserID, Provider:"password", ProviderUID: lower(email), Email: email, PasswordHash: hash})` → `raw, thash, _ := auth.NewToken()` → `CreateEmailToken(thash, identityID, "verify", now, now.Add(verifyTTL))` → `h.SendMail(RenderVerify(baseURL, raw, name))`.
- Гонка двойного сабмита: второй `CreateIdentity` упрётся в `UNIQUE(provider, provider_uid)` (лог, не паника); `CreateUser` мог оставить сиротский `users`-ряд — редко, `Slow` 3/час это гасит. Задокументировать, не чинить.
- В тестах `Deps.Async` синхронный → фейковый `SendMail` ловит письмо сразу после ответа, без `time.Sleep`.

**`login`** `{email, password}`:
- `locked(h.Fails, lower(email))` → `401` generic (лок отдельно не раскрываем), пароль не проверяем.
- `allow(h.Login, "ip:"+ip)` + `allow(h.LoginEmail, lower(email))` → нет → `429 {"error":"too many attempts"}`.
- `IdentityByProviderUID("password", lower(email))` → нет → `h.Fails.Fail(...)` (если `h.Fails != nil`) + `401` generic. Есть → `auth.VerifyPassword(id.PasswordHash, password)`:
  - неверно → `h.Fails.Fail(...)` + `401` generic;
  - верно + `id.EmailVerifiedAt == ""` → `h.Fails.Reset(...)` + фоново переслать verify-письмо (новый токен) + `403 {"error":"email_unverified"}`;
  - верно + подтверждён → `h.Fails.Reset(...)` + `issueSession` + `200 {sessionUserDTO}`.

  (`h.Fails.Fail/Reset` заворачивать в `if h.Fails != nil` или добавить хелперы `fail(f, key)` / `reset(f, key)` рядом с `allow`/`locked`.)

**`logout`:** `authFrom` → есть `SessionHash` → `Store.DeleteSession(hash)`; `clearSessionCookie`; `204`. (не требует `requireAuth` — сам мягко.)

**`logoutAll`:** (за `requireAuth`) `Store.DeleteUserSessions(ctx.UserID)` затем заново `issueSession` для текущего устройства (или просто удалить все и `clearSessionCookie` + `204` — проще; спека допускает). **Решение:** удалить все, `clearSessionCookie`, `204` — пусть перелогинится.

**`currentSession`:** `authFrom` → нет → `401`; есть → собрать `sessionUserDTO` (из `ctx.Summary` или дозагрузить) → `200`.

- [ ] **Step 1: тесты** `auth_test.go` (fake mailer sink, fake clock, лимитеры реальные с тестовыми настройками через новый `Deps`-хелпер или `nil`):
  - `TestRegisterCreatesUnverifiedAndSendsVerify` — `POST /register` → `200`; sink: 1 письмо, тема verify; в сторе — юзер + `password` identity с `email_verified_at` пустым.
  - `TestRegisterExistingEmailGenericAndAlreadyMail` — второй `register` тем же email → `200` generic; sink: письмо `already_registered`; второго юзера нет.
  - `TestRegisterBadInput` — короткий пароль / кривой email → `400`.
  - `TestLoginUnverified403` — зарегистрировать, не подтверждать, `login` верным паролем → `403 email_unverified`; sink: повторное verify-письмо.
  - `TestLoginWrongPassword401Generic` — `401`, тело `неверная почта или пароль`.
  - `TestLoginSuccessSetsCookie` — подтвердить (через стор `SetEmailVerified`), `login` → `200`, `Set-Cookie` с сессией; `GET /api/auth/session` с этой кукой → `200 {user}`.
  - `TestLoginRateLimited` — форсить лимит → `429`.
  - `TestLoginSoftLock` — 10 неверных → 11-й `login` даже с верным паролем → `401` (лок); после сдвига часов на 16 мин → проходит.
  - `TestLogoutClearsSession` — `login`, `logout` → `204` + кука очищена + сессии в сторе нет.
  - `TestLogoutAllKillsEverySession`.
- [ ] **Step 2: падает.**
- [ ] **Step 3: реализация.**
- [ ] **Step 4: проходит.** `go test ./server/internal/api/ -v`.
- [ ] **Step 5: коммит** — `feat(api): register / login / logout / session endpoints`.

---

## Task 13: `auth.go` — verify / resend-verification

**`verifyEmail`** `GET /api/auth/verify?token=`:
- `thash := auth.HashToken(r.URL.Query().Get("token"))` → `Store.UseEmailToken(thash, "verify", now)` → `sql.ErrNoRows` → редирект `302` на `cfg.AppBaseURL + "/verify?err=1"`. Успех → вернулся `identityID` → `Store.SetEmailVerified(identityID, now)` → редирект `302` на `/verify?ok=1`.
- Никогда не 500 наружу с токеном в теле; токен не логируем.

**`resendVerification`** `POST {email}`:
- `Slow.Allow(...)` (ip+email) → нет → generic `200`.
- Фон: `IdentityByProviderUID("password", lower(email))` → есть и не подтверждён → новый `verify`-токен + письмо. Есть и подтверждён / нет → ничего.
- Всегда `200 {"status":"ok"}`.

- [ ] **Step 1: тесты** (в `auth_test.go`):
  - `TestVerifyEmailHappyRedirect` — зарегистрировать → достать сырой токен из sink (тест-mailer должен сохранять и распарсенный `?token=` из письма) → `GET /verify?token=` → `302` на `/verify?ok=1`; в сторе `email_verified_at` проставлен; повторный тот же токен → `302 /verify?err=1`.
  - `TestVerifyEmailBadToken` — мусорный token → `302 /verify?err=1`.
  - `TestResendVerificationGeneric` — `200` для существующего неподтверждённого (sink +1) и для несуществующего (sink +0), тело одинаковое.
- [ ] Steps 2–5 как обычно. Коммит — `feat(api): email verification + resend`.

---

## Task 14: `auth.go` — forgot / reset

**`forgotPassword`** `POST {email}`:
- `Slow.Allow(...)` → нет → generic `200`.
- Фон: `IdentityByProviderUID("password", lower(email))`:
  - есть → `reset`-токен (`now+1h`) + `SendMail(RenderReset(...))`;
  - нет → ничего (тайминг: уравнять — как в register, лишних тяжёлых операций тут нет, так что просто вернуть сразу).
- Всегда `200 {"status":"ok"}` сразу.

**`resetPassword`** `POST {token, password}`:
- Пароль 8–128 → нет → `400`.
- `Store.UseEmailToken(auth.HashToken(token), "reset", now)` → `ErrNoRows` → `400 {"error":"invalid_token"}`.
- Успех → `identityID` → `hash, _ := auth.HashPassword(password)` → `Store.SetPasswordHash(identityID, hash)` → `Store.DeleteIdentityTokens(identityID, "reset")` → нужен `userID` по `identityID` (добавить `Store.IdentityByID(id) (Identity, error)` — **мелкое дополнение стора, включить в Task 9** или отдельным шагом здесь) → `Store.DeleteUserSessions(userID)` → `issueSession(w, r, userID)` (автологин) → `SendMail(RenderPasswordChanged(...))` (фон) → `200 {sessionUserDTO}`.

> **Дополнение к Task 9:** также `Store.IdentityByID(id string) (Identity, error)`. Внести туда.

- [ ] **Step 1: тесты:**
  - `TestForgotGeneric` — `200` для существующего (sink: reset-письмо) и несуществующего (sink 0), одинаковое тело.
  - `TestResetChangesPasswordAndKillsSessions` — зарегистрировать+подтвердить, залогиниться (сессия A), `forgot` → взять reset-токен из sink → `reset` новым паролем → `200` + новая кука; старая сессия A мертва (`GET /session` кукой A → `401`); `login` старым паролем → `401`, новым → `200`; sink: password_changed письмо.
  - `TestResetBadToken` → `400 invalid_token`. `TestResetShortPassword` → `400`.
- [ ] Steps 2–5. Коммит — `feat(api): forgot-password + reset`.

---

## Task 15: `auth.go` — telegram

**`telegramLogin`** `POST {init_data}` (Mini App) **или** `{...widget fields...}` (Login Widget):
- `cfg.TelegramEnabled()` == false → `503 {"error":"telegram_disabled"}`.
- Определить формат: есть строковое поле `init_data` → `auth.VerifyInitData(initData, cfg.TelegramBotToken, now, 24h)`; иначе трактовать тело как map полей виджета → `auth.VerifyWidget(fields, token, now, 24h)`.
  - `ErrBadHash` / `ErrStale` / `ErrMalformed` → `401 {"error":"bad_telegram_auth"}`.
- Получили `TelegramUser{ID, Username}`. `tgID := strconv.FormatInt(u.ID, 10)`.
- `Store.IdentityByProviderUID("telegram", tgID)`:
  - есть → `issueSession(w, r, identity.UserID)` → `200 {sessionUserDTO}`.
  - нет → `Store.AttachPendingTelegram(u.Username, tgID)` (часть 1) → вернулось `true` → снова `IdentityByProviderUID("telegram", tgID)` (теперь есть) → `issueSession` → `200`.
  - `AttachPendingTelegram` → `false` (нет pending по этому username) → **создать нового юзера**: `Store.CreateUser(displayName)` (displayName = `u.Username` или `u.FirstName` или `"tg"+tgID`) → `Store.CreateIdentity(Identity{ID: auth.NewIdentityID(), UserID, Provider:"telegram", ProviderUID: tgID, TgUsername: u.Username})` → `issueSession` → `200`.

- [ ] **Step 1: тесты:**
  - `TestTelegramDisabled503` — `Config` без токена → `503`.
  - `TestTelegramLoginNewUser` — `Config` с тестовым токеном, валидный `init_data` (подписать хелпером из Task 3) для `tg_id=555, username=neo` → `200` + кука; в сторе новый юзер + `telegram` identity `provider_uid="555"`.
  - `TestTelegramLoginExisting` — повторный тот же `init_data` → тот же `user_id`, не плодит юзера.
  - `TestTelegramClaimsPending` — предварительно `CreateUser("Гриша")` + `CreateIdentity(telegram, "pending:llladnooo")`; `init_data` c `username=llladnooo, tg_id=999` → `200`, залогинен как «Гриша», identity теперь `provider_uid="999"`.
  - `TestTelegramBadHash401`.
- [ ] Steps 2–5. Коммит — `feat(api): Telegram login (Mini App + widget), pending-account claim`.

---

## Task 16: `api/me.go`

**Files:** Modify `server/internal/api/me.go` (наполнить заглушки), create `me_test.go`

Все за `requireAuth` → `ctx, _ := authFrom(r)`; `us := Store.User(ctx.UserID)`.

**`GET /me`** → `meDTO`: `UserByID` (name), `IdentityForUser(uid,"password")` (email, email_verified — `EmailVerifiedAt != ""`), `IdentityForUser(uid,"telegram")` (linked, username), `ListUserSessions(uid)` → `[]deviceDTO` (`ID` = `token_hash[:12]`, `Current` = `token_hash == ctx.SessionHash`).

**`PATCH /me`** `{name}` → `NormalizeName`, ≤40, непустой → `Store.RenameUser(uid, name)` → `200 {sessionUserDTO}`.

**`POST /me/password`** `{current, new}`:
- `new` 8–128 → нет → `400`.
- `IdentityForUser(uid,"password")`:
  - есть → `current` обязателен, `auth.VerifyPassword(id.PasswordHash, current)` неверно → `403 {"error":"wrong_password"}`; верно → `SetPasswordHash(id.ID, HashPassword(new))` → `DeleteUserSessionsExcept(uid, ctx.SessionHash)` → `SendMail(password_changed)` → `200`.
  - нет (TG-only) → `current` не нужен; но нужен email → тело должно нести `{email, new}` (расширить: `{current?, new, email?}`). Нет email → `400 {"error":"email_required"}`. Есть → `CreateIdentity(password, lower(email), hash, email_verified пусто)` → `verify`-токен + письмо → `DeleteUserSessionsExcept` → `200 {"status":"verify_sent"}`.

**`POST /me/link/telegram`** `{init_data}`/widget:
- `cfg.TelegramEnabled()` == false → `503`.
- Верифицировать подпись (как Task 15). `tgID`.
- `IdentityByProviderUID("telegram", tgID)`:
  - есть и `UserID == uid` → `200` (уже привязан);
  - есть и `UserID != uid` → `409 {"error":"telegram_taken"}`;
  - нет → `CreateIdentity(telegram, tgID, tg_username)` для `uid` → `200 {sessionUserDTO}`.

**`DELETE /me/telegram`:**
- `IdentitiesForUser(uid)` → если `telegram` — единственная → `409 {"error":"only_login_method"}` (иначе юзер запрётся). Иначе `DELETE FROM identities WHERE user_id=? AND provider='telegram'` — нужен `Store.DeleteIdentity(id)` или `Store.DeleteUserIdentity(uid, provider)` — **мелкое дополнение стора (Task 9)**. → `204`.

**`DELETE /me/sessions/{id}`** — `id` = 12-символьный префикс; `ListUserSessions(uid)` → найти сессию с `token_hash[:12] == id` → `DeleteSession(full hash)` → `204`. Не нашли → `404`.

**`DELETE /me`** `{password?}`:
- `IdentityForUser(uid,"password")` есть → `password` обязателен и верный, иначе `403`.
- нет (TG-only) → разрешить если текущая сессия свежая: `now - session.created_at < 5min` (нужно `created_at` сессии — `SessionByHash` его отдаёт) иначе `403 {"error":"reauth_required"}`.
- ОК → `Store.DeleteUser(uid)` (часть 1: транзакция, всё состояние + identities/sessions/tokens) → `clearSessionCookie` → `204`.

> **Дополнения к Task 9 (итог):** `IdentityForUser`, `IdentityByID`, `DeleteUserIdentity(userID, provider string) error`.

- [ ] **Step 1: тесты** `me_test.go` (везде `authed()` кука):
  - `TestGetMeShape` — name/email/email_verified/telegram/sessions; `current:true` ровно на одной.
  - `TestPatchMeRenames`.
  - `TestChangePasswordWrongCurrent403` / `TestChangePasswordHappyKillsOtherSessions` (вторая сессия B мертва, текущая жива, sink: password_changed).
  - `TestChangePasswordTgOnlyRequiresEmail` — юзер только с TG → `{new}` без email → `400 email_required`; с email → `200 verify_sent` + verify-письмо.
  - `TestLinkTelegramConflict409` — привязать tg_id, который уже у другого юзера.
  - `TestUnlinkTelegramOnlyMethod409` — TG-only юзер → `DELETE /me/telegram` → `409`.
  - `TestDeleteSessionByHandle` / `TestDeleteSessionUnknown404`.
  - `TestDeleteMeWrongPassword403` / `TestDeleteMeHappy` (юзер и всё состояние исчезли, кука очищена).
- [ ] Steps 2–5. Коммит — `feat(api): /api/me — profile, password, telegram link, devices, delete`.

---

## Task 17: `main.go` — провязка

**Files:** Modify `server/main.go`, create/extend a smoke check

- Собрать `cfg := config.Load()`.
- `var mailer mail.Mailer`; `if cfg.SMTPEnabled() { mailer = mail.NewSMTPMailer(cfg.SMTP) } else { mailer = mail.LogMailer{} }`.
- `async := func(f func()) { go f() }`.
- `sendMail := func(to, subject, text, html string) { /* runs inside an Async goroutine already — synchronous here */ ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second); defer cancel(); if err := mailer.Send(ctx, to, subject, text, html); err != nil { ctx2, c2 := context.WithTimeout(context.Background(), 15*time.Second); defer c2(); if err2 := mailer.Send(ctx2, to, subject, text, html); err2 != nil { log.Printf("mail send failed after retry: %v", err2) } } }` — лог только на повторной ошибке, без `to`/тела.
- Лимитеры: `login := ratelimit.NewLimiter(5.0/60, 5)`, `loginEmail := ratelimit.NewLimiter(10.0/3600, 10)`, `slow := ratelimit.NewLimiter(3.0/3600, 3)`, `fails := ratelimit.NewFailCounter(10, 15*time.Minute)`.
- Housekeeping: `go func() { t := time.NewTicker(1*time.Hour); for range t.C { if _, err := st.DeleteExpiredSessions(time.Now()); err != nil { log.Printf("session sweep: %v", err) } } }()`.
- Передать всё в `api.Handler(api.Deps{... Config: cfg, SendMail: sendMail, Async: async, Login: login, LoginEmail: loginEmail, Slow: slow, Fails: fails})`.
- `flag` `-import-sqlite` и прочее — не трогать.

- [ ] **Step 1:** правки `main.go`.
- [ ] **Step 2: сборка + смоук:** `go build -o /tmp/srpski ./server && DATABASE_URL= APP_BASE_URL=http://localhost:8080 /tmp/srpski -addr :18080 -db /tmp/smoke.db &` затем:
  - `curl -s localhost:18080/api/health` → `{"status":"ok",...}`;
  - `curl -si -XPOST localhost:18080/api/auth/register -H 'Origin: http://localhost:8080' -d '{"email":"a@b.io","password":"password123","name":"Smoke"}'` → `200`; в stderr сервера — строка `LogMailer` с verify-ссылкой;
  - `curl -s "localhost:18080/api/auth/verify?token=<из лога>"` → `302` на `/verify?ok=1`;
  - `curl -si -XPOST localhost:18080/api/auth/login -H 'Origin: http://localhost:8080' -d '{"email":"a@b.io","password":"password123"}'` → `200` + `Set-Cookie: session=`;
  - `curl -s localhost:18080/api/progress --cookie 'session=<...>'` → `200` (не 401).
  - убить процесс, удалить `/tmp/smoke.db`.
- [ ] **Step 3:** зафиксировать вывод смоука в отчёте.
- [ ] **Step 4: коммит** — `feat(server): wire auth config, mailer, rate limiters, session sweep`.

---

## Финальная проверка (часть 2)

- [ ] `go test ./server/... && go build ./... && go vet ./... && gofmt -l server/` — всё зелёное (кроме пред-существующего `checker_test.go`).
- [ ] `make test-pg` — зелёное (или отложить на ревьюера).
- [ ] `go test -race ./server/internal/ratelimit/ ./server/internal/api/` — гонок нет.
- [ ] Security-чеклист спеки §2 (та часть, что про бэкенд): кука `HttpOnly`+`SameSite=Lax`+префикс; токен сессии не в теле/логах, в БД только sha256; `register`/`forgot` — одинаковый ответ и тайминг; `verify`/`reset` одноразовы и с TTL; CSP `frame-ancestors` пускает Telegram; rate-limit срабатывает; `/api/users` удалён; `PasswordHash` не сериализуется ни в один DTO (проверить `grep -n PasswordHash server/internal/api/`).
- [ ] Мост `X-User` ещё жив (часть 3 снимет) — старый фронт продолжает работать: тест `TestRequireAuthBridgeXUser` зелёный.

> **Раскатка:** часть 2 катится вместе с частью 1 одним релизом (Deploy A). После него бэкенд отдаёт и `/api/auth/*`, и мост `X-User` — старый Vue-фронт продолжает работать до Deploy B (часть 3). `TELEGRAM_BOT_TOKEN` и `SMTP_*` в env можно не задавать сразу — `/auth/telegram` вернёт 503, письма уйдут в лог; часть 4 их включает.
