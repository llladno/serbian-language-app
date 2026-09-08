# Уроки с шагами + лёгкие задания — дизайн

Переработка курса: разбить крупные уроки на маленькие **шаги** внутри
урока, сделать задания проще и добавить лёгкие типы, поставить жёсткое
правило «в задании только уже пройденные слова», обезличить ядро курса
и расширить карту курса до разговорного уровня (A1→B1).

- **Стек:** без изменений — Go (`net/http`, `yaml.v3`, `modernc.org/sqlite`,
  `fsnotify`) + Vue 3 / Vite / TS.
- **Репозиторий:** `/Users/grisha/plans/serbian-app/`.
- **Предыдущий дизайн:** `2026-09-06-serbian-app-design.md` (в силе, это
  надстройка).

## Мотивация

Обратная связь пользователя:

1. Программа слишком сложная, темы вводятся слишком быстро — растянуть.
2. Сделать курс универсальнее: шире темы/лексика **и** обезличить ядро
   (личное — имя, город, работа — отдельным слоем).
3. Не давать заданий со словами, которых ещё не было; только то, что уже
   в теме или раньше.
4. Главное: разбить уроки на поменьше, задания — попроще. Усложнять —
   потом.
5. Больше уроков и тем.
6. **Цель курса:** любой человек после курса уверенно говорит по-сербски
   на разговорном уровне (A1→B1), не только «выживание».

## Цели

1. **Модель шагов.** Урок = упорядоченный список шагов (`teach` /
   `practice` / `reading` / `checkpoint`). Каждый шаг — самостоятельный
   объект с заголовком, прогрессом и экраном.
2. **Манифест урока.** `content/lessons/NN.yaml` описывает урок и его
   шаги; фрагменты теории — `content/lessons/NN/*.md`; упражнения — инлайн
   в шаге. Старая схема (`NN-slug.md` + `exercises/NN.yaml`) продолжает
   работать через фолбэк.
3. **Лёгкие типы заданий:** `choice` (выбор), `word_bank` (собери фразу),
   `match` (пары).
4. **Порядок сложности** внутри шага `practice`: от узнавания к переводу;
   нарушение — ошибка сборки (с явным опт-аутом).
5. **Гвардия лексики.** Тест сборки: сербские слова в `accept` / `options`
   / `bank` / `pairs` / `say` — только из накопительного словаря к этому
   уроку либо из явных списков-исключений. Нарушение — красная сборка.
6. **Пошаговый плеер** на экране урока + прогресс по шагам.
7. **Обезличивание ядра** + личный слой (`persona.yaml`, подстановка
   `{name}` / `{city}` / `{job}`).
8. **Карта курса A1→B1** в `course.yaml` и `serbian/plan.md`.
9. **Эталон:** уроки 01–03 целиком переведены в новую модель; 04–05
   работают через фолбэк без переписывания.

## Не-цели (сейчас)

- Наполнение уроков 06+ контентом (только карта: title/subtitle/planned).
- Перенумерация / переработка «падежных» уроков старой Фазы B.
- Новый экран онбординга для `persona.yaml` (правится как файл; `{name}`
  берётся из имени аккаунта).
- Изменения SRS-алгоритма, дашборда фаз, лидерборда.
- Подсказки в упражнениях, смягчение checker'а (порядок слов/диакритика).
- Озвучка новых слов уроков 03 (`scripts/tts.py` — отдельным прогоном).

---

## Секция 1 — Контент-модель

### Раскладка файлов

```
content/
  course.yaml                 # фазы/уровни + список уроков (расширяем)
  lessons/
    01.yaml                   # НОВОЕ — манифест
    01/                        # НОВОЕ — фрагменты теории
      1-pozdravi.md
      3-ti-vi.md
      5-citanje.md
    02.yaml   02/…
    03.yaml   03/…
    04-brojevi-i-novac.md      # СТАРАЯ модель — не трогаем
    05-u-prodavnici.md         # СТАРАЯ модель — не трогаем
  exercises/
    04.yaml  05.yaml           # СТАРАЯ модель — не трогаем
    _TEMPLATE.yaml             # обновить под новые типы/манифест
  vocab.yaml
  false-friends.yaml
  allow-words.yaml             # НОВОЕ — глобальные исключения токенов
  persona.yaml                 # НОВОЕ — личный слой
```

`course.yaml` в записи урока: поле `file:`. Расширение решает модель:
`.yaml` → манифест, `.md` → сегодняшняя схема как есть.

### Манифест `content/lessons/01.yaml`

```yaml
lesson: "01"
title: "Поздрави и први контакт"
subtitle: "поздороваться, ты/вы, вежливые слова"
teaches: [zdravo, cao, dobar-dan, dobro-jutro, dobro-vece, laku-noc,
          dovidjenja, prijatno, molim, hvala, izvini, izvolite, nema-na-cemu]
steps:
  - id: "01.1"
    kind: teach
    title: "Здороваемся"
    md: "01/1-pozdravi.md"

  - id: "01.2"
    kind: practice
    title: "Узнай приветствие"
    also_ok: []                # доп. токены для этого шага
    exercises:
      - id: "01.2.1"
        type: choice
        prompt: "«Добрый день» —"
        options: ["Dobar dan", "Dobro veče", "Laku noć"]
        answer: "Dobar dan"

  - id: "01.5"
    kind: reading
    title: "В пекаре"
    md: "01/5-citanje.md"      # '---' внутри делит SR / RU (текущая механика)

  - id: "01.8"
    kind: checkpoint
    title: "Проверка"
    exercises: [ … ]
```

Правила манифеста:

- `lesson` совпадает с ключом в `course.yaml`.
- `teaches` — id из `vocab.yaml`, слова, которые вводит урок. Могут
  отсутствовать в `vocab.yaml` только если это ошибка → сборка падает.
- `steps` непустой. `id` шага уникален в уроке, начинается с `"<lesson>."`.
- `kind ∈ {teach, practice, reading, checkpoint}`.
- `teach` требует `md`, запрещает `exercises`.
- `practice` / `checkpoint` требуют непустой `exercises`; `md`
  необязателен (короткое напоминание).
- `reading` требует `md` (с `---`); `exercises` необязательны.
- `also_ok` (только у шагов с упражнениями) — список строк-токенов,
  разрешённых гвардии лексики в этом шаге (падежные формы, имена).
- `mixed: true` (только `practice`) — опт-аут из проверки порядка
  сложности.

### Виды шагов

| kind | содержимое | условие «Дальше» |
|---|---|---|
| `teach` | markdown, одна мысль, ≤ ~180 слов | открыт |
| `practice` | 3–8 упражнений | на все дан ответ |
| `reading` | текст SR/RU + опц. 2–3 вопроса | открыт / отвечены |
| `checkpoint` | смешанные типы, слова всего урока, + `free`-ролёвка | на все дан ответ |

### Изменения кода — `server/internal/content`

**`types.go`:**

```go
type Lesson struct {
    ID, Title, Subtitle string
    Planned  bool
    File     string          // относительный путь из course.yaml
    Manifest bool            // true = новая модель
    Steps    []Step
    Teaches  []string        // id слов
    // Legacy-совместимость: заполняются и в новой модели (конкатенация)
    Markdown, Reading, ReadingRU string
    MarkdownPath string
}

type Step struct {
    ID, Kind, Title string
    Markdown        string   // teach/practice/reading
    MarkdownRU      string   // reading: перевод
    AlsoOK          []string
    Mixed           bool
    Exercises       []Exercise
}
```

`Course.Exercises map[string][]ExerciseBlock` — оставляем для легаси;
у манифест-уроков заполняется по одному синтетическому блоку на шаг
`practice`/`checkpoint` (id блока = id шага), чтобы не ломать
`GET /api/lessons/{id}/exercises`, `LessonAttempts`, `WeakExercises`.

**`load.go`:**

- Диспатч в загрузке урока по расширению `file:`.
- Манифест: читать YAML, для каждого шага с `md:` — прочитать фрагмент
  (`teach`/`practice`: как есть в `Step.Markdown`; `reading`: прогнать
  `extractReading` по `---`). Инлайн-упражнения — тем же декодером, что и
  `exercises/*.yaml`, плюс новые типы.
- Легаси `.md`: как сейчас, затем синтезировать `Steps`:
  `[teach(весь markdown без reading)]` + по одному `practice` на каждый
  блок из `exercises/NN.yaml` + `[reading(...)]`, если есть reading-блок.
- Валидация манифеста (список правил выше). Ошибки — с путём файла и id
  шага/упражнения.
- Собрать `Course.Exercises` для манифест-уроков из шагов.

**`watch.go`:** добавить рекурсивный обход `content/lessons/` — при старте
watcher'а пройти `filepath.WalkDir` и `w.Add` каждую поддиректорию; на
события `Create` каталога — `w.Add` новый каталог. Дебаунс/атомарный
снапшот без изменений.

**`_TEMPLATE.yaml`:** переписать как манифест-пример со всеми видами
шагов и всеми типами упражнений (включая новые).

---

## Секция 2 — Типы заданий

Существующие: `translate`, `fill_blank`, `fix_error`, `conjugate`,
`free`, `listen`. Добавляем три.

### `choice` — выбор одного верного

```yaml
- id: "01.2.1"
  type: choice
  prompt: "«Спасибо» —"
  options: ["Hvala", "Molim", "Izvini"]
  answer: "Hvala"
  explain: "..."
```

- Хранение: `Exercise.Options []string`, `Exercise.Answer string`.
  `answer` обязан быть одним из `options` (иначе сборка падает).
  `options` ≥ 2, уникальны.
- DTO: отдаём `options` (в исходном порядке; перемешивание — на клиенте),
  **не отдаём** `answer`.
- Checker: `CheckChoice(answer, correct)` — `Normalize`-равенство,
  `Result{OK, Expected: correct}`.

### `word_bank` — собрать фразу из фишек

```yaml
- id: "01.4.3"
  type: word_bank
  prompt: "Соберите: «Меня зовут Ана»"
  bank: ["Zovem", "se", "Ana", "Ja", "sam"]     # можно с отвлекалками
  accept: ["Zovem se Ana", "Ja sam Ana"]
```

- Хранение: `Exercise.Bank []string` + существующий `Accept`.
  `bank` ≥ 2; каждый токен любого `accept` (после `Normalize`/split)
  обязан присутствовать в `bank` (иначе фразу не собрать) → проверка
  сборки.
- DTO: отдаём `bank` (перемешивание — на клиенте), `accept` — нет.
- Checker: склеить присланные фишки через пробел → существующий
  `checker.Check(joined, accept)`. Ответ в попытке — собранная строка.

### `match` — соединить пары

```yaml
- id: "01.6.2"
  type: match
  prompt: "Соедините"
  pairs:
    - ["Dobro jutro", "Доброе утро"]
    - ["Laku noć", "Спокойной ночи"]
```

- Хранение: `Exercise.Pairs [][2]string`. 2–6 пар, левые уникальны,
  правые уникальны.
- DTO: `left []string` (в исходном порядке) + `right []string`
  (перемешивание на клиенте). Соответствие не отдаём.
- Checker: `CheckMatch(got map[string]string, pairs) (ok bool, per
  map[string]bool)`. Ответ в попытке — JSON-строка `{"left":"right"}`
  (для `AddAttempt.Answer` и показа в «слабых местах»).
- Компонент MVP: у каждой левой строки `<select>` с правыми вариантами.
- **Приоритет ниже** `choice` и `word_bank` — при нехватке времени
  вырезается из плана (не из спеки).

### Порядок сложности

Ранги: `choice`=1; `match`,`fill_blank`=2; `word_bank`,`fix_error`=3;
`conjugate`,`listen`=4; `translate`=5; `free`=6.

Проверка сборки: в шаге `kind: practice` без `mixed: true` ранги
упражнений идут не убывая. Иначе — ошибка сборки с указанием урока/шага.
`checkpoint` и `mixed: true` — исключены.

### Прятанье ответов — контракт

`GET /api/lessons/{id}/exercises` и любой ответ, кроме `POST .../check`,
**никогда** не содержит: `accept`, `answer`, соответствие `pairs`, `say`.
Отдаются: `options`, `bank`, `left`, `right`. Гард-тест на DTO
(проверяет отсутствие полей в JSON) — обязателен.

### Изменения кода

- `content/load.go`: парсинг `options`/`answer`/`bank`/`pairs`; валидация
  выше; `autoTypes` += `choice`, `word_bank`, `match`.
- `content/types.go`: новые поля `Exercise`.
- `checker/checker.go`: `CheckChoice`, `CheckMatch`; `word_bank` идёт
  через `Check`.
- `api/dto.go`: `exerciseDTO` += `Options`, `Bank`, `Left`, `Right`
  (`omitempty`); НИКОГДА не сериализовать `answer`/`accept`/`pairs`.
- `api/api.go` `checkExercise`: ветки новых типов; запись попытки как
  сейчас (`block` = id шага).
- фронт: `web/src/components/exercises/ChoiceAnswer.vue`,
  `WordBankAnswer.vue`, `MatchAnswer.vue`; ветки в `ExerciseItem.vue`;
  типы в `web/src/types.ts`.

---

## Секция 3 — Фронт: пошаговый плеер

`web/src/views/LessonView.vue` из «вся теория + все блоки на одной
странице» → плеер по шагам.

- Роут `/lesson/:id`, опционально `?step=<id>` для диплинка.
- **Шапка:** название урока, полоса «шаг k из N», выход (прогресс
  сохранён).
- **Тело — один шаг:**
  - `teach` → `MarkdownView` + «Дальше».
  - `practice` / `checkpoint` → упражнения шага (переиспользуем
    `ExerciseBlock` / `ExerciseItem`); «Дальше» активна, когда на все
    упражнения есть ответ (не обязательно верный).
  - `reading` → `ReadingText` + опц. вопросы.
- **Низ:** «Назад» / «Дальше»; на последнем шаге — «Завершить урок» →
  `POST /api/lessons/{id}/complete`.
- **Возобновление:** открыть первый незавершённый шаг (по прогрессу
  шагов); всё завершено → первый шаг.
- **Тоггл «весь урок сразу»** для повторения — низкий приоритет,
  вырезается при нехватке времени.
- `web/src/stores/course.ts`: `lesson` теперь с `steps`; состояние
  текущего шага; экшены отметки шага.
- `web/src/types.ts`: `Step`, `Lesson.steps`.
- Экран курса (`CourseView.vue`) — без изменений (уровень уроков), но
  строка урока может показывать «5/8».

---

## Секция 4 — Прогресс по шагам

### Схема (в `store.schemaSQL`)

```sql
CREATE TABLE IF NOT EXISTS lesson_step_progress (
    user_name    TEXT NOT NULL,
    lesson       TEXT NOT NULL,
    step         TEXT NOT NULL,
    status       TEXT NOT NULL,          -- in_progress | done
    completed_at TEXT,
    PRIMARY KEY (user_name, lesson, step)
);
```

`CREATE TABLE IF NOT EXISTS` — как остальные, ручных миграций нет. Для
Postgres проходит через тот же `schemaSQL(pg)` (нет автоинкремента —
отличий нет).

### Store (`server/internal/store/store.go`)

- `SetStepStatus(lesson, step, status string, now time.Time) error` —
  upsert, `completed_at` ставится один раз при `done`.
- `StepStatuses(lesson string) (map[string]string, error)`.
- `ResetLesson` / `ResetExercises` — дополнительно чистят
  `lesson_step_progress`.
- `import.go` — SQLite→PG перенос: добавить таблицу в список
  копируемых.

### API

- `POST /api/lessons/{id}/steps/{step}` тело `{"status":"in_progress"|"done"}`
  → `SetStepStatus`. Первый `in_progress` любого шага → апсертит урок в
  `in_progress` (существующий `SetLessonStatus`). Последний шаг `done`
  фронт не завершает автоматически — вызывает `.../complete` как сейчас.
- `GET /api/lessons/{id}` (`lessonDTO`) += `steps []stepDTO` со `status`
  каждого шага. `stepDTO` НЕ содержит `answer`/`accept` вложенных
  упражнений (упражнения отдаёт отдельный эндпоинт как сейчас), только
  метаданные шага: `id`, `kind`, `title`, `markdown`, `markdown_ru`,
  `status`, `exercise_ids`.
- Существующие пройденные уроки: строк в `lesson_step_progress` нет →
  плеер стартует с шага 1, `lesson_progress.status` не трогается,
  урок остаётся `done`.

---

## Секция 5 — Гвардия лексики (тест сборки)

Новый `server/internal/content/lexicon_test.go`.

### Накопительный словарь урока N

Порядок уроков — из `course.yaml` (`phases[].lessons` по порядку).
Известные для урока N токены:

1. `latin` каждого `vocab.yaml` с `lesson` ≤ N (по порядку курса), в
   т.ч. многословные — разбиваются на токены.
2. `teaches:` урока N.
3. `allow-words.yaml` — глобальные: имена собственные (Ana, Marko,
   Miloš, Beograd, Novi Sad, Srbija, Rusija…), числа-цифры, латинские
   буквы-заглушки, служебные частицы/союзы, звукоподражания.
4. `also_ok:` конкретного шага.

Токен = `checker.Normalize` + split по пробелу; финальная пунктуация уже
срезана `Normalize`.

### Что проверяем

Для каждого авто-упражнения (`translate`, `fill_blank`, `fix_error`,
`word_bank`, `choice`, `listen`, `match`) манифест-урока:

- **строго (падает сборка):** все токены из `accept`, `options`, `bank`,
  сербской стороны `pairs` (левая, если правая — русская; иначе обе),
  `say`. Токен проходит, если: точно в наборе; ИЛИ известный токен —
  префикс токена длиной ≥ 3 (`grad` → `gradu`, `grada`); ИЛИ в
  `also_ok`/allow-list.
- **мягко (t.Log, не падает):** сербские фрагменты `prompt` — вырезаем
  то, что в кавычках `«…»` / после `— `, проверяем тем же способом,
  расхождения только логируем (промпты двуязычные, токенайз неточен).
- Легаси-уроки (04–05) — целиком в режиме `t.Log`.

Сообщение: `урок 03 шаг 03.4 упр 03.4.2: неизвестное слово "sestru"
(добавь в also_ok шага или в teaches урока)`.

### Токенайзер

В `internal/api/lookup.go` уже есть словарный лукап с приведением формы.
Вынести общую нормализацию/стемминг в
`server/internal/content/lexicon.go` (экспортируемая функция), лукап
переиспользует.

---

## Секция 6 — Карта курса A1→B1

### `course.yaml`

- `phases` → 5 «уровней» (id `1`…`5`), поле `title` вида
  `"Уровень 1 — Первый контакт (A1.1)"`. Ключ структуры не меняется
  (`id/title/lessons`), дашборд читает как есть.
- `lessons` расширить до ~30 записей новой карты (ниже). Уроки 01–05 —
  `file:` на манифест (01–03) / старый `.md` (04–05). Остальные — только
  `title` + `subtitle`, без `file:` → `Planned=true` как сейчас.

### Карта (черновик, уточняется при наполнении)

**Уровень 1 — Первый контакт (A1.1):**
01 Поздрави и први контакт · 02 Ко си ти · 03 Људи око мене ·
04 Бројеви, цена, сати · 05 Куповина · 06 Провера 1

**Уровень 2 — Быт (A1.2):**
07 Дан по дан · 08 Кући · 09 Храна и пиће · 10 Град око мене ·
11 Време и природа · 12 Провера 2

**Уровень 3 — Связная речь (A2.1):**
13 Прошлост (перфект) · 14 Планови (футур) · 15 Кретање ·
16 Договори и термини · 17 Тело и здравље · 18 Провера 3

**Уровень 4 — Мнения и жизнь (A2.2):**
19 Осећања и мишљења · 20 Прича о себи · 21 Телефон и поруке ·
22 Проблеми и решења · 23 Слободно време · 24 Провера 4

**Уровень 5 — Уверенно (B1.1):**
25 Посао · 26 Папирологија (личный слой) · 27 Standard vs. novosadski ·
28 Дуги разговор · 29 Читање и слух · 30 Велика провера

Падежи/грамматика — нитками по темам, не отдельными уроками. Полный
модуль «вопросы» (`li`-инверсия, `jel'`, `koji/kakav/čiji`,
`zašto/zato što/jer/zbog`) — на Уровне 2. Текущий урок 03 «Pitanja» →
становится «Људи око мене»; материал вопросов переезжает.

### `serbian/plan.md`

Переписать под новую карту (человекочитаемый источник). `serbian/vocab.md`
и `serbian/lessons/*` (устарели, приложение — источник правды) →
заменить содержимое на короткий указатель к `serbian-app/content/`.

### Личный слой

`content/persona.yaml`:

```yaml
name: "Гриша"
name_latin: "Griša"
city: "Novi Sad"
city_ru: "Нови-Сад"
job: "programer"
job_ru: "программист"
native_ru: "русский"
```

Подстановка на этапе загрузки контента: в `Step.Markdown`,
`Exercise.Prompt`, `Exercise.Sample` заменять `{name}` / `{name_latin}` /
`{city}` / `{city_ru}` / `{job}` / `{job_ru}` / `{native_ru}`.
`accept`/`answer`/`bank` — **не** подставлять (ответы фиксированы; если
имя нужно в ответе — писать явным словом и добавлять в `also_ok`).
Ядро уроков 01–03 пишем на нейтральном персонаже (Ana/Marko), `{...}`
используем точечно там, где речь идёт «от лица ученика».

---

## Секция 7 — Уроки 01–03 (эталон)

Новые слова уроков капают в `content/vocab.yaml` с нужным `lesson`;
падежные формы — в `also_ok` шагов. После наполнения — `scripts/tts.py`.

### 01 — Поздрави и први контакт (~8 шагов)

1. `teach` Здороваемся — zdravo, dobar dan, dobro jutro/veče, ćao,
   laku noć, doviđenja, prijatno
2. `practice` — choice + match (узнавание)
3. `teach` Ты и вы — ti/vi, kako si / kako ste, dobro hvala, može,
   ne žalim se
4. `practice` — choice + fill_blank + word_bank
5. `teach` Вежливые слова — molim, hvala, izvini/oprosti, izvolite,
   nema na čemu, nema problema
6. `practice` — word_bank (мини-реплики)
7. `reading` «в пекаре» + 2 choice на понимание
8. `checkpoint` — mixed + 1 free («зайди в пекару, поздоровайся и
   попрощайся»)

Глагол `jesam` целиком и порядок клитик **уходят в 02**. В 01 — только
готовые `ja sam …`, `ti si …` во фразах, без таблицы.

### 02 — Ко си ти (~8 шагов)

1. `teach` `jesam`: я / ти / он — 3 формы + `nisam/nisi/nije`
2. `practice`
3. `teach` Откуда я — `iz Rusije/Srbije/…`, `živim u …`, страны/города
4. `practice`
5. `teach` Языки и работа — `govorim srpski/ruski/engleski`,
   `radim kao …`
6. `practice`
7. `reading` «знакомство»
8. `checkpoint`

`jesam` mi/vi/oni + «клитика на второе место» + `da li si` vs `jesi li`
→ последний teach-шаг 02 или урок 06. В основной части 02 — только ед.
число.

### 03 — Људи око мене (~8 шагов, новая тема)

1. `teach` Семья — mama, tata, brat, sestra, muž, žena, dete, deca,
   roditelji, baba, deda
2. `practice` — match + choice
3. `teach` `imam / nemam` + кого — акузатив только для этих слов,
   готовыми формами (`imam brata`, `imam sestru`)
4. `practice`
5. `teach` Мой / твой / его — moj/moja/moje, tvoj…, njegov/njen
   (им. падеж, 3 рода)
6. `practice` — fill_blank + word_bank
7. `reading` «моя семья»
8. `checkpoint` — mixed + 1 free («расскажи про свою семью: 3 фразы»)

---

## Влияние на существующий код

| Файл/область | Изменение |
|---|---|
| `content/types.go` | `Lesson.Steps/Teaches/Manifest`, `Step`, поля `Exercise` |
| `content/load.go` | диспатч модели, парсинг манифеста, синтез шагов для легаси, новые типы, валидация |
| `content/watch.go` | рекурсивный watch `lessons/` |
| `content/real_test.go` | переписать ожидания под шаги |
| `content/lexicon_test.go`, `lexicon.go` | новые |
| `checker/checker.go` | `CheckChoice`, `CheckMatch` |
| `store/store.go` | `lesson_step_progress`, `SetStepStatus`, `StepStatuses`, чистка в reset |
| `store/import.go` | новая таблица в переносе |
| `api/api.go` | роут `POST /api/lessons/{id}/steps/{step}`, ветки типов в `checkExercise` |
| `api/dto.go` | `stepDTO`, `lessonDTO.steps`, `exerciseDTO` новые поля, гард на утечку |
| `web/views/LessonView.vue` | плеер по шагам |
| `web/stores/course.ts`, `types.ts` | модель шагов |
| `web/components/exercises/*` | 3 новых компонента + ветки |
| `content/course.yaml` | карта A1→B1 |
| `content/lessons/01–03*`, `vocab.yaml` | эталонный контент |
| `content/allow-words.yaml`, `persona.yaml`, `_TEMPLATE.yaml` | новые/обновить |
| `serbian/plan.md`, `serbian/vocab.md`, `serbian/lessons/*` | карта + указатели |

## Порядок работ (для плана реализации)

1. Контент-модель: типы, loader, синтетические шаги для легаси, watch,
   `real_test.go`. Уроки 04–05 работают.
2. Новые типы заданий: loader + checker + DTO + гард на утечку (без
   фронта).
3. Прогресс шагов: схема, store, API, `lessonDTO.steps`.
4. Фронт: плеер `LessonView` + store (работает на синтетических шагах
   04–05).
5. Фронт: `ChoiceAnswer` / `WordBankAnswer` / `MatchAnswer`.
6. Гвардия лексики: `lexicon.go` + тест (сначала warn-only, потом строгий
   для манифест-уроков).
7. Карта курса: `course.yaml`, `serbian/plan.md`, `persona.yaml` +
   подстановка.
8. Эталонный контент: уроки 01, 02, 03 + слова в `vocab.yaml` + TTS.
9. `_TEMPLATE.yaml`, README/CLAUDE.md — обновить раздел «Добавить урок».

## Открытые вопросы

- Нужен ли `match` в первой итерации или отложить (сейчас: включён,
  низкий приоритет).
- Порог префиксного совпадения в гвардии лексики (сейчас: ≥ 3 символа) —
  подстроить по факту ложных срабатываний.
- `{name}`-подстановка на бэке (при загрузке) vs на фронте (из аккаунта):
  сейчас на бэке из `persona.yaml`; аккаунт-специфичность — позже.
