# Уведомления в приложении (in-app notifications из админки)

Дата: 2026-09-23

## Контекст

Сейчас у Гриши нет способа сообщить что-то пользователям ucimo прямо в
приложении — только Telegram-рассылка (`bot_outbox`, см.
`2026-09-22-bot-messages-design.md`), которая достаёт только тех, кто
привязал Telegram. Нужен канал внутри самого сайта: раздел в админке, где
можно выбрать получателей (всех / нескольких / одного) и текст, и колокольчик
в приложении, который показывает непрочитанные уведомления.

Задача пересекает [[serbian-app]] (хранит уведомления, отдаёт их
пользователю, отмечает прочитанными) и [[ucimo-content-admin]] (раздел
композера). Граница между ними — та же, что уже пробита вчерашней задачей
рассылок: админка пишет в прод-БД serbian-app напрямую через
`ProdDbService`/роль `ucimo_admin_ro`, с точечным `INSERT`-грантом на две
новые таблицы. Никакого нового сервиса и воркера не нужно — в отличие от
`bot_outbox`, здесь никто ничего не "отправляет" во внешний API: это просто
таблица, которую serbian-app сам же читает для залогиненного пользователя.

Без WebSocket в v1 — список обновляется поллингом. Архитектура его не
исключает: когда WebSocket появится, он просто заменит поллинг в
`lib/notifications.ts`, схема данных и API не меняются.

## Архитектура

```
ucimo-content-admin                          serbian-app (Postgres)
 «Уведомления» → композер
  (ProdDbService, точечный INSERT) ──INSERT──► notifications
                                                notification_recipients ◄──┐
                                                                            │
serbian-app Go API                                                        │
 GET  /api/me/notifications          ──reads───────────────────────────────┘
 POST /api/me/notifications/mark-read ──UPDATE read_at──────────────────────┘
   │
   ▼
Vue: NotificationBell.vue (поллинг 45с, без WebSocket — задел на будущее)
```

Альтернатива — одна строка `notifications` с JSON-списком получателей
вместо join-таблицы `notification_recipients` — экономит место при
рассылке "всем", но ломает уже принятую в `bot_outbox` конвенцию
"материализовать по получателю" и усложняет запрос на двух диалектах
(SQLite слабее с JSON). Не берём.

## Схема данных (serbian-app) — миграция `010_notifications.sql`

```sql
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

- `text` — сырой ввод админа, может содержать `[текст](url)`-ссылки;
  парсится только на фронте (см. ниже), в БД хранится как есть.
- При отправке админка вставляет одну строку `notifications` и по одной
  строке `notification_recipients` на каждого получателя — "всем" тоже
  материализуется списком на момент отправки (будущие регистрации не
  попадают, как договорились).
- `read_at = ''` — непрочитано (та же конвенция, что `sent_at`/`error` в
  `bot_outbox`); при прочтении — `now` в RFC3339.
- Хранение 30 дней **после** прочтения — фильтр на чтении
  (`read_at = '' OR read_at > now-30d`), не отдельная job на удаление;
  непрочитанные хранятся бессрочно. Реального удаления строк в v1 нет —
  YAGNI, как и с историей рассылок в bot-messages design.
- Права на проде: `GRANT INSERT ON notifications, notification_recipients TO ucimo_admin_ro;`
  Ни `SELECT`, ни `UPDATE` админке не нужны — она никогда не читает эти
  таблицы обратно (нет отчёта "кто прочитал").

## Go-изменения (serbian-app)

- `server/internal/store/notifications.go`:
  - `type Notification struct { ID int64; Text string; CreatedAt time.Time; Read bool }`
  - `ListNotificationsForUser(userID string, now time.Time) ([]Notification, error)` —
    джойн `notifications`/`notification_recipients`, фильтр по ретеншену
    выше, `ORDER BY created_at DESC LIMIT 50`.
  - `MarkNotificationsRead(userID string, now time.Time) error` —
    `UPDATE notification_recipients SET read_at = ? WHERE user_id = ? AND read_at = ''`.
  - Функции создания уведомления в Go **не нужны** — пишет только админка
    напрямую в БД, serbian-app только читает.
- `server/internal/api/notifications.go` (по образцу `support.go`):
  - `GET /api/me/notifications` (`requireSession`) → `{items: [{id, text, created_at, read}], unread_count}`.
  - `POST /api/me/notifications/mark-read` (`requireSession`), без тела →
    помечает все непрочитанные пользователя прочитанными, `{status: "ok"}`.
  - Регистрация в `api.go` рядом с остальными `/api/me*`-роутами.

## Эндпоинты и Frontend (ucimo-content-admin)

- Новый модуль `server/src/notifications/` (по образцу `broadcasts/`):
  - `GET /notifications/candidates` — `SELECT id, name FROM users ORDER BY name`
    через `ProdDbService` (**все** пользователи, не только с Telegram —
    в отличие от `broadcasts/candidates`).
  - `POST /notifications` `{recipients: {mode:'all'}|{mode:'users', userIds}, text, dryRun?}`:
    `dryRun` — только считает получателей; иначе — `INSERT` одной
    `notifications`-строки + batch `INSERT` в `notification_recipients`
    (в одной транзакции). Пустой/только-пробельный `text` → 400.
- Новый раздел сайдбара **«Уведомления»** — отдельно от «Бот» (не про
  Telegram).
- `NotificationsView.vue` — копия UX `BroadcastsView.vue`: поиск +
  чекбоксы получателей / переключатель «Всем пользователям (N)»,
  textarea с текстом, кнопка «Отправить» → dry-run → модалка
  подтверждения → реальный `POST`. Под textarea — подсказка:
  > Чтобы добавить ссылку: `[текст ссылки](https://...)`
- Без истории отправленных уведомлений / статистики прочтений / удаления —
  YAGNI, как и с рассылками.

## Frontend (serbian-app)

- `web/src/lib/notifications.ts` — composable по образцу
  `lib/telegramStart.ts`: `setInterval` каждые 45с вызывает
  `GET /api/me/notifications`, хранит `items`/`unreadCount` в `ref`,
  отдаёт `refresh()` и `markRead()`; `clearInterval` при unmount.
- `web/src/components/NotificationBell.vue` — иконка-кнопка в ряду
  иконок `AppNav.vue` (рядом с переключателем темы/поддержкой/профилем).
  Колокольчик (`lucide-vue-next` `Bell`) + точка `background: var(--accent)`
  при `unreadCount > 0`. Клик открывает дропдаун-карточку (стиль как у
  существующих дропдаунов/модалок); при открытии сразу вызывает
  `markRead()` — весь видимый список помечается прочитанным разом (не по
  клику на каждое). Непрочитанные элементы до пометки — фон
  `var(--accent-soft)`.
- Рендер текста — **не** через `MarkdownView.vue`/полный `markdown-it`
  (это дало бы заголовки/списки/таблицы, чего мы осознанно не хотим).
  Локальный хелпер `renderNotificationText(text: string): string` в
  `lib/notifications.ts`: экранирует HTML, затем regex на `[текст](url)`
  (`url` обязан начинаться с `http://`, `https://` или `/` — защита от
  `javascript:`-вставок), заменяет на `<a target="_blank" rel="noopener">`,
  оставшиеся `\n` → `<br>`. Рендерится через `v-html` — безопасно, так как
  вся HTML-генерация проходит через этот единственный контролируемый
  хелпер, а не произвольный markdown.

## Тестирование

- **serbian-app**:
  - `internal/store/notifications_test.go` — `ListNotificationsForUser`
    возвращает только строки пользователя, фильтрует по ретеншену
    (непрочитанное — всегда, прочитанное >30 дней назад — нет);
    `MarkNotificationsRead` помечает только непрочитанные этого
    пользователя, идемпотентен.
  - `internal/api` — `notifications_test.go`: `GET` без сессии → 401;
    `GET` возвращает `unread_count`, совпадающий с числом `read: false`;
    `POST mark-read` переводит все в `read: true` при следующем `GET`.
  - `go test ./server/...` на SQLite, `make test-pg` — миграция 010
    накатывается на оба диалекта.
- **ucimo-content-admin**: `notifications`-модуль тестами как у
  `broadcasts` — `candidates` возвращает всех пользователей; `dryRun`
  считает получателей и не пишет строк; `mode:'all'`/`mode:'users'`
  создают правильное число строк `notification_recipients` с
  `read_at=''`; пустой текст → 400.
- **Frontend**: компонентный тест `NotificationBell.test.ts` (по образцу
  существующих `*.test.ts` во `views`/`components`) — точка показывается
  при `unread_count > 0`; открытие дропдауна вызывает `mark-read`;
  `renderNotificationText` — юнит-тест на экранирование HTML и на то, что
  `javascript:`-ссылка не превращается в `<a href>`.

## Вне скоупа

- WebSocket — поллинг остаётся до отдельной задачи на это.
- Отчёт "кто прочитал" / история отправленных уведомлений в админке.
- Редактирование и удаление уже отправленного уведомления.
- Пометка прочитанным по клику на отдельное уведомление (вместо всего
  списка при открытии дропдауна).
- Пагинация дальше последних 50 уведомлений.
- Полный markdown (жирный/списки/заголовки) — только ссылки.
- Реальное удаление строк старше ретеншена — только фильтрация на чтении.
