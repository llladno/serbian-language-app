# Srpski App

Персональное приложение для изучения сербского. Переносит курс из
Obsidian (`../serbian/`) в веб-формат: читалка уроков, интерактивные
упражнения с автопроверкой, тренажёр слов на интервальном повторении
(SRS), дашборд прогресса.

Стек: Go (бэкенд, JSON API + отдача SPA) + Vue 3 / Vite / TypeScript.

## Разработка

```bash
make dev
```

Поднимает Vite-дев-сервер на `http://localhost:5173` (проксирует
`/api` на Go-сервер `:8080`).

## Продакшн

```bash
make build
./serbian-app -addr :8080
```

Собранный фронт вшивается в бинарь через `go:embed` — один файл,
никаких зависимостей.

### База данных

Один код, два бэкенда (`store.Open`):

- **PostgreSQL** — прод. Выбирается `DATABASE_URL` (или `-dsn`) вида
  `postgres://…`. Отдельный сервис, редеплой приложения его не трогает.
- **SQLite** — локальная разработка и тесты. Дефолт: файл `./data/app.db`
  (флаг `-db`). Ставить ничего не нужно.

Схема создаётся при старте сама. Перенос старой SQLite-базы в Postgres —
флаг `-import-sqlite <path>` (разовый, только на пустой базе).

### Docker / деплой

```bash
docker build -t srpski .
# Postgres:
docker run --rm -p 8080:8080 -e DATABASE_URL='postgres://user:pass@host:5432/db' srpski
# или SQLite на volume:
docker run --rm -p 8080:8080 -v srpski-data:/app/data srpski
```

Один контейнер: API + SPA на порту `8080`. Деплой на Dokploy (приложение +
сервис PostgreSQL) — см. [docs/DEPLOY.md](docs/DEPLOY.md).

## Контент

Источник правды — каталог `content/`:

- `course.yaml` — уровни и порядок уроков.
- `content/lessons/NN.yaml` — манифест урока: список **шагов** (`teach` /
  `practice` / `reading` / `checkpoint`), см. `lessons/_TEMPLATE.yaml`.
  Фрагменты теории — `content/lessons/NN/*.md`.
  Старая модель (`lessons/NN-slug.md` + `exercises/NN.yaml`) ещё
  поддерживается — загрузчик синтезирует из неё шаги.
- `content/vocab.yaml` — словарь.
- `content/allow-words.yaml` — имена/числа/частицы для гвардии лексики.
- `content/persona.yaml` — личный слой (`{name}` и т.п.).
- `content/false-friends.yaml` — ложные друзья.
- `content/images/<id>.jpg` — фото к слову (опц.), раздаётся на `/img/`.
- `content/audio/<id>.mp3` — озвучка слова (опц.), раздаётся на `/audio/`.
  Генерится скриптом `scripts/tts.py` (edge-tts, сербский нейроголос,
  без ключа): `pip install edge-tts pyyaml && python3 scripts/tts.py`.
  Поле `audio` в API проставляется само, если файл есть.

Правки в `content/` подхватываются на лету (сервер следит за файлами).
Изменяемое состояние (аккаунты, SRS, попытки, прогресс) — в базе
(PostgreSQL в проде, SQLite локально; см. «База данных»).

## Аккаунты

Вход — просто по имени (без пароля). Имя хранится в `localStorage`
браузера и в таблице `users`; всё состояние (карточки, попытки,
прогресс) привязано к аккаунту. Фронт шлёт имя в заголовке `X-User`
(percent-encoded). Сменить пользователя — по имени в правом верхнем углу.
Старая база без аккаунтов мигрирует автоматически под именем «Гриша».

### Добавить урок

1. В `content/course.yaml` у нужного номера — `file: "lessons/NN.yaml"`.
2. Скопируй `content/lessons/_TEMPLATE.yaml` в `content/lessons/NN.yaml`,
   собери шаги. Упражнения в `practice`-шаге — от лёгких к сложным
   (`choice` < `fill_blank`/`match` < `word_bank` < `translate` < `free`);
   иначе `mixed: true`.
3. Теорию для `teach`/`reading` — в `content/lessons/NN/*.md`.
4. Новые слова — в `content/vocab.yaml` (`lesson: "NN"`), их id — в
   `teaches:` манифеста. Падежные формы — в `also_ok:` шага.
5. Озвучка — `python3 scripts/tts.py` (дописывает недостающие
   `content/audio/*.mp3` для слов и `listen`-упражнений).
6. `make test` — гвардии проверят структуру, лексику и порядок сложности.

Тесты: `make test` (стор — на SQLite in-memory).
`make test-pg` — тесты стора против настоящего Postgres
(`TEST_DATABASE_URL`, по умолчанию `postgres://localhost/srpski_test`).

Дизайн: `docs/superpowers/specs/2026-09-06-serbian-app-design.md`.
План: `docs/superpowers/plans/2026-09-06-serbian-app.md`.
