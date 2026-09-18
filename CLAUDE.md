# serbian-app (ucimo) — заметки для Claude

Персональное веб-приложение для изучения сербского с нуля (курс Гриши, бренд
«ucimo»). Go-бэкенд + Vue-фронт в одном бинаре, контент — файлы в `content/`,
состояние (аккаунты, SRS, попытки, прогресс) — PostgreSQL/SQLite. Плюс
отдельный статический лендинг на Nuxt (`landing/`), встроенный в тот же
бинарь и отданный с того же домена.

## Архитектура

- **`server/`** — Go 1.25, stdlib `net/http`. Собирает оба фронта в бинарь
  через `go:embed` (`server/web/dist`, `server/landing/dist`).
  - `internal/content` — грузит и валидирует `content/` (уроки, упражнения,
    словарь, ложные друзья). Hot-reload через fsnotify.
  - `internal/store` — персистентность (аккаунты, identities/сессии, SRS,
    попытки, прогресс).
  - `internal/auth` — примитивы: id/токены, bcrypt-хеши паролей, Telegram
    login-widget/`/start`-подпись, pending-store для `/start`-флоу.
  - `internal/srs` — планировщик повторений (SM-2).
  - `internal/checker` — нормализация и проверка ответов.
  - `internal/mail`, `internal/ratelimit` — почта (верификация/сброс пароля),
    троттлинг логина/регистрации.
  - `internal/api` — JSON API (`/api/...`), см. «Аутентификация» ниже.
  - `main.go` — на `/` сперва пробует статику Nuxt-лендинга (`landing.FS()`,
    напр. `/`, `/privacy`, `/robots.txt`), иначе — Vue SPA
    (`spaHandler(web.FS())`, отдаёт `index.html` для клиентских роутов).
- **`web/`** — Vue 3 + Vite + Pinia + Tailwind. Сам таргетируемый app:
  логин/регистрация, курс, урок (пошаговый плеер), повторение (SRS),
  словарь (табы «Слова»/«Ловушки», один экран — `VocabView.vue`),
  профиль/рейтинг.
- **`landing/`** — Nuxt 4, отдельный `package.json`/`npm run build`, чисто
  статический маркетинговый сайт (SEO/GEO, аналитика, `/privacy`, `/terms`).
  Не трогает `/api` и не завязан на аутентификацию.
- **`content/`** — единственный источник правды по учебному материалу
  (см. `README.md` → «Контент»). Уроки — манифесты `lessons/NN.yaml`
  (шаги) + фрагменты `lessons/NN/*.md`; легаси `lessons/NN-*.md` +
  `exercises/NN.yaml` ещё поддерживается. Плюс `vocab.yaml`,
  `allow-words.yaml`, `persona.yaml`, `false-friends.yaml`, `images/`,
  `audio/`.

## Аутентификация

Сессионная, не по имени: email+пароль, и/или вход через Telegram-бота
(`/start`-диплинк — открывает бота, тот дёргает вебхук, фронт поллит
`/api/auth/telegram/poll`). Email требует подтверждения (иначе роут-гвард
фронта загоняет на `/verify`); Telegram-only аккаунт email не имеет и
подтверждения не требует. Сессия — httpOnly-кука, `internal/store` хранит
её хешированный токен; пароль — bcrypt.

- Публичные (без сессии): `POST /api/auth/{register,login,logout}`,
  `GET /api/auth/verify`, `POST /api/auth/{resend-verification,forgot,reset}`,
  `POST /api/auth/telegram(/start)`, `GET /api/auth/telegram/poll`,
  `POST /api/telegram/webhook` (аутентифицирован отдельным секретом от
  Telegram, не сессией).
- `requireSession` (строго кука, без легаси-моста) — всё account-scoped:
  `/api/me*`, `/api/auth/{logout-all,session}`.
- `requireAuth` (кука ИЛИ легаси `X-User`-заголовок) — весь остальной
  `/api/*`: курс, уроки, упражнения, SRS-очередь, словарь, лидерборд.
  `X-User` жив только для переходного периода — не должен доходить ни до
  чего account-scoped.
- Троттлинг — `internal/ratelimit` (по IP и по email/ключу), плюс
  soft-lock после N неудачных попыток.

## База данных — PostgreSQL (прод) / SQLite (локально)

**Один код, два бэкенда.** `store.Open(dsn)`:

- `dsn` начинается с `postgres://` / `postgresql://` → **PostgreSQL** (драйвер
  `pgx/v5/stdlib`).
- иначе → **SQLite** (`modernc.org/sqlite`, чистый Go), `dsn` = путь к файлу
  или `:memory:`.

Запросы пишутся с плейсхолдерами `?`; для Postgres они переписываются в `$N`
функцией `rebind` в `store.go`. Схема — `schemaSQL(pg bool)`, отличие только в
колонке автоинкремента. Idempotent `CREATE TABLE IF NOT EXISTS`, миграций
руками нет (кроме `internal/store/migrations` — версионированные шаги схемы,
`migrate.go`).

### Прод (Dokploy)

Отдельный сервис **PostgreSQL** в проекте, приложение получает
`DATABASE_URL` через env. Редеплой приложения базу не трогает. Полностью —
`docs/DEPLOY.md`. Деплоится через Dokploy API (`git push` сам по себе не
триггерит редеплой — см. память «Serbian app deploy access»).

### Локально

По умолчанию SQLite-файл `./data/app.db` — ничего ставить не нужно.
`make dev` поднимает фронт (5173) и бэк (8080). Локальный тестовый аккаунт
можно создать через `/register` и вручную проставить
`identities.email_verified_at` в SQLite, чтобы обойти письмо с
подтверждением.

### Тесты

- `make test` / `go test ./server/...` — стор гоняется на SQLite in-memory.
- Против настоящего Postgres: `TEST_DATABASE_URL='postgres://localhost/srpski_test?sslmode=disable' go test ./server/internal/store/`
  (таблицы `TRUNCATE` между тестами). `make test-pg` — то же.
- Локальный тест-Postgres: `createdb srpski_test` (нужен запущенный
  `postgresql@17` из brew) или свой docker.
- Фронт: `cd web && npx vitest run` (юнит/компонентные), `npx vue-tsc -b` (типы).

## Команды

```bash
make dev        # фронт :5173 (проксирует /api) + бэк :8080, SQLite
make test       # go + vitest (SQLite)
make test-pg    # go-тесты стора против TEST_DATABASE_URL
make build      # npm ci (web/) + vite build + go build -> ./serbian-app
```

Озвучка: `python3 scripts/tts.py` (edge-tts, `sr-RS-SophieNeural`, без ключа) —
дописывает недостающие `content/audio/*.mp3` для слов из `vocab.yaml` **и** для
упражнений `type: listen` / диалоговых реплик (`NN.M-tK.mp3`).

## Деплой

Прод — Dokploy, домен `ucimo.ru`. `docs/DEPLOY.md` описывает Auto Deploy на
push в `main`, но на практике GitHub-вебхук у Dokploy не срабатывает —
деплоить нужно руками через Dokploy API (`POST /api/application.deploy`).
Ключ/IDs — в памяти «Serbian app deploy access», туда же не заглядывай без
необходимости (секрет). Контент и озвучка внутри образа, база — снаружи.

## Модель урока: манифест + шаги

У урока две модели, `content/load.go` выбирает по расширению `file:` в
`course.yaml`:

- **манифест** (`file: "lessons/NN.yaml"`) — новая, единственная в реальном
  контенте сейчас. Урок = упорядоченный список **шагов**:
  `teach` (теория), `practice` (упражнения), `reading` (текст + опц.
  упражнения), `dialogue` (пошаговый диалог, см. ниже), `checkpoint`
  (упражнения-проверка). Теория — фрагменты `content/lessons/NN/*.md`,
  упражнения — инлайн в шаге. Пример со всеми видами —
  `content/lessons/_TEMPLATE.yaml`.
- **легаси** (`file: "lessons/NN-slug.md"` + `content/exercises/NN.yaml`)
  — загрузчик синтезирует из неё шаги (teach → practice-блоки → reading).
  В реальном `content/` легаси-уроков не осталось; синтез шагов покрыт
  тестом на фикстуре (`testdata/content`).

Прогресс — по шагам (`store.lesson_step_progress`), экран урока —
пошаговый плеер (`web/src/views/LessonView.vue`, мобайл-first): один
экран = один шаг, либо (для `practice`/`reading`/`checkpoint` с
несколькими упражнениями) одно упражнение за раз, листается «Дальше».
Кнопка «Назад» в заголовке листает назад в том же гранулярном порядке —
внутри диалогового шага она отменяет последний ответ **этой сессии**
(шаг за шагом), а не сразу выходит из шага (см. `DialogueStep.vue`
`stepBack()` + `LessonView.vue` `goBack()`).

### Диалоговый шаг (`kind: dialogue`)

Реплики (`turns:`) чередуют `npc`/`me`; каждая `me`-реплика несёт
`exercise:` (только `choice`/`translate`/`fill_blank`). Разговор
раскрывается сверху вниз по мере ответов (`DialogueStep.vue`), без
постраничной навигации — весь шаг живёт на одном экране. У шага общий
`step.Exercises` (как и у practice/checkpoint, `collectExercises` в
`manifest.go`), поэтому назад-навигация и «один Проверить видим
одновременно» логика в `LessonView.vue` учитывают dialogue отдельно
(`activeDialogueExercise`), а не через обычный `paginatesExercises`.

**`fill_blank` в диалоге:** `sr`/`ru` самой реплики скрыты до ответа
(см. `Turn.SR`/`Turn.RU`), так что `prompt` — единственная подсказка.
Голый сербский промпт без русского ориентира (`"Malo. ___ srpski."`)
угадывается неоднозначно, если на пропущенное место грамматически
подходит больше одного слова (было: `02.10.3`, `05.18.2`, `09.16.3`,
исправлено русской подсказкой `«...» → ...` по образцу остальных
`fill_blank`). Без русской подсказки оставляй только когда ответ
вынужден грамматикой (`06.8.6`: время только с `u`) или дословно
повторяет слово из видимой предыдущей реплики (`10.18.2`: эхо
услышанного). Иначе — либо добавляй `«русский» → сербский с ___`, либо
меняй на `choice` с русским `prompt`, как у соседних реплик того же
диалога.

## Добавить урок (манифест)

1. `content/course.yaml` — запись урока с `file: "lessons/NN.yaml"`.
2. `content/lessons/NN.yaml` — манифест (скопируй `_TEMPLATE.yaml`).
3. `content/lessons/NN/*.md` — фрагменты теории для `teach`/`reading`.
4. Новые слова → `content/vocab.yaml` (`lesson: "NN"`), продублируй их
   id в `teaches:` манифеста.
5. Падежные формы, имена → `also_ok:` конкретного шага; общие
   имена/числа/частицы → `content/allow-words.yaml`.
6. `python3 scripts/tts.py` — озвучка новых слов и `listen`-упражнений/
   диалоговых реплик.

**Гард-тесты** (`server/internal/content/`):
- `real_test.go` — структура уроков, наличие reading/checkpoint, аудио
  для `listen`.
- `lexicon_test.go` — в манифест-уроке `accept` / `options` / `bank` /
  сербская сторона `pairs` / `say` используют только слова из
  накопительного словаря к этому уроку (+ `teaches` / `also_ok` /
  `allow-words`).
- порядок сложности в `practice`-шаге не убывает (иначе `mixed: true`);
  у `dialogue` порядок — сценарный, не проверяется.

### Текст для чтения

Манифест: шаг `kind: reading`, `md:` на фрагмент, где строка `---`
делит сербский текст и русский перевод. Попадает в `Step.Markdown` /
`Step.MarkdownRU`.

Клик по слову → карточка из словаря — общий компонент `GlossedText.vue`
(токенайзер `lib/reading.ts`, общий кэш `lib/lookup.ts`, эндпоинт
`/api/lookup?q=`). Используется в тексте для чтения, в промптах упражнений
`fill_blank` / `fix_error` и в репликах диалога. Промпты `translate`
(русские, ответ спойлить нельзя) и `free` — без глоссов. В карточке слова —
кнопка «＋ в повторение» (`POST /api/review/add {vocab_id}`).

### Типы упражнений

`translate`, `fill_blank`, `fix_error`, `conjugate`, `free`, `listen`
(диктант — `say:` синтезируется, `accept:` печатают), плюс лёгкие:
`choice` (`options` + `answer`), `word_bank` (`bank` из фишек +
`accept`), `match` (`pairs` [sr, ru]). Клиенту НИКОГДА не уходят
`accept` / `answer` / соответствие `pairs` / `say` — только
`options` / `bank` / `left` / `right`, перемешиваются на клиенте.
Проверка — `POST /api/lessons/{id}/exercises/{exId}/check`. Компоненты —
`web/src/components/exercises/*Answer.vue`; `translate`/`fill_blank`/
`fix_error`/`listen` делят `TextAnswer.vue`.

Каждый Answer-компонент, у которого своя кнопка «Проверить», рисует её
через общий `BottomBar.vue` (закреплён снизу) — `LessonView.vue`
(`showOwnBottomButton`) следит, чтобы одновременно был виден только один
нижний бар (свой или общий «Дальше»).

### Личный слой

`content/persona.yaml` (`name`, `city`, `job`, …). Загрузчик подставляет
`{ключ}` в теорию, промпты и `sample` — но **не** в ответы. Отсутствие
файла — no-op.

## SRS-очередь повторения (`/api/review/queue`, `ReviewView.vue`)

Карточки — слова (`vocab:<id>`) и ложные друзья (`ff:<id>`), SM-2
(`internal/srs`). Новые слова не выдаются пачкой — гейтятся по одному в
порядке уроков (`nextNewVocabCardIDs`): пока пройдено < 15 слов (
`beginnerWordCount`), весь дневной лимит новых карт (`newPerDay = 15`) —
только словарь, ложные друзья не подмешиваются вообще; после — лимит
делится пополам. У новой карточки первый показ — quiz на узнавание
(4 варианта, `buildOptions`), а не открытый ответ.

## Стиль

- Go: как в соседних файлах, короткие доки-комментарии на экспортных
  символах, ошибки оборачивай (`fmt.Errorf("...: %w")`).
- Комментарии в коде — по-английски, тексты для пользователя и контент —
  по-русски/сербски.
- Тесты рядом с кодом, table-driven где уместно.
- Front: без бордеров у карточек/инпутов/тегов-переключателей — разделение
  только фоном (`--card`/`--bg-soft`/`--accent-soft`) и `box-shadow`
  (`.card`); граница только там, где несёт смысл (напр. подсветка
  выбранного/правильного/неправильного варианта).
