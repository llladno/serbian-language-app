# serbian-app — заметки для Claude

Веб-приложение для изучения сербского с нуля (курс Гриши). Go-бэкенд +
Vue-фронт в одном бинаре, контент — файлы в `content/`, состояние — PostgreSQL.

## Архитектура

- **`server/`** — Go 1.25, stdlib `net/http`. Собирает фронт в бинарь через
  `go:embed` (`server/web/dist`).
  - `internal/content` — грузит и валидирует `content/` (уроки, упражнения,
    словарь, ложные друзья). Hot-reload через fsnotify.
  - `internal/store` — персистентность (аккаунты, SRS, попытки, прогресс).
  - `internal/srs` — планировщик повторений (SM-2).
  - `internal/checker` — нормализация и проверка ответов.
  - `internal/api` — JSON API (`/api/...`), заголовок `X-User` = имя аккаунта.
- **`web/`** — Vue 3 + Vite + Pinia + Tailwind. Роуты: курс, урок, упражнения,
  повторение (SRS), словарь, ложные друзья, прогресс.
- **`content/`** — единственный источник правды по учебному материалу
  (см. `README.md` → «Контент»). Уроки `.md`, упражнения `.yaml`,
  `vocab.yaml`, `false-friends.yaml`, картинки `images/`, озвучка `audio/`.

## База данных — PostgreSQL (прод) / SQLite (локально)

**Один код, два бэкенда.** `store.Open(dsn)`:

- `dsn` начинается с `postgres://` / `postgresql://` → **PostgreSQL** (драйвер
  `pgx/v5/stdlib`).
- иначе → **SQLite** (`modernc.org/sqlite`, чистый Go), `dsn` = путь к файлу
  или `:memory:`.

Запросы пишутся с плейсхолдерами `?`; для Postgres они переписываются в `$N`
функцией `rebind` в `store.go`. Схема — `schemaSQL(pg bool)`, отличие только в
колонке автоинкремента. Idempotent `CREATE TABLE IF NOT EXISTS`, миграций
руками нет. Путь `v1→v2` (старые беспарольные таблицы) — только SQLite,
легаси.

### Прод (Dokploy)

Отдельный сервис **PostgreSQL** в проекте, приложение получает
`DATABASE_URL` через env. Редеплой приложения базу не трогает. Полностью —
`docs/DEPLOY.md`. Перенос старой SQLite-базы: флаг `-import-sqlite <path>`
(разово, срабатывает только на пустой базе).

### Локально

По умолчанию SQLite-файл `./data/app.db` — ничего ставить не нужно.
`make dev` поднимает фронт (5173) и бэк (8080).

### Тесты

- `make test` / `go test ./server/...` — стор гоняется на SQLite in-memory.
- Против настоящего Postgres: `TEST_DATABASE_URL='postgres://localhost/srpski_test?sslmode=disable' go test ./server/internal/store/`
  (таблицы `TRUNCATE` между тестами). `make test-pg` — то же.
- Локальный тест-Postgres: `createdb srpski_test` (нужен запущенный
  `postgresql@17` из brew) или свой docker.

## Команды

```bash
make dev        # фронт :5173 (проксирует /api) + бэк :8080, SQLite
make test       # go + vitest (SQLite)
make test-pg    # go-тесты стора против TEST_DATABASE_URL
make build      # npm ci + vite build + go build -> ./serbian-app
```

Озвучка новых слов: `python3 scripts/tts.py` (edge-tts, `sr-RS-SophieNeural`,
без ключа) — дописывает недостающие `content/audio/*.mp3`.

## Деплой

Auto Deploy включён: push в `main` → Dokploy пересобирает образ. Контент и
озвучка внутри образа, база — снаружи. Подробности и перенос данных —
`docs/DEPLOY.md`.

## Добавить урок

`README.md` → «Добавить урок». Кратко: `content/course.yaml` (`file:`),
`content/lessons/NN-slug.md`, `content/exercises/NN.yaml`, слова в
`content/vocab.yaml`, `python3 scripts/tts.py`. Гард-тест —
`server/internal/content/real_test.go`.

### Текст для чтения в уроке

В `.md` урока можно добавить блок между маркерами `<!-- reading -->` и
`<!-- /reading -->` (каждый на своей строке). Строка `---` внутри отделяет
сербский текст от русского перевода. Загрузчик (`content/reading.go`)
вырезает блок из markdown в поля `Lesson.Reading` / `Lesson.ReadingRU`;
фронт рендерит его карточкой «📖 Текст для чтения».

Клик по слову → карточка из словаря — общий компонент `GlossedText.vue`
(токенайзер `lib/reading.ts`, общий кэш `lib/lookup.ts`, эндпоинт
`/api/lookup?q=`). Используется в тексте для чтения и в промптах упражнений
`fill_blank` / `fix_error` (`TextAnswer.vue`). Промпты `translate` (русские,
и ответ спойлить нельзя) и `free` — без глоссов.

## Стиль

- Go: как в соседних файлах, короткие доки-комментарии на экспортных
  символах, ошибки оборачивай (`fmt.Errorf("...: %w")`).
- Комментарии в коде — по-английски, тексты для пользователя и контент —
  по-русски/сербски.
- Тесты рядом с кодом, table-driven где уместно.
