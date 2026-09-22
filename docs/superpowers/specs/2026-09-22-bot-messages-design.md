# Дружелюбные и редактируемые тексты Telegram-бота

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

Задача пересекает [[serbian-app]] (бот живёт там) и
[[ucimo-content-admin]] (туда добавляется страница редактирования). Граница
между ними — админка читает прод-БД serbian-app read-only через
`ProdDbService`/роль `ucimo_admin_ro` (design `2026-09-20-...`). Эта задача
**первой** пробивает эту границу: для одной новой таблицы роль получает
`INSERT`/`UPDATE` в дополнение к `SELECT` (см. «Права на проде» ниже) — то
есть `ucimo_admin_ro` перестаёт быть строго read-only, оставаясь read-only
для всего остального.

## Архитектура

```
Пользователь -> Telegram -> POST /api/telegram/webhook (serbian-app)
                                    |
                                    v
                     telegramWebhook читает нужный текст
                     из bot_messages (Postgres serbian-app),
                     подставляет {name}, шлёт sendMessage
                     (+ inline-кнопка) через Bot API
                                    ^
                                    | INSERT/UPDATE (тот же bot_messages)
                                    |
                     ucimo-content-admin, страница «Бот» → «Сообщения»,
                     пишет через ProdDbService (роль ucimo_admin_ro,
                     теперь с точечным грантом на эту таблицу)
```

- **serbian-app** — владеет таблицей `bot_messages`, хранит дефолтные
  тексты как fallback в коде (на случай отсутствующей строки), читает
  таблицу на каждый вызов вебхука (трафик низкий — правки видны сразу, без
  редеплоя и без кеша).
- **ucimo-content-admin** — редактор поверх той же таблицы. Не хранит
  копию текстов у себя (никакого дублирования источника истины) — только
  статичный список ключей/подписей/подсказок для формы (8 записей, они не
  меняются без ручной правки кода в обоих репозиториях — это устраивает,
  учитывая частоту таких правок).

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

Дефолтные тексты (ими же будет засеяна таблица; редактируются в админке):

**`start_greeting`**
> Привет, {name}! 👋 Это бот Учимо — сервиса для изучения сербского с нуля.
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
> Попробуй войти ещё раз с сайта ucimo.ru. Если не получится — напиши нам, поможем.

**`link_success`**
> Готово! 🎉 Telegram привязан к твоему аккаунту, {name}.
>
> Теперь можно входить в Учимо и через Telegram — возвращайся на сайт.

**`link_taken`**
> Этот Telegram уже привязан к другому аккаунту Учимо.
>
> Если это ошибка — напиши нам, разберёмся.

**`link_error`**
> Упс, не получилось привязать Telegram 😔
>
> Попробуй ещё раз с сайта, а если не поможет — напиши нам.

Вне скоупа этой задачи: периодические bot-напоминания (`reminders.go`,
`allDoneMessages`/`inactivityMessages`) — у них уже есть пул дружелюбных
вариантов, это отдельная фича с другим триггером (не реакция на
пользовательское действие, а cron-джоба).

## Схема данных (serbian-app, миграция `007_bot_messages.sql`)

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

### Права на проде

```sql
GRANT SELECT, INSERT, UPDATE ON bot_messages TO ucimo_admin_ro;
```

Точечный грант на одну таблицу — по образцу уже принятого подхода
"гранты по колонкам/таблицам, не блокет" (`2026-09-20-...design.md`). Всё
остальное для `ucimo_admin_ro` остаётся read-only. Выполняется руками через
psql после того, как миграция создаст таблицу на проде (тот же процесс,
которым были выданы текущие гранты).

## Go-изменения (serbian-app)

- `server/internal/store/bot_messages.go`: `BotMessageText(key string) (text string, ok bool, err error)`.
- `server/internal/telegram/messages.go`: карта дефолтов (8 ключей) + `Substitute(text, name string) string` (`strings.ReplaceAll(text, "{name}", name)`) + `DisplayName(firstName, username string) string` (first_name → @username → "друг").
- `server/internal/telegram/telegram.go`: новая `SendMessageWithButton(botToken string, chatID int64, text string, button *InlineButton) error`, где
  ```go
  type InlineButton struct {
  	Label     string
  	WebAppURL string // если задан — кнопка типа web_app
  	URL       string // иначе, если задан — обычная url-кнопка
  }
  ```
  собирает `reply_markup.inline_keyboard`. `nil` кнопка = обычный `sendMessage` без клавиатуры.
- `Deps.SendTelegramMessage` меняет сигнатуру на `func(chatID int64, text string, button *telegram.InlineButton)` — правится `main.go` (wiring) и `telegram_bot_test.go` (`tgSink` начинает захватывать и кнопку).
- `telegram.IsBareStart(text string) bool` — true для `"/start"`/`"/start@bot"` без токена (сейчас `ParseStartToken` на этот случай просто возвращает `false, false`, и вызывающий код ничего не шлёт).
- `telegramWebhook`:
  - если `ParseStartToken` вернул `false` и `IsBareStart(text)` — шлёт `start_greeting` + кнопка на Mini App (`cfg.AppBaseURL + "/profile"`, тот же URL, что уже использует `SetChatMenuButton`), выходит.
  - `Lookup(token)` не нашёл — вместо молчания шлёт `login_token_expired`.
  - `resolveTelegramLogin` — использует возвращённый `created` вместо `_`: `login_success_new` или `login_success_existing`.
  - остальные ветки (`login_error`, `link_success`, `link_taken`, `link_error`) — та же логика, что сейчас, но через новый механизм текстов + кнопки.
  - Кнопка "Написать в поддержку" — константа `telegram.SupportURL = "https://t.me/ucimosupport"` (то же значение, что `web/src/lib/supportModal.ts`, задублировано намеренно — Go и Vue не делят константы).
- Смена сигнатуры `SendTelegramMessage` задевает и `RunReminderSweep`
  (`reminders.go`, 2 текущих вызова) — они просто передают `nil` кнопкой,
  поведение и тексты напоминаний не меняются (см. «Вне скоупа»).

## Эндпоинты и Frontend (ucimo-content-admin)

- Новый модуль `server/src/bot-messages/` (по образцу `links`):
  - Статичный список из 8 `{key, label, hint}` (без текста — текст всегда
    берётся из БД, она всегда заполнена после миграции).
  - `GET /bot-messages` — джойнит статичный список с `SELECT key, text, updated_at FROM bot_messages` через `ProdDbService`.
  - `PUT /bot-messages/:key` `{text}` — 404 если `key` не из списка восьми, 400 на пустой текст после trim, иначе `UPDATE bot_messages SET text=$1, updated_at=$2 WHERE key=$3`.
- Новый раздел сайдбара «Бот» (`AdminSidebar.vue`, по образцу секции
  «Ссылки» — отдельно от «Данные», это CRUD, а не read-only отчёт), пункт
  «Сообщения» → `/bot/messages`.
- `BotMessagesView.vue`: 8 карточек (подпись + подсказка + textarea с
  текущим текстом + кнопка «Сохранить»), без превью подстановки `{name}` —
  YAGNI, текст короткий и `{name}` виден прямо в поле.

## Тестирование

- **serbian-app**: `internal/telegram` — юнит-тесты на `Substitute`,
  `DisplayName`, `IsBareStart`, JSON `InlineButton`. `internal/api` —
  расширить `telegram_bot_test.go`: голый `/start` шлёт `start_greeting` с
  кнопкой на Mini App; новый аккаунт получает `login_success_new`,
  повторный логин — `login_success_existing`; просроченный токен больше не
  молчит; override из `bot_messages` перебивает дефолт (тест пишет строку
  в стор перед вызовом вебхука). `go test ./server/...` на SQLite,
  `make test-pg` — миграция 007 накатывается на оба диалекта.
- **ucimo-content-admin**: `bot-messages`-модуль — тесты как у `links`
  (реальный Postgres через `ProdDbService`, без моков): список содержит
  все 8 ключей, `PUT` с неизвестным ключом → 404, с пустым текстом → 400,
  успешный `PUT` виден в следующем `GET`.

## Вне скоупа

- Периодические bot-напоминания (`reminders.go`) — свой пул текстов,
  другой триггер, не трогаем.
- Кнопка «Написать в поддержку» **внутри бота** не открывает никакой
  формы — просто url-ссылка на существующий `t.me/ucimosupport`, как на
  сайте.
- История правок / откат к дефолту в админке — не просят, YAGNI.
- Локализация (сербский и т.п.) — все тексты бота только на русском, как
  и сейчас.
