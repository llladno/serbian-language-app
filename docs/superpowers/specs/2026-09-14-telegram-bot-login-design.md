# Telegram-бот вход через /start — дизайн

## Мотивация

Текущий вход через Telegram — Login Widget (`Telegram.Login.auth`) — требует
`/setdomain` у BotFather и работает только с фиксированного, заранее
зарегистрированного домена. Взамен — вход через глубокую ссылку на бота
(`t.me/<bot>?start=<token>`): пользователь жмёт кнопку на сайте, попадает в
Telegram, жмёт Start, возвращается на сайт уже залогиненным. Не требует
`/setdomain`, работает одинаково на любом домене и локально.

Silent-автовход внутри Telegram Mini App (через `initData`) не меняется —
это отдельный, уже рабочий механизм.

## Не-цели

- Защита от «чужой ссылки» (кто-то прислал вам токен, сгенерированный в
  своей вкладке, вы жмёте Start в Telegram — залогинится их вкладка).
  Известный класс риска у этой схемы, для двух доверенных пользователей
  сочли не критичным — без кода-подтверждения.
- Reply-клавиатуры, инлайн-кнопки, любой другой функционал бота, кроме
  обработки `/start <token>`.

## Поток

### Вход (нет активной сессии)

1. `POST /api/auth/telegram/start` (публичный) → сервер генерирует
   одноразовый токен (`auth.NewToken()`, тот же примитив, что и у
   verify/reset), кладёт в `Pending` с `{purpose: "login", userID: ""}`,
   отвечает `{"url": "https://t.me/ucimoappbot?start=<raw>"}`.
2. Фронт открывает URL в новой вкладке/окне, начинает поллить
   `GET /api/auth/telegram/poll?token=<raw>` каждые 1.5 с.
3. Юзер жмёт Start → Telegram шлёт апдейт на наш вебхук. Вебхук парсит
   `/start <token>` из `message.text`, берёт `message.from.id` /
   `.username` / `.first_name` — это уже аутентифицированные Telegram-
   данные (подпись не нужна, это server-to-server вызов от самого
   Telegram). Резолвит токен, резолвит аккаунт **той же логикой, что и
   сейчас** (`IdentityByProviderUID` → `AttachPendingTelegram` →
   `CreateUser`+`CreateIdentity`), помечает `Pending` как
   `{status: done, userID: <resolved>}`. Отвечает в чат сообщением
   «Готово! Вернитесь на сайт.» (`sendMessage`).
4. Следующий `poll`, увидев `done`, **на этом самом ответе** (обычный
   запрос из браузера того же юзера, не вебхук) ставит сессионную куку
   (`issueSession`) и отдаёт `{status:"ok", user: {...}}` — тем же DTO,
   что `POST /api/auth/telegram` сейчас.
5. Токен одноразовый: как только `poll` вернул `done`, запись удаляется
   из `Pending`. TTL — 5 минут; просроченный/неизвестный токен → `poll`
   отвечает `{status:"expired"}`.

### Привязка (уже залогинен по паролю)

То же самое, но `POST /api/auth/telegram/start` требует сессию
(`requireSession`) и кладёт в `Pending` текущий `userID`. Вебхук при
резолве токена с непустым `userID` вызывает не `finishTelegramLogin`, а
логику `linkTelegram` (уже привязан кому-то другому → `poll` вернёт
`{status:"error", error:"telegram_taken"}`, а не тихо перелогинивает).

## Данные

**В памяти**, не в БД — один инстанс (`replicas: 1`), токены живут
минуты, потеря при рестарте не страшна (юзер просто нажмёт кнопку снова).
Новый тип в `internal/auth` (уже отвечает за токены/сессии):

```go
type PendingTelegram struct {
    UserID  string    // "" для login, реальный id для link
    Status  string    // "pending" | "done" | "error"
    Error   string    // заполнено только при status == "error"
    Resolved string   // user_id, к которому резолвился вход/линковка — заполнено только при status == "done"
    Expires time.Time
}
```

(DTO для ответа `poll` собирается в хендлере через уже существующий
`summaryFor(resolvedUserID)` + `summaryToDTO` — `Pending` хранит только
`user_id`, не готовый DTO, чтобы не дублировать источник правды.)

Хранилище — мьютекс + мапа `map[tokenHash]PendingTelegram`, TTL-очистка
как в `internal/ratelimit` (та же паттерн: `evictLocked` по таймеру/при
обращении). Ключ — `sha256` от сырого токена (как у sessions/email_tokens
— сырой токен никогда не лежит в памяти читаемым, хотя это ниже риском
чем БД, всё равно единообразно).

## Новый пакет `internal/telegram`

Тонкий HTTP-клиент к Bot API — **не** трогает существующий
`internal/auth/telegram.go` (HMAC-проверка `initData`/widget для Mini App
остаётся).

```go
package telegram

func SendMessage(botToken string, chatID int64, text string) error
func SetWebhook(botToken, url, secretToken string) error
```

Апдейт от Telegram парсится прямо в хендлере `internal/api` — только те
поля, что реально нужны (`message.chat.id`, `message.text`, `message.from.{id,username,first_name}`).

## Username бота

`t.me/<username>?start=` нужен username, а не числовой `bot_id` (тот, что
уже отдаётся на `/api/health`, выводится из токена сдвигом до `:` — этого
для ссылки недостаточно). Username при старте не задан нигде — получаем
его тем же самым `getMe`-вызовом Bot API, что часто делают для проверки
токена. `internal/telegram.GetMe(botToken) (username string, err error)`,
вызывается один раз при старте (там же, где `SetWebhook`), результат
кладётся в `Deps`/держится в памяти процесса. Ошибка — как и с вебхуком,
логируем, не роняем сервер (бот-функциональность просто не работает,
остальное приложение не должно от этого зависеть).

## Вебхук

`POST /api/telegram/webhook` (публичный, вне `requireAuth`/`requireSession`
— это не пользовательский запрос). Проверяет заголовок
`X-Telegram-Bot-Api-Secret-Token` против случайного секрета,
сгенерированного при регистрации вебхука — без совпадения `401`, никакой
обработки. Не начинающееся с `/start ` сообщение — `200 OK` без действия
(Telegram обязательно ждёт `200`, иначе ретраит).

**Регистрация вебхука** — при старте сервера, если заданы
`TELEGRAM_BOT_TOKEN` и `APP_BASE_URL` (`https://`): сервер сам генерирует
случайный `secret_token` (в памяти процесса, не персистентный — при
рестарте создаётся новый и регистрируется заново, старый автоматически
инвалидируется через тот же `setWebhook`) и дёргает
`setWebhook(APP_BASE_URL + "/api/telegram/webhook", secret_token)`.
Ошибка регистрации — залогировать, не `log.Fatal` (бот — не критичная для
старта зависимость, как и SMTP).

## Эндпойнты (сводка)

| Метод | Путь | Auth | Поведение |
|---|---|---|---|
| POST | `/api/auth/telegram/start` | публичный | Генерит login-токен |
| POST | `/api/me/telegram/start` | `requireSession` | Генерит link-токен |
| GET | `/api/auth/telegram/poll` | публичный (токен = секрет) | `{status: pending\|done\|expired\|error, user?, error?}` |
| POST | `/api/telegram/webhook` | `X-Telegram-Bot-Api-Secret-Token` | Принимает апдейты от Telegram |

`POST /api/auth/telegram` (Mini App, `{init_data}`) и `POST
/api/me/link/telegram` (Mini App-линковка) — **остаются без изменений**,
это отдельный, независимый путь.

## Что удаляется

- `web/src/components/TelegramLoginButton.vue` + его тест
- Кнопка в `LoginView.vue` (Login Widget), связанный код/тест
- `auth.VerifyWidget` в `server/internal/auth/telegram.go` + его тесты —
  после удаления кнопки виджетный payload никто не шлёт
- Виджетная ветка в `verifyTelegramPayload` (`server/internal/api/auth.go`)
  — оставляем только `init_data`-путь (Mini App)

## Тесты

- `internal/telegram`: `SendMessage`/`SetWebhook` — httptest-сервер вместо
  реального Bot API, проверка URL/тела запроса.
- Pending-хранилище (`internal/auth`): create → poll(pending) →
  resolve → poll(done, one-shot — второй poll того же токена не находит
  ничего вместо повторного `done`) → TTL-истечение.
- `internal/api`: вебхук — валидный secret + `/start <token>` резолвит
  login и link токены; невалидный secret → `401`; не-/start сообщение →
  `200` без побочных эффектов; `poll` для несуществующего/просроченного
  токена → `expired`; login-резолюция переиспользует существующую логику
  (identity уже есть / pending-claim по username / новый юзер) — те же
  кейсы, что уже покрыты для `POST /api/auth/telegram`, только через
  вебхук вместо прямого HMAC.
- Frontend: `LoginView`/`ProfileView` — клик по кнопке зовёт `start`,
  открывает `window.open` на вернувшийся `url`, поллит, на `done`
  простав­ляет `session.user`/обновляет профиль и останавливает поллинг.
