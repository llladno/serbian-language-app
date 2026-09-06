# Srpski App — дизайн

Персональное веб-приложение для изучения сербского. Переносит курс из
Obsidian (`/Users/grisha/plans/serbian/`) в формат приложения: читалка
уроков, интерактивные упражнения с автопроверкой, тренажёр слов на
интервальном повторении (SRS), дашборд прогресса.

- **Стек:** Go (бэкенд, JSON API + отдача статики) + Vue 3 / Vite /
  TypeScript (SPA).
- **Пользователь:** один (Гриша). Без авторизации.
- **Где работает:** локально на Mac. Архитектура держит в уме будущий
  Telegram mini-app (чистый JSON API, адаптивная вёрстка) — но сам TG
  mini-app вне этой задачи.
- **Репозиторий:** `/Users/grisha/plans/serbian-app/` (новый git-репо).

## Цели и не-цели

**Цели:**

1. Все 4 режима: читалка уроков, упражнения, SRS-тренажёр слов, дашборд.
2. Контент — структурные файлы в репо (`content/`), автор — Клод правит
   их в чате. Никакого редактора контента в UI.
3. Изменяемое состояние (SRS, попытки, прохождение) — в SQLite.
4. Один бинарь на проде (`go:embed` собранного фронта), запуск одной
   командой.
5. Миграция уже существующего контента: уроки 01–02, `vocab.md`,
   `false-friends.md`.

**Не-цели (сейчас):**

- Авторизация, мультипользовательность, хостинг в интернете.
- Telegram mini-app.
- Редактор контента в интерфейсе.
- Аудио/произношение, TTS.
- Написание новых уроков (03+) — добавляются позже правкой файлов.
- Синхронизация обратно в Obsidian `.md`.

## Архитектура

```
serbian-app/
  content/                     # источник правды по контенту (git)
    course.yaml
    lessons/
      01-pozdravi-i-upoznavanje.md
      02-zamenice-i-prezent.md
    exercises/
      01.yaml
      02.yaml
    vocab.yaml
    false-friends.yaml
  server/                      # Go
    main.go
    internal/content/          # загрузка и парсинг content/
    internal/store/            # SQLite: srs, attempts, progress
    internal/srs/              # алгоритм SM-2
    internal/api/              # HTTP-хендлеры
    internal/checker/          # нормализация и сверка ответов
    web/embed.go               # go:embed dist/
  web/                         # Vue SPA
    src/
    ...
  data/
    app.db                     # SQLite (gitignore)
  Makefile
  README.md
```

Поток данных:

- При старте сервер читает `content/` целиком в память
  (`content.Course`). Фон: fsnotify-вотчер на `content/` перечитывает
  при изменении файла (правки без перезапуска). Ошибка парсинга не
  роняет сервер — логируется, держится последняя валидная версия.
- Фронт запрашивает контент через `/api/*`, рендерит.
- Действия пользователя (ответ на упражнение, оценка карты, отметка
  урока) → `POST /api/*` → запись в SQLite.
- Дашборд читает агрегаты из SQLite + метаданные курса.

## Модель контента

### `course.yaml`

```yaml
title: "Српски језик"
phases:
  - id: A
    title: "Фаза A — Выживание"
    lessons: ["01", "02", "03", "04", "05", "06", "07", "08", "09"]
  - id: B
    title: "Фаза B — Грамматика в деле"
    lessons: ["10", ...]
  - id: C
    title: "Фаза C — Жизнь и работа"
    lessons: ["21", ...]
lessons:
  "01":
    title: "Pozdravi i upoznavanje"
    subtitle: "приветствия, знакомство, глагол jesam, ты/вы"
    file: "lessons/01-pozdravi-i-upoznavanje.md"
    status_default: "available"     # available | locked (пока все available)
```

Урок, которого нет в файле (03–30), в API отдаётся как «запланирован»
из списка `phases[].lessons` + описания из `plan.md` (перенесём
названия в `course.yaml`). Такой урок открывается, но показывает
«скоро».

### `lessons/NN-*.md`

Чистый markdown: заголовки, таблицы, цитаты, диалоги. **Без** блока
`## Упражнения` и `## Ответы` / `<details>` — упражнения переезжают в
`exercises/NN.yaml`. Рендерится на фронте через `markdown-it`
(+ `markdown-it` таблицы включены по умолчанию).

Первая строка H1 игнорируется при рендере (заголовок берётся из
`course.yaml`).

### `exercises/NN.yaml`

```yaml
lesson: "01"
blocks:
  - id: "A"
    title: "Переведи на сербский (латиница)"
    instruction: "Пиши латиницей."
    exercises:
      - id: "01-A-1"
        type: translate
        prompt: "Привет! Как дела?"
        accept:
          - "Zdravo! Kako si?"
          - "Ćao! Kako si?"
        explain: "«Ćao» — неформально, тоже ок."
      - id: "01-A-7"
        type: translate
        prompt: "Я не устал. (муж. род)"
        accept: ["Nisam umoran."]
  - id: "B"
    title: "Вставь форму глагола jesam"
    exercises:
      - id: "01-B-1"
        type: fill_blank
        prompt: "Ja ___ iz Rusije."
        accept: ["sam"]
  - id: "C"
    title: "Исправь ошибку"
    exercises:
      - id: "01-C-1"
        type: fix_error
        prompt: "Sam iz Rusije."
        accept: ["Ja sam iz Rusije.", "Iz Rusije sam."]
  - id: "D"
    title: "Мини-ситуация"
    exercises:
      - id: "01-D-1"
        type: free
        prompt: "Ты заходишь в пекару утром. Поздоровайся, на выходе — попрощайся."
        sample: "Dobro jutro! … Hvala, prijatno! / Doviđenja!"
```

**Типы упражнений:**

| type | ввод | автопроверка | как показывается |
|------|------|--------------|------------------|
| `translate` | текст | да (`accept`) | prompt (ru) → поле ввода |
| `fill_blank` | текст | да | prompt с `___` → поле |
| `fix_error` | текст | да | неверная фраза → поле |
| `conjugate` | 6 полей | да (`accept` — список из 6, поле за полем) | инфинитив → 6 полей (ja/ti/on/mi/vi/oni) |
| `free` | textarea | нет | prompt → textarea → кнопка «показать образец» → «справился / нет» |

`conjugate` формат:

```yaml
- id: "02-A-1"
  type: conjugate
  prompt: "govoriti"
  forms: ["ja", "ti", "on/ona/ono", "mi", "vi", "oni/one/ona"]
  accept: [["govorim"], ["govoriš"], ["govori"], ["govorimo"], ["govorite"], ["govore"]]
  meta: "тип I"
```

### `vocab.yaml`

```yaml
- id: "zdravo"
  latin: "zdravo"
  cyrillic: "здраво"
  ru: "привет / здравствуйте"
  note: "нейтрально, на все случаи"
  lesson: "01"
  tags: ["greetings"]
  pos: "interj"          # опционально
  gender: null           # m | f | n | null
  aspect: null           # sv | nesv | null
```

`id` — уникальный слаг (обычно = `latin`, при коллизии с суффиксом).

### `false-friends.yaml`

```yaml
- id: "pravo"
  sr: "pravo"
  means: "прямо / право (юр.)"
  not: "направо"
  correct: "desno"       # как сказать «то самое», опционально
  group: "top"           # top | shop | small
```

## Бэкенд (Go)

- Модуль: `github.com/grisha/serbian-app` (локальный, не публикуется).
- Go ≥ 1.22 (роутинг `http.ServeMux` с методами и путевыми
  параметрами).
- Зависимости:
  - `gopkg.in/yaml.v3` — парсинг контента.
  - `modernc.org/sqlite` — SQLite без CGO.
  - `github.com/fsnotify/fsnotify` — вотчер контента.
  - Больше ничего. Markdown → HTML делает фронт.

### Пакет `internal/content`

- `Load(dir string) (*Course, error)` — читает всё, валидирует
  (каждый `exercises/NN.yaml` ссылается на существующий урок; `accept`
  непустой для авто-типов; `vocab` id уникальны).
- `Course` держит: фазы, уроки (+ сырой markdown), упражнения по
  урокам, словарь, ложные друзья, индексы по id.
- `Watcher(dir, onReload)` — дебаунс 300 мс, перечитывает, при успехе
  атомарно подменяет указатель (`atomic.Pointer[Course]`).

### Пакет `internal/checker`

- `Normalize(s string) string` — trim, схлопнуть пробелы, нижний
  регистр (Unicode), убрать финальную `.`/`!`/`?`, нормализовать
  типографские кавычки/дефисы. Латиницу и кириллицу **не**
  конвертируем между собой (ответы в `accept` пишем латиницей —
  как в уроках).
- `Check(answer string, accept []string) Result` — `Result{OK bool,
  Best string, Diff []DiffChunk}`. `Diff` — пословный diff ответа с
  ближайшим (по расстоянию Левенштейна) вариантом из `accept`, для
  подсветки на фронте.
- Для `conjugate` — проверка поле-за-полем, `Result` на каждое.

### Пакет `internal/srs`

Алгоритм **SM-2** (упрощённый, как в Anki-lite):

- Карта: `ease float64` (старт 2.5, мин 1.3), `interval_days int`,
  `reps int`, `lapses int`, `due date`, `state` (`new`|`learning`|`review`).
- Оценки: `again` (0), `hard` (1), `good` (2), `easy` (3).
- Новая карта: `again` → повтор сегодня (10 мин, в рамках сессии),
  `good` → 1 день, `easy` → 4 дня.
- Review-карта:
  - `again` → `lapses++`, `ease -= 0.2`, `interval = 1`,
    `state = learning`.
  - `hard` → `ease -= 0.15`, `interval = round(interval * 1.2)`.
  - `good` → `interval = round(interval * ease)`.
  - `easy` → `ease += 0.15`, `interval = round(interval * ease * 1.3)`.
- `interval` кэпается 365 днями.
- `Schedule(card Card, grade Grade, now time.Time) Card` — чистая
  функция, полностью покрыта тестами.

Карты создаются лениво: при первом обращении к `/api/review/queue`
сервер сверяет `vocab.yaml` + `false-friends.yaml` с таблицей
`srs_cards` и заводит недостающие как `new`.

Очередь на день: все `due <= today` (review + learning) +
до `N` новых карт (по умолчанию `N = 15`, настройка в
`data/config` не нужна — константа). Порядок: сперва просроченные,
потом новые, лёгкое перемешивание.

### Пакет `internal/store` (SQLite)

Схема (миграции — простые `CREATE TABLE IF NOT EXISTS` при старте,
версия схемы в `PRAGMA user_version`):

```sql
srs_cards (
  card_id TEXT PRIMARY KEY,        -- "vocab:zdravo" | "ff:pravo"
  kind TEXT NOT NULL,              -- vocab | ff
  ref_id TEXT NOT NULL,
  ease REAL NOT NULL DEFAULT 2.5,
  interval_days INTEGER NOT NULL DEFAULT 0,
  reps INTEGER NOT NULL DEFAULT 0,
  lapses INTEGER NOT NULL DEFAULT 0,
  state TEXT NOT NULL DEFAULT 'new',
  due TEXT,                        -- ISO date, NULL для new
  updated_at TEXT NOT NULL
);

reviews (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  card_id TEXT NOT NULL,
  grade INTEGER NOT NULL,
  reviewed_at TEXT NOT NULL
);

attempts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  exercise_id TEXT NOT NULL,
  lesson TEXT NOT NULL,
  block TEXT NOT NULL,
  answer TEXT NOT NULL,
  correct INTEGER NOT NULL,        -- 0 | 1  (для free — самооценка)
  attempted_at TEXT NOT NULL
);

lesson_progress (
  lesson TEXT PRIMARY KEY,
  status TEXT NOT NULL,            -- in_progress | done
  started_at TEXT,
  completed_at TEXT
);
```

### HTTP API

Все ответы — JSON, префикс `/api`. Ошибки: `{ "error": "..." }` +
статус.

| Метод + путь | Назначение |
|---|---|
| `GET /api/course` | фазы, уроки с метаданными и статусом (join с `lesson_progress`) |
| `GET /api/lessons/{id}` | markdown теории + метаданные + статус |
| `GET /api/lessons/{id}/exercises` | блоки упражнений (без `accept` — не светим ответы клиенту) |
| `POST /api/lessons/{id}/exercises/{exId}/check` | `{answer}` → `{ok, diff, explain, expected, sample?}`; пишет `attempts` |
| `POST /api/lessons/{id}/complete` | статус урока → `done` |
| `GET /api/vocab` | весь словарь (query: `?lesson=`, `?tag=`, `?q=`) |
| `GET /api/false-friends` | список (query: `?group=`, `?q=`) |
| `GET /api/review/queue` | карты на сегодня (полные данные слова/ложного друга) |
| `POST /api/review/grade` | `{card_id, grade}` → следующий `due`; пишет `reviews` |
| `GET /api/progress` | агрегаты: по фазам, слабые упражнения, статистика SRS, серия дней |

**Важно:** список `accept` не уходит на клиент в payload упражнений
(нельзя подсмотреть все ответы разом). Проверка — только на сервере
(`/check`). В ответе `/check` сервер возвращает `expected` —
ближайший правильный вариант (для показа после попытки) — и `sample`
для `free`.

`GET /api/progress` отдаёт:

```json
{
  "phases": [{ "id": "A", "done": 1, "total": 9 }],
  "srs": { "due_today": 12, "new_available": 15, "reviewed_today": 8,
           "total_cards": 74, "known": 40 },
  "weak_exercises": [
    { "exercise_id": "02-C-1", "lesson": "02", "wrong": 3, "total": 4,
      "prompt": "..." }
  ],
  "streak_days": 3,
  "recent_lessons": [{ "lesson": "01", "status": "done" }]
}
```

«Слабые» — упражнения с долей неверных ≥ 0.5 при ≥ 2 попытках, топ-10
по числу ошибок.

### Отдача фронта

- `web/embed.go`: `//go:embed all:dist` → `fs.FS`.
- Прод: `http.FileServer` с SPA-fallback (неизвестный путь без
  расширения → `index.html`).
- Дев: если `dist` пустой (сборки нет) — сервер логирует подсказку
  «запусти vite», API работает.
- Флаги: `-addr :8080`, `-content ./content`, `-db ./data/app.db`.

## Фронтенд (Vue)

- Vue 3 `<script setup>` + TypeScript, Vite, Vue Router, Pinia.
- Tailwind CSS (адаптив, тёмная тема по `prefers-color-scheme`).
- `markdown-it` (таблицы, `linkify` выкл.) для теории.
- Фетч через тонкую обёртку `src/api.ts` (типизированные функции).
- Никакого SSR.

### Роуты и экраны

| Роут | Экран | Содержимое |
|---|---|---|
| `/` | Дашборд | карточка «Повторить слова» (N due), прогресс по фазам (полоски), слабые места, «продолжить урок», серия дней |
| `/course` | Курс | фазы → уроки, статус-бейджи, клик → урок |
| `/lesson/:id` | Урок | теория (markdown) + блоки упражнений внизу; для запланированных — «скоро» |
| `/review` | SRS-тренажёр | колода: лицо (слово или ru) → «показать» → 4 кнопки оценки; прогресс сессии; горячие клавиши 1–4, Space |
| `/vocab` | Словарь | поиск, фильтр по уроку/тегу, тумблер latin/кириллица, «учить выборку» → режим флешкарт |
| `/false-friends` | Ложные друзья | таблица с фильтром по группе и поиском |

### Компоненты упражнений

- `ExerciseBlock.vue` — заголовок + инструкция + список.
- `ExerciseItem.vue` — диспетчер по `type`.
  - `translate` / `fill_blank` / `fix_error` — общий `TextAnswer.vue`:
    поле, Enter = проверить, показывает ✓ / ✗ + подсветку diff +
    `explain`, кнопка «ещё раз».
  - `conjugate` — `ConjugateAnswer.vue`: 6 подписанных полей, проверка
    по кнопке, ✓/✗ на каждое.
  - `free` — `FreeAnswer.vue`: textarea, «показать образец», затем
    «справился / не справился» (пишет `attempt`).
- Состояние ответов — локально в компоненте; результат уходит на
  сервер через `/check`. Блок показывает счёт `N / M`.

### SRS-тренажёр (`/review`)

- Грузит очередь один раз (`GET /api/review/queue`).
- Карта показывает `latin` (+ кириллица мелким), по «показать» —
  `ru` + `note`. Направление фиксированное sr→ru для v1
  (проще; ru→sr даёт тренажёр в словаре).
- 4 кнопки: «Опять / Трудно / Хорошо / Легко» → `POST
  /api/review/grade` → следующая карта. Карту с `again` возвращаем в
  конец текущей сессии.
- Конец очереди — экран «на сегодня всё», ссылка на дашборд.

### Стор (Pinia)

- `useCourse` — кэш `/api/course`, инвалидация после `complete`.
- `useReview` — очередь, индекс, счётчики сессии.
- Прочее — по запросу, без глобального стора.

## Обработка ошибок

- Бэкенд: ошибка парсинга контента при старте → сервер не стартует
  с понятным сообщением (какой файл, строка). При hot-reload → лог +
  держим прошлую версию, флаг в `/api/health` (`content_stale: true`).
- Неизвестный `lesson`/`exId` → 404 JSON.
- Фронт: обёртка `api.ts` кидает `ApiError`; экраны показывают
  инлайн-баннер «не удалось загрузить, повторить».
- SQLite недоступен (заблокирован) → 503; фронт предлагает повтор.

## Тестирование

**Go (`go test ./...`):**

- `checker`: нормализация (регистр, пробелы, пунктуация, кавычки),
  `Check` на точный / вариативный / неверный ответ, diff-чанки,
  `conjugate` поле-за-полем.
- `srs`: `Schedule` — таблица кейсов на все переходы состояний и
  оценок; кэп интервала; пол ease.
- `content`: загрузка фикстур (мини-`content/` в `testdata/`),
  валидация (битый `accept`, дубль id, ссылка на несуществующий урок).
- `api`: `httptest` — ключевые эндпоинты на фикстурах + временной
  SQLite (`:memory:` или tmp-файл): `/check` пишет attempt,
  `/review/grade` двигает `due`, `/progress` считает агрегаты.

**Фронт (`vitest`):**

- `api.ts` — разбор ответов и ошибок (мок fetch).
- diff-подсветка (компонент `TextAnswer`) — рендер ✓/✗ по пропсам.
- `useReview` — переход по очереди, возврат `again`-карты в конец.

TDD: тест → реализация, по каждому пакету.

## Запуск

`Makefile`:

- `make dev` — параллельно `vite` (:5173, прокси `/api` → :8080) и
  `go run ./server -addr :8080`. (через `npx concurrently` или два
  фоновых процесса + trap).
- `make build` — `cd web && npm run build` → `web/dist`, копия в
  `server/web/dist`, затем `go build -o ../serbian-app ./server`.
- `make test` — `go test ./...` + `cd web && npm run test`.
- `make seed` — не нужен (карты заводятся лениво).

`README.md`: как запустить, где контент, как добавить урок
(шаблон `exercises/NN.yaml`).

`.gitignore`: `data/`, `web/dist/`, `server/web/dist/`, `node_modules/`,
`serbian-app` (бинарь).

## Миграция существующего контента

Часть этой задачи — перенести:

- `serbian/lessons/01-*.md` → `content/lessons/01-*.md` (убрать блоки
  «Упражнения» и «Ответы») + `content/exercises/01.yaml` (блоки A/B/C/D
  из урока, `accept` из раздела «Ответы»).
- `serbian/lessons/02-*.md` → аналогично + `content/exercises/02.yaml`
  (блоки A–E).
- `serbian/vocab.md` → `content/vocab.yaml` (уроки 01–02, ~80 слов, с
  пометками рода/вида/типа спряжения в `note`/`tags`).
- `serbian/reference/false-friends.md` → `content/false-friends.yaml`
  (группы top / shop / small).
- `serbian/plan.md` → названия и подзаголовки уроков 03–30 в
  `course.yaml` (как «запланировано»).

Исходную папку `serbian/` не трогаем (останется как есть); в её
`README.md` добавим строку-указатель на приложение.

## Порядок реализации (для плана)

1. Скелет репо: `go.mod`, `web/` (Vite+Vue+TS+Tailwind), `Makefile`,
   `.gitignore`, `README`.
2. `content` пакет + фикстуры + миграция реального контента в
   `content/`.
3. `checker` пакет (TDD).
4. `srs` пакет (TDD).
5. `store` пакет + схема (TDD).
6. `api` пакет + сервер + embed (TDD через httptest).
7. Фронт: обёртка `api.ts`, роутер, layout.
8. Экраны: Курс → Урок (+ компоненты упражнений) → SRS → Словарь →
   Ложные друзья → Дашборд.
9. `make dev` / `make build` / `make test`, прогон, README.
```
