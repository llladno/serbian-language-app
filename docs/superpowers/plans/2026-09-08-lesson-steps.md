# Уроки с шагами + лёгкие задания — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Разбить крупные уроки на маленькие шаги (`teach`/`practice`/`reading`/`checkpoint`) через манифест урока, добавить лёгкие типы заданий (`choice`/`word_bank`/`match`), поставить гвардию лексики, сделать пошаговый плеер урока и переработать уроки 01–03 как эталон.

**Architecture:** Go-загрузчик контента получает вторую модель урока — `content/lessons/NN.yaml` (манифест со списком шагов) рядом со старой (`NN-slug.md` + `exercises/NN.yaml`). Старые уроки заворачиваются в синтетические шаги, поэтому фронт-плеер и API одинаковы для обеих моделей. Прогресс по шагам — новая таблица в том же двухбэкендовом сторе (SQLite/Postgres). Лёгкие типы заданий и гвардия лексики (тест сборки) — поверх модели.

**Tech Stack:** Go 1.25 (`net/http` ServeMux, `gopkg.in/yaml.v3`, `modernc.org/sqlite`, `github.com/jackc/pgx/v5`, `github.com/fsnotify/fsnotify`); Vue 3 + `<script setup>` + TS + Vite + Pinia + Tailwind; Vitest.

**Spec:** `docs/superpowers/specs/2026-09-08-lesson-steps-design.md`

## Global Constraints

- Модуль Go: `github.com/grisha/serbian-app`. Версия: `go 1.25.0`. Бинарь `go` — `/opt/homebrew/bin/go` (Makefile уже кладёт в `PATH`).
- Бэкенд-зависимости не добавляем: только `yaml.v3`, `modernc.org/sqlite`, `pgx/v5`, `fsnotify`. Нет веб-фреймворка, ORM, серверного рендера markdown.
- Списки `accept`, поля `answer`, соответствие `pairs`, `say` **не появляются** ни в одном ответе API, кроме `POST .../check`. Гард-тест на JSON DTO обязателен.
- Ответы (`accept`/`answer`/`bank`) пишутся **латиницей**. Checker не транслитерирует латиницу↔кириллицу.
- Контент — источник правды в `content/`. Редактора контента в UI нет. SQLite — `data/app.db` (в .gitignore).
- Даты/время в БД — строки ISO-8601 (UTC). «Сегодня» для прогресса — локальная дата сервера.
- Фронт: Vue 3 `<script setup>` + TS, без Options API, без SSR.
- SQL пишется с плейсхолдерами `?`; `rebind` переписывает в `$N` для Postgres. Схема — `schemaSQL(pg bool)`, идемпотентные `CREATE TABLE IF NOT EXISTS`, ручных миграций нет.
- Тексты для пользователя и контент — по-русски/сербски; комментарии в коде — по-английски. Каждая сербская фраза в контенте глоссируется по-русски по месту (правило курса).
- Тесты: `make test` (`go test ./server/...` + `npm run test` в `web/`). Стор-тесты гоняются на SQLite in-memory.
- Репозиторий: `/Users/grisha/plans/serbian-app/`. Ветка: `feature/lesson-steps`. Каталог `/Users/grisha/plans/serbian/` — **не git-репозиторий**, правки там без коммита.

## Порядок и вехи

- **Веха 1 (задачи 1–9)** — движок шагов. По завершении: все уроки (включая легаси 04–05) проигрываются пошаговым плеером, прогресс по шагам пишется, карта курса A1→B1 в `course.yaml`. Отгружаемо.
- **Веха 2 (задачи 10–18)** — лёгкие типы заданий, гвардия лексики, уроки 01–03 как эталон. Отгружаемо.

## Карта файлов

**Бэкенд — `server/internal/content/`**
- `types.go` — MODIFY: `Step`, поля `Lesson` (`Steps`, `Teaches`, `Manifest`), поля `Exercise` (`Options`, `Answer`, `Bank`, `Pairs`, `AlsoOK` на шаге).
- `manifest.go` — CREATE: парсинг `lessons/NN.yaml` → `[]Step`; синтез шагов для легаси-уроков.
- `load.go` — MODIFY: диспатч модели по расширению `file:`, сбор `Course.Exercises` из шагов, `autoTypes` += новые типы.
- `persona.go` — CREATE: загрузка `persona.yaml`, подстановка `{...}`.
- `lexicon.go` — CREATE: нормализация/токенизация сербского, построение накопительного набора слов.
- `watch.go` — MODIFY: рекурсивный watch `content/lessons/`.
- `real_test.go` — MODIFY: ожидания под шаги и 5 фаз.
- `manifest_test.go`, `persona_test.go`, `lexicon_test.go` — CREATE.
- `testdata/content/` — MODIFY: добавить фикстуру-манифест урока `90`.

**Бэкенд — `server/internal/checker/`**
- `checker.go` — MODIFY: `CheckChoice`, `CheckMatch`.
- `checker_test.go` — MODIFY.

**Бэкенд — `server/internal/store/`**
- `store.go` — MODIFY: таблица `lesson_step_progress`, `SetStepStatus`, `StepStatuses`, чистка в `ResetLesson`/`ResetExercises`.
- `import.go` — MODIFY: таблица в списке копирования.
- `store_test.go` — MODIFY: `TRUNCATE` + новые тесты.

**Бэкенд — `server/internal/api/`**
- `dto.go` — MODIFY: `stepDTO`, `lessonDTO.Steps`, поля `exerciseDTO` (`Options`, `Bank`, `Left`, `Right`).
- `api.go` — MODIFY: роут `POST /api/lessons/{id}/steps/{step}`, ветки типов в `checkExercise`, `getLesson` отдаёт шаги.
- `api_test.go` — MODIFY: гард на утечку + тест роута шага.

**Фронт — `web/src/`**
- `types.ts` — MODIFY: `Step`, `Lesson.steps`, `ExerciseType` += 3, поля `Exercise`.
- `api.ts` — MODIFY: `setStepStatus`.
- `stores/course.ts` — MODIFY (при необходимости — хелпер статуса шага не нужен, состояние держит вью).
- `views/LessonView.vue` — REWRITE: пошаговый плеер.
- `components/exercises/ExerciseItem.vue` — MODIFY: ветки новых типов.
- `components/exercises/ChoiceAnswer.vue`, `WordBankAnswer.vue`, `MatchAnswer.vue` — CREATE (+ по тесту).

**Контент — `content/`**
- `course.yaml` — REWRITE: карта A1→B1 (5 уровней, ~30 уроков).
- `allow-words.yaml`, `persona.yaml` — CREATE.
- `lessons/01.yaml` + `lessons/01/*.md`, `lessons/02.yaml` + `lessons/02/*.md`, `lessons/03.yaml` + `lessons/03/*.md` — CREATE.
- `lessons/01-pozdravi-i-upoznavanje.md`, `lessons/02-zamenice-i-prezent.md`, `lessons/03-pitanja.md`, `exercises/01.yaml`, `exercises/02.yaml`, `exercises/03.yaml` — DELETE (заменены манифестами; Pitanja-материал восстанавливается из git для будущего модуля «вопросы»).
- `exercises/_TEMPLATE.yaml` — MODIFY: пример-манифест.
- `vocab.yaml` — MODIFY: слова уроков 01–03 (в т.ч. новые слова темы «семья»).

**Курс-репозиторий — `/Users/grisha/plans/serbian/`** (без коммита)
- `plan.md` — REWRITE под карту A1→B1.
- `vocab.md`, `lessons/*.md`, `README.md` — заменить содержимое коротким указателем на `serbian-app/content/`.

**Документация — корень репо**
- `CLAUDE.md`, `README.md` — обновить раздел «Добавить урок».

---

## Задача 1: Модель манифеста — типы и парсинг `lessons/NN.yaml`

**Files:**
- Modify: `server/internal/content/types.go`
- Create: `server/internal/content/manifest.go`
- Modify: `server/internal/content/load.go:80-140` (загрузка урока из `course.yaml`)
- Create: `server/internal/content/manifest_test.go`
- Create: `server/internal/content/testdata/content/lessons/90.yaml`
- Create: `server/internal/content/testdata/content/lessons/90/1-intro.md`
- Create: `server/internal/content/testdata/content/lessons/90/3-citanje.md`
- Modify: `server/internal/content/testdata/content/course.yaml` (добавить урок `90`)

**Interfaces:**
- Produces:
  - `content.Step{ID, Kind, Title string; Markdown, MarkdownRU string; AlsoOK []string; Mixed bool; Exercises []Exercise}`
  - `content.Lesson` новые поля: `Manifest bool`, `Steps []Step`, `Teaches []string`
  - `content.Exercise` новые поля: `Options []string`, `Answer string`, `Bank []string`, `Pairs [][2]string`
  - `content.parseManifest(dir, rel string) (*Lesson, error)` — внутренняя, вызывается из `load.go`
- Consumes: существующий декодер упражнений в `load.go` (переиспользуется для инлайн-упражнений).

**Заметки для реализации:**
- `course.yaml` в записи урока — поле `file:`. Диспатч: `strings.HasSuffix(file, ".yaml")` → манифест, иначе легаси.
- Инлайн-упражнения в шаге парсятся тем же кодом, что `exercises/*.yaml`. Вынеси разбор одного упражнения из `Load` в функцию `decodeExercise(rel string, e exerciseYAML) (Exercise, error)` в `load.go`, вызывай из обоих мест.
- `reading`-шаг: `md:` прогоняется через существующий `extractReading` (`reading.go`) — сербский текст → `Step.Markdown`, перевод → `Step.MarkdownRU`.
- Валидация манифеста (ошибка с путём файла + id шага):
  - `lesson` == ключ в `course.yaml`;
  - `steps` непустой; `id` шага уникален, `strings.HasPrefix(step.ID, lesson+".")`;
  - `kind ∈ {teach, practice, reading, checkpoint}`;
  - `teach`: `md` задан, `exercises` пуст;
  - `practice`/`checkpoint`: `exercises` непустой;
  - `reading`: `md` задан;
  - id упражнений уникальны в пределах урока.
- Пока НЕ трогаем `Course.Exercises` для манифест-уроков (задача 2 соберёт синтетические блоки). В этой задаче `parseManifest` заполняет только `Lesson.Steps`.

- [ ] **Шаг 1: Тест — манифест парсится в шаги**

`server/internal/content/manifest_test.go`:
```go
package content

import "testing"

func TestManifestLessonLoadsSteps(t *testing.T) {
	c, err := Load("testdata/content")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	l := c.Lessons["90"]
	if l == nil {
		t.Fatal("lesson 90 missing")
	}
	if !l.Manifest {
		t.Error("lesson 90: Manifest = false, want true")
	}
	if len(l.Steps) != 4 {
		t.Fatalf("lesson 90: %d steps, want 4", len(l.Steps))
	}
	if l.Steps[0].Kind != "teach" || l.Steps[0].Markdown == "" {
		t.Errorf("step 0: kind=%q md-empty=%v", l.Steps[0].Kind, l.Steps[0].Markdown == "")
	}
	if l.Steps[1].Kind != "practice" || len(l.Steps[1].Exercises) == 0 {
		t.Errorf("step 1: kind=%q exercises=%d", l.Steps[1].Kind, len(l.Steps[1].Exercises))
	}
	r := l.Steps[2]
	if r.Kind != "reading" || r.Markdown == "" || r.MarkdownRU == "" {
		t.Errorf("step 2 reading: md=%q ru=%q", r.Markdown, r.MarkdownRU)
	}
	if l.Steps[1].Exercises[0].Type != "choice" || l.Steps[1].Exercises[0].Answer != "Da" {
		t.Errorf("choice exercise not parsed: %+v", l.Steps[1].Exercises[0])
	}
}

func TestManifestRejectsTeachWithExercises(t *testing.T) {
	_, err := parseManifest("testdata/content", "lessons/bad.yaml")
	if err == nil {
		t.Skip("fixture bad.yaml not present; covered by inline table test below")
	}
}
```

Фикстуры:

`server/internal/content/testdata/content/lessons/90.yaml`:
```yaml
lesson: "90"
title: "Fixture lesson"
subtitle: "manifest smoke test"
teaches: []
steps:
  - id: "90.1"
    kind: teach
    title: "Intro"
    md: "90/1-intro.md"
  - id: "90.2"
    kind: practice
    title: "Practice"
    exercises:
      - id: "90.2.1"
        type: choice
        prompt: "«Да» —"
        options: ["Da", "Ne"]
        answer: "Da"
      - id: "90.2.2"
        type: fill_blank
        prompt: "Ja ___ ovde."
        accept: ["sam"]
  - id: "90.3"
    kind: reading
    title: "Reading"
    md: "90/3-citanje.md"
  - id: "90.4"
    kind: checkpoint
    title: "Checkpoint"
    exercises:
      - id: "90.4.1"
        type: translate
        prompt: "Да."
        accept: ["Da."]
```

`server/internal/content/testdata/content/lessons/90/1-intro.md`:
```markdown
# Intro

Kratak tekst. *(краткий текст)*
```

`server/internal/content/testdata/content/lessons/90/3-citanje.md`:
```markdown
Zdravo. Kako si?

---

Привет. Как дела?
```

`server/internal/content/testdata/content/course.yaml` — добавь в `lessons:`:
```yaml
  "90":
    title: "Fixture lesson"
    file: "lessons/90.yaml"
```
и в `phases[0].lessons` допиши `"90"`.

- [ ] **Шаг 2: Запусти тест — падает**

Run: `/opt/homebrew/bin/go test ./server/internal/content/ -run TestManifest -v`
Expected: FAIL (`l.Manifest` undefined / `parseManifest` undefined).

- [ ] **Шаг 3: Добавь типы в `types.go`**

В `Lesson` добавь:
```go
	Manifest bool     // true = загружен из lessons/NN.yaml
	Steps    []Step   // упорядоченные шаги урока
	Teaches  []string // id слов из vocab.yaml, которые вводит урок
```
Новый тип рядом с `ExerciseBlock`:
```go
// Step is one screen of a lesson: a teach card, a practice block, a
// reading text, or a mixed checkpoint.
type Step struct {
	ID    string
	Kind  string // teach | practice | reading | checkpoint
	Title string

	Markdown   string // teach/practice reminder, or reading (Serbian side)
	MarkdownRU string // reading: Russian translation

	AlsoOK    []string // extra tokens the lexicon guard allows in this step
	Mixed     bool     // practice: opt out of the difficulty-order check
	Exercises []Exercise
}
```
В `Exercise` добавь:
```go
	Options []string   // choice: shown options (answer hidden from clients)
	Answer  string     // choice: the correct option
	Bank    []string   // word_bank: shuffled chips (accept holds full answers)
	Pairs   [][2]string // match: [Serbian, Russian] pairs (order hidden)
```

- [ ] **Шаг 4: Напиши `manifest.go`**

```go
package content

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type manifestFile struct {
	Lesson   string   `yaml:"lesson"`
	Title    string   `yaml:"title"`
	Subtitle string   `yaml:"subtitle"`
	Teaches  []string `yaml:"teaches"`
	Steps    []struct {
		ID        string      `yaml:"id"`
		Kind      string      `yaml:"kind"`
		Title     string      `yaml:"title"`
		MD        string      `yaml:"md"`
		AlsoOK    []string    `yaml:"also_ok"`
		Mixed     bool        `yaml:"mixed"`
		Exercises []exerciseYAML `yaml:"exercises"`
	} `yaml:"steps"`
}

var stepKinds = map[string]bool{"teach": true, "practice": true, "reading": true, "checkpoint": true}

// parseManifest loads a lessons/NN.yaml manifest into a Lesson with Steps.
// Course.Exercises is populated separately (see synthesizeExercises).
func parseManifest(dir, rel string) (*Lesson, error) {
	var mf manifestFile
	if err := readYAML(filepath.Join(dir, rel), &mf); err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	l := &Lesson{
		ID: mf.Lesson, Title: mf.Title, Subtitle: mf.Subtitle,
		Manifest: true, MarkdownPath: rel, Teaches: mf.Teaches,
	}
	seenStep, seenEx := map[string]bool{}, map[string]bool{}
	for _, s := range mf.Steps {
		if !stepKinds[s.Kind] {
			return nil, fmt.Errorf("%s: step %q: unknown kind %q", rel, s.ID, s.Kind)
		}
		if s.ID == "" || !hasLessonPrefix(s.ID, mf.Lesson) {
			return nil, fmt.Errorf("%s: step %q: id must start with %q.", rel, s.ID, mf.Lesson)
		}
		if seenStep[s.ID] {
			return nil, fmt.Errorf("%s: duplicate step id %q", rel, s.ID)
		}
		seenStep[s.ID] = true

		step := Step{ID: s.ID, Kind: s.Kind, Title: s.Title, AlsoOK: s.AlsoOK, Mixed: s.Mixed}
		if s.MD != "" {
			md, err := os.ReadFile(filepath.Join(dir, "lessons", s.MD))
			if err != nil {
				return nil, fmt.Errorf("%s: step %s: %w", rel, s.ID, err)
			}
			if s.Kind == "reading" {
				_, step.Markdown, step.MarkdownRU = extractReading("<!-- reading -->\n" + string(md) + "\n<!-- /reading -->")
			} else {
				step.Markdown = string(md)
			}
		}
		for _, e := range s.Exercises {
			if seenEx[e.ID] {
				return nil, fmt.Errorf("%s: duplicate exercise id %q", rel, e.ID)
			}
			seenEx[e.ID] = true
			ex, err := decodeExercise(rel, e)
			if err != nil {
				return nil, err
			}
			step.Exercises = append(step.Exercises, ex)
		}
		switch s.Kind {
		case "teach":
			if step.Markdown == "" || len(step.Exercises) > 0 {
				return nil, fmt.Errorf("%s: step %s: teach needs md and no exercises", rel, s.ID)
			}
		case "practice", "checkpoint":
			if len(step.Exercises) == 0 {
				return nil, fmt.Errorf("%s: step %s: %s needs exercises", rel, s.ID, s.Kind)
			}
		case "reading":
			if step.Markdown == "" {
				return nil, fmt.Errorf("%s: step %s: reading needs md", rel, s.ID)
			}
		}
		l.Steps = append(l.Steps, step)
	}
	if len(l.Steps) == 0 {
		return nil, fmt.Errorf("%s: no steps", rel)
	}
	// Legacy-compat view: expose the first reading step as Lesson.Reading so
	// the existing DTO field and content guard-tests keep working.
	for _, s := range l.Steps {
		if s.Kind == "reading" && l.Reading == "" {
			l.Reading, l.ReadingRU = s.Markdown, s.MarkdownRU
		}
	}
	return l, nil
}

func hasLessonPrefix(id, lesson string) bool {
	return len(id) > len(lesson)+1 && id[:len(lesson)+1] == lesson+"."
}
```

- [ ] **Шаг 5: Вынеси `decodeExercise` и `exerciseYAML` в `load.go`**

В `load.go` замени анонимную структуру внутри `exerciseFile.Blocks[].Exercises` на именованный тип и функцию:
```go
type exerciseYAML struct {
	ID      string    `yaml:"id"`
	Type    string    `yaml:"type"`
	Prompt  string    `yaml:"prompt"`
	Explain string    `yaml:"explain"`
	Sample  string    `yaml:"sample"`
	Meta    string    `yaml:"meta"`
	Say     string    `yaml:"say"`
	Forms   []string  `yaml:"forms"`
	Options []string  `yaml:"options"`
	Answer  string    `yaml:"answer"`
	Bank    []string  `yaml:"bank"`
	Pairs   [][]string `yaml:"pairs"`
	Accept  yaml.Node `yaml:"accept"`
}

// decodeExercise turns one YAML exercise into a content.Exercise, decoding
// the polymorphic `accept` field per type. `rel` is the source path for errors.
func decodeExercise(rel string, e exerciseYAML) (Exercise, error) {
	ex := Exercise{
		ID: e.ID, Type: e.Type, Prompt: e.Prompt, Explain: e.Explain,
		Sample: e.Sample, Meta: e.Meta, Forms: e.Forms, Say: e.Say,
		Options: e.Options, Answer: e.Answer, Bank: e.Bank,
	}
	for _, p := range e.Pairs {
		if len(p) != 2 {
			return Exercise{}, fmt.Errorf("%s: exercise %s: each pair needs [sr, ru]", rel, e.ID)
		}
		ex.Pairs = append(ex.Pairs, [2]string{p[0], p[1]})
	}
	switch {
	case e.Type == "conjugate":
		if err := e.Accept.Decode(&ex.AcceptForms); err != nil {
			return Exercise{}, fmt.Errorf("%s: exercise %s: accept must be a list of lists: %w", rel, e.ID, err)
		}
		if len(ex.AcceptForms) != len(e.Forms) {
			return Exercise{}, fmt.Errorf("%s: exercise %s: %d forms but %d accept rows", rel, e.ID, len(e.Forms), len(ex.AcceptForms))
		}
	case e.Type == "choice":
		if len(e.Options) < 2 || e.Answer == "" {
			return Exercise{}, fmt.Errorf("%s: exercise %s: choice needs >=2 options and an answer", rel, e.ID)
		}
		if !slicesContains(e.Options, e.Answer) {
			return Exercise{}, fmt.Errorf("%s: exercise %s: answer %q not among options", rel, e.ID, e.Answer)
		}
	case e.Type == "match":
		if len(ex.Pairs) < 2 || len(ex.Pairs) > 6 {
			return Exercise{}, fmt.Errorf("%s: exercise %s: match needs 2..6 pairs", rel, e.ID)
		}
	case autoTypes[e.Type]:
		if err := e.Accept.Decode(&ex.Accept); err != nil {
			return Exercise{}, fmt.Errorf("%s: exercise %s: accept must be a list of strings: %w", rel, e.ID, err)
		}
		if len(ex.Accept) == 0 {
			return Exercise{}, fmt.Errorf("%s: exercise %s: %s needs a non-empty accept list", rel, e.ID, e.Type)
		}
		if e.Type == "word_bank" && len(ex.Bank) < 2 {
			return Exercise{}, fmt.Errorf("%s: exercise %s: word_bank needs a bank of >=2 chips", rel, e.ID)
		}
		if e.Type == "listen" && e.Say == "" {
			return Exercise{}, fmt.Errorf("%s: exercise %s: listen needs say", rel, e.ID)
		}
	case e.Type == "free":
	default:
		return Exercise{}, fmt.Errorf("%s: exercise %s: unknown type %q", rel, e.ID, e.Type)
	}
	return ex, nil
}

func slicesContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
```
Обнови `autoTypes`:
```go
var autoTypes = map[string]bool{
	"translate": true, "fill_blank": true, "fix_error": true, "listen": true,
	"word_bank": true, "choice": true, "match": true,
}
```
> Примечание: `listen` в манифесте пока без проверки файла аудио (задача-контент запустит `tts.py`). Проверку `os.Stat(audio)` из старого `Load` оставь только в легаси-ветке.

В существующем `Load` перепиши цикл разбора блоков `exercises/*.yaml` на вызов `decodeExercise(rel, e)`.

- [ ] **Шаг 6: Диспатч в `load.go`**

В цикле `for id, le := range cf.Lessons` замени тело на:
```go
		var l *Lesson
		switch {
		case le.File == "":
			l = &Lesson{ID: id, Title: le.Title, Subtitle: le.Subtitle, Planned: true}
		case strings.HasSuffix(le.File, ".yaml"):
			ml, err := parseManifest(dir, le.File)
			if err != nil {
				return nil, err
			}
			ml.Title, ml.Subtitle = pick(ml.Title, le.Title), pick(ml.Subtitle, le.Subtitle)
			l = ml
		default:
			l = &Lesson{ID: id, Title: le.Title, Subtitle: le.Subtitle, MarkdownPath: le.File}
			md, err := os.ReadFile(filepath.Join(dir, le.File))
			if err != nil {
				return nil, fmt.Errorf("%s: %w", le.File, err)
			}
			l.Markdown, l.Reading, l.ReadingRU = extractReading(string(md))
		}
		c.Lessons[id] = l
```
Хелпер:
```go
func pick(a, b string) string { if a != "" { return a }; return b }
```
Добавь `"strings"` в импорт, если нет.

- [ ] **Шаг 7: Запусти тесты**

Run: `/opt/homebrew/bin/go test ./server/internal/content/ -run TestManifest -v`
Expected: PASS.
Run: `/opt/homebrew/bin/go test ./server/internal/content/ -v`
Expected: `TestRealContentLoads` может упасть на счётчиках блоков (легаси-синтез — задача 2). Остальное — PASS. Если падает только `TestRealContentLoads` — ок, продолжай.

- [ ] **Шаг 8: Коммит**

```bash
git add server/internal/content/
git commit -m "feat(content): lesson manifest model (lessons/NN.yaml -> steps)"
```

---

## Задача 2: Синтетические шаги для легаси-уроков + `Course.Exercises`

**Files:**
- Modify: `server/internal/content/manifest.go` (добавить `synthesizeSteps`, `collectExercises`)
- Modify: `server/internal/content/load.go` (вызвать после загрузки всех уроков)
- Modify: `server/internal/content/real_test.go`

**Interfaces:**
- Consumes: `Lesson.Steps` (задача 1), `Lesson.Markdown/Reading/ReadingRU`, `Course.Exercises` (легаси-разбор `exercises/*.yaml`).
- Produces:
  - `synthesizeSteps(l *Lesson, blocks []ExerciseBlock) []Step` — для легаси-урока строит `[teach(markdown)] + practice(блок)… + [reading]`.
  - После `Load`: у **каждого** урока с контентом заполнен `Lesson.Steps`; `Course.Exercises[id]` заполнен и для манифест-уроков (по одному синтетическому блоку `ExerciseBlock{ID: step.ID, Title: step.Title}` на каждый `practice`/`checkpoint`-шаг).

**Заметки:**
- Легаси-урок: `Steps` строится так —
  1. если `l.Markdown != ""` → шаг `{ID: id+".teach", Kind: "teach", Title: "Теория", Markdown: l.Markdown}`;
  2. по каждому блоку из `Course.Exercises[id]` → `{ID: id+"."+block.ID, Kind: "practice", Title: block.Title, Exercises: block.Exercises}` (запомни `block.Instruction` в `Step.Markdown`? нет — оставь пустым, инструкция уедет в синтетический блок обратно);
  3. если `l.Reading != ""` → `{ID: id+".reading", Kind: "reading", Title: "Текст для чтения", Markdown: l.Reading, MarkdownRU: l.ReadingRU}`.
- Манифест-урок: `Course.Exercises[id]` собирается из шагов —
  `for _, s := range l.Steps { if len(s.Exercises) > 0 { blocks = append(blocks, ExerciseBlock{ID: s.ID, Title: s.Title, Exercises: s.Exercises}) } }`.
- `findExercise` в `api.go` итерирует `Course.Exercises` — не трогаем, работает для обеих моделей, `block` == id шага.
- Порядок в `Load`: сначала весь текущий разбор (`course.yaml`, `exercises/*.yaml`, `vocab`, `false-friends`), затем финальный проход по `c.Lessons`: для манифеста — `collectExercises`, для легаси с контентом — `synthesizeSteps`.

- [ ] **Шаг 1: Тест — легаси-урок 04 получает синтетические шаги**

Добавь в `real_test.go`:
```go
func TestLegacyLessonSynthesizesSteps(t *testing.T) {
	c, err := Load("../../../content")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	l := c.Lessons["04"] // legacy .md + exercises/04.yaml
	if l.Manifest {
		t.Fatal("lesson 04 should be legacy")
	}
	if len(l.Steps) < 3 {
		t.Fatalf("lesson 04: %d synthetic steps, want >=3", len(l.Steps))
	}
	if l.Steps[0].Kind != "teach" || l.Steps[0].Markdown == "" {
		t.Errorf("step 0 not a teach card: %+v", l.Steps[0])
	}
	last := l.Steps[len(l.Steps)-1]
	if last.Kind != "reading" || last.MarkdownRU == "" {
		t.Errorf("last step not reading: %+v", last)
	}
	hasPractice := false
	for _, s := range l.Steps {
		if s.Kind == "practice" && len(s.Exercises) > 0 {
			hasPractice = true
		}
	}
	if !hasPractice {
		t.Error("no practice step with exercises")
	}
}

func TestManifestExercisesCollected(t *testing.T) {
	c, _ := Load("testdata/content")
	blocks := c.Exercises["90"]
	if len(blocks) != 2 { // practice 90.2 + checkpoint 90.4
		t.Fatalf("lesson 90: %d exercise blocks, want 2", len(blocks))
	}
	if blocks[0].ID != "90.2" {
		t.Errorf("block 0 id = %q, want 90.2", blocks[0].ID)
	}
}
```

- [ ] **Шаг 2: Запусти — падает**

Run: `/opt/homebrew/bin/go test ./server/internal/content/ -run 'TestLegacyLessonSynthesizes|TestManifestExercisesCollected' -v`
Expected: FAIL (`l.Steps` пуст для легаси; `c.Exercises["90"]` пуст).

- [ ] **Шаг 3: Реализуй `synthesizeSteps` и `collectExercises` в `manifest.go`**

```go
func collectExercises(l *Lesson) []ExerciseBlock {
	var out []ExerciseBlock
	for _, s := range l.Steps {
		if len(s.Exercises) > 0 {
			out = append(out, ExerciseBlock{ID: s.ID, Title: s.Title, Exercises: s.Exercises})
		}
	}
	return out
}

func synthesizeSteps(l *Lesson, blocks []ExerciseBlock) []Step {
	var steps []Step
	if l.Markdown != "" {
		steps = append(steps, Step{ID: l.ID + ".teach", Kind: "teach", Title: "Теория", Markdown: l.Markdown})
	}
	for _, b := range blocks {
		steps = append(steps, Step{ID: l.ID + "." + b.ID, Kind: "practice", Title: b.Title, Exercises: b.Exercises})
	}
	if l.Reading != "" {
		steps = append(steps, Step{ID: l.ID + ".reading", Kind: "reading", Title: "Текст для чтения",
			Markdown: l.Reading, MarkdownRU: l.ReadingRU})
	}
	return steps
}
```
> Синтетические practice-шаги легаси используют id блока `b.ID` ("A"/"B"…), значит `Step.ID` = `"04.A"`. Атрибуты попыток пишутся с `block = "04.A"` — согласовано с `findExercise`, который вернёт `block.ID` из `Course.Exercises`. Для легаси `Course.Exercises` НЕ пересобираем — оставляем как разобрано из `exercises/04.yaml` (id блоков "A"/"B"…). Значит в `api.findExercise` `block` вернётся как "A", а `Step.ID` — "04.A". **Расхождение.** Решение: для легаси синтетические шаги берут `Step.ID = b.ID` (без префикса урока), т.е. "A". Аналогично teach → `Step.ID = "teach"`, reading → `"reading"`. Фронт всё равно ключует по `lesson.id + step.id`.

Перепиши `synthesizeSteps` с `Step.ID: b.ID` (не `l.ID+"."+b.ID`), teach → `"teach"`, reading → `"reading"`.

- [ ] **Шаг 4: Вызов в конце `Load`**

Перед `return c, nil`:
```go
	for _, l := range c.Lessons {
		switch {
		case l.Manifest:
			c.Exercises[l.ID] = collectExercises(l)
		case !l.Planned:
			l.Steps = synthesizeSteps(l, c.Exercises[l.ID])
		}
	}
```

- [ ] **Шаг 5: Обнови существующие ожидания `real_test.go`**

`TestRealContentLoads` — замени проверки числа блоков на проверку шагов; `phases != 3` пока оставь (карта — задача 9). Конкретно:
- `len(c.Exercises["01"]) < 4` → оставь (01 ещё легаси, блоки A–E).
- добавь: для `id` из `{"01","02","03","04","05"}` — `len(c.Lessons[id].Steps) >= 3` и первый шаг `Kind=="teach"`.

- [ ] **Шаг 6: Запусти весь пакет**

Run: `/opt/homebrew/bin/go test ./server/internal/content/ -v`
Expected: PASS (все, включая `TestRealContentLoads`).

- [ ] **Шаг 7: Коммит**

```bash
git add server/internal/content/
git commit -m "feat(content): synthesize steps for legacy lessons; collect manifest exercises"
```

---

## Задача 3: Рекурсивный watch `content/lessons/`

**Files:**
- Modify: `server/internal/content/watch.go`
- Modify: `server/internal/content/load_test.go` (или новый `watch_test.go`)

**Interfaces:**
- Consumes: `Load` (задачи 1–2).
- Produces: watcher, который перечитывает контент при изменении файла в любой поддиректории `content/lessons/**` и при создании новой поддиректории добавляет её в отслеживание.

- [ ] **Шаг 1: Тест — правка фрагмента в подпапке триггерит reload**

`server/internal/content/watch_test.go`:
```go
package content

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchReloadsOnNestedFragmentChange(t *testing.T) {
	dir := t.TempDir()
	mustCopyTree(t, "testdata/content", dir) // helper: recursive copy
	get, _, err := Watch(dir)
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	before := get().Lessons["90"].Steps[0].Markdown

	frag := filepath.Join(dir, "lessons", "90", "1-intro.md")
	if err := os.WriteFile(frag, []byte("# Intro\n\nIzmenjeno. *(изменено)*\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if get().Lessons["90"].Steps[0].Markdown != before {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("content did not reload after nested fragment change")
}
```
Хелпер `mustCopyTree` — рекурсивная копия через `filepath.WalkDir` + `os.MkdirAll`/`os.ReadFile`/`os.WriteFile` (напиши в тесте).

- [ ] **Шаг 2: Запусти — падает**

Run: `/opt/homebrew/bin/go test ./server/internal/content/ -run TestWatchReloadsOnNestedFragmentChange -v`
Expected: FAIL (watcher не видит `lessons/90/`).

- [ ] **Шаг 3: Рекурсивная подписка в `watch.go`**

Замени блок `for _, sub := range []string{"lessons", "exercises"}` на обход:
```go
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			_ = w.Add(p)
		}
		return nil
	})
```
В обработчике событий, при `event.Op&fsnotify.Create != 0` и если `event.Name` — каталог, `_ = w.Add(event.Name)` перед постановкой таймера reload. Добавь импорты `os`, `io/fs` если нужно; `fsnotify` уже есть.

- [ ] **Шаг 4: Запусти**

Run: `/opt/homebrew/bin/go test ./server/internal/content/ -run TestWatch -v`
Expected: PASS.

- [ ] **Шаг 5: Коммит**

```bash
git add server/internal/content/
git commit -m "feat(content): recursive watch for lessons/ subdirectories"
```

---

## Задача 4: Персона — `persona.yaml` + подстановка `{...}`

**Files:**
- Create: `server/internal/content/persona.go`
- Create: `server/internal/content/persona_test.go`
- Modify: `server/internal/content/load.go` (применить подстановку в конце)
- Modify: `server/internal/content/types.go` (`Course.Persona map[string]string` — опционально, для дебага)
- Create: `content/persona.yaml`
- Create: `server/internal/content/testdata/content/persona.yaml`

**Interfaces:**
- Produces:
  - `loadPersona(dir string) (map[string]string, error)` — читает `persona.yaml` (отсутствует → пустая мапа, без ошибки).
  - `interpolate(s string, p map[string]string) string` — заменяет `{key}` на `p[key]`; неизвестные `{key}` оставляет как есть.
  - Подстановка применяется к `Step.Markdown`, `Step.MarkdownRU`, `Exercise.Prompt`, `Exercise.Sample`, `ExerciseBlock` инструкциям, `Lesson.Subtitle`. **НЕ** применяется к `Accept`, `Answer`, `Bank`, `Pairs`, `Options`, `Say`, `Cyrillic`.

- [ ] **Шаг 1: Тест**

`server/internal/content/persona_test.go`:
```go
package content

import "testing"

func TestInterpolatePersona(t *testing.T) {
	p := map[string]string{"name_latin": "Griša", "city_ru": "Нови-Сад"}
	got := interpolate("Zdravo, ja sam {name_latin}. Živim u {city_ru}. {unknown}", p)
	want := "Zdravo, ja sam Griša. Živim u Нови-Сад. {unknown}"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestPersonaAppliedToPrompts(t *testing.T) {
	c, err := Load("testdata/content")
	if err != nil {
		t.Fatal(err)
	}
	// fixture persona.yaml sets name_latin: Test; fixture 90.yaml has no {name} —
	// assert interpolation is wired without error and leaves plain text intact.
	if c.Lessons["90"].Steps[0].Markdown == "" {
		t.Fatal("step markdown lost after interpolation")
	}
}
```

Фикстура `server/internal/content/testdata/content/persona.yaml`:
```yaml
name: "Тест"
name_latin: "Test"
city: "Novi Sad"
city_ru: "Нови-Сад"
job: "programer"
job_ru: "программист"
native_ru: "русский"
```

- [ ] **Шаг 2: Запусти — падает** (`interpolate` undefined)

Run: `/opt/homebrew/bin/go test ./server/internal/content/ -run 'TestInterpolate|TestPersona' -v`

- [ ] **Шаг 3: `persona.go`**

```go
package content

import (
	"path/filepath"
	"regexp"
)

var personaRe = regexp.MustCompile(`\{([a-z_]+)\}`)

func loadPersona(dir string) (map[string]string, error) {
	var raw map[string]string
	if err := readYAML(filepath.Join(dir, "persona.yaml"), &raw); err != nil {
		return map[string]string{}, nil // absent or unreadable -> no personalization
	}
	return raw, nil
}

func interpolate(s string, p map[string]string) string {
	if len(p) == 0 || s == "" {
		return s
	}
	return personaRe.ReplaceAllStringFunc(s, func(m string) string {
		key := m[1 : len(m)-1]
		if v, ok := p[key]; ok {
			return v
		}
		return m
	})
}
```

- [ ] **Шаг 4: Применение в `Load`**

После сборки шагов (конец `Load`):
```go
	persona, _ := loadPersona(dir)
	for _, l := range c.Lessons {
		l.Subtitle = interpolate(l.Subtitle, persona)
		for i := range l.Steps {
			l.Steps[i].Markdown = interpolate(l.Steps[i].Markdown, persona)
			l.Steps[i].MarkdownRU = interpolate(l.Steps[i].MarkdownRU, persona)
			for j := range l.Steps[i].Exercises {
				l.Steps[i].Exercises[j].Prompt = interpolate(l.Steps[i].Exercises[j].Prompt, persona)
				l.Steps[i].Exercises[j].Sample = interpolate(l.Steps[i].Exercises[j].Sample, persona)
			}
		}
		l.Markdown = interpolate(l.Markdown, persona)     // legacy view
		l.Reading = interpolate(l.Reading, persona)
		l.ReadingRU = interpolate(l.ReadingRU, persona)
	}
	// rebuild Course.Exercises from interpolated steps for manifest lessons
	for _, l := range c.Lessons {
		if l.Manifest {
			c.Exercises[l.ID] = collectExercises(l)
		}
	}
```

- [ ] **Шаг 5: `content/persona.yaml`** (реальный)

```yaml
name: "Гриша"
name_latin: "Griša"
city: "Novi Sad"
city_ru: "Нови-Сад"
job: "programer"
job_ru: "программист"
native_ru: "русский"
```

- [ ] **Шаг 6: Запусти пакет**

Run: `/opt/homebrew/bin/go test ./server/internal/content/ -v`
Expected: PASS.

- [ ] **Шаг 7: Коммит**

```bash
git add server/internal/content/ content/persona.yaml
git commit -m "feat(content): persona.yaml + {placeholder} interpolation"
```

---

## Задача 5: Прогресс по шагам — стор

**Files:**
- Modify: `server/internal/store/store.go` (schema + методы + reset)
- Modify: `server/internal/store/import.go`
- Modify: `server/internal/store/store_test.go` (`TRUNCATE` + тесты)

**Interfaces:**
- Produces (на `*UserStore`):
  - `SetStepStatus(lesson, step, status string, now time.Time) error` — upsert в `lesson_step_progress`; `completed_at` ставится один раз при первом `status=="done"`.
  - `StepStatuses(lesson string) (map[string]string, error)` — `step -> status`.
- `ResetLesson(lesson)` и `ResetExercises()` дополнительно удаляют строки `lesson_step_progress`.

- [ ] **Шаг 1: Тест**

В `store_test.go` (хелпер `newUser(t) (*Store, *UserStore)` уже есть, аккаунт «Гриша»):
```go
func TestStepProgress(t *testing.T) {
	s, u := newUser(t)
	_ = s
	now := time.Now()

	if err := u.SetStepStatus("01", "01.1", "in_progress", now); err != nil {
		t.Fatal(err)
	}
	if err := u.SetStepStatus("01", "01.1", "done", now); err != nil {
		t.Fatal(err)
	}
	if err := u.SetStepStatus("01", "01.2", "in_progress", now); err != nil {
		t.Fatal(err)
	}
	m, err := u.StepStatuses("01")
	if err != nil {
		t.Fatal(err)
	}
	if m["01.1"] != "done" || m["01.2"] != "in_progress" {
		t.Fatalf("statuses = %v", m)
	}
	if err := u.ResetLesson("01"); err != nil {
		t.Fatal(err)
	}
	if m, _ = u.StepStatuses("01"); len(m) != 0 {
		t.Fatalf("after reset: %v", m)
	}
}
```
`"time"` уже импортирован в `store_test.go`.

- [ ] **Шаг 2: Запусти — падает**

Run: `/opt/homebrew/bin/go test ./server/internal/store/ -run TestStepProgress -v`

- [ ] **Шаг 3: Схема**

В `schemaSQL` перед `CREATE INDEX`:
```sql
CREATE TABLE IF NOT EXISTS lesson_step_progress (
	user_name    TEXT NOT NULL,
	lesson       TEXT NOT NULL,
	step         TEXT NOT NULL,
	status       TEXT NOT NULL,
	completed_at TEXT,
	PRIMARY KEY (user_name, lesson, step)
);
```
Добавь эту же таблицу в список `stmts` внутри `migrateV1toV2` (после `schemaSQL(false)` она уже создастся; отдельный INSERT не нужен — в v1 её не было).

- [ ] **Шаг 4: Методы**

Рядом с `SetLessonStatus`:
```go
// SetStepStatus upserts one lesson step's status ("in_progress" | "done").
func (u *UserStore) SetStepStatus(lesson, step, status string, now time.Time) error {
	iso := now.UTC().Format(time.RFC3339)
	var completed any
	if status == "done" {
		completed = iso
	}
	_, err := u.db.Exec(`
INSERT INTO lesson_step_progress (user_name, lesson, step, status, completed_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(user_name, lesson, step) DO UPDATE SET
	status = excluded.status,
	completed_at = COALESCE(lesson_step_progress.completed_at, excluded.completed_at)`,
		u.user, lesson, step, status, completed)
	return err
}

// StepStatuses returns step id -> status for one lesson.
func (u *UserStore) StepStatuses(lesson string) (map[string]string, error) {
	rows, err := u.db.Query(`SELECT step, status FROM lesson_step_progress WHERE user_name = ? AND lesson = ?`, u.user, lesson)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var s, st string
		if err := rows.Scan(&s, &st); err != nil {
			return nil, err
		}
		out[s] = st
	}
	return out, rows.Err()
}
```
В `ResetLesson` (в транзакции) добавь:
```go
	if _, err := tx.Exec(`DELETE FROM lesson_step_progress WHERE user_name = ? AND lesson = ?`, u.user, lesson); err != nil {
		return err
	}
```
В `ResetExercises`:
```go
	if _, err := tx.Exec(`DELETE FROM lesson_step_progress WHERE user_name = ?`, u.user); err != nil {
		return err
	}
```

- [ ] **Шаг 5: `import.go`**

Добавь в срез `tables`:
```go
		{`SELECT user_name, lesson, step, status, completed_at FROM lesson_step_progress`,
			`INSERT INTO lesson_step_progress (user_name, lesson, step, status, completed_at) VALUES (?, ?, ?, ?, ?)`, 5},
```

- [ ] **Шаг 6: `store_test.go` TRUNCATE**

`TRUNCATE users, srs_cards, reviews, attempts, lesson_progress` → добавь `, lesson_step_progress`.

- [ ] **Шаг 7: Запусти**

Run: `/opt/homebrew/bin/go test ./server/internal/store/ -v`
Expected: PASS.

- [ ] **Шаг 8: Коммит**

```bash
git add server/internal/store/
git commit -m "feat(store): per-step lesson progress table + methods"
```

---

## Задача 6: API — шаги в `lessonDTO`, роут статуса шага

**Files:**
- Modify: `server/internal/api/dto.go`
- Modify: `server/internal/api/api.go` (`getLesson`, новый роут, регистрация)
- Modify: `server/internal/api/api_test.go`

**Interfaces:**
- Consumes: `content.Lesson.Steps`, `store.SetStepStatus`, `store.StepStatuses`.
- Produces:
  - `GET /api/lessons/{id}` → `lessonDTO` c полем `steps []stepDTO`.
  - `stepDTO{ID, Kind, Title, Markdown, MarkdownRU string; ExerciseIDs []string; Status string}` — **без** вложенных `Exercise` (упражнения отдаёт отдельный эндпоинт).
  - `POST /api/lessons/{id}/steps/{step}` тело `{"status":"in_progress"|"done"}` → `204`; невалидный статус → `400`; неизвестный урок → `404`. Побочно: если статус урока ещё не `done`, апсертит `lesson_progress` в `in_progress`.

- [ ] **Шаг 1: Тесты в `api_test.go`**

Хелперы уже есть: `newTestAPI(t) (http.Handler, *store.Store)` (грузит `../content/testdata/content`, аккаунт `"tester"`), `do(h, method, path, body)`, `doAs(...)`, `decodeBody[T](t, rr)`. Фикстура-урок `90` добавлена в задаче 1.
```go
func TestGetLessonIncludesSteps(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "GET", "/api/lessons/90", "")
	if rr.Code != 200 {
		t.Fatalf("code %d: %s", rr.Code, rr.Body)
	}
	var dto struct {
		Steps []struct {
			ID, Kind, Status string
			ExerciseIDs      []string `json:"exercise_ids"`
		} `json:"steps"`
	}
	json.Unmarshal(rr.Body.Bytes(), &dto)
	if len(dto.Steps) != 4 || dto.Steps[0].Kind != "teach" {
		t.Fatalf("steps: %+v", dto.Steps)
	}
	if len(dto.Steps[1].ExerciseIDs) == 0 || dto.Steps[1].ExerciseIDs[0] != "90.2.1" {
		t.Errorf("practice step exercise ids: %v", dto.Steps[1].ExerciseIDs)
	}
	if body := rr.Body.String(); strings.Contains(body, `"answer"`) || strings.Contains(body, `"accept"`) {
		t.Error("lesson payload leaked answer/accept")
	}
}

func TestSetStepStatus(t *testing.T) {
	h, _ := newTestAPI(t)
	if rr := do(h, "POST", "/api/lessons/90/steps/90.1", `{"status":"done"}`); rr.Code != 204 {
		t.Fatalf("code %d: %s", rr.Code, rr.Body)
	}
	if rr := do(h, "POST", "/api/lessons/90/steps/90.1", `{"status":"nope"}`); rr.Code != 400 {
		t.Fatalf("bad status: code %d", rr.Code)
	}
	if rr := do(h, "POST", "/api/lessons/99/steps/x", `{"status":"done"}`); rr.Code != 404 {
		t.Fatalf("unknown lesson: code %d", rr.Code)
	}
}
```

- [ ] **Шаг 2: Запусти — падает**

Run: `/opt/homebrew/bin/go test ./server/internal/api/ -run 'TestGetLessonIncludesSteps|TestSetStepStatus' -v`

- [ ] **Шаг 3: DTO**

В `dto.go`:
```go
type stepDTO struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	Title       string   `json:"title"`
	Markdown    string   `json:"markdown,omitempty"`
	MarkdownRU  string   `json:"markdown_ru,omitempty"`
	ExerciseIDs []string `json:"exercise_ids,omitempty"`
	Status      string   `json:"status"` // not_started | in_progress | done
}
```
В `lessonDTO` добавь `Steps []stepDTO \`json:"steps,omitempty"\``.

- [ ] **Шаг 4: `getLesson` отдаёт шаги**

После получения `st` (статуса урока):
```go
	stepSt, _ := us.StepStatuses(id)
	steps := make([]stepDTO, 0, len(l.Steps))
	for _, s := range l.Steps {
		d := stepDTO{ID: s.ID, Kind: s.Kind, Title: s.Title, Markdown: s.Markdown, MarkdownRU: s.MarkdownRU}
		for _, e := range s.Exercises {
			d.ExerciseIDs = append(d.ExerciseIDs, e.ID)
		}
		if d.Status = stepSt[s.ID]; d.Status == "" {
			d.Status = "not_started"
		}
		steps = append(steps, d)
	}
```
и передай `Steps: steps` в `lessonDTO{...}`.

- [ ] **Шаг 5: роут статуса шага**

Регистрация в `Handler`:
```go
	mux.HandleFunc("POST /api/lessons/{id}/steps/{step}", h.setStepStatus)
```
Хендлер:
```go
func (h handlers) setStepStatus(w http.ResponseWriter, r *http.Request) {
	us, ok := h.user(w, r)
	if !ok {
		return
	}
	id, step := r.PathValue("id"), r.PathValue("step")
	if h.Course().Lessons[id] == nil {
		fail(w, 404, "unknown lesson")
		return
	}
	var req struct{ Status string `json:"status"` }
	if err := decode(r, &req); err != nil || (req.Status != "in_progress" && req.Status != "done") {
		fail(w, 400, "status must be in_progress or done")
		return
	}
	now := h.Now()
	if err := us.SetStepStatus(id, step, req.Status, now); err != nil {
		fail(w, 500, err.Error())
		return
	}
	if st, _ := us.LessonStatus(id); st != "done" {
		_ = us.SetLessonStatus(id, "in_progress", now)
	}
	w.WriteHeader(204)
}
```

- [ ] **Шаг 6: Запусти пакет API**

Run: `/opt/homebrew/bin/go test ./server/internal/api/ -v`
Expected: PASS.

- [ ] **Шаг 7: Коммит**

```bash
git add server/internal/api/
git commit -m "feat(api): lesson steps in DTO + POST /lessons/{id}/steps/{step}"
```

---

## Задача 7: Фронт — типы и API-клиент шагов

**Files:**
- Modify: `web/src/types.ts`
- Modify: `web/src/api.ts`
- Modify: `web/src/api.test.ts` (если покрывает форму `Lesson`)

**Interfaces:**
- Produces:
  - `Step` интерфейс; `Lesson.steps: Step[]`.
  - `ExerciseType` += `'choice' | 'word_bank' | 'match'`.
  - `Exercise` += `options?: string[]`, `bank?: string[]`, `left?: string[]`, `right?: string[]`.
  - `api.setStepStatus(lesson, step, status)` → `POST /lessons/{lesson}/steps/{step}`.

- [ ] **Шаг 1: Тест (Vitest) в `web/src/api.test.ts`**

Добавь кейс: мок `fetch` для `POST /api/lessons/01/steps/01.2` возвращает 204; `api.setStepStatus('01','01.2','done')` резолвится без throw и дергает правильный URL/метод/тело. (Повтори паттерн существующих тестов в файле.)

- [ ] **Шаг 2: Запусти — падает**

Run: `cd web && npx vitest run src/api.test.ts`

- [ ] **Шаг 3: Типы**

```ts
export type StepKind = 'teach' | 'practice' | 'reading' | 'checkpoint'

export interface Step {
  id: string
  kind: StepKind
  title: string
  markdown?: string
  markdown_ru?: string
  exercise_ids?: string[]
  status: LessonStatus
}
```
В `Lesson` добавь `steps?: Step[]`.
`ExerciseType` → `'translate' | 'fill_blank' | 'fix_error' | 'conjugate' | 'free' | 'listen' | 'choice' | 'word_bank' | 'match'`.
В `Exercise` добавь `options?: string[]`, `bank?: string[]`, `left?: string[]`, `right?: string[]`.

- [ ] **Шаг 4: API-клиент**

В `api.ts` рядом с `completeLesson`:
```ts
  setStepStatus: (lesson: string, step: string, status: 'in_progress' | 'done') =>
    request<void>(`/lessons/${lesson}/steps/${step}`, {
      method: 'POST',
      body: JSON.stringify({ status }),
    }),
```

- [ ] **Шаг 5: Запусти**

Run: `cd web && npx vitest run src/api.test.ts && npx vue-tsc --noEmit`
Expected: PASS, без ошибок типов.

- [ ] **Шаг 6: Коммит**

```bash
git add web/src/types.ts web/src/api.ts web/src/api.test.ts
git commit -m "feat(web): step types + setStepStatus client"
```

---

## Задача 8: Фронт — пошаговый плеер урока

**Files:**
- Rewrite: `web/src/views/LessonView.vue`
- Create: `web/src/components/StepProgress.vue` (тонкая полоса «k из N»)
- Modify: `web/src/stores/course.ts` (не обязательно; статус урока уже есть)

**Interfaces:**
- Consumes: `api.lesson(id)` (теперь со `steps`), `api.exercises(id)`, `api.lessonAttempts(id)`, `api.setStepStatus`, `api.completeLesson`, `store.markDone`.
- Поведение:
  - грузит урок; строит массив шагов из `lesson.steps`;
  - для `practice`/`checkpoint` берёт упражнения из `api.exercises(id)` (сопоставление по `step.exercise_ids`);
  - текущий шаг = первый со `status != 'done'`; всё done → шаг 0; `?step=<id>` перекрывает;
  - `teach`/`reading`: кнопка «Дальше» → `setStepStatus(id, step.id, 'done')`, следующий шаг;
  - `practice`/`checkpoint`: «Дальше» активна, когда на все упражнения шага есть ответ (свой `graded`-учёт или `priors`); нажатие → `setStepStatus('done')`;
  - последний шаг: кнопка «Завершить урок» → `store.markDone(id)` + конфетти (перенести из текущей вьюхи);
  - `planned` урок — прежняя заглушка;
  - легаси-урок работает: `lesson.steps` придёт синтетический (teach/practice…/reading).

- [ ] **Шаг 1: Тест-компонент (Vitest + @vue/test-utils)** `web/src/views/LessonView.test.ts`

Смоук: замокай `api.lesson` → урок с 3 шагами (`teach`,`practice`,`reading`), `api.exercises` → один блок с 1 упражнением. Проверь:
- рендерится только первый шаг (teach markdown виден, practice — нет);
- клик «Дальше» вызывает `api.setStepStatus` с `('X','X.1','done')` и показывает шаг 2.
(Опирайся на существующие тесты компонентов, напр. `TextAnswer.test.ts`, для стиля моков.)

- [ ] **Шаг 2: Запусти — падает**

Run: `cd web && npx vitest run src/views/LessonView.test.ts`

- [ ] **Шаг 3: `StepProgress.vue`**

```vue
<script setup lang="ts">
defineProps<{ current: number; total: number }>()
</script>
<template>
  <div class="flex items-center gap-2 text-xs text-[var(--muted)]">
    <div class="h-1.5 flex-1 rounded-full bg-[var(--border)]">
      <div class="h-full rounded-full bg-[var(--accent)] transition-all"
           :style="{ width: `${Math.round((current / total) * 100)}%` }" />
    </div>
    <span>{{ current }} / {{ total }}</span>
  </div>
</template>
```

- [ ] **Шаг 4: Перепиши `LessonView.vue`**

Полная реализация (следуй существующим утилитам вью: `MarkdownView`, `ReadingText`, `ExerciseBlockView`, `Confetti`, стор). Ключевые куски:

```ts
const steps = computed<Step[]>(() => lesson.value?.steps ?? [])
const idx = ref(0)
const exByStep = ref<Record<string, ExerciseBlock>>({})   // step.id -> synthetic block

function firstUnfinished(): number {
  const i = steps.value.findIndex((s) => s.status !== 'done')
  return i === -1 ? 0 : i
}

async function loadLesson(id: string) {
  /* fetch lesson; if !planned fetch exercises+attempts;
     map blocks by id === step.id; idx.value = route step or firstUnfinished() */
}

const cur = computed(() => steps.value[idx.value])
const curBlock = computed(() => (cur.value ? exByStep.value[cur.value.id] : undefined))

const canAdvance = computed(() => {
  const s = cur.value
  if (!s) return false
  if (s.kind === 'teach' || s.kind === 'reading') return true
  const ids = s.exercise_ids ?? []
  return ids.every((eid) => eid in graded)   // graded: Record<string,boolean>, seeded from priors
})

async function next() {
  const s = cur.value
  if (!s) return
  if (s.status !== 'done') {
    await api.setStepStatus(lesson.value!.id, s.id, 'done')
    s.status = 'done'
  }
  if (idx.value < steps.value.length - 1) idx.value++
  else await finish()
}

async function finish() {
  await store.markDone(lesson.value!.id)
  lesson.value!.status = 'done'
  celebrate.value = true
}
```

Шаблон: шапка с `StepProgress`, тело — `v-if` по `cur.kind`:
- `teach` → `<MarkdownView :source="cur.markdown" />`
- `practice` / `checkpoint` → `<ExerciseBlockView :lesson="lesson.id" :block="curBlock" :priors="priors" @graded=... />` (ExerciseBlock уже эмитит через ExerciseItem — добавь проброс `@graded` из ExerciseBlock, если его нет: он ведёт `graded` внутри; подними наверх через `defineExpose` или новый проп-emit. Проще: в `ExerciseBlock.vue` добавь `const emit = defineEmits<{ graded: [id: string, ok: boolean] }>()` и вызывай в `onGraded`.)
- `reading` → `<ReadingText :serbian="cur.markdown" :translation="cur.markdown_ru" />` + при наличии `curBlock` — упражнения.
Низ: «Назад» (`idx--`, disabled на 0), «Дальше»/«Завершить урок».

- [ ] **Шаг 5: Проброс `graded` из `ExerciseBlock.vue`**

Добавь `emit('graded', id, ok)` в `onGraded`; в `LessonView` слушай и веди `graded[eid]=ok`.

- [ ] **Шаг 6: Запусти фронт-тесты + типы**

Run: `cd web && npx vitest run && npx vue-tsc --noEmit`
Expected: PASS.

- [ ] **Шаг 7: Ручная проверка**

Run: `make dev`, открой `http://localhost:5173`, залогинься, открой урок 04 (легаси) — должен идти по шагам: теория → блоки → чтение → «Завершить урок». Открой урок 02 — то же. Прогресс сохраняется при перезагрузке.

- [ ] **Шаг 8: Коммит**

```bash
git add web/src/
git commit -m "feat(web): step-by-step lesson player"
```

---

## Задача 9: Карта курса A1→B1

**Files:**
- Rewrite: `content/course.yaml`
- Modify: `server/internal/content/real_test.go` (`phases` → 5; ожидания карты)
- Modify: `server/internal/content/types.go` — без изменений (форма `Phase` та же)
- Rewrite: `/Users/grisha/plans/serbian/plan.md` (без коммита)
- Replace: `/Users/grisha/plans/serbian/vocab.md`, `/Users/grisha/plans/serbian/lessons/*.md`, `/Users/grisha/plans/serbian/README.md` — короткий указатель (без коммита)

**Interfaces:**
- Consumes: `content.Load` (диспатч по `file:`).
- Produces: `course.yaml` с 5 «уровнями» (`phases[].id` = `"1"`..`"5"`), ~30 уроками. Уроки 01–05 сохраняют `file:` на текущие легаси-файлы (манифесты подключит Веха 2). Уроки 06–30 — только `title`/`subtitle` (→ `Planned`).

**Заметки:**
- `phaseDTO`/`phaseProgressDTO` читают `id/title/lessons` — форму не меняем, дашборд не ломается.
- Не удаляй `file:` у 03 в этой задаче (Pitanja-контент ещё нужен тесту). Переименование 03 → «Људи око мене» произойдёт в задаче 17.

- [ ] **Шаг 1: Обнови тест карты в `real_test.go`**

```go
	if len(c.Phases) != 5 {
		t.Errorf("phases = %d, want 5", len(c.Phases))
	}
	if got := len(c.Lessons); got < 28 {
		t.Errorf("lessons = %d, want >= 28", got)
	}
	for _, id := range []string{"06", "12", "18", "24", "30"} {
		if !c.Lessons[id].Planned {
			t.Errorf("checkpoint lesson %s should be planned", id)
		}
	}
```
Убери старую строку `if len(c.Phases) != 3`.

- [ ] **Шаг 2: Запусти — падает**

Run: `/opt/homebrew/bin/go test ./server/internal/content/ -run TestRealContentLoads -v`

- [ ] **Шаг 3: Перепиши `content/course.yaml`**

```yaml
title: "Српски језик"

phases:
  - id: "1"
    title: "Уровень 1 — Первый контакт (A1.1)"
    lessons: ["01", "02", "03", "04", "05", "06"]
  - id: "2"
    title: "Уровень 2 — Быт (A1.2)"
    lessons: ["07", "08", "09", "10", "11", "12"]
  - id: "3"
    title: "Уровень 3 — Связная речь (A2.1)"
    lessons: ["13", "14", "15", "16", "17", "18"]
  - id: "4"
    title: "Уровень 4 — Мнения и жизнь (A2.2)"
    lessons: ["19", "20", "21", "22", "23", "24"]
  - id: "5"
    title: "Уровень 5 — Уверенно (B1.1)"
    lessons: ["25", "26", "27", "28", "29", "30"]

lessons:
  "01":
    title: "Поздрави и први контакт"
    subtitle: "поздороваться, ты/вы, вежливые слова"
    file: "lessons/01-pozdravi-i-upoznavanje.md"
  "02":
    title: "Ко си ти"
    subtitle: "jesam, откуда я, языки, профессии"
    file: "lessons/02-zamenice-i-prezent.md"
  "03":
    title: "Pitanja"
    subtitle: "вопросы: da li, ko/šta/gde, переспрос"
    file: "lessons/03-pitanja.md"
  "04":
    title: "Бројеви, цена, сати"
    subtitle: "числа 0–1000, цена, возраст, часы, телефон"
    file: "lessons/04-brojevi-i-novac.md"
  "05":
    title: "Куповина"
    subtitle: "магазин, количество, оплата, возврат"
    file: "lessons/05-u-prodavnici.md"
  "06": { title: "Провера 1", subtitle: "повторение уровня 1 + ролёвки" }
  "07": { title: "Дан по дан", subtitle: "рутина, настоящее время целиком, volim da" }
  "08": { title: "Кући", subtitle: "дом, комнаты, вещи, u/na" }
  "09": { title: "Храна и пиће", subtitle: "продукты, в кафе, заказ" }
  "10": { title: "Град око мене", subtitle: "где что, ima/nema, локатив базово, вопросы 2" }
  "11": { title: "Време и природа", subtitle: "погода, времена года, sviđa mi se" }
  "12": { title: "Провера 2", subtitle: "повторение уровня 2" }
  "13": { title: "Прошлост", subtitle: "перфект, juče sam…" }
  "14": { title: "Планови", subtitle: "футур, sutra ću…" }
  "15": { title: "Кретање", subtitle: "транспорт, направления, kako da dođem" }
  "16": { title: "Договори и термини", subtitle: "записаться, позвать, отменить" }
  "17": { title: "Тело и здравље", subtitle: "у врача, в аптеке, boli me" }
  "18": { title: "Провера 3", subtitle: "повторение уровня 3" }
  "19": { title: "Осећања и мишљења", subtitle: "mislim da, slažem se, нравится/раздражает" }
  "20": { title: "Прича о себи", subtitle: "биография, этапы жизни" }
  "21": { title: "Телефон и поруке", subtitle: "звонки, переписка, голосовые" }
  "22": { title: "Проблеми и решења", subtitle: "жалоба, мастер, pokvarilo se" }
  "23": { title: "Слободно време", subtitle: "хобби, спорт, приглашения, hajde da" }
  "24": { title: "Провера 4", subtitle: "повторение уровня 4" }
  "25": { title: "Посао", subtitle: "рабочее место, задачи, собеседование базово" }
  "26": { title: "Папирологија", subtitle: "банк, boravak, МУП (личный слой)" }
  "27": { title: "Standard vs. novosadski", subtitle: "войводинские словечки, беглость" }
  "28": { title: "Дуги разговор", subtitle: "рассказать историю, поддержать беседу" }
  "29": { title: "Читање и слух", subtitle: "новости, вывески, сериалы" }
  "30": { title: "Велика провера", subtitle: "итоговые ролёвки, план дальше" }
```

- [ ] **Шаг 4: Запусти весь бэкенд**

Run: `/opt/homebrew/bin/go test ./server/... -v 2>&1 | tail -30`
Expected: PASS.

- [ ] **Шаг 5: `serbian/plan.md` + указатели** (без коммита — не git-репо)

Перепиши `/Users/grisha/plans/serbian/plan.md` под карту выше (человекочитаемо, с грамматическими нитками по темам — см. спеку §6). Замени содержимое `/Users/grisha/plans/serbian/vocab.md`, `/Users/grisha/plans/serbian/README.md` и файлов в `/Users/grisha/plans/serbian/lessons/` на одну-две строки: «Актуальный курс и словарь — в приложении: `/Users/grisha/plans/serbian-app/content/`. Этот каталог — архив».

- [ ] **Шаг 6: Коммит (только репо приложения)**

```bash
git add content/course.yaml server/internal/content/real_test.go
git commit -m "content: A1->B1 course map (5 levels, 30 lessons)"
```

---
---

# ВЕХА 2 — Лёгкие задания, гвардия лексики, эталонные уроки

---

## Задача 10: checker — `CheckChoice` и `CheckMatch`

**Files:**
- Modify: `server/internal/checker/checker.go`
- Modify: `server/internal/checker/checker_test.go`

**Interfaces:**
- Produces:
  - `CheckChoice(answer, correct string) Result` — `OK` при `Normalize`-равенстве; `Expected: correct`.
  - `CheckMatch(got map[string]string, pairs [][2]string) (ok bool, per map[string]bool)` — `per[left]` = `Normalize(got[left]) == Normalize(right)`; `ok` = все `true` и `len(got) == len(pairs)`.

- [ ] **Шаг 1: Тесты**

```go
func TestCheckChoice(t *testing.T) {
	if r := CheckChoice("Hvala", "hvala"); !r.OK {
		t.Errorf("case-insensitive choice should pass: %+v", r)
	}
	if r := CheckChoice("Molim", "Hvala"); r.OK {
		t.Error("wrong choice passed")
	}
}

func TestCheckMatch(t *testing.T) {
	pairs := [][2]string{{"Dobro jutro", "Доброе утро"}, {"Laku noć", "Спокойной ночи"}}
	ok, per := CheckMatch(map[string]string{
		"Dobro jutro": "Доброе утро", "Laku noć": "Спокойной ночи",
	}, pairs)
	if !ok || !per["Dobro jutro"] || !per["Laku noć"] {
		t.Errorf("all-correct match failed: ok=%v per=%v", ok, per)
	}
	ok, per = CheckMatch(map[string]string{
		"Dobro jutro": "Спокойной ночи", "Laku noć": "Доброе утро",
	}, pairs)
	if ok || per["Dobro jutro"] {
		t.Errorf("swapped match should fail: ok=%v per=%v", ok, per)
	}
}
```

- [ ] **Шаг 2: Запусти — падает**

Run: `/opt/homebrew/bin/go test ./server/internal/checker/ -run 'TestCheckChoice|TestCheckMatch' -v`

- [ ] **Шаг 3: Реализация**

```go
// CheckChoice grades a single-choice answer.
func CheckChoice(answer, correct string) Result {
	if Normalize(answer) == Normalize(correct) {
		return Result{OK: true, Expected: correct}
	}
	return Result{OK: false, Expected: correct}
}

// CheckMatch grades a pair-matching answer. got maps each left item to the
// right item the learner picked.
func CheckMatch(got map[string]string, pairs [][2]string) (bool, map[string]bool) {
	per := make(map[string]bool, len(pairs))
	all := len(got) == len(pairs)
	for _, p := range pairs {
		ok := Normalize(got[p[0]]) == Normalize(p[1])
		per[p[0]] = ok
		if !ok {
			all = false
		}
	}
	return all, per
}
```

- [ ] **Шаг 4: Запусти**

Run: `/opt/homebrew/bin/go test ./server/internal/checker/ -v`
Expected: PASS.

- [ ] **Шаг 5: Коммит**

```bash
git add server/internal/checker/
git commit -m "feat(checker): CheckChoice + CheckMatch"
```

---

## Задача 11: content — валидация новых типов + порядок сложности

**Files:**
- Modify: `server/internal/content/load.go` (`decodeExercise` — уже принял поля в задаче 1; добавить проверку `word_bank` bank⊇accept-токенов)
- Create: `server/internal/content/difficulty.go`
- Create: `server/internal/content/difficulty_test.go`
- Modify: `server/internal/content/manifest.go` (вызвать проверку порядка для `practice`-шагов)

**Interfaces:**
- Produces:
  - `exerciseRank(t string) int` — `choice`=1; `match`,`fill_blank`=2; `word_bank`,`fix_error`=3; `conjugate`,`listen`=4; `translate`=5; `free`=6; неизвестный → `99`.
  - `checkDifficultyOrder(rel string, step Step) error` — для `kind=="practice"` без `Mixed`: ранги не убывают, иначе ошибка с id шага и парой упражнений.
- Проверка `word_bank`: каждый токен (`checker.Normalize` + split) любого `accept` присутствует в `Bank` (нормализованно). Иначе ошибка сборки.

- [ ] **Шаг 1: Тесты**

```go
func TestDifficultyOrderRejectsRegression(t *testing.T) {
	step := Step{ID: "x.2", Kind: "practice", Exercises: []Exercise{
		{ID: "x.2.1", Type: "translate"},
		{ID: "x.2.2", Type: "choice"},
	}}
	if err := checkDifficultyOrder("x.yaml", step); err == nil {
		t.Fatal("translate before choice should be rejected")
	}
	step.Mixed = true
	if err := checkDifficultyOrder("x.yaml", step); err != nil {
		t.Fatalf("mixed step should pass: %v", err)
	}
	step.Mixed = false
	step.Kind = "checkpoint"
	if err := checkDifficultyOrder("x.yaml", step); err != nil {
		t.Fatalf("checkpoint should be exempt: %v", err)
	}
}

func TestWordBankRequiresBankCoversAnswer(t *testing.T) {
	_, err := decodeExercise("x.yaml", exerciseYAML{
		ID: "x.1", Type: "word_bank", Prompt: "p",
		Bank:   []string{"Ja", "sam"},
		Accept: yamlList("Ja sam Ana"), // helper building a yaml.Node list
	})
	if err == nil {
		t.Fatal("bank missing 'Ana' should be rejected")
	}
}
```
(Хелпер `yamlList(...string) yaml.Node` — собери `yaml.Node` kind `SequenceNode`; или прими `Accept` как уже декодированный в тестовом вызове — проверь сигнатуру `decodeExercise` и при необходимости вынеси проверку bank в отдельную функцию `validateWordBank(bank, accept []string) error`, которую и тестируй напрямую.)

- [ ] **Шаг 2: Запусти — падает**

- [ ] **Шаг 3: `difficulty.go`**

```go
package content

import "fmt"

var ranks = map[string]int{
	"choice": 1, "match": 2, "fill_blank": 2, "word_bank": 3, "fix_error": 3,
	"conjugate": 4, "listen": 4, "translate": 5, "free": 6,
}

func exerciseRank(t string) int {
	if r, ok := ranks[t]; ok {
		return r
	}
	return 99
}

// checkDifficultyOrder enforces non-decreasing exercise difficulty within a
// practice step. checkpoint steps and steps flagged `mixed` are exempt.
func checkDifficultyOrder(rel string, s Step) error {
	if s.Kind != "practice" || s.Mixed {
		return nil
	}
	prev := 0
	prevID := ""
	for _, e := range s.Exercises {
		r := exerciseRank(e.Type)
		if r < prev {
			return fmt.Errorf("%s: step %s: %s (%s) is easier than the preceding %s — reorder or set `mixed: true`",
				rel, s.ID, e.ID, e.Type, prevID)
		}
		prev, prevID = r, e.ID
	}
	return nil
}
```

- [ ] **Шаг 4: `validateWordBank` + вызовы**

В `decodeExercise`, ветка `word_bank` (внутри `case autoTypes[e.Type]:` после декодирования `ex.Accept`):
```go
		if e.Type == "word_bank" {
			if err := validateWordBank(ex.Bank, ex.Accept); err != nil {
				return Exercise{}, fmt.Errorf("%s: exercise %s: %w", rel, e.ID, err)
			}
		}
```
```go
// in difficulty.go — imports "strings" and
// "github.com/grisha/serbian-app/server/internal/checker"
func validateWordBank(bank, accept []string) error {
	have := map[string]bool{}
	for _, b := range bank {
		for _, tok := range strings.Fields(checker.Normalize(b)) {
			have[tok] = true
		}
	}
	for _, a := range accept {
		for _, tok := range strings.Fields(checker.Normalize(a)) {
			if !have[tok] {
				return fmt.Errorf("bank is missing %q (needed for accepted answer %q)", tok, a)
			}
		}
	}
	return nil
}
```
> `content` не импортирует `checker` сегодня. Добавь импорт `github.com/grisha/serbian-app/server/internal/checker` в `difficulty.go`. Цикла нет: `checker` не зависит от `content`. (Задача 14 тоже будет импортировать `checker` в `lexicon.go`.)

В `manifest.go`, в `parseManifest` после сборки `step`:
```go
		if err := checkDifficultyOrder(rel, step); err != nil {
			return nil, err
		}
```

- [ ] **Шаг 5: Запусти пакет**

Run: `/opt/homebrew/bin/go test ./server/internal/content/ -v`
Expected: PASS (фикстура `90.yaml` — `90.2` идёт choice→fill_blank, ранги 1→2, ок).

- [ ] **Шаг 6: Коммит**

```bash
git add server/internal/content/
git commit -m "feat(content): new-type validation + difficulty-order check"
```

---

## Задача 12: API — новые типы в `checkExercise` + DTO + гард на утечку

**Files:**
- Modify: `server/internal/api/dto.go`
- Modify: `server/internal/api/api.go` (`getExercises`, `checkExercise`)
- Modify: `server/internal/api/api_test.go`

**Interfaces:**
- Consumes: `checker.CheckChoice`, `checker.CheckMatch`, `checker.Check`.
- Produces:
  - `exerciseDTO` += `Options []string`, `Bank []string`, `Left []string`, `Right []string` (все `omitempty`).
  - `getExercises` наполняет их: `choice`→`Options` (как есть, порядок с сервера; клиент перемешивает), `word_bank`→`Bank`, `match`→`Left` (левые в порядке) + `Right` (правые; **порядок с сервера — клиент перемешивает**).
  - `checkExercise` ветки:
    - `choice`: `req.Answer` → `CheckChoice(req.Answer, ex.Answer)`.
    - `word_bank`: `req.Answer` (клиент шлёт собранную строку) → `checker.Check(req.Answer, ex.Accept)`.
    - `match`: `req.Pairs map[string]string` → `CheckMatch`; `resp.Forms` не используется, добавь `resp.Match map[string]bool`; `recordAnswer` = JSON от `req.Pairs`.
  - `checkRequest` += `Pairs map[string]string \`json:"pairs"\``.
  - `checkResultDTO` += `Match map[string]bool \`json:"match,omitempty"\``.

- [ ] **Шаг 1: Тесты** (фикстура-урок `90` содержит `choice` `90.2.1` и `fill_blank` `90.2.2`)

```go
func TestExercisesEndpointHidesNewAnswers(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "GET", "/api/lessons/90/exercises", "")
	body := rr.Body.String()
	for _, leak := range []string{`"answer"`, `"accept"`, `"pairs"`, `"say"`} {
		if strings.Contains(body, leak) {
			t.Errorf("exercises endpoint leaked %s", leak)
		}
	}
	if !strings.Contains(body, `"options"`) {
		t.Error("choice options not delivered")
	}
}

func TestCheckChoiceEndpoint(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "POST", "/api/lessons/90/exercises/90.2.1/check", `{"answer":"Da"}`)
	var res struct {
		OK bool `json:"ok"`
	}
	json.Unmarshal(rr.Body.Bytes(), &res)
	if !res.OK {
		t.Fatalf("choice check: %s", rr.Body)
	}
	rr = do(h, "POST", "/api/lessons/90/exercises/90.2.1/check", `{"answer":"Ne"}`)
	json.Unmarshal(rr.Body.Bytes(), &res)
	if res.OK {
		t.Fatal("wrong choice graded OK")
	}
}
```

- [ ] **Шаг 2: Запусти — падает**

- [ ] **Шаг 3: DTO + `getExercises`**

`dto.go`: добавь поля в `exerciseDTO`. В `getExercises`:
```go
		for _, e := range b.Exercises {
			d := exerciseDTO{ID: e.ID, Type: e.Type, Prompt: e.Prompt, Forms: e.Forms, Meta: e.Meta, Audio: e.Audio}
			switch e.Type {
			case "choice":
				d.Options = e.Options
			case "word_bank":
				d.Bank = e.Bank
			case "match":
				for _, p := range e.Pairs {
					d.Left = append(d.Left, p[0])
					d.Right = append(d.Right, p[1])
				}
			}
			bd.Exercises = append(bd.Exercises, d)
		}
```
> `answer`/`accept`/`say` не были в `exerciseDTO` и раньше — не добавляем. `Pairs` наружу не отдаём (только `Left`/`Right` по отдельности).

- [ ] **Шаг 4: `checkExercise` ветки**

```go
	case "choice":
		res := checker.CheckChoice(req.Answer, ex.Answer)
		resp.OK, resp.Expected = res.OK, res.Expected
		recordAnswer, correct = req.Answer, res.OK
	case "word_bank":
		res := checker.Check(req.Answer, ex.Accept)
		resp.OK, resp.Diff, resp.Expected, resp.NearMiss = res.OK, res.Diff, res.Expected, res.NearMiss
		recordAnswer, correct = req.Answer, res.OK
	case "match":
		ok, per := checker.CheckMatch(req.Pairs, ex.Pairs)
		resp.OK, resp.Match = ok, per
		b, _ := json.Marshal(req.Pairs)
		recordAnswer, correct = string(b), ok
```
Добавь `Pairs map[string]string \`json:"pairs"\`` в `checkRequest`, `Match map[string]bool \`json:"match,omitempty"\`` в `checkResultDTO`. `"encoding/json"` уже импортирован.

- [ ] **Шаг 5: Запусти пакет**

Run: `/opt/homebrew/bin/go test ./server/internal/api/ -v`
Expected: PASS.

- [ ] **Шаг 6: Коммит**

```bash
git add server/internal/api/
git commit -m "feat(api): choice/word_bank/match in check + exercise DTO"
```

---

## Задача 13: Фронт — компоненты `choice` / `word_bank` / `match`

**Files:**
- Create: `web/src/components/exercises/ChoiceAnswer.vue`
- Create: `web/src/components/exercises/WordBankAnswer.vue`
- Create: `web/src/components/exercises/MatchAnswer.vue`
- Modify: `web/src/components/exercises/ExerciseItem.vue`
- Create: `web/src/components/exercises/ChoiceAnswer.test.ts`, `WordBankAnswer.test.ts`

**Interfaces:**
- Каждый компонент принимает `{ lesson, exerciseId, prompt, prior? }` + свои данные (`options` / `bank` / `left`+`right`), эмитит `graded: [ok: boolean]`, шлёт `api.check(lesson, exerciseId, payload)`:
  - Choice: `{ answer: <clicked option> }`.
  - WordBank: `{ answer: <joined picked chips> }`.
  - Match: `{ pairs: { [left]: <picked right> } }` — читает `result.match` для пораздельной подсветки.
- Перемешивание `options` / `bank` / `right` — на клиенте, один раз при монтировании (`ref` от `[...arr].sort(() => Math.random() - 0.5)`).

- [ ] **Шаг 1: Тесты (Choice, WordBank)**

`ChoiceAnswer.test.ts`: смонтировать с `options=['A','B']`, замокать `api.check` → `{ok:true}`; клик по варианту → `api.check` вызван с `{answer:'A'}`, эмит `graded true`, показан «Верно».
`WordBankAnswer.test.ts`: `bank=['sam','Ja','Ana']`; клики `Ja`,`sam`,`Ana` → tray = "Ja sam Ana"; submit → `api.check` с `{answer:'Ja sam Ana'}`.

- [ ] **Шаг 2: Запусти — падает**

Run: `cd web && npx vitest run src/components/exercises/ChoiceAnswer.test.ts src/components/exercises/WordBankAnswer.test.ts`

- [ ] **Шаг 3: `ChoiceAnswer.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { api } from '../../api'
import type { CheckResult, LessonAttempt } from '../../types'

const props = defineProps<{
  lesson: string; exerciseId: string; prompt: string
  options: string[]; prior?: LessonAttempt
}>()
const emit = defineEmits<{ graded: [ok: boolean] }>()

const shuffled = ref([...props.options].sort(() => Math.random() - 0.5))
const picked = ref<string | null>(props.prior?.answer ?? null)
const result = ref<CheckResult | null>(null)
const pending = ref(false)

async function choose(opt: string) {
  if (pending.value || result.value) return
  picked.value = opt
  pending.value = true
  try {
    result.value = await api.check(props.lesson, props.exerciseId, { answer: opt })
    emit('graded', !!result.value.ok)
  } finally {
    pending.value = false
  }
}
function retry() { result.value = null; picked.value = null }
</script>

<template>
  <div class="card p-3.5">
    <p class="mb-2 whitespace-pre-wrap">{{ prompt }}</p>
    <div class="flex flex-wrap gap-2">
      <button v-for="o in shuffled" :key="o"
        class="btn btn-ghost"
        :class="{
          'ring-2 ring-[var(--good)]': result?.ok && picked === o,
          'ring-2 ring-[var(--bad)]': result && !result.ok && picked === o,
        }"
        :disabled="!!result"
        @click="choose(o)">{{ o }}</button>
    </div>
    <div v-if="result" class="pop mt-2 text-sm">
      <p class="font-semibold" :class="result.ok ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        {{ result.ok ? '✓ Верно' : '✗ Не то' }}
      </p>
      <p v-if="!result.ok && result.expected">Правильно: <span class="serbian font-semibold">{{ result.expected }}</span></p>
      <button class="mt-1 font-medium text-[var(--accent)]" @click="retry">Ещё раз</button>
    </div>
  </div>
</template>
```

- [ ] **Шаг 4: `WordBankAnswer.vue`**

Ряд «банк» (кнопки-фишки, при клике уходят в лоток), лоток (упорядоченные выбранные, клик — вернуть), кнопка «Проверить» → `api.check` с `{ answer: picked.join(' ') }`. Показ результата как в `TextAnswer.vue` (diff/expected/explain). Seed shuffled как в Choice.

- [ ] **Шаг 5: `MatchAnswer.vue`**

Левый столбец фиксирован; напротив каждого — `<select>` с `right` (перемешан). Кнопка «Проверить» → `api.check` с `{ pairs: Object.fromEntries(rows.map(r => [r.left, r.pick])) }`. После ответа — по строкам подсветка из `result.match`. Кнопка «Ещё раз».

- [ ] **Шаг 6: `ExerciseItem.vue`**

```ts
import ChoiceAnswer from './ChoiceAnswer.vue'
import WordBankAnswer from './WordBankAnswer.vue'
import MatchAnswer from './MatchAnswer.vue'
```
Убери `'listen'`? нет — оставь в `textTypes`. Добавь ветки перед `TextAnswer`:
```vue
  <ChoiceAnswer v-else-if="exercise.type === 'choice'"
    :lesson="lesson" :exercise-id="exercise.id" :prompt="exercise.prompt"
    :options="exercise.options ?? []" :prior="prior" @graded="$emit('graded', $event)" />
  <WordBankAnswer v-else-if="exercise.type === 'word_bank'"
    :lesson="lesson" :exercise-id="exercise.id" :prompt="exercise.prompt"
    :bank="exercise.bank ?? []" :prior="prior" @graded="$emit('graded', $event)" />
  <MatchAnswer v-else-if="exercise.type === 'match'"
    :lesson="lesson" :exercise-id="exercise.id" :prompt="exercise.prompt"
    :left="exercise.left ?? []" :right="exercise.right ?? []" :prior="prior" @graded="$emit('graded', $event)" />
```

- [ ] **Шаг 7: Запусти фронт**

Run: `cd web && npx vitest run && npx vue-tsc --noEmit`
Expected: PASS.

- [ ] **Шаг 8: Коммит**

```bash
git add web/src/components/exercises/
git commit -m "feat(web): choice / word_bank / match exercise components"
```

---

## Задача 14: Гвардия лексики

**Files:**
- Create: `server/internal/content/lexicon.go`
- Create: `server/internal/content/lexicon_test.go`
- Create: `content/allow-words.yaml`
- Create: `server/internal/content/testdata/content/allow-words.yaml`

**Interfaces:**
- Produces:
  - `knownWords(c *Course, lessonID string) map[string]bool` — накопительный набор нормализованных токенов: все `vocab` с `lesson` в порядке курса ≤ `lessonID`, `Teaches` этого урока, `allow-words.yaml`.
  - `unknownTokens(text string, known map[string]bool, alsoOK []string) []string` — токены `text` (после `checker.Normalize`), которых нет в `known`, не покрытых префиксом (известный токен длиной ≥ 3 — префикс данного) и не в `alsoOK`.
  - `TestLexiconGuard` — для каждого манифест-урока проверяет `accept`/`options`/`bank`/`pairs[sr]`/`say` строго (`t.Errorf`), сербские вставки `prompt` — мягко (`t.Logf`). Легаси-уроки — целиком мягко.

- [ ] **Шаг 1: `allow-words.yaml`**

```yaml
# Токены, всегда допустимые в упражнениях (имена, числа, служебное).
names: [ana, marko, milan, milos, jovana, nikola, petar, srbija, rusija,
        beograd, novi, sad, griša, grisa, srbin, ruskinja]
particles: [i, a, ali, ili, ne, da, li, je, se, su, sam, si, smo, ste,
            u, na, o, za, iz, do, od, s, sa, po, pa, evo, eto]
digits: ["0","1","2","3","4","5","6","7","8","9"]
```
Формат парсится в один плоский набор (склей все списки).

- [ ] **Шаг 2: Тест**

```go
func TestUnknownTokens(t *testing.T) {
	known := map[string]bool{"grad": true, "zdravo": true}
	got := unknownTokens("Zdravo, idem u grad danas", known, []string{"idem"})
	// "u" not known, not alsoOK -> unknown; "danas" unknown; "grad" ok; "zdravo" ok; "idem" alsoOK
	want := map[string]bool{"u": true, "danas": true}
	for _, g := range got {
		if !want[g] {
			t.Errorf("unexpected unknown %q (got %v)", g, got)
		}
		delete(want, g)
	}
	if len(want) != 0 {
		t.Errorf("missed unknowns: %v", want)
	}
}

func TestLexiconGuardRealContent(t *testing.T) {
	c, err := Load("../../../content")
	if err != nil {
		t.Fatal(err)
	}
	strictFail := 0
	for _, ph := range c.Phases {
		for _, id := range ph.Lessons {
			l := c.Lessons[id]
			if l == nil || !l.Manifest {
				continue
			}
			known := knownWords(c, id)
			for _, s := range l.Steps {
				for _, e := range s.Exercises {
					for _, txt := range strictStrings(e) {
						for _, u := range unknownTokens(txt, known, s.AlsoOK) {
							t.Errorf("lesson %s step %s ex %s: unknown word %q (add to step also_ok or lesson teaches)", id, s.ID, e.ID, u)
							strictFail++
						}
					}
				}
			}
		}
	}
	if strictFail > 0 {
		t.Logf("%d lexicon violations", strictFail)
	}
}
```
`strictStrings(e Exercise) []string` — собирает `e.Accept`, `e.Options`, `e.Bank`, `e.Say`, и левые (или обе, если правая не кириллица) стороны `e.Pairs`.

- [ ] **Шаг 3: Запусти — падает** (undefined). После реализации на этом этапе манифест-уроков ещё нет (01–03 — задачи 15–17) → тест зелёный вхолостую. Это ок: тест станет боевым, когда появятся манифесты.

- [ ] **Шаг 4: `lexicon.go`**

```go
package content

import (
	"strings"
	"unicode"

	"github.com/grisha/serbian-app/server/internal/checker"
)

func lexTokens(s string) []string {
	return strings.Fields(checker.Normalize(s))
}

func isCyrillic(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Cyrillic, r) {
			return true
		}
	}
	return false
}

func knownWords(c *Course, upTo string) map[string]bool {
	order := map[string]int{}
	n := 0
	for _, ph := range c.Phases {
		for _, id := range ph.Lessons {
			order[id] = n
			n++
		}
	}
	limit, ok := order[upTo]
	if !ok {
		limit = 1 << 30
	}
	known := map[string]bool{}
	for _, v := range c.Vocab {
		if o, ok := order[v.Lesson]; ok && o <= limit {
			for _, tok := range lexTokens(v.Latin) {
				known[tok] = true
			}
		}
	}
	if l := c.Lessons[upTo]; l != nil {
		for _, w := range l.Teaches {
			for _, v := range c.Vocab {
				if v.ID == w {
					for _, tok := range lexTokens(v.Latin) {
						known[tok] = true
					}
				}
			}
		}
	}
	for _, w := range c.allowWords { // loaded in Load()
		known[w] = true
	}
	return known
}

func unknownTokens(text string, known map[string]bool, alsoOK []string) []string {
	ok := map[string]bool{}
	for _, a := range alsoOK {
		for _, tok := range lexTokens(a) {
			ok[tok] = true
		}
	}
	var out []string
	for _, tok := range lexTokens(text) {
		if known[tok] || ok[tok] || coveredByPrefix(tok, known) {
			continue
		}
		out = append(out, tok)
	}
	return out
}

func coveredByPrefix(tok string, known map[string]bool) bool {
	for k := range known {
		if len(k) >= 3 && strings.HasPrefix(tok, k) {
			return true
		}
	}
	return false
}
```
Добавь в `Course` поле `allowWords []string` (не в DTO). В `Load` — прочитать `allow-words.yaml` (плоский склеенный список) в `c.allowWords`; отсутствие файла → пустой список.

- [ ] **Шаг 5: `strictStrings` в `lexicon.go`**

```go
func strictStrings(e Exercise) []string {
	var out []string
	out = append(out, e.Accept...)
	out = append(out, e.Options...)
	out = append(out, e.Bank...)
	if e.Say != "" {
		out = append(out, e.Say)
	}
	for _, p := range e.Pairs {
		if isCyrillic(p[1]) {
			out = append(out, p[0]) // left is Serbian, right is Russian
		} else {
			out = append(out, p[0], p[1])
		}
	}
	return out
}
```

- [ ] **Шаг 6: Запусти пакет**

Run: `/opt/homebrew/bin/go test ./server/internal/content/ -v`
Expected: PASS.

- [ ] **Шаг 7: Коммит**

```bash
git add server/internal/content/ content/allow-words.yaml
git commit -m "feat(content): lexicon guard (exercises use only taught words)"
```

---

## Задача 15: Эталон — урок 01 «Поздрави и први контакт»

**Files:**
- Create: `content/lessons/01.yaml`
- Create: `content/lessons/01/1-pozdravi.md`, `01/3-ti-vi.md`, `01/5-vezljivost.md`, `01/7-citanje.md`
- Delete: `content/lessons/01-pozdravi-i-upoznavanje.md`, `content/exercises/01.yaml`
- Modify: `content/course.yaml` (`"01".file` → `lessons/01.yaml`, убрать `subtitle`-дубль если нужно)
- Modify: `content/vocab.yaml` (проставить `lesson: "01"` уже есть; добавить недостающие слова урока, если появятся)
- Modify: `server/internal/content/real_test.go` (ожидания для 01: ≥ 6 шагов, есть checkpoint)
- Run: `python3 scripts/tts.py`

**Структура шагов (8):** см. спеку §7. Правила:
- Каждый `teach` ≤ ~180 слов, глоссы по-русски по месту, одна мысль.
- `practice`: 4–7 упражнений, ранги не убывают (`choice`→`match`→`fill_blank`→`word_bank`→`translate`).
- Только слова из `teaches` урока 01 + `allow-words` + `also_ok` шага. `jesam` только готовыми `ja sam`/`ti si` во фразах — без парадигмы.
- `checkpoint`: 5–7 смешанных + 1 `free`.
- **≥ 3 упражнения `listen`** на урок (существующий гард `real_test.go`): отдельный `practice`-шаг «Диктант» или `listen` в checkpoint. `say` — только из известной лексики. `listen` ранг 4 — ставь после `word_bank`/`fix_error` либо в свой шаг с `mixed: true`.

**Пример — шаг `01.2` (полностью рабочий):**
```yaml
  - id: "01.2"
    kind: practice
    title: "Узнай приветствие"
    also_ok: []
    exercises:
      - id: "01.2.1"
        type: choice
        prompt: "«Добрый день» —"
        options: ["Dobar dan", "Dobro veče", "Laku noć"]
        answer: "Dobar dan"
      - id: "01.2.2"
        type: choice
        prompt: "Прощаешься вечером, уходя из кафе. Говоришь —"
        options: ["Prijatno", "Dobro jutro", "Molim"]
        answer: "Prijatno"
      - id: "01.2.3"
        type: match
        prompt: "Соедини приветствие и перевод"
        pairs:
          - ["Dobro jutro", "Доброе утро"]
          - ["Laku noć", "Спокойной ночи"]
          - ["Doviđenja", "До свидания"]
          - ["Ćao", "Привет / пока"]
      - id: "01.2.4"
        type: fill_blank
        prompt: "Утро, заходишь в пекару: «___ jutro!»"
        accept: ["Dobro", "dobro"]
```

**Пример — шаг `01.1` (`content/lessons/01/1-pozdravi.md`):**
```markdown
# Здороваемся

Универсальное приветствие — **zdravo** *(здраво — привет / здравствуйте)*
и **dobar dan** *(добар дан — добрый день)*. Оба годятся почти всегда.

По времени суток: **dobro jutro** *(добро јутро — доброе утро)* примерно
до 10, **dobro veče** *(добро вече — добрый вечер)* — после 17–18.

Неформально, со сверстниками: **ćao** *(ћао)* — это и «привет», и «пока».

Прощание: **doviđenja** *(довиђења — до свидания)*, на ночь —
**laku noć** *(лаку ноћ — спокойной ночи)*. Уходя из магазина или кафе,
часто говорят **prijatno** *(пријатно — всего доброго)* — отвечай тем же.
```

- [ ] **Шаг 1: Обнови `real_test.go` для 01**

```go
	{
		l := c.Lessons["01"]
		if !l.Manifest {
			t.Fatal("lesson 01 should be a manifest")
		}
		if len(l.Steps) < 6 {
			t.Errorf("lesson 01: %d steps, want >= 6", len(l.Steps))
		}
		kinds := map[string]int{}
		for _, s := range l.Steps {
			kinds[s.Kind]++
		}
		if kinds["teach"] < 2 || kinds["practice"] < 2 || kinds["checkpoint"] < 1 {
			t.Errorf("lesson 01 step kinds: %v", kinds)
		}
	}
```
Убери старую строку `len(c.Exercises["01"]) < 4` (заменена).

- [ ] **Шаг 2: Запусти — падает**

Run: `/opt/homebrew/bin/go test ./server/internal/content/ -run TestRealContentLoads -v`

- [ ] **Шаг 3: Напиши `content/lessons/01/*.md`** (4 фрагмента; следуй примерам и §7)

- [ ] **Шаг 4: Напиши `content/lessons/01.yaml`** (8 шагов; используй пример `01.2`)

- [ ] **Шаг 5: Удали легаси-файлы урока 01, переключи `course.yaml`**

```bash
git rm content/lessons/01-pozdravi-i-upoznavanje.md content/exercises/01.yaml
```
В `course.yaml`: `"01": { title: ..., subtitle: ..., file: "lessons/01.yaml" }`.

- [ ] **Шаг 6: Прогони гвардию и тесты**

Run: `/opt/homebrew/bin/go test ./server/internal/content/ -v 2>&1 | grep -E 'FAIL|PASS|unknown word'`
Чини нарушения: добавляй падежные формы/имена в `also_ok` шага или новые слова — в `content/vocab.yaml` с `lesson: "01"` и в `teaches`.

- [ ] **Шаг 7: Озвучка**

Run: `python3 scripts/tts.py`
Проверь: новые `content/audio/*.mp3` для `listen`-упражнений урока 01 (если есть) и для новых слов.

- [ ] **Шаг 8: Ручная проверка**

Run: `make dev` → урок 01 идёт по 8 шагам, задания лёгкие, всё проверяется.

- [ ] **Шаг 9: Коммит**

```bash
git add content/ server/internal/content/real_test.go
git commit -m "content: rebuild lesson 01 as gentle micro-steps"
```

---

## Задача 16: Эталон — урок 02 «Ко си ти»

**Files:** как задача 15, для `02` (`content/lessons/02.yaml` + `02/*.md`, удалить `02-zamenice-i-prezent.md` + `exercises/02.yaml`, `course.yaml`, `vocab.yaml`, `real_test.go`, tts).

**Структура (8 шагов):** спека §7.
- Шаг 1 `teach`: `jesam` — только `ja sam / ti si / on je` + `nisam/nisi/nije`. Полная парадигма (mi/vi/oni) и порядок клитик — НЕ здесь (упомянуть «остальные формы — позже»).
- `practice`-шаги: `conjugate` не использовать для `jesam` (это тип 4, сломает порядок) — только `choice`/`fill_blank`/`word_bank`.
- Слова: страны/города (`Rusija, Srbija, Novi Sad`…), `živim u`, `iz`, `govorim`, `radim kao`, языки/профессии.
- Один `conjugate` уместен в шаге 5 (языки/работа) для регулярного глагола `raditi` — тогда шаг помечается `mixed: true` **или** `conjugate` идёт последним перед `translate`. Предпочти: отдельный шаг только с `conjugate` + `fill_blank` (ранги 4→2 — нельзя) → значит `conjugate` последним, либо `mixed: true`.
- **≥ 3 `listen`** на урок (как в задаче 15).

- [ ] **Шаг 1: `real_test.go` — блок для 02.** Убери строку `len(c.Exercises["02"]) < 5`. Добавь:
```go
	{
		l := c.Lessons["02"]
		if !l.Manifest {
			t.Fatal("lesson 02 should be a manifest")
		}
		if len(l.Steps) < 6 {
			t.Errorf("lesson 02: %d steps, want >= 6", len(l.Steps))
		}
		kinds := map[string]int{}
		for _, s := range l.Steps {
			kinds[s.Kind]++
		}
		if kinds["teach"] < 2 || kinds["practice"] < 2 || kinds["checkpoint"] < 1 {
			t.Errorf("lesson 02 step kinds: %v", kinds)
		}
	}
```
- [ ] **Шаг 2: Запусти — падает.**
- [ ] **Шаг 3: `content/lessons/02/*.md`.**
- [ ] **Шаг 4: `content/lessons/02.yaml`.**
- [ ] **Шаг 5: `git rm` легаси 02, `course.yaml` → `lessons/02.yaml`.**
- [ ] **Шаг 6: Гвардия + тесты, чинить нарушения.**
- [ ] **Шаг 7: `python3 scripts/tts.py`.**
- [ ] **Шаг 8: `make dev` — ручная проверка.**
- [ ] **Шаг 9: Коммит** `content: rebuild lesson 02 as gentle micro-steps`.

---

## Задача 17: Эталон — урок 03 «Људи око мене» (новая тема)

**Files:**
- Create: `content/lessons/03.yaml` + `03/*.md`
- Delete: `content/lessons/03-pitanja.md`, `content/exercises/03.yaml`
- Modify: `content/course.yaml` (`"03"` → title «Људи око мене», subtitle «семья, imam/nemam, мой/твой», `file: "lessons/03.yaml"`)
- Modify: `content/vocab.yaml` — **новые слова**: `mama, tata, brat, sestra, muž, žena, dete, deca, roditelji, baba, deda, sin, ćerka` с `lesson: "03"`, `pos: noun`, `gender`; акузативные формы (`brata, sestru, muža, ženu, sina, ćerku, dete`) — в `also_ok` соответствующих шагов, НЕ в vocab.
- Modify: `server/internal/content/real_test.go` — блок для 03; `vocab` счётчик (`>= 60` уже с запасом; убедись не упал).
- Modify: `content/false-friends.yaml` — при желании добавить `žena` (жена/женщина) заметку; не обязательно.
- Run: `python3 scripts/tts.py` (новые слова + listen).

**Структура (8 шагов):** спека §7. `imam/nemam` + акузатив — только для перечисленных слов, готовыми формами (`imam brata` *(имам брата — у меня есть брат)*). `moj/tvoj/njegov` — им. падеж, 3 рода. **≥ 3 `listen`** на урок (как в задаче 15).

**Восстановление Pitanja-материала:** `content/lessons/03-pitanja.md` и `exercises/03.yaml` удаляются — они восстановимы из git (`git show HEAD~1:content/lessons/03-pitanja.md`) для будущего модуля «вопросы» (урок 10). Добавь строку в `serbian/plan.md`: «Материал старого урока 03 (Pitanja) → база для урока 10 „Град око мене / вопросы 2“, лежит в git-истории до коммита rebuild lesson 03».

- [ ] **Шаг 1: `real_test.go` — блок для 03.** Существующий цикл по `{"01".."05"}` уже проверяет reading — оставь 03 в нём. Добавь:
```go
	{
		l := c.Lessons["03"]
		if !l.Manifest || l.Title == "Pitanja" {
			t.Fatalf("lesson 03 should be the manifest 'Ljudi oko mene', got title=%q manifest=%v", l.Title, l.Manifest)
		}
		if len(l.Steps) < 6 {
			t.Errorf("lesson 03: %d steps, want >= 6", len(l.Steps))
		}
		kinds := map[string]int{}
		for _, s := range l.Steps {
			kinds[s.Kind]++
		}
		if kinds["reading"] < 1 || kinds["checkpoint"] < 1 {
			t.Errorf("lesson 03 step kinds: %v", kinds)
		}
	}
```
- [ ] **Шаг 2: Запусти — падает.**
- [ ] **Шаг 3: `content/vocab.yaml` — добавь 10–13 слов семьи** (латиница + кириллица + ru + gender; глоссы по-русски).
- [ ] **Шаг 4: `content/lessons/03/*.md`** (4–5 фрагментов).
- [ ] **Шаг 5: `content/lessons/03.yaml`** (8 шагов).
- [ ] **Шаг 6: `git rm` легаси 03; `course.yaml` → манифест + новый заголовок.**
- [ ] **Шаг 7: Гвардия + тесты; акузативы в `also_ok`.**
- [ ] **Шаг 8: `python3 scripts/tts.py`.**
- [ ] **Шаг 9: `make dev` — ручная проверка; `serbian/plan.md` — заметка о Pitanja.**
- [ ] **Шаг 10: Коммит** `content: replace lesson 03 with "Ljudi oko mene" (family)`.

---

## Задача 18: Обновить шаблон и документацию

**Files:**
- Rewrite: `content/exercises/_TEMPLATE.yaml` → `content/lessons/_TEMPLATE.yaml` (пример-манифест)
- Modify: `CLAUDE.md` (раздел «Добавить урок» + «Текст для чтения»)
- Modify: `README.md` (раздел «Контент»)
- Modify: `server/internal/content/load.go` — убедись, что `_TEMPLATE.yaml` в `lessons/` игнорируется (как сейчас в `exercises/`)

**Interfaces:** нет кода — документация. Но: loader должен пропускать `lessons/_TEMPLATE.yaml` (по имени файла) при перечислении — сейчас он грузит уроки только по `course.yaml`, так что файл-шаблон и так не подхватится. Проверь и, если `Glob` по `lessons/*.yaml` где-то появился, добавь skip.

- [ ] **Шаг 1: `content/lessons/_TEMPLATE.yaml`**

Полный пример-манифест: по одному шагу каждого `kind`, по одному упражнению каждого типа (`translate, fill_blank, fix_error, conjugate, free, listen, choice, word_bank, match`), с комментариями. Удали старый `content/exercises/_TEMPLATE.yaml` (`git rm`).

- [ ] **Шаг 2: `CLAUDE.md` — перепиши «Добавить урок»**

Новый порядок: `content/course.yaml` (`file: lessons/NN.yaml`) → `content/lessons/NN.yaml` (манифест) → `content/lessons/NN/*.md` (фрагменты) → слова в `content/vocab.yaml` (`lesson: "NN"`, продублировать в `teaches:`) → падежные формы/имена в `also_ok` шагов → `python3 scripts/tts.py`. Гард-тесты: `server/internal/content/real_test.go`, `lexicon_test.go`. Правило порядка сложности: упражнения в `practice`-шаге от лёгких к сложным, иначе `mixed: true`.

- [ ] **Шаг 3: `README.md` — обнови раздел про `content/`** (манифест-модель + легаси-фолбэк одной фразой).

- [ ] **Шаг 4: Тесты целиком**

Run: `make test`
Expected: PASS (Go + Vitest).

- [ ] **Шаг 5: Полная сборка**

Run: `make build`
Expected: бинарь `./serbian-app` собран без ошибок.

- [ ] **Шаг 6: Коммит**

```bash
git add -A
git commit -m "docs: manifest lesson authoring; new _TEMPLATE.yaml"
```

---

## Финальная проверка (после задачи 18)

- [ ] `make test` — зелёно.
- [ ] `make build` — бинарь собирается.
- [ ] `make dev` — уроки 01–03 идут по микро-шагам с лёгкими заданиями; 04–05 идут по синтетическим шагам; прогресс шагов и урока сохраняется; SRS-слова уроков 01–03 попадают в тренажёр; дашборд показывает 5 уровней.
- [ ] `git log --oneline` — по коммиту на задачу, ветка `feature/lesson-steps`.
- [ ] Ручная проверка гвардии: временно добавь в упражнение урока 01 слово `automobil` (нет в словаре) → `go test ./server/internal/content/ -run TestLexiconGuard` падает с понятным сообщением. Откати.

## Self-review notes (для автора плана)

- **Покрытие спеки:** §1 → задачи 1–3; §2 → задачи 1, 10–13; §3 → задачи 7–8; §4 → задачи 5–6; §5 → задача 14; §6 → задача 9; §7 → задачи 15–17; персона → задача 4; `_TEMPLATE`/доки → задача 18.
- **Открытый вопрос спеки (match в первой итерации):** включён (задачи 10, 12, 13). Если Веха 2 поджимает — `match` можно пометить как отложенный, компоненты и checker останутся.
- **Открытый вопрос (префикс ≥ 3):** зашит в `coveredByPrefix`; крутить при ложных срабатываниях в задачах 15–17.
- **Риск:** синтетические id шагов легаси (`"A"`, `"teach"`, `"reading"`) vs манифестные (`"01.2"`). Согласовано: фронт ключует по `lesson.id + step.id`, атрибуты попыток — по `block` из `Course.Exercises`. Тесты задач 2 и 6 это фиксируют.
