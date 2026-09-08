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
  (см. `README.md` → «Контент»). Уроки — манифесты `lessons/NN.yaml`
  (шаги) + фрагменты `lessons/NN/*.md`; легаси `lessons/NN-*.md` +
  `exercises/NN.yaml` ещё поддерживается. Плюс `vocab.yaml`,
  `allow-words.yaml`, `persona.yaml`, `false-friends.yaml`, `images/`,
  `audio/`.

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

Озвучка: `python3 scripts/tts.py` (edge-tts, `sr-RS-SophieNeural`, без ключа) —
дописывает недостающие `content/audio/*.mp3` для слов из `vocab.yaml` **и** для
упражнений `type: listen` (файл по id упражнения, напр. `01-E-1.mp3`).

## Деплой

Auto Deploy включён: push в `main` → Dokploy пересобирает образ. Контент и
озвучка внутри образа, база — снаружи. Подробности и перенос данных —
`docs/DEPLOY.md`.

## Модель урока: манифест + шаги

У урока две модели, `content/load.go` выбирает по расширению `file:` в
`course.yaml`:

- **манифест** (`file: "lessons/NN.yaml"`) — новая. Урок = упорядоченный
  список **шагов** (`teach` / `practice` / `reading` / `checkpoint`).
  Теория — фрагменты `content/lessons/NN/*.md`, упражнения — инлайн в
  шаге. Пример со всеми видами — `content/lessons/_TEMPLATE.yaml`.
- **легаси** (`file: "lessons/NN-slug.md"` + `content/exercises/NN.yaml`)
  — старая. Загрузчик синтезирует из неё шаги (teach → practice-блоки →
  reading), поэтому фронт-плеер работает одинаково. В реальном `content/`
  легаси-уроков не осталось — блоки 1–2 (уроки 01–12) целиком манифесты;
  синтез шагов покрыт тестом на фикстуре (`testdata/content`).

Прогресс — по шагам (`store.lesson_step_progress`), экран урока —
пошаговый плеер (`web/src/views/LessonView.vue`).

## Добавить урок (манифест)

1. `content/course.yaml` — запись урока с `file: "lessons/NN.yaml"`.
2. `content/lessons/NN.yaml` — манифест (скопируй `_TEMPLATE.yaml`).
3. `content/lessons/NN/*.md` — фрагменты теории для `teach`/`reading`.
4. Новые слова → `content/vocab.yaml` (`lesson: "NN"`), продублируй их
   id в `teaches:` манифеста.
5. Падежные формы, имена → `also_ok:` конкретного шага; общие
   имена/числа/частицы → `content/allow-words.yaml`.
6. `python3 scripts/tts.py` — озвучка новых слов и `listen`-упражнений
   (файл по id упражнения, напр. `NN.8.1.mp3`).

**Гард-тесты** (`server/internal/content/`):
- `real_test.go` — структура уроков 01–05, наличие reading/checkpoint,
  аудио для `listen`.
- `lexicon_test.go` — в манифест-уроке `accept` / `options` / `bank` /
  сербская сторона `pairs` / `say` используют только слова из
  накопительного словаря к этому уроку (+ `teaches` / `also_ok` /
  `allow-words`). Легаси-уроки — только предупреждения.
- порядок сложности в `practice`-шаге не убывает (иначе `mixed: true`).

### Текст для чтения

Манифест: шаг `kind: reading`, `md:` на фрагмент, где строка `---`
делит сербский текст и русский перевод. Легаси: блок между маркерами
`<!-- reading -->` / `<!-- /reading -->` в `.md` урока. И то, и другое
попадает в `Step.Markdown` / `Step.MarkdownRU` (и, для совместимости, в
`Lesson.Reading` / `Lesson.ReadingRU`).

Клик по слову → карточка из словаря — общий компонент `GlossedText.vue`
(токенайзер `lib/reading.ts`, общий кэш `lib/lookup.ts`, эндпоинт
`/api/lookup?q=`). Используется в тексте для чтения и в промптах упражнений
`fill_blank` / `fix_error` (`TextAnswer.vue`). Промпты `translate` (русские,
и ответ спойлить нельзя) и `free` — без глоссов. В карточке слова — кнопка
«＋ в повторение» (`POST /api/review/add {vocab_id}` → `store.ActivateCard`:
переводит `new`-карточку в `learning` due-сегодня, мимо дневного лимита новых;
`lib/review.ts` держит добавленные id в рамках сессии).

### Типы упражнений

`translate`, `fill_blank`, `fix_error`, `conjugate`, `free`, `listen`
(диктант — `say:` синтезируется, `accept:` печатают), плюс лёгкие:
`choice` (`options` + `answer`), `word_bank` (`bank` из фишек +
`accept`), `match` (`pairs` [sr, ru]). Клиенту НИКОГДА не уходят
`accept` / `answer` / соответствие `pairs` / `say` — только
`options` / `bank` / `left` / `right`, перемешиваются на клиенте.
Проверка — `POST /api/lessons/{id}/exercises/{exId}/check`
(`checker.CheckChoice` / `CheckMatch` / `Check`). Компоненты —
`web/src/components/exercises/*Answer.vue`.

Аудио `listen` — `scripts/tts.py` кладёт `content/audio/<id>.mp3`
(id упражнения) из кириллической транслитерации `say`, коммитится в
репо. Скрипт сканирует и `exercises/*.yaml`, и `lessons/*.yaml`.

### Личный слой

`content/persona.yaml` (`name`, `city`, `job`, …). Загрузчик подставляет
`{ключ}` в теорию, промпты и `sample` — но **не** в ответы. Отсутствие
файла — no-op.

## Стиль

- Go: как в соседних файлах, короткие доки-комментарии на экспортных
  символах, ошибки оборачивай (`fmt.Errorf("...: %w")`).
- Комментарии в коде — по-английски, тексты для пользователя и контент —
  по-русски/сербски.
- Тесты рядом с кодом, table-driven где уместно.
