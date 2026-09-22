# ucimo (serbian-app)

Персональное веб-приложение для изучения сербского с нуля — бренд
**«ucimo»** (домен `ucimo.ru`). Переносит курс из Obsidian (`../serbian/`,
теперь архив) в веб-формат: уроки по шагам, интерактивные упражнения с
автопроверкой, тренажёр слов на интервальном повторении (SRS), словарь,
рейтинг («Люди»), профиль.

Стек: **Go** (stdlib `net/http`, `pgx/v5` + `modernc.org/sqlite`) +
**Vue 3 / Vite / TypeScript / Pinia / Tailwind**. Плюс отдельный
статический маркетинговый лендинг на **Nuxt 4** (`landing/`). Всё
собирается в один бинарь через `go:embed`.

## Архитектура

- **`server/`** — Go-бэкенд. `internal/store` — персистентность
  (аккаунты, сессии, SRS, попытки, прогресс, UTM-атрибуция),
  `internal/content` — загрузка и hot-reload `content/`, `internal/auth` —
  сессии/пароли/Telegram, `internal/api` — JSON API.
- **`web/`** — сам таргетируемый Vue-app (курс, урок, повторение, словарь,
  профиль, рейтинг), мобайл-first, с Telegram Mini App shim.
- **`landing/`** — отдельный статический Nuxt-сайт (SEO/маркетинг,
  `/privacy`, `/terms`), не завязан на API/аутентификацию.
- **`content/`** — единственный источник правды по учебному материалу
  (см. «Контент» ниже). Правки подхватываются на лету (fsnotify).

## Аккаунты и вход

Полноценная аутентификация: email + пароль (с подтверждением по почте) и/или
вход через Telegram-бота (Mini App или `/start`-диплинк). Сессия — httpOnly
cookie. Троттлинг логина/регистрации по IP и email
(`internal/ratelimit`). *(Старая версия без пароля — «войти просто по
имени» — давно заменена; легаси `X-User`-заголовок ещё жив только для
части контентных роутов, но не для аккаунтных.)*

При регистрации (email или Telegram Mini App) с ссылки с `utm_*`
параметрами первый источник трафика сохраняется на аккаунте
(first-touch) — это читает раздел «Ссылки» в
[ucimo-content-admin](../ucimo-content-admin/README.md), см.
`../ucimo-content-admin/docs/superpowers/specs/2026-09-22-utm-link-tracking-design.md`.

## Разработка

```bash
make dev        # фронт :5173 (проксирует /api) + бэк :8080, SQLite
make test       # go test + vitest, на SQLite
make test-pg    # тесты стора против настоящего Postgres (TEST_DATABASE_URL)
make build      # web + landing + go build -> ./serbian-app
```

### База данных

Один код, два бэкенда (`store.Open(dsn)`):

- **PostgreSQL** — прод (`DATABASE_URL=postgres://…`, драйвер `pgx/v5`).
- **SQLite** — локально и в тестах, дефолт `./data/app.db`, ничего ставить
  не нужно.

Схема — идемпотентная база (`CREATE TABLE IF NOT EXISTS`) плюс
версионированные шаги в `server/internal/store/migrations/*.sql`
(`migrate.go` применяет их по номеру при старте на обоих бэкендах).

## Контент

- `content/course.yaml` — уровни и порядок уроков.
- `content/lessons/NN.yaml` — манифест урока: список **шагов** (`teach` /
  `practice` / `reading` / `dialogue` / `checkpoint`), см.
  `content/lessons/_TEMPLATE.yaml`. Фрагменты теории —
  `content/lessons/NN/*.md`. Легаси-модель (`lessons/NN-slug.md` +
  `exercises/NN.yaml`) ещё поддерживается загрузчиком, но в реальном
  контенте не осталось.
- `content/vocab.yaml`, `content/false-friends.yaml`,
  `content/allow-words.yaml`, `content/persona.yaml` — словарь, ложные
  друзья, разрешённая лексика, личный слой (`{name}` и т.п.).
- `content/images/*`, `content/audio/*` — картинки к словам и озвучка
  (`python3 scripts/tts.py`, edge-tts, без ключа).

Изменяемое состояние (аккаунты, SRS, попытки, прогресс) — в базе, не в
`content/`.

## Деплой

Прод — Dokploy на `147.45.153.154`, домен **`ucimo.ru`**. GitHub:
[llladno/serbian-language-app](https://github.com/llladno/serbian-language-app),
ветка `main`. Автодеплой по вебхуку не настроен — после `git push` в `main`
редеплой нужно триггерить руками через Dokploy API
(`POST /api/application.deploy`). Подробности инфраструктуры (ID
ресурсов, ключи) — не в этом репозитории, см. память Claude
(«serbian-app deploy access»).

`docs/DEPLOY.md` описывает общую схему деплоя (может отставать в
деталях типа домена — сверяйся с этим README и с памятью Claude).

## Связанные проекты

- **[ucimo-content-admin](../ucimo-content-admin)** — соседняя админка
  бренда UCIMO: пайплайн постов для соцсетей + read-only аналитика по
  прод-БД этого приложения (раздел «Данные») + управление
  UTM-ссылками (раздел «Ссылки»). Никогда не пишет в БД serbian-app
  напрямую.
- **[../serbian/](../serbian)** — архив исходных материалов курса
  (Obsidian). Больше не источник правды — приложение с ним не
  синхронизируется.

## Тесты и добавление урока

`make test` гоняет гвардии структуры/лексики/сложности контента.
Подробный процесс добавления урока и все типы упражнений — см.
`CLAUDE.md` в этом репозитории (более полная и техническая версия этого
README, для агентов).

Дизайн/история решений: `docs/superpowers/specs/`,
`docs/superpowers/plans/`.
