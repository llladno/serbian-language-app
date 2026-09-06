# Деплой на Dokploy — один контейнер

Приложение собирается в **один Docker-образ**: Go-бинарь + вшитый фронт (`go:embed`),
слушает порт `8080`, отдаёт и API, и SPA. Отдельная БД не нужна — состояние в
SQLite-файле на персистентном volume.

- Образ: multi-stage `Dockerfile` в корне репозитория (~48 МБ).
- Данные: `/app/data/app.db` — держать на volume `srpski-data`.
- Порт: `8080` (HTTP). TLS вешает Traefik/Dokploy.
- Реплик: **строго 1** (SQLite — один писатель).

---

## 0. Запушить репозиторий

Сейчас репо локальное, без remote. Dokploy тянет код из Git — залей на
GitHub / GitLab / Gitea:

```bash
cd /Users/grisha/plans/serbian-app
git remote add origin git@github.com:<user>/serbian-app.git
git push -u origin build-app        # или master, если ветка уже влита
```

(Либо в Dokploy выбрать провайдер **Git**, вставить SSH-URL и добавить
показанный deploy key в настройки репозитория.)

---

## 1. Создать приложение в Dokploy

1. **Projects → Create Project** → имя `srpski`.
2. Внутри проекта → **Create Service → Application**.
3. **Provider:**
   - GitHub (подключить аккаунт) или **Git** (SSH-URL + deploy key).
   - Repository: `serbian-app`. Branch: `build-app` (или `master`).
4. **Build Type: `Dockerfile`**. Dockerfile Path: `Dockerfile`. Build Context: `.`

## 2. Volume под базу

**Advanced → Volumes → Add Mount:**

| поле | значение |
|---|---|
| Mount Type | **Volume Mount** |
| Volume Name | `srpski-data` |
| Mount Path (в контейнере) | `/app/data` |

Без этого SQLite-файл пропадёт при каждом редеплое.

## 3. Домен

**Domains → Add Domain:**

| поле | значение |
|---|---|
| Host | `srpski.твойдомен.рф` (или сгенерённый `*.traefik.me`) |
| Container Port | `8080` |
| HTTPS | on (Let's Encrypt) |
| Redirect HTTP → HTTPS | on |

## 4. (опционально) Health check

В образе уже есть `HEALTHCHECK` на `/api/health`. Дополнительно в Dokploy
**Advanced → Health Check**: Path `/api/health`, Port `8080`.

## 5. Переменные окружения

Не требуются. `TZ=Europe/Belgrade` уже зашит в образ (для корректных «сегодня»
в SRS/серии). Флаги можно переопределить в **Advanced → Command**:

```
-addr :8080 -content /app/content -db /app/data/app.db
```

## 6. Deploy

Нажать **Deploy**. Первая сборка ~2–3 мин (npm ci + go build). Смотреть
**Deployments → Logs**. Успех — в логах `listening on :8080`, health зелёный.

Открыть домен → экран «Кто занимается?» → ввести имя. Готово.

---

## Обновления

- **Auto Deploy:** в настройках приложения включить webhook — тогда
  `git push` в ветку сам триггерит редеплой.
- Новые уроки/слова едут внутри образа (лежат в `content/` репозитория) —
  просто коммит + push.
- Volume `srpski-data` при редеплое сохраняется, прогресс не теряется.

## Бэкапы БД

- **Dokploy → приложение → Backups (Volume Backups):** расписание + S3-совместимое
  хранилище.
- Вручную на сервере:
  ```bash
  docker run --rm -v srpski-data:/d -v "$PWD":/out alpine \
    cp /d/app.db /out/app-$(date +%F).db
  ```

## Миграция схемы

При первом запуске на непустой старой базе (без аккаунтов) схема
автоматически мигрирует v1→v2: все прежние карточки/попытки/прогресс
переезжают под аккаунт **«Гриша»**. Ничего делать не нужно.

## Почему не Postgres

Приложение однопользовательское по нагрузке (несколько имён-аккаунтов, один
писатель). SQLite на volume проще, быстрее и без отдельного сервиса.
Postgres понадобится только при нескольких репликах бэкенда — это отдельная
переделка `server/internal/store` (замена `modernc.org/sqlite` на `pgx`,
переписывание SQLite-специфики: `INSERT OR IGNORE`, `substr`, `RANDOM()`,
`PRAGMA`).

## Локальная проверка образа

```bash
docker build -t srpski .
docker run --rm -p 8080:8080 -v srpski-data:/app/data srpski
# открыть http://localhost:8080
```
