# Дружелюбные и редактируемые тексты Telegram-бота + рассылки

Дата: 2026-09-22

## Контекст

Сейчас бот `@ucimoappbot` отвечает сухими служебными фразами
("Готово! Вернитесь на сайт.") только на `/start <token>` — диплинк,
который генерирует сам сайт для входа/привязки через Telegram. У этого два
недостатка:

1. Тон не дружелюбный, и сценарий "уже зарегистрирован" никак не отличается
   от "успешная новая регистрация" — `resolveTelegramLogin` уже возвращает
   `created bool`, но `telegramWebhook` его отбрасывает (`_`).
2. Тексты зашиты в Go-код — чтобы поправить формулировку, нужен деплой
   serbian-app. Хочется редактировать их через админку без правки кода.

Плюс два случая сейчас **молчат** совсем: голый `/start` без токена (бота
открыли напрямую, а не по ссылке с сайта) и просроченный/неизвестный токен.

Отдельно: хочется уметь слать сообщения из админки — конкретным
пользователям или всем сразу (рассылка). Это должно быть безопасно с точки
зрения лимитов Bot API — единая очередь на отправку, с приоритетом:
`/start`-флоу (вход/привязка) всегда важнее массовой рассылки.

Задача пересекает [[serbian-app]] (бот и очередь отправки живут там) и
[[ucimo-content-admin]] (туда добавляются страницы редактирования текстов и
рассылок). Граница между ними — админка читает прод-БД serbian-app
read-only через `ProdDbService`/роль `ucimo_admin_ro` (design
`2026-09-20-...`). Эта задача **первой** пробивает эту границу: роль
получает точечные `INSERT`/`UPDATE` на две новые таблицы (см. «Права на
проде» ниже), оставаясь read-only для всего остального.

## Лимиты Telegram Bot API (официальный FAQ)

https://core.telegram.org/bots/faq — раздел "My bot is hitting limits, how
do I avoid this?":

- не больше **1 сообщения в секунду** в один и тот же чат;
- не больше **30 сообщений в секунду** суммарно при массовой рассылке
  разным пользователям (soft-limit — Telegram может пропустить кратковременный
  всплеск, но при продолжении начнёт отвечать `429 Too Many Requests` с
  `retry_after` в секундах);
- отдельный, более жёсткий лимит на группы (20 сообщений/минуту) — не
  актуален, у бота только приватные чаты.

Берём **25 сообщений/сек** как рабочий потолок (запас от 30) — константа в
коде (см. вопрос про настраиваемость в разделе «Вне скоупа»). Лимит на чат
(1/сек) отдельно не форсируем: то, что один и тот же chat_id получит два
сообщения от бота в одну и ту же секунду, на практике не случается (одно
действие пользователя — один ответ), а рассылка по определению шлёт в каждый
чат ровно одно сообщение.

## Архитектура

```
                    ┌─ POST /api/telegram/webhook (вход/привязка) ──┐
                    │                                                │  priority=high
ucimo-content-admin │                                                ▼
  «Бот»→«Рассылки» ─┼─ INSERT в bot_outbox (priority=normal) ──► bot_outbox (Postgres serbian-app)
  (ProdDbService,    │                                                │
   роль с точечным   │                                                │ воркер, тикер ~25/сек,
   INSERT/UPDATE)    │                                                │ приоритет ASC, потом FIFO
                    │                                                ▼
  «Бот»→«Сообщения» ─┴─ UPDATE bot_messages ──────────►  telegram.SendMessageWithButton
   (тексты 8 шаблонов,                                    (Bot API), учитывает 429/retry_after
    та же роль)
```

- **serbian-app** — единственный, кто реально дёргает Bot API. Владеет
  таблицами `bot_messages` (тексты шаблонов) и `bot_outbox` (очередь на
  отправку) и фоновым воркером, который её разбирает с учётом приоритета и
  общего рейт-лимита. И `/start`-вебхук, и periodic-напоминания
  (`reminders.go`), и админские рассылки — все теперь просто кладут строку
  в `bot_outbox`, реальную отправку и лимиты видит только воркер.
- **ucimo-content-admin** — два новых раздела: редактор текстов (пишет в
  `bot_messages`) и композер рассылок (пишет в `bot_outbox` напрямую,
  priority=normal). Не хранит копию текстов/очереди у себя.

## Сообщения — 8 штук, все с `{name}`

`{name}` — `first_name` из Telegram, иначе `@username`, иначе слово «друг».
Все тексты — на "ты", как и остальной фронт (см. `ProgressDashboard.vue`,
`FreeAnswer.vue`).

| Ключ | Когда отправляется | Кнопка |
|---|---|---|
| `start_greeting` | Голый `/start` (бот открыт напрямую, без диплинка) — **сейчас бот молчит** | «Открыть Учимо» → Mini App |
| `login_success_new` | `/start <token>` с сайта, аккаунт создан впервые (**сценарий 1**) | «Открыть Учимо» → Mini App |
| `login_success_existing` | То же, но аккаунт уже существовал (**сценарий 2**) | «Открыть Учимо» → Mini App |
| `login_token_expired` | Токен неизвестен/просрочен — **сейчас бот молчит** | нет |
| `login_error` | Внутренняя ошибка при входе (**сценарий 3**) | «Написать в поддержку» → t.me/ucimosupport |
| `link_success` | Telegram привязан к уже залогиненному аккаунту (флоу из профиля) | «Открыть Учимо» → Mini App |
| `link_taken` | Этот Telegram уже привязан к другому аккаунту | «Написать в поддержку» |
| `link_error` | Внутренняя ошибка при привязке | «Написать в поддержку» |

Дефолтные тексты (ими же будет засеяна таблица; редактируются в админке).
Сообщения с упоминанием поддержки несут и текстовую ссылку `@ucimosupport`
(Telegram сам делает её кликабельной), и отдельную inline-кнопку — на
случай, если клиент режет автолинк.

**`start_greeting`**
> Zdravo (привет), {name}! 👋 Это бот Учимо — сервиса для изучения сербского с нуля.
>
> Жми на кнопку ниже, чтобы открыть приложение и начать учиться.

**`login_success_new`**
> Готово, {name}! 🎉 Регистрация прошла успешно — добро пожаловать в Учимо.
>
> Заходи на сайт ucimo.ru или сразу открывай приложение здесь, в Telegram, — и начинай первый урок!

**`login_success_existing`**
> С возвращением, {name}! 👋 Ты уже с нами — продолжай изучать сербский.
>
> Открывай приложение и вперёд: тебя ждут уроки и повторение слов.

**`login_token_expired`**
> Кажется, эта ссылка уже устарела 🙈
>
> Зайди на ucimo.ru и попробуй войти через Telegram ещё раз.

**`login_error`**
> Упс, что-то пошло не так 😔
>
> Попробуй войти ещё раз с сайта ucimo.ru. Если не получится — напиши нам: @ucimosupport

**`link_success`**
> Готово! 🎉 Telegram привязан к твоему аккаунту, {name}.
>
> Теперь можно входить в Учимо и через Telegram — возвращайся на сайт.

**`link_taken`**
> Этот Telegram уже привязан к другому аккаунту Учимо.
>
> Если это ошибка — напиши нам: @ucimosupport

**`link_error`**
> Упс, не получилось привязать Telegram 😔
>
> Попробуй ещё раз с сайта, а если не поможет — напиши нам: @ucimosupport

Периодические bot-напоминания (`reminders.go`, `allDoneMessages`/
`inactivityMessages`) свои тексты не меняют и в `bot_messages` не попадают —
у них уже есть пул дружелюбных вариантов, это отдельная фича с другим
триггером (cron-джоба, не реакция на действие пользователя). В область этой
задачи они попадают только с одной стороны — начинают идти через общую
очередь `bot_outbox` (см. ниже), чтобы не создавать собственный
неконтролируемый всплеск запросов к Bot API при рассылке 20-минутного
свипа на всех активных пользователей разом.

## Схема данных (serbian-app)

### `bot_messages` — миграция `007_bot_messages.sql`

```sql
CREATE TABLE IF NOT EXISTS bot_messages (
	key        TEXT PRIMARY KEY,
	text       TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

INSERT INTO bot_messages (key, text, updated_at) VALUES
	('start_greeting', '<дефолт выше>', '<now>'),
	('login_success_new', '<дефолт выше>', '<now>'),
	('login_success_existing', '<дефолт выше>', '<now>'),
	('login_token_expired', '<дефолт выше>', '<now>'),
	('login_error', '<дефолт выше>', '<now>'),
	('link_success', '<дефолт выше>', '<now>'),
	('link_taken', '<дефолт выше>', '<now>'),
	('link_error', '<дефолт выше>', '<now>')
ON CONFLICT (key) DO NOTHING;
```

Сидинг прямо в миграции — чтобы в админке сразу было что показать/править,
без отдельного ручного шага после деплоя. `internal/telegram` держит те же
8 текстов как константы — используются только если строка в таблице вдруг
отсутствует (defensive fallback, в норме не задействуется).

### `bot_outbox` — миграция `008_bot_outbox.sql`

```sql
CREATE TABLE IF NOT EXISTS bot_outbox (
	id            {{.AutoID}},
	chat_id       BIGINT NOT NULL,
	text          TEXT NOT NULL,
	button_label  TEXT,
	button_type   TEXT,                     -- 'web_app' | 'url' | NULL (нет кнопки)
	button_target TEXT,                     -- URL для button_type
	priority      SMALLINT NOT NULL,        -- 0 = high (вход/привязка), 1 = normal (напоминания, рассылки)
	status        TEXT NOT NULL DEFAULT 'pending', -- pending | sent | failed
	error         TEXT,
	created_at    TEXT NOT NULL,
	sent_at       TEXT
);
CREATE INDEX IF NOT EXISTS bot_outbox_pending_idx ON bot_outbox (status, priority, created_at);
```

`button_type` различает Mini-App-кнопку (`web_app`, открывает `/profile`
как Mini App с авто-логином через initData) от обычной url-кнопки
(«Написать в поддержку»). Обе строки кладёт только serbian-app —
`telegramWebhook` заполняет оба поля по фиксированным сценариям; рассылки
из админки всегда пишут `button_type = NULL` (только текст, см. «Вне
скоупа»). Воркер при разборе строки собирает `*telegram.InlineButton` из этих двух
колонок.

### Права на проде

```sql
GRANT SELECT, INSERT, UPDATE ON bot_messages TO ucimo_admin_ro;
GRANT INSERT ON bot_outbox TO ucimo_admin_ro;
-- Ни SELECT, ни UPDATE на bot_outbox admin'у не нужны — приложение
-- (candidates/dryRun) считает получателей по users+identities, не по
-- очереди; статус/ошибки строк меняет исключительно воркер serbian-app;
-- посмотреть саму очередь руками можно через psql напрямую.
```

Точечные гранты на две таблицы — по образцу уже принятого подхода "гранты
по колонкам/таблицам, не блоком" (`2026-09-20-...design.md`). Всё
остальное для `ucimo_admin_ro` остаётся read-only. Выполняется руками через
psql после того, как миграции создадут таблицы на проде (тот же процесс,
которым были выданы текущие гранты).

## Go-изменения (serbian-app)

### Тексты шаблонов

- `server/internal/store/bot_messages.go`: `BotMessageText(key string) (text string, ok bool, err error)`.
- `server/internal/telegram/messages.go`: карта дефолтов (8 ключей) + `Substitute(text, name string) string` (`strings.ReplaceAll(text, "{name}", name)`) + `DisplayName(firstName, username string) string` (first_name → @username → "друг").
- `telegram.IsBareStart(text string) bool` — true для `"/start"`/`"/start@bot"` без токена (сейчас `ParseStartToken` на этот случай просто возвращает `false, false`, и вызывающий код ничего не шлёт).
- Кнопка "Написать в поддержку" — константа `telegram.SupportURL = "https://t.me/ucimosupport"` (то же значение, что `web/src/lib/supportModal.ts`, задублировано намеренно — Go и Vue не делят константы).

### Очередь на отправку (новое: `internal/outbox`)

- `store.EnqueueBotMessage(chatID int64, text string, button *telegram.InlineButton, priority int, at time.Time) error` — INSERT в `bot_outbox`.
- `telegram.InlineButton{Label, WebAppURL, URL string}` — как раньше планировалось для прямой отправки, теперь просто поля, которые кладутся в очередь вместе с текстом; `SendMessageWithButton(botToken string, chatID int64, text string, button *InlineButton) error` остаётся низкоуровневым вызовом Bot API, но вызывает его теперь только воркер, не хендлеры.
- `internal/outbox/worker.go`, `RunOutboxWorker(st *store.Store, botToken string, now func() time.Time)`, запускается горутиной в `main.go` (как sweep сессий/напоминаний). Тикер `time.Second/25` (см. «Лимиты» выше):
  - на каждый тик: `SELECT ... WHERE status='pending' ORDER BY priority ASC, created_at ASC LIMIT 1` (работает на обоих диалектах через существующий `rebind`);
  - нет строки — ничего не делает, ждёт следующий тик;
  - есть строка — `SendMessageWithButton`; успех → `status='sent', sent_at=now`; `429` → **не трогает статус** (строка остаётся `pending`, будет подхвачена повторно), воркер спит `retry_after` секунд сверху; любая другая ошибка → `status='failed', error=...`, без ретраев (v1, см. «Вне скоупа»).
- `Deps.SendTelegramMessage func(chatID int64, text string)` заменяется на `Deps.EnqueueTelegramMessage func(chatID int64, text string, button *telegram.InlineButton, priority int)` — тонкая обёртка над `store.EnqueueBotMessage`. Правится `main.go` (wiring), `telegram_bot_test.go` и `reminders_test.go` (там, где раньше ловили `sink.msgs`, теперь читают строки `bot_outbox` через тестовый `Store`).
- `telegram.PriorityHigh = 0`, `telegram.PriorityNormal = 1` — константы, чтобы вызывающий код не путался в сырых числах.

### `telegramWebhook`

- если `ParseStartToken` вернул `false` и `IsBareStart(text)` — enqueue `start_greeting` (priority high) + кнопка на Mini App (`cfg.AppBaseURL + "/profile"`, тот же URL, что уже использует `SetChatMenuButton`), выходит.
- `Lookup(token)` не нашёл — вместо молчания enqueue `login_token_expired` (priority high).
- `resolveTelegramLogin` — использует возвращённый `created` вместо `_`: `login_success_new` или `login_success_existing`.
- остальные ветки (`login_error`, `link_success`, `link_taken`, `link_error`) — та же логика, что сейчас, но через enqueue + приоритет high.
- Вебхук по-прежнему отвечает 200 сразу (до всякой обработки) — теперь ответ ещё меньше зависит от сети: раньше блокировался на HTTP-вызове к Bot API, теперь только на быстрый INSERT.

### `reminders.go`

Два текущих вызова `SendTelegramMessage` меняются на `EnqueueTelegramMessage(..., priority=Normal)` — тексты и логика выбора получателей не меняются, просто перестают быть нерегулируемым прямым вызовом Bot API (сейчас 20-минутный свип может дёрнуть Bot API для всех активных пользователей разом без какого-либо троттлинга — это и есть тот скрытый риск, который заодно закрывает общая очередь).

## Эндпоинты и Frontend (ucimo-content-admin)

### Редактор текстов — раздел «Бот» → «Сообщения»

- Новый модуль `server/src/bot-messages/` (по образцу `links`):
  - Статичный список из 8 `{key, label, hint}` (без текста — текст всегда
    берётся из БД, она всегда заполнена после миграции).
  - `GET /bot-messages` — джойнит статичный список с `SELECT key, text, updated_at FROM bot_messages` через `ProdDbService`.
  - `PUT /bot-messages/:key` `{text}` — 404 если `key` не из списка восьми, 400 на пустой текст после trim, иначе `UPDATE bot_messages SET text=$1, updated_at=$2 WHERE key=$3`.
- `BotMessagesView.vue`: 8 карточек (подпись + подсказка + textarea с
  текущим текстом + кнопка «Сохранить»), без превью подстановки `{name}` —
  YAGNI, текст короткий и `{name}` виден прямо в поле.

### Рассылки — раздел «Бот» → «Рассылки»

- Новый модуль `server/src/broadcasts/`:
  - `GET /broadcasts/candidates` — список пользователей с привязанным
    Telegram для селектора: `SELECT u.id, u.name, i.provider_uid FROM users u JOIN identities i ON i.user_id = u.id AND i.provider = 'telegram'` через `ProdDbService` (те же колонки `identities`, что уже разрешены роли). Пользователи без Telegram в список не попадают — им физически некуда слать.
  - `POST /broadcasts` body `{ recipients: {mode: 'all'} | {mode: 'users', userIds: string[]}, text: string, dryRun?: boolean }`:
    - `mode: 'all'` → берёт все `provider_uid` из `candidates`-запроса;
    - `mode: 'users'` → те же, отфильтрованные по `userIds`;
    - `dryRun: true` — только считает получателей, ничего не пишет (для подтверждения на фронте: «Отправить всем N?»);
    - иначе — batch `INSERT INTO bot_outbox (chat_id, text, priority, status, created_at) VALUES ...` (priority=normal) через `ProdDbService`, возвращает `{queued: N}`.
  - Пустой/только-пробельный `text` → 400. Ключи `key` из `bot-messages` тут ни при чём — это свободный текст, не шаблон.
- Новый раздел сайдбара «Бот» (`AdminSidebar.vue`, по образцу секции
  «Ссылки» — отдельно от «Данные», это CRUD/действие, а не read-only отчёт):
  пункты «Сообщения» (`/bot/messages`) и «Рассылки» (`/bot/broadcasts`).
- `BroadcastsView.vue`: поиск/мулти-select получателей (по имени, из
  `candidates`) + переключатель «Всем пользователям с Telegram (N)» +
  textarea с текстом + кнопка «Отправить». Клик → сперва `dryRun`-запрос,
  модалка «Отправить N получателям? [Да, отправить всем N] [Отмена]»,
  только после подтверждения — настоящий `POST`. После успеха — тост
  «Поставлено в очередь: N».

## Тестирование

- **serbian-app**:
  - `internal/telegram` — юнит-тесты на `Substitute`, `DisplayName`,
    `IsBareStart`, JSON `InlineButton`.
  - `internal/outbox` — юнит-тесты воркера: приоритет `high` уходит раньше
    `normal` даже если добавлен позже; `429` с `retry_after` не помечает
    строку `failed` и не теряет её; прочая ошибка помечает `failed` без
    ретрая; тикер не шлёт больше одного сообщения за тик (без реального Bot
    API — HTTP-клиент мокается через `httptest`, как уже сделано в
    `telegram_test.go`).
  - `internal/api` — расширить `telegram_bot_test.go`: голый `/start`
    кладёт в очередь `start_greeting` с кнопкой на Mini App; новый аккаунт
    получает `login_success_new`, повторный логин — `login_success_existing`;
    просроченный токен больше не молчит; override из `bot_messages`
    перебивает дефолт. `reminders_test.go` — проверяет, что напоминания
    теперь тоже идут через `bot_outbox` (priority normal), поведение выбора
    получателей не меняется.
  - `go test ./server/...` на SQLite, `make test-pg` — миграции 007/008
    накатываются на оба диалекта.
- **ucimo-content-admin**:
  - `bot-messages`-модуль — тесты как у `links` (реальный Postgres через
    `ProdDbService`, без моков): список содержит все 8 ключей, `PUT` с
    неизвестным ключом → 404, с пустым текстом → 400, успешный `PUT` виден
    в следующем `GET`.
  - `broadcasts`-модуль: `candidates` возвращает только Telegram-привязанных
    пользователей; `dryRun` считает получателей и не пишет строк;
    `mode:'all'`/`mode:'users'` кладут правильное число строк с
    `priority=1, status='pending'`; пустой текст → 400.

## Вне скоупа

- Периодические bot-напоминания (`reminders.go`) — свои тексты не трогаем,
  меняется только механизм доставки (через очередь).
- Кнопка «Написать в поддержку» **внутри бота** не открывает никакой
  формы — просто url-ссылка/текстовое упоминание существующего
  `t.me/ucimosupport` (`@ucimosupport`), как на сайте.
- История правок / откат к дефолту текстов в админке — не просят, YAGNI.
- История отправленных рассылок в админке (кто/когда/скольким слал) — не
  просят, YAGNI; сами строки остаются в `bot_outbox` с `status`, при
  необходимости смотрятся руками через БД.
- Кнопка/вложение в тексте рассылки — только текст, v1.
- Ретраи для `failed` (кроме `429`, который и так не считается провалом) —
  не делаем; типичная причина провала — пользователь заблокировал бота,
  повторять бессмысленно.
- Лимит скорости отправки (сообщ./сек) не редактируется в админке —
  зашит в коде как безопасная константа (см. «Лимиты Telegram Bot API»).
- Локализация (сербский и т.п.) — все тексты бота только на русском, как
  и сейчас (кроме самого слова "Zdravo" в приветствии).
