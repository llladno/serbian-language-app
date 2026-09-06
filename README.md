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

## Контент

Источник правды — каталог `content/`:

- `course.yaml` — фазы и порядок уроков.
- `content/lessons/NN-*.md` — теория урока (markdown).
- `content/exercises/NN.yaml` — упражнения урока (см.
  `content/exercises/_TEMPLATE.yaml`).
- `content/vocab.yaml` — словарь.
- `content/false-friends.yaml` — ложные друзья.

Правки в `content/` подхватываются на лету (сервер следит за файлами).
Изменяемое состояние (SRS, попытки, прогресс) — в SQLite `data/app.db`.

Дизайн: `docs/superpowers/specs/2026-09-06-serbian-app-design.md`.
