# Деплой на Dokploy

Приложение — **один Docker-образ**: Go-бинарь + вшитый фронт (`go:embed`),
слушает `8080`, отдаёт и API, и SPA. Состояние (аккаунты, SRS, попытки,
прогресс) — в **отдельной базе PostgreSQL** (сервис в том же проекте Dokploy),
поэтому редеплой приложения данные не трогает.

- Образ: multi-stage `Dockerfile` в корне.
- БД: PostgreSQL, подключается через переменную `DATABASE_URL`.
- Порт: `8080` (HTTP). TLS вешает Traefik/Dokploy.
- Реплик приложения: можно >1 (писатель теперь Postgres).

> SQLite всё ещё поддерживается для локальной разработки и как фолбэк: если
> `DATABASE_URL` не задан, используется файл из `-db` (по умолчанию
> `/app/data/app.db`). Для прода — только Postgres.

---

## 1. Приложение в Dokploy

1. **Projects → Create Project** → `srpski`.
2. **Create Service → Application**.
3. **Provider:** GitHub (или Git по SSH-URL + deploy key). Repo:
   `llladno/serbian-language-app`, branch `main`.
4. **Build Type: `Dockerfile`**, path `Dockerfile`, context `.`.

## 2. База — сервис PostgreSQL

В том же проекте: **Create Service → Database → PostgreSQL**.

| поле | значение |
|---|---|
| Name | `srpski-db` |
| Postgres user / db | `srpski` / `srpski` (или как удобно) |
| Version | 16+ |

Dokploy сам заводит volume под данные Postgres и держит их отдельно от
приложения. Скопируй **внутренний** connection string (вида
`postgres://srpski:<pass>@srpski-db:5432/srpski`).

## 3. Переменные окружения приложения

**Application → Environment:**

```
DATABASE_URL=postgres://srpski:<pass>@srpski-db:5432/srpski?sslmode=disable
```

`sslmode=disable` — трафик внутри Docker-сети Dokploy. `TZ=Europe/Belgrade`
уже зашит в образ. Больше ничего не нужно.

> Схема создаётся сама при старте (`CREATE TABLE IF NOT EXISTS`). Миграций
> руками нет.

## 4. Домен

**Domains → Add Domain:** Host `serbianapp.pockets-money.ru`,
Container Port `8080`, HTTPS on, Redirect HTTP→HTTPS on.

## 5. Deploy

**Deploy**. Первая сборка ~2–3 мин. Успех — в логах `listening on :8080`,
health (`/api/health`) зелёный. Открыть домен → «Кто занимается?» → имя.

---

## Обновления

- **Auto Deploy** включён: `git push` в `main` триггерит редеплой.
- Новые уроки / слова / озвучка едут внутри образа (`content/`) — просто
  коммит + push.
- База — отдельный сервис, редеплой приложения её не касается.

## Перенос данных со старой SQLite

Если раньше крутилась SQLite-версия и нужно перенести прогресс:

1. Достань файл со старого контейнера:
   ```bash
   docker cp "$(docker ps -qf name=srpski-app):/app/data/app.db" ./app.db
   ```
2. Положи его в новый контейнер приложения (`docker cp ./app.db <app>:/tmp/app.db`).
3. Разово переопредели команду запуска (**Advanced → Command**), добавив
   `-import-sqlite /tmp/app.db`, и сделай Redeploy. В логах будет
   `imported N rows from /tmp/app.db`. Импорт срабатывает только если в
   Postgres ещё нет аккаунтов — повторный запуск безвреден.
4. Убери `-import-sqlite` из команды.

## Бэкапы БД

**Dokploy → сервис `srpski-db` → Backups:** расписание + S3-совместимое
хранилище. Ручной дамп:
```bash
docker exec "$(docker ps -qf name=srpski-db)" pg_dump -U srpski srpski > srpski-$(date +%F).sql
```

## Локальная проверка образа

```bash
docker build -t srpski .
docker run --rm --network host -e DATABASE_URL='postgres://localhost/srpski?sslmode=disable' srpski
# открыть http://localhost:8080
```

Или без Postgres — на SQLite:
```bash
docker run --rm -p 8080:8080 -v srpski-data:/app/data srpski
```
