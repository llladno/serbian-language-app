# Srpski App Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a local Vue + Go web app that turns the Obsidian Serbian
course into a lesson reader, interactive auto-checked exercises, an
SRS vocabulary trainer, and a progress dashboard.

**Architecture:** Go HTTP server loads structured content files
(`content/*.yaml` + `content/lessons/*.md`) into memory, exposes a JSON
API, and persists mutable state (SRS schedule, exercise attempts,
lesson progress) in SQLite. A Vue 3 SPA consumes the API. Production is
a single Go binary with the built SPA embedded via `go:embed`.

**Tech Stack:** Go 1.27 (`net/http` ServeMux, `gopkg.in/yaml.v3`,
`modernc.org/sqlite`, `github.com/fsnotify/fsnotify`); Vue 3 +
TypeScript + Vite + Vue Router + Pinia + Tailwind CSS + `markdown-it`.

**Spec:** `docs/superpowers/specs/2026-09-06-serbian-app-design.md`

## Global Constraints

- Go module path: `github.com/grisha/serbian-app`. Go version floor:
  `go 1.27`.
- Backend third-party deps limited to exactly: `gopkg.in/yaml.v3`,
  `modernc.org/sqlite`, `github.com/fsnotify/fsnotify`. No web
  framework, no ORM, no server-side markdown renderer.
- The `accept` answer lists MUST NOT appear in any response body except
  `POST .../check`. The exercises list endpoint strips them.
- All API responses are JSON under the `/api` prefix. Errors:
  `{"error": "<message>"}` with a non-2xx status.
- Answer-accept strings are authored in **latin** script. The checker
  never transliterates between latin and cyrillic.
- Content is the source of truth in `content/`; there is no content
  editor in the UI. SQLite lives at `data/app.db` (gitignored).
- Dates/times stored as ISO-8601 strings (UTC) in SQLite. "Today" for
  SRS due comparisons is the server's local date.
- Frontend: Vue 3 `<script setup>` + TS only, no Options API, no SSR.
- Repo root is `/Users/grisha/plans/serbian-app/`. `go` is at
  `/opt/homebrew/bin/go` — ensure `PATH` includes `/opt/homebrew/bin`.

---

## File Structure

```
serbian-app/
  go.mod
  Makefile
  README.md
  .gitignore
  content/
    course.yaml
    lessons/01-pozdravi-i-upoznavanje.md
    lessons/02-zamenice-i-prezent.md
    exercises/01.yaml
    exercises/02.yaml
    vocab.yaml
    false-friends.yaml
  server/
    main.go                      # flag parsing, wiring, ListenAndServe
    internal/content/
      types.go                   # Course, Lesson, ExerciseBlock, Exercise, Vocab, FalseFriend
      load.go                    # Load(dir) (*Course, error) + validation
      load_test.go
      watch.go                   # Watcher(dir, func()) using fsnotify
      testdata/content/...       # minimal fixture tree
    internal/checker/
      checker.go                 # Normalize, Check, diff
      checker_test.go
    internal/srs/
      srs.go                     # Card, Grade, Schedule
      srs_test.go
    internal/store/
      store.go                   # Open, schema, methods
      store_test.go
    internal/api/
      api.go                     # Handler(deps) http.Handler, route table
      api_test.go
      dto.go                     # response structs
    web/
      embed.go                   # //go:embed all:dist  (build tag-free; dist has .gitkeep)
      dist/.gitkeep
  web/
    index.html
    package.json
    vite.config.ts
    tsconfig.json
    tailwind.config.js
    postcss.config.js
    src/
      main.ts
      App.vue
      router.ts
      api.ts
      api.test.ts
      types.ts
      stores/course.ts
      stores/review.ts
      components/AppNav.vue
      components/MarkdownView.vue
      components/exercises/ExerciseBlock.vue
      components/exercises/ExerciseItem.vue
      components/exercises/TextAnswer.vue
      components/exercises/TextAnswer.test.ts
      components/exercises/ConjugateAnswer.vue
      components/exercises/FreeAnswer.vue
      views/DashboardView.vue
      views/CourseView.vue
      views/LessonView.vue
      views/ReviewView.vue
      views/VocabView.vue
      views/FalseFriendsView.vue
```

---

## Task 1: Repo skeleton — Go module + Vite/Vue frontend that boots

**Files:**
- Create: `go.mod`, `Makefile`, `.gitignore`, `README.md`
- Create: `server/main.go`, `server/internal/api/api.go`,
  `server/web/embed.go`, `server/web/dist/.gitkeep`
- Create: `web/` Vite project (`package.json`, `vite.config.ts`,
  `tsconfig.json`, `tailwind.config.js`, `postcss.config.js`,
  `index.html`, `src/main.ts`, `src/App.vue`, `src/router.ts`,
  `src/style.css`)

**Interfaces:**
- Produces: `api.Handler(deps Deps) http.Handler` where
  `type Deps struct{}` for now (fields added in Task 7). Route
  `GET /api/health` → `{"status":"ok","content_stale":false}`.
- Produces: `web.FS() fs.FS` returning the embedded `dist` subtree.

- [ ] **Step 1: Create `go.mod`**

```
module github.com/grisha/serbian-app

go 1.27
```

- [ ] **Step 2: Create `server/web/embed.go`**

```go
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// FS returns the built SPA files rooted at dist/.
func FS() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
```

Create `server/web/dist/.gitkeep` (empty) so the embed compiles before
the frontend is built.

- [ ] **Step 3: Create `server/internal/api/api.go`**

```go
package api

import (
	"encoding/json"
	"net/http"
)

type Deps struct{}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func Handler(deps Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "content_stale": false})
	})
	return mux
}
```

- [ ] **Step 4: Create `server/main.go`**

```go
package main

import (
	"flag"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/grisha/serbian-app/server/internal/api"
	"github.com/grisha/serbian-app/server/web"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	_ = flag.String("content", "./content", "content directory")
	_ = flag.String("db", "./data/app.db", "sqlite path")
	flag.Parse()

	mux := http.NewServeMux()
	mux.Handle("/api/", api.Handler(api.Deps{}))
	mux.Handle("/", spaHandler(web.FS()))

	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

// spaHandler serves static files and falls back to index.html for
// client-side routes (paths without a file extension).
func spaHandler(files fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(files))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(files, p); err != nil {
			if !strings.Contains(p, ".") {
				r = r.Clone(r.Context())
				r.URL.Path = "/"
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 5: Scaffold the Vite frontend**

Run:
```bash
cd /Users/grisha/plans/serbian-app
npm create vite@latest web -- --template vue-ts
cd web && npm install
npm install vue-router@4 pinia markdown-it
npm install -D @types/markdown-it tailwindcss@^3 postcss autoprefixer vitest @vue/test-utils jsdom
npx tailwindcss init -p
```

- [ ] **Step 6: Configure Tailwind + Vite proxy + Vitest**

`web/tailwind.config.js` `content`: `["./index.html","./src/**/*.{vue,ts}"]`.

`web/src/style.css` starts with the three `@tailwind` directives.

`web/vite.config.ts`:
```ts
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: { proxy: { '/api': 'http://localhost:8080' } },
  build: { outDir: 'dist' },
  test: { environment: 'jsdom' },
})
```
Add `/// <reference types="vitest" />` at the top of the file.
Add `"test": "vitest run"` to `web/package.json` scripts.

- [ ] **Step 7: Minimal router + App shell**

`web/src/router.ts`:
```ts
import { createRouter, createWebHistory } from 'vue-router'

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'dashboard', component: () => import('./views/DashboardView.vue') },
    { path: '/course', name: 'course', component: () => import('./views/CourseView.vue') },
    { path: '/lesson/:id', name: 'lesson', component: () => import('./views/LessonView.vue') },
    { path: '/review', name: 'review', component: () => import('./views/ReviewView.vue') },
    { path: '/vocab', name: 'vocab', component: () => import('./views/VocabView.vue') },
    { path: '/false-friends', name: 'false-friends', component: () => import('./views/FalseFriendsView.vue') },
  ],
})
```

`web/src/main.ts` creates the app, uses `createPinia()` and the router,
mounts `#app`, imports `./style.css`.

`web/src/App.vue` renders `<AppNav />` + `<RouterView />`.

Create `web/src/components/AppNav.vue` with `RouterLink`s to all six
routes, and six stub views each rendering an `<h1>` with the screen
name. Create empty stub `src/api.ts` exporting `export const API = ''`.

- [ ] **Step 8: Makefile**

```makefile
GO ?= /opt/homebrew/bin/go
export PATH := /opt/homebrew/bin:$(PATH)

.PHONY: dev build test tidy

dev:
	@echo "frontend: http://localhost:5173  (proxies /api to :8080)"
	@( cd web && npm run dev ) & \
	 $(GO) run ./server -addr :8080 ; \
	 kill %1 2>/dev/null || true

build:
	cd web && npm ci && npm run build
	rm -rf server/web/dist && cp -r web/dist server/web/dist
	$(GO) build -o serbian-app ./server

test:
	$(GO) test ./server/...
	cd web && npm run test

tidy:
	$(GO) mod tidy
```

- [ ] **Step 9: `.gitignore` + `README.md`**

`.gitignore` (append to existing): already has `data/`, `web/dist/`,
`server/web/dist/`, `node_modules/`, `/serbian-app`, `*.db`. Add
`!server/web/dist/.gitkeep`.

`README.md`: one paragraph on what it is, `make dev` to develop,
`make build` then `./serbian-app` to run, "content lives in
`content/`", and a pointer to the spec.

- [ ] **Step 10: Verify both halves boot**

Run:
```bash
cd /Users/grisha/plans/serbian-app
/opt/homebrew/bin/go mod tidy
/opt/homebrew/bin/go build ./server/...
/opt/homebrew/bin/go run ./server -addr :8080 &
sleep 1 && curl -s localhost:8080/api/health && kill %1
cd web && npm run build
```
Expected: `go build` succeeds; curl prints
`{"status":"ok","content_stale":false}`; `npm run build` writes
`web/dist/index.html`.

- [ ] **Step 11: Commit**

```bash
git add -A
git commit -m "chore: repo skeleton — Go server + Vue/Vite frontend boot"
```

---

## Task 2: Content types + loader + validation

**Files:**
- Create: `server/internal/content/types.go`
- Create: `server/internal/content/load.go`
- Create: `server/internal/content/load_test.go`
- Create: `server/internal/content/testdata/content/` fixture tree

**Interfaces:**
- Consumes: nothing.
- Produces:
  ```go
  type Course struct {
      Title       string
      Phases      []Phase
      Lessons     map[string]*Lesson          // key: "01"
      Exercises   map[string][]ExerciseBlock  // key: lesson id
      Vocab       []Vocab
      FalseFriends []FalseFriend
  }
  type Phase struct { ID, Title string; Lessons []string }
  type Lesson struct {
      ID, Title, Subtitle string
      Planned bool          // true when no markdown file exists
      MarkdownPath string
      Markdown string       // "" when Planned
  }
  type ExerciseBlock struct {
      ID, Title, Instruction string
      Exercises []Exercise
  }
  type Exercise struct {
      ID, Type, Prompt, Explain, Sample, Meta string
      Accept  []string    // for non-conjugate auto types
      Forms   []string    // conjugate: labels
      AcceptForms [][]string // conjugate: accepted per form
  }
  type Vocab struct {
      ID, Latin, Cyrillic, RU, Note, Lesson, POS, Gender, Aspect string
      Tags []string
  }
  type FalseFriend struct { ID, SR, Means, Not, Correct, Group string }

  func Load(dir string) (*Course, error)
  ```
- `Load` returns an error whose message names the offending file when:
  a referenced markdown path is set but missing AND not intended as
  planned (planned lessons simply have no `file:`); an `exercises/NN.yaml`
  targets an unknown lesson; a non-`free`/non-`conjugate` exercise has
  empty `Accept`; a `conjugate` exercise's `AcceptForms` length ≠
  `Forms` length; duplicate `Vocab.ID`; duplicate `FalseFriend.ID`.

- [ ] **Step 1: Write `types.go`** with the structs above plus their
  `yaml:"..."` tags. `course.yaml` shape:

```yaml
title: "Српски језик"
phases:
  - { id: A, title: "Фаза A — Выживание", lessons: ["01","02"] }
lessons:
  "01": { title: "Pozdravi i upoznavanje", subtitle: "...", file: "lessons/01-pozdravi-i-upoznavanje.md" }
  "02": { title: "Zamenice i prezent", subtitle: "...", file: "lessons/02-zamenice-i-prezent.md" }
  "03": { title: "Pitanja", subtitle: "..." }   # no file => Planned
```

Add a private mirror struct for YAML decode
(`courseFile`, `lessonEntry`) then map into the exported `Course`.

- [ ] **Step 2: Write the fixture tree** under
  `server/internal/content/testdata/content/`:
  - `course.yaml`: 1 phase `A` with lessons `["01","02"]`; lesson `01`
    with `file: lessons/01.md`; lesson `02` planned (no file).
  - `lessons/01.md`: `# Test\n\nHello **svet**.\n`
  - `exercises/01.yaml`: one `A` block with a `translate`
    (`accept: ["Zdravo"]`) and a `free` (`sample: "..."`), plus a
    `conjugate` with `forms` len 6 and `accept` len 6.
  - `vocab.yaml`: two entries (`zdravo`, `svet`).
  - `false-friends.yaml`: one entry (`pravo`).

- [ ] **Step 3: Write failing tests** in `load_test.go`:

```go
func TestLoadValidFixture(t *testing.T) {
	c, err := Load("testdata/content")
	if err != nil { t.Fatalf("Load: %v", err) }
	if c.Title != "Српски језик" { t.Errorf("title = %q", c.Title) }
	if len(c.Phases) != 1 || len(c.Phases[0].Lessons) != 2 { t.Fatalf("phases: %+v", c.Phases) }
	if c.Lessons["01"].Planned { t.Error("01 should not be planned") }
	if !c.Lessons["02"].Planned { t.Error("02 should be planned") }
	if !strings.Contains(c.Lessons["01"].Markdown, "svet") { t.Error("markdown not loaded") }
	if len(c.Exercises["01"]) != 1 { t.Fatalf("blocks: %d", len(c.Exercises["01"])) }
	if len(c.Vocab) != 2 || len(c.FalseFriends) != 1 { t.Error("vocab/ff counts") }
}

func TestLoadRejectsExerciseForUnknownLesson(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"course.yaml": "title: x\nphases: []\nlessons: {}\n",
		"exercises/99.yaml": "lesson: \"99\"\nblocks: []\n",
		"vocab.yaml": "[]\n",
		"false-friends.yaml": "[]\n",
	})
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "99") {
		t.Fatalf("want error mentioning 99, got %v", err)
	}
}

func TestLoadRejectsMissingAccept(t *testing.T) { /* translate with empty accept => err mentions exercise id */ }
func TestLoadRejectsDuplicateVocabID(t *testing.T) { /* two vocab id "x" => err mentions "x" */ }
func TestLoadRejectsConjugateArityMismatch(t *testing.T) { /* forms len 6, accept len 5 => err */ }
```

Write the `writeTree` helper (creates dirs, writes files) in the test
file. Fill in the three stubbed tests with concrete fixtures like the
first two.

- [ ] **Step 4: Run — expect FAIL** (`Load` undefined).
Run: `PATH=/opt/homebrew/bin:$PATH go test ./server/internal/content/ -run TestLoad -v`

- [ ] **Step 5: Implement `load.go`** — read `course.yaml`, then for
  each `lessons` entry decide `Planned = file == ""`; read markdown
  when present (error if `file` set but unreadable). Glob
  `exercises/*.yaml`, decode, validate lesson exists and per-exercise
  arity/accept rules. Decode `vocab.yaml`, `false-friends.yaml`, check
  duplicate IDs. Wrap every error with `fmt.Errorf("%s: %w", relPath, err)`.

- [ ] **Step 6: Run — expect PASS.**
Run: `PATH=/opt/homebrew/bin:$PATH go test ./server/internal/content/ -v`

- [ ] **Step 7: `go mod tidy` then commit**

```bash
git add -A && git commit -m "feat(content): structured content loader with validation"
```

---

## Task 3: Migrate the real Obsidian content into `content/`

**Files:**
- Create: `content/course.yaml`
- Create: `content/lessons/01-pozdravi-i-upoznavanje.md`,
  `content/lessons/02-zamenice-i-prezent.md`
- Create: `content/exercises/01.yaml`, `content/exercises/02.yaml`
- Create: `content/vocab.yaml`, `content/false-friends.yaml`
- Modify: `/Users/grisha/plans/serbian/README.md` (add pointer line)

**Interfaces:**
- Consumes: `content.Load` from Task 2.
- Produces: a real `content/` tree that `Load` accepts.

- [ ] **Step 1: `course.yaml`** — phases A/B/C with the full lesson-id
  lists from `serbian/plan.md` (`01`–`09`, `10`–`20`, `21`–`30`).
  `lessons:` entries for `01`–`30` with `title` + `subtitle` taken
  from `plan.md`; only `01` and `02` get a `file:`.

- [ ] **Step 2: Lesson markdown** — copy
  `serbian/lessons/01-pozdravi-i-upoznavanje.md` and
  `02-zamenice-i-prezent.md` into `content/lessons/`, **deleting** the
  `## 7. Упражнения` / `## Упражнения` section and the `## Ответы`
  `<details>` block from each. Leave everything else verbatim.

- [ ] **Step 3: `exercises/01.yaml`** — encode blocks A, B, C, D from
  lesson 01. `accept` values come from the lesson's "Ответы" section
  (e.g. A1 → `["Zdravo! Kako si?"]`, A5 → `["Drago mi je. — Takođe.",
  "Drago mi je. Takođe.", "I meni je drago."]`). B items are
  `fill_blank`. C items are `fix_error`. D is `free` with `sample`.
  For multi-part prompts (B3), split into separate exercises.

- [ ] **Step 4: `exercises/02.yaml`** — blocks A (conjugate ×4), B
  (translate ×10), C (fill_blank ×8), D (transform → model as `free`
  with `sample` showing both question forms), E (fix_error ×5). Use
  the answer key at the bottom of lesson 02.

- [ ] **Step 5: `vocab.yaml`** — every entry from `serbian/vocab.md`
  lessons 01 and 02. Verbs get `pos: verb` and `note` carrying the
  conjugation type (`тип I`, `тип E (idem)`, etc.). Nouns with a
  marked gender get `gender:`. `tags` group them (`greetings`,
  `politeness`, `intro`, `verbs`, `time`, `adverbs`).

- [ ] **Step 6: `false-friends.yaml`** — every row from
  `serbian/reference/false-friends.md`. `group: top` for "Топ-опасные",
  `group: shop` for "Магазин / еда", `group: small` for "Мелкие частые".
  `not:` = the "А НЕ" column, `correct:` = the "«То самое»" column when
  present.

- [ ] **Step 7: Add a test that the real content loads**

`server/internal/content/real_test.go`:
```go
func TestRealContentLoads(t *testing.T) {
	c, err := Load("../../../content")
	if err != nil { t.Fatalf("real content: %v", err) }
	if len(c.Vocab) < 60 { t.Errorf("vocab = %d, want >= 60", len(c.Vocab)) }
	if len(c.Exercises["01"]) < 4 { t.Errorf("lesson 01 blocks = %d", len(c.Exercises["01"])) }
	if c.Lessons["03"].Planned != true { t.Error("03 should be planned") }
}
```

- [ ] **Step 8: Run the loader tests**
Run: `PATH=/opt/homebrew/bin:$PATH go test ./server/internal/content/ -v`
Expected: PASS including `TestRealContentLoads`.

- [ ] **Step 9: Pointer line in the Obsidian README** — add under the
  title of `/Users/grisha/plans/serbian/README.md`:
  `> 📱 Приложение по этому курсу: /Users/grisha/plans/serbian-app (make dev)`

- [ ] **Step 10: Commit**

```bash
git add -A && git commit -m "content: migrate lessons 01-02, vocab, false-friends from Obsidian"
```

---

## Task 4: `checker` package — normalize + check + diff

**Files:**
- Create: `server/internal/checker/checker.go`
- Create: `server/internal/checker/checker_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  ```go
  type Chunk struct { Text string; OK bool }   // OK=false => highlight as wrong
  type Result struct {
      OK       bool
      Expected string     // closest accept entry (for display after attempt)
      Diff     []Chunk    // word-level diff of answer vs Expected
  }
  func Normalize(s string) string
  func Check(answer string, accept []string) Result
  func CheckForms(answers []string, acceptForms [][]string) []Result
  ```

- [ ] **Step 1: Failing tests** in `checker_test.go`:

```go
func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"  Zdravo!  ":        "zdravo",
		"Kako   si?":         "kako si",
		"Iz Rusije sam.":     "iz rusije sam",
		"“Ćao”":              "ćao",
		"Drago mi je . ":     "drago mi je",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckExactAndVariant(t *testing.T) {
	accept := []string{"Zdravo! Kako si?", "Ćao! Kako si?"}
	if !Check("zdravo kako si", accept).OK { t.Error("normalized exact should pass") }
	if !Check("Ćao! Kako si?", accept).OK { t.Error("second variant should pass") }
}

func TestCheckWrongProducesDiffAndExpected(t *testing.T) {
	r := Check("Zdravo! Kako sti?", []string{"Zdravo! Kako si?"})
	if r.OK { t.Fatal("should fail") }
	if r.Expected != "Zdravo! Kako si?" { t.Errorf("Expected = %q", r.Expected) }
	var wrong int
	for _, c := range r.Diff { if !c.OK { wrong++ } }
	if wrong == 0 { t.Error("expected at least one wrong chunk") }
}

func TestCheckForms(t *testing.T) {
	rs := CheckForms(
		[]string{"govorim", "govoris", "govori"},
		[][]string{{"govorim"}, {"govoriš"}, {"govori"}},
	)
	if !rs[0].OK || rs[1].OK || !rs[2].OK {
		t.Errorf("results = %+v", rs)
	}
}
```

- [ ] **Step 2: Run — expect FAIL.**
Run: `PATH=/opt/homebrew/bin:$PATH go test ./server/internal/checker/ -v`

- [ ] **Step 3: Implement.**
  - `Normalize`: `strings.TrimSpace`, `strings.ToLower` (Unicode via
    `strings.ToLower` is fine for Serbian latin), replace curly quotes
    `“”„‟` and `«»` with nothing, replace `–—` with `-`,
    `strings.Fields`+`strings.Join` to collapse whitespace, then trim a
    trailing run of `.!?,;:` from the joined string.
  - `Check`: normalize answer; for each accept entry, normalize and
    compare; on match `OK=true, Expected=<original accept>`. On no
    match pick the accept entry with the smallest Levenshtein distance
    (word list) as `Expected`, build `Diff` by aligning answer words to
    Expected words (simple LCS; non-matching answer words get
    `OK:false`, matching get `OK:true`).
  - `CheckForms`: element-wise `Check(answers[i], acceptForms[i])`;
    missing answers (`i >= len(answers)`) → `Result{OK:false}`.
  - Add an unexported `levenshtein([]string,[]string) int` and
    `lcsDiff` helper.

- [ ] **Step 4: Run — expect PASS.**
Run: `PATH=/opt/homebrew/bin:$PATH go test ./server/internal/checker/ -v`

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat(checker): answer normalization, matching, word diff"
```

---

## Task 5: `srs` package — SM-2 scheduler

**Files:**
- Create: `server/internal/srs/srs.go`
- Create: `server/internal/srs/srs_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  ```go
  type Grade int
  const ( Again Grade = 0; Hard Grade = 1; Good Grade = 2; Easy Grade = 3 )
  type State string
  const ( New State = "new"; Learning State = "learning"; Review State = "review" )
  type Card struct {
      Ease         float64
      IntervalDays int
      Reps         int
      Lapses       int
      State        State
      Due          time.Time  // zero for New
  }
  // Schedule returns the updated card after a review at time now.
  func Schedule(c Card, g Grade, now time.Time) Card
  func NewCard() Card   // Ease 2.5, State New
  ```

- [ ] **Step 1: Failing tests** in `srs_test.go` — a table covering:

```go
var day = 24 * time.Hour
func d(c Card) int { return c.IntervalDays }

func TestNewCardGoodThenGood(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	c := Schedule(NewCard(), Good, now)          // new + good => 1 day
	if d(c) != 1 || c.State != Review { t.Fatalf("%+v", c) }
	c = Schedule(c, Good, now.Add(day))          // 1 * 2.5 = 3 (rounded)
	if d(c) != 3 { t.Errorf("interval = %d, want 3", d(c)) }
}

func TestNewCardEasy(t *testing.T)   { /* new + easy => 4 days, State Review */ }
func TestNewCardAgain(t *testing.T)  { /* new + again => State Learning, Due == now (same day) */ }

func TestReviewAgainIsLapse(t *testing.T) {
	now := time.Now().UTC()
	c := Card{Ease: 2.5, IntervalDays: 10, Reps: 3, State: Review}
	c = Schedule(c, Again, now)
	if c.Lapses != 1 || c.IntervalDays != 1 || c.State != Learning { t.Fatalf("%+v", c) }
	if c.Ease > 2.31 { t.Errorf("ease not decremented: %v", c.Ease) }
}

func TestReviewHardGoodEasy(t *testing.T) { /* interval multipliers + ease deltas per spec */ }

func TestEaseFloorAndIntervalCap(t *testing.T) {
	c := Card{Ease: 1.3, IntervalDays: 300, Reps: 9, State: Review}
	c = Schedule(c, Again, time.Now().UTC())
	if c.Ease < 1.3 { t.Errorf("ease below floor: %v", c.Ease) }
	c2 := Card{Ease: 3.0, IntervalDays: 300, Reps: 9, State: Review}
	c2 = Schedule(c2, Easy, time.Now().UTC())
	if c2.IntervalDays > 365 { t.Errorf("interval above cap: %d", c2.IntervalDays) }
}

func TestDueIsIntervalDaysAhead(t *testing.T) {
	now := time.Date(2026, 9, 6, 8, 0, 0, 0, time.UTC)
	c := Schedule(Card{Ease: 2.5, IntervalDays: 2, Reps: 1, State: Review}, Good, now)
	wantDay := now.Add(time.Duration(c.IntervalDays) * day).YearDay()
	if c.Due.YearDay() != wantDay { t.Errorf("due = %v", c.Due) }
}
```

- [ ] **Step 2: Run — expect FAIL.**
Run: `PATH=/opt/homebrew/bin:$PATH go test ./server/internal/srs/ -v`

- [ ] **Step 3: Implement per spec §"Пакет internal/srs".**
  - `clampEase(e) = max(1.3, e)`; `capInterval(n) = min(365, n)`.
  - New/Learning card: `Again` → `State=Learning, IntervalDays=0,
    Due=now`; `Hard`/`Good` → `State=Review, IntervalDays=1, Due=now+1d`;
    `Easy` → `State=Review, IntervalDays=4, Due=now+4d`.
  - Review card:
    - `Again`: `Lapses++`, `Ease=clamp(Ease-0.2)`, `IntervalDays=1`,
      `State=Learning`.
    - `Hard`: `Ease=clamp(Ease-0.15)`, `IntervalDays=cap(round(IntervalDays*1.2))`.
    - `Good`: `IntervalDays=cap(round(IntervalDays*Ease))`.
    - `Easy`: `Ease=Ease+0.15`, `IntervalDays=cap(round(IntervalDays*Ease*1.3))`.
  - Always `Reps++` (except keep `Reps` semantics simple: increment on
    every non-Again grade), set `Due=now.AddDate(0,0,IntervalDays)`
    truncated to date (use `time.Date(y,m,d,0,0,0,0,now.Location())`
    then add days) — but for `Again` on new card `Due=now`.
  - `round` = `int(math.Round(x))`, min 1 for review intervals.

- [ ] **Step 4: Run — expect PASS.** Iterate until the table is green.
Run: `PATH=/opt/homebrew/bin:$PATH go test ./server/internal/srs/ -v`

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat(srs): SM-2 scheduler"
```

---

## Task 6: `store` package — SQLite persistence

**Files:**
- Create: `server/internal/store/store.go`
- Create: `server/internal/store/store_test.go`

**Interfaces:**
- Consumes: `srs.Card`, `srs.Grade`, `srs.Schedule` (Task 5).
- Produces:
  ```go
  type Store struct { /* *sql.DB */ }
  func Open(path string) (*Store, error)   // applies schema; ":memory:" ok
  func (s *Store) Close() error

  type CardRow struct {
      CardID, Kind, RefID string
      srs.Card
  }
  // EnsureCards inserts missing cards (state "new") for the given ids.
  func (s *Store) EnsureCards(ids []struct{ CardID, Kind, RefID string }) error
  // DueQueue returns due/learning cards (Due <= today) plus up to
  // newLimit "new" cards, ordered overdue-first then new.
  func (s *Store) DueQueue(today time.Time, newLimit int) ([]CardRow, error)
  // GradeCard loads the card, runs srs.Schedule, saves it, writes a
  // reviews row, and returns the updated card.
  func (s *Store) GradeCard(cardID string, g srs.Grade, now time.Time) (srs.Card, error)
  func (s *Store) ReviewedToday(today time.Time) (int, error)
  func (s *Store) StreakDays(today time.Time) (int, error)

  type Attempt struct { ExerciseID, Lesson, Block, Answer string; Correct bool }
  func (s *Store) AddAttempt(a Attempt, now time.Time) error
  type WeakExercise struct { ExerciseID, Lesson string; Wrong, Total int }
  func (s *Store) WeakExercises(limit int) ([]WeakExercise, error)

  func (s *Store) SetLessonStatus(lesson, status string, now time.Time) error
  func (s *Store) LessonStatuses() (map[string]string, error)
  func (s *Store) CardStats() (total, known int, err error)  // known = state "review" & interval >= 7
  ```

- [ ] **Step 1: Failing tests** in `store_test.go` (use `Open(":memory:")`
  or `Open(filepath.Join(t.TempDir(), "t.db"))`):

```go
func TestEnsureCardsIdempotent(t *testing.T) {
	s := mustOpen(t)
	ids := []struct{ CardID, Kind, RefID string }{{"vocab:zdravo","vocab","zdravo"}}
	if err := s.EnsureCards(ids); err != nil { t.Fatal(err) }
	if err := s.EnsureCards(ids); err != nil { t.Fatal(err) }
	total, _, _ := s.CardStats()
	if total != 1 { t.Errorf("total = %d", total) }
}

func TestDueQueueOrdersOverdueThenNew(t *testing.T) {
	s := mustOpen(t)
	today := time.Date(2026,9,6,0,0,0,0,time.UTC)
	s.EnsureCards(ids3(t)) // vocab:a, vocab:b, vocab:c all "new"
	// grade a to Review due in the past by manipulating via GradeCard w/ old now, then...
	// simpler: expose a test helper setDue; instead assert new cards limited:
	q, _ := s.DueQueue(today, 2)
	if len(q) != 2 { t.Fatalf("want 2 new, got %d", len(q)) }
}

func TestGradeCardPersistsAndLogsReview(t *testing.T) {
	s := mustOpen(t)
	s.EnsureCards([]struct{ CardID, Kind, RefID string }{{"vocab:x","vocab","x"}})
	now := time.Date(2026,9,6,10,0,0,0,time.UTC)
	c, err := s.GradeCard("vocab:x", srs.Good, now)
	if err != nil { t.Fatal(err) }
	if c.State != srs.Review { t.Errorf("state = %s", c.State) }
	n, _ := s.ReviewedToday(now)
	if n != 1 { t.Errorf("reviewed today = %d", n) }
}

func TestAddAttemptAndWeakExercises(t *testing.T) {
	s := mustOpen(t)
	now := time.Now().UTC()
	s.AddAttempt(Attempt{"02-C-1","02","C","x",false}, now)
	s.AddAttempt(Attempt{"02-C-1","02","C","y",false}, now)
	s.AddAttempt(Attempt{"02-C-1","02","C","radi",true}, now)
	w, _ := s.WeakExercises(10)
	if len(w) != 1 || w[0].Wrong != 2 || w[0].Total != 3 { t.Fatalf("%+v", w) }
}

func TestLessonStatusRoundTrip(t *testing.T) {
	s := mustOpen(t)
	s.SetLessonStatus("01","done", time.Now().UTC())
	m, _ := s.LessonStatuses()
	if m["01"] != "done" { t.Errorf("m = %+v", m) }
}

func TestStreakDays(t *testing.T) { /* reviews on today and today-1 => streak 2; gap => reset */ }
```

Add `mustOpen`, `ids3` helpers. For `TestDueQueueOrdersOverdueThenNew`
and `TestStreakDays` add an unexported test-only helper in
`store_test.go` that inserts rows via `s.db` — expose the `*sql.DB` as
lowercase `db` field and access from the same package test.

- [ ] **Step 2: Run — expect FAIL.**
Run: `PATH=/opt/homebrew/bin:$PATH go test ./server/internal/store/ -v`

- [ ] **Step 3: Implement `store.go`.**
  - Import `_ "modernc.org/sqlite"`; `sql.Open("sqlite", path)`.
  - `applySchema` runs the four `CREATE TABLE IF NOT EXISTS` from the
    spec + `PRAGMA user_version`. Set `PRAGMA journal_mode=WAL` and
    `PRAGMA busy_timeout=5000`.
  - Store dates as `now.UTC().Format(time.RFC3339)`; store SRS `Due` as
    `Due.Format("2006-01-02")`.
  - `DueQueue`: `SELECT ... WHERE state IN ('learning','review') AND
    due <= ? ORDER BY due ASC` then a second query for
    `state='new' LIMIT ?`; concat.
  - `GradeCard`: `SELECT` row → build `srs.Card` → `srs.Schedule` →
    `UPDATE` + `INSERT INTO reviews`. Wrap in a transaction.
  - `StreakDays`: `SELECT DISTINCT date(reviewed_at)` desc; walk from
    `today` backward counting consecutive days.
  - `WeakExercises`: `SELECT exercise_id, lesson, SUM(1-correct) wrong,
    COUNT(*) total FROM attempts GROUP BY exercise_id HAVING total >= 2
    AND wrong*2 >= total ORDER BY wrong DESC LIMIT ?`.
  - `CardStats`: `total = COUNT(*)`, `known = COUNT(*) WHERE
    state='review' AND interval_days >= 7`.

- [ ] **Step 4: Run — expect PASS.**
Run: `PATH=/opt/homebrew/bin:$PATH go test ./server/internal/store/ -v`

- [ ] **Step 5: `go mod tidy` then commit**

```bash
git add -A && git commit -m "feat(store): SQLite persistence for SRS, attempts, progress"
```

---

## Task 7: `api` package — wire content + store + srs + checker

**Files:**
- Create: `server/internal/api/dto.go`
- Modify: `server/internal/api/api.go`
- Create: `server/internal/api/api_test.go`
- Modify: `server/main.go` (build real `Deps`, start watcher)
- Create: `server/internal/content/watch.go`

**Interfaces:**
- Consumes: `content.Course` + `content.Load`, `store.Store`,
  `srs`, `checker`.
- Produces the HTTP API from the spec. `Deps`:
  ```go
  type Deps struct {
      Course func() *content.Course   // returns current snapshot
      Store  *store.Store
      Now    func() time.Time         // injectable clock for tests
      Stale  func() bool
  }
  ```

- [ ] **Step 1: `content/watch.go`**

```go
package content

import (
	"log"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watch loads dir into an atomic snapshot and reloads on file changes
// (300ms debounce). stale reports whether the last reload failed.
func Watch(dir string) (get func() *Course, stale func() bool, err error) {
	first, err := Load(dir)
	if err != nil { return nil, nil, err }
	var snap atomic.Pointer[Course]
	var bad atomic.Bool
	snap.Store(first)
	w, err := fsnotify.NewWatcher()
	if err != nil { return nil, nil, err }
	_ = w.Add(dir)
	for _, sub := range []string{"lessons", "exercises"} {
		_ = w.Add(filepath.Join(dir, sub))
	}
	go func() {
		var t *time.Timer
		for range w.Events {
			if t != nil { t.Stop() }
			t = time.AfterFunc(300*time.Millisecond, func() {
				c, err := Load(dir)
				if err != nil { log.Printf("content reload failed: %v", err); bad.Store(true); return }
				snap.Store(c); bad.Store(false); log.Printf("content reloaded")
			})
		}
	}()
	return snap.Load, bad.Load, nil
}
```

- [ ] **Step 2: `dto.go`** — response structs that intentionally omit
  `accept`: `courseDTO`, `lessonDTO`, `exerciseBlockDTO`
  (`exerciseDTO` has `ID,Type,Prompt,Forms,Meta` — **no** `Accept`,
  `Explain`, `Sample`), `checkResultDTO{OK bool; Diff []checker.Chunk;
  Expected, Explain, Sample string}`, `vocabDTO`, `falseFriendDTO`,
  `reviewCardDTO`, `progressDTO`.

- [ ] **Step 3: Failing tests** in `api_test.go` — build a `Deps` from
  the `content` fixture tree + an in-memory `store`, fixed clock:

```go
func newTestAPI(t *testing.T) (http.Handler, *store.Store) {
	c, err := content.Load("../content/testdata/content")
	if err != nil { t.Fatal(err) }
	st, _ := store.Open(":memory:")
	h := Handler(Deps{
		Course: func() *content.Course { return c },
		Store:  st,
		Now:    func() time.Time { return time.Date(2026,9,6,12,0,0,0,time.UTC) },
		Stale:  func() bool { return false },
	})
	return h, st
}

func TestGetCourseHasStatuses(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "GET", "/api/course", "")
	if rr.Code != 200 { t.Fatal(rr.Code) }
	// body contains phases and lesson "01"
}

func TestExercisesEndpointHidesAccept(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "GET", "/api/lessons/01/exercises", "")
	if strings.Contains(rr.Body.String(), "Zdravo") { // accept value from fixture
		t.Error("accept leaked to client")
	}
}

func TestCheckRecordsAttemptAndReturnsResult(t *testing.T) {
	h, st := newTestAPI(t)
	rr := do(h, "POST", "/api/lessons/01/exercises/01-A-1/check", `{"answer":"Zdravo"}`)
	if rr.Code != 200 { t.Fatal(rr.Code) }
	// body has "ok":true
	w, _ := st.WeakExercises(10) // no weak yet
	_ = w
}

func TestReviewQueueThenGrade(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "GET", "/api/review/queue", "")
	if rr.Code != 200 { t.Fatal(rr.Code) }
	// pick a card id from body, then:
	rr2 := do(h, "POST", "/api/review/grade", `{"card_id":"vocab:zdravo","grade":2}`)
	if rr2.Code != 200 { t.Fatalf("grade: %d %s", rr2.Code, rr2.Body) }
}

func TestLessonCompleteThenProgress(t *testing.T) {
	h, _ := newTestAPI(t)
	do(h, "POST", "/api/lessons/01/complete", "")
	rr := do(h, "GET", "/api/progress", "")
	if !strings.Contains(rr.Body.String(), `"done":1`) { t.Errorf("progress: %s", rr.Body) }
}

func TestUnknownLesson404(t *testing.T) {
	h, _ := newTestAPI(t)
	if do(h, "GET", "/api/lessons/zz", "").Code != 404 { t.Error("want 404") }
}
```

Add `do(h, method, path, body)` returning `*httptest.ResponseRecorder`.
Ensure the fixture `exercises/01.yaml` from Task 2 uses ids `01-A-1`
etc. and vocab id `zdravo`.

- [ ] **Step 4: Run — expect FAIL.**
Run: `PATH=/opt/homebrew/bin:$PATH go test ./server/internal/api/ -v`

- [ ] **Step 5: Implement handlers in `api.go`.** Route table:

```go
mux.HandleFunc("GET /api/health", h.health)
mux.HandleFunc("GET /api/course", h.getCourse)
mux.HandleFunc("GET /api/lessons/{id}", h.getLesson)
mux.HandleFunc("GET /api/lessons/{id}/exercises", h.getExercises)
mux.HandleFunc("POST /api/lessons/{id}/exercises/{exId}/check", h.checkExercise)
mux.HandleFunc("POST /api/lessons/{id}/complete", h.completeLesson)
mux.HandleFunc("GET /api/vocab", h.getVocab)
mux.HandleFunc("GET /api/false-friends", h.getFalseFriends)
mux.HandleFunc("GET /api/review/queue", h.reviewQueue)
mux.HandleFunc("POST /api/review/grade", h.reviewGrade)
mux.HandleFunc("GET /api/progress", h.getProgress)
```

  - `getCourse`: merge `Course().Phases`/`Lessons` with
    `store.LessonStatuses()` → `status` per lesson (`not_started` default).
  - `getLesson`: 404 if id not in `Course().Lessons`; return
    `{id,title,subtitle,planned,markdown,status}`.
  - `getExercises`: map blocks → DTO without `accept`/`explain`/`sample`.
  - `checkExercise`: find exercise by id within lesson (404 if absent);
    `free` → always `OK=true` after echoing `sample`, still record
    attempt with `correct` from request body field `self` (bool,
    default true); `conjugate` → body `{answers:[...]}` →
    `checker.CheckForms`; else `checker.Check(body.answer, ex.Accept)`.
    Record `store.AddAttempt`. Response includes `explain` + (for
    `free`) `sample`.
  - `completeLesson`: `store.SetLessonStatus(id,"done",now)`; 404 if
    unknown lesson.
  - `getVocab` / `getFalseFriends`: filter by query params
    (`lesson`,`tag`,`q` / `group`,`q`); case-insensitive `q` substring
    over latin+cyrillic+ru (vocab) or sr+means (ff).
  - `reviewQueue`: build card ids from `Course().Vocab` (`vocab:<id>`)
    and `FalseFriends` (`ff:<id>`); `store.EnsureCards`; `DueQueue(now,
    15)`; hydrate each card with its vocab/ff payload; DTO.
  - `reviewGrade`: `store.GradeCard(body.card_id, srs.Grade(body.grade),
    now)`; return `{due, interval_days, state}`.
  - `getProgress`: assemble `progressDTO` per spec (phases done/total,
    srs block from `CardStats`+`DueQueue` counts+`ReviewedToday`,
    `WeakExercises(10)` hydrated with prompts from `Course()`,
    `StreakDays`, recent lessons from statuses).
  - `health`: `{"status":"ok","content_stale": deps.Stale()}`.
  - Helper `h.decode(r, &body)` + `h.fail(w, code, msg)`.

- [ ] **Step 6: Wire `main.go`** — `content.Watch(*contentDir)`,
  `store.Open(*dbPath)` (create `data/` dir first), pass real `Deps`
  with `Now: time.Now`. Keep `spaHandler`.

- [ ] **Step 7: Run — expect PASS; then full backend suite.**
Run: `PATH=/opt/homebrew/bin:$PATH go test ./server/...`

- [ ] **Step 8: Manual smoke**

```bash
PATH=/opt/homebrew/bin:$PATH go run ./server -addr :8080 &
sleep 1
curl -s localhost:8080/api/course | head -c 200
curl -s localhost:8080/api/review/queue | head -c 200
curl -s -XPOST localhost:8080/api/lessons/01/exercises/01-A-1/check -d '{"answer":"Zdravo! Kako si?"}'
kill %1
```

- [ ] **Step 9: Commit**

```bash
git add -A && git commit -m "feat(api): full JSON API wiring content, store, srs, checker"
```

---

## Task 8: Frontend foundation — typed API client, types, layout

**Files:**
- Modify: `web/src/api.ts`, `web/src/types.ts`, `web/src/App.vue`,
  `web/src/components/AppNav.vue`, `web/src/style.css`
- Create: `web/src/api.test.ts`

**Interfaces:**
- Produces `web/src/types.ts` mirroring the DTOs, and `web/src/api.ts`:
  ```ts
  export class ApiError extends Error { constructor(public status: number, msg: string) }
  export const api = {
    course(): Promise<Course>,
    lesson(id: string): Promise<Lesson>,
    exercises(id: string): Promise<ExerciseBlock[]>,
    check(lesson: string, exId: string, payload: CheckPayload): Promise<CheckResult>,
    completeLesson(id: string): Promise<void>,
    vocab(params?: {lesson?: string; tag?: string; q?: string}): Promise<Vocab[]>,
    falseFriends(params?: {group?: string; q?: string}): Promise<FalseFriend[]>,
    reviewQueue(): Promise<ReviewCard[]>,
    grade(cardId: string, grade: number): Promise<GradeResult>,
    progress(): Promise<Progress>,
  }
  ```

- [ ] **Step 1: Failing test** `web/src/api.test.ts`:

```ts
import { describe, it, expect, vi } from 'vitest'
import { api, ApiError } from './api'

describe('api', () => {
  it('parses JSON on 200', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ title: 'X', phases: [], lessons: [] }), { status: 200 })
    ))
    const c = await api.course()
    expect(c.title).toBe('X')
  })
  it('throws ApiError on non-2xx', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ error: 'nope' }), { status: 404 })
    ))
    await expect(api.lesson('zz')).rejects.toBeInstanceOf(ApiError)
  })
})
```

- [ ] **Step 2: Run — expect FAIL.**  Run: `cd web && npm run test`

- [ ] **Step 3: Implement `api.ts`** — a `request<T>(path, init?)`
  helper: `fetch('/api' + path, ...)`, on `!res.ok` read `{error}` and
  throw `new ApiError(res.status, error)`, else `res.json()` (or
  `undefined` for 204). Build query strings for `vocab`/`falseFriends`.
  Write `types.ts`.

- [ ] **Step 4: Run — expect PASS.**  Run: `cd web && npm run test`

- [ ] **Step 5: Layout** — `App.vue`: sticky top `<AppNav>` (flex, wraps
  on mobile) + `<main class="mx-auto max-w-3xl p-4">`. `AppNav.vue`:
  `RouterLink`s with active class. Basic Tailwind dark-mode-aware
  palette in `style.css` (`@media (prefers-color-scheme: dark)` body
  colors). Keep it plain.

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "feat(web): typed API client, DTO types, app layout"
```

---

## Task 9: Course + Lesson screens with exercise components

**Files:**
- Modify: `web/src/views/CourseView.vue`, `web/src/views/LessonView.vue`
- Create: `web/src/components/MarkdownView.vue`
- Create: `web/src/stores/course.ts`
- Create: `web/src/components/exercises/{ExerciseBlock,ExerciseItem,TextAnswer,ConjugateAnswer,FreeAnswer}.vue`
- Create: `web/src/components/exercises/TextAnswer.test.ts`

**Interfaces:**
- Consumes: `api` (Task 8).
- Produces: `useCourseStore` with `phases`, `lessons`, `load()`,
  `markDone(id)`.

- [ ] **Step 1: `useCourseStore`** (Pinia) — `load()` calls
  `api.course()` and caches; `markDone(id)` calls
  `api.completeLesson(id)` then patches local status to `done`.

- [ ] **Step 2: `CourseView.vue`** — on mount `store.load()`; render
  each phase as a section with a lesson list; each lesson row: number,
  title, subtitle, status badge (`not_started`/`in_progress`/`done`),
  `RouterLink` to `/lesson/:id`. "Запланирован" tag when `planned`.

- [ ] **Step 3: `MarkdownView.vue`** — prop `source: string`; render
  with `markdown-it` (`{ html: false, linkify: false, typographer: true }`),
  `v-html` the result into a `prose`-ish container (hand-rolled
  `:deep()` styles for `table`, `th/td` borders, `blockquote`, `code`).

- [ ] **Step 4: `TextAnswer.vue` + its test.** Props:
  `{ lesson: string; exerciseId: string; type: 'translate'|'fill_blank'|'fix_error'; prompt: string }`.
  State: `answer`, `result: CheckResult | null`, `pending`. Enter or
  the "Проверить" button → `api.check(lesson, exerciseId, { answer })`
  → store `result`. Render prompt; on result show ✓/✗, the `Diff`
  chunks (wrong chunks get `bg-red-200`/`line-through`), `Expected`
  when `!ok`, `explain` when present, a "Ещё раз" button that clears.
  Emit `graded(ok: boolean)`.

  `TextAnswer.test.ts` (Vitest + `@vue/test-utils`, mock `api.check`):
  ```ts
  it('shows expected answer and diff after a wrong check', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: false, expected: 'Zdravo! Kako si?',
      diff: [{ text: 'zdravo', ok: true }, { text: 'kako', ok: true }, { text: 'sti', ok: false }] } as any)
    const w = mount(TextAnswer, { props: { lesson: '01', exerciseId: '01-A-1', type: 'translate', prompt: 'x' } })
    await w.find('input').setValue('zdravo kako sti')
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(w.text()).toContain('Zdravo! Kako si?')
    expect(w.find('.chunk-wrong').exists()).toBe(true)
  })
  ```

- [ ] **Step 5: Run the component test — FAIL then implement then PASS.**
Run: `cd web && npm run test`

- [ ] **Step 6: `ConjugateAnswer.vue`** — props include
  `forms: string[]`, `meta?: string`. Render one labelled input per
  form; "Проверить" → `api.check(lesson, exId, { answers })`; response
  is `CheckResult[]` for conjugate (adjust `api.check` return typing to
  `CheckResult | CheckResult[]`). Mark each field ✓/✗.

- [ ] **Step 7: `FreeAnswer.vue`** — textarea; "Показать образец" →
  `api.check(lesson, exId, { answer, self: true })` returns `sample`;
  then show "Справился" / "Не справился" buttons → second `check` call
  with `{ answer, self: boolean }` to record the real attempt. Emit
  `graded`.

- [ ] **Step 8: `ExerciseItem.vue`** — switch on `type` → the right
  component. `ExerciseBlock.vue` — title, instruction, `v-for`
  `ExerciseItem`, running score `N / M` updated via `graded` events.

- [ ] **Step 9: `LessonView.vue`** — route param `id`; fetch
  `api.lesson(id)` + `api.exercises(id)`. Render `<MarkdownView>` then
  the blocks. If `planned` → a "Скоро" panel, no exercises. Sticky
  footer button "Отметить пройденным" → `store.markDone(id)` →
  toast/inline confirmation. When first exercise is answered, fire
  `api.completeLesson`? No — only the explicit button; but set
  `in_progress` implicitly server-side on first attempt (already
  happens via attempts? add: `completeLesson` only sets done; for
  `in_progress`, `AddAttempt` handler also upserts status
  `in_progress` if not `done` — add that to Task 7 Step 5 checkExercise).

- [ ] **Step 10: Full frontend test + manual check**
Run: `cd web && npm run test`
Then `make dev`, open `/course`, open lesson 01, answer an exercise.

- [ ] **Step 11: Commit**

```bash
git add -A && git commit -m "feat(web): course list, lesson reader, interactive exercises"
```

> **Note for Task 7 executor if reading later:** `checkExercise` must
> also call `store.SetLessonStatus(lesson, "in_progress", now)` when the
> lesson's current status is not `"done"`. Add `LessonStatus(id)` to
> store or read from `LessonStatuses()`.

---

## Task 10: Review screen (SRS trainer)

**Files:**
- Modify: `web/src/views/ReviewView.vue`
- Create: `web/src/stores/review.ts`
- Create: `web/src/stores/review.test.ts`

**Interfaces:**
- Consumes: `api.reviewQueue`, `api.grade`.
- Produces: `useReviewStore` — `queue: ReviewCard[]`, `index`,
  `current`, `sessionCount`, `load()`, `grade(g: number)`.

- [ ] **Step 1: Failing test** `review.test.ts`:

```ts
it('re-queues an "again" card to the end and advances otherwise', async () => {
  const store = useReviewStore()
  store.$patch({ queue: [card('a'), card('b')], index: 0 })
  vi.spyOn(api, 'grade').mockResolvedValue({ due: '', interval_days: 1, state: 'learning' })
  await store.grade(0)            // Again on 'a'
  expect(store.current?.card_id).toBe('b')
  expect(store.queue.map(c => c.card_id)).toEqual(['b', 'a'])
  await store.grade(2)            // Good on 'b'
  expect(store.current?.card_id).toBe('a')
})
```

- [ ] **Step 2: Run — FAIL.**  Run: `cd web && npm run test`

- [ ] **Step 3: Implement `review.ts`** — `grade(g)` calls
  `api.grade(current.card_id, g)`, increments `sessionCount`, if
  `g === 0` push a copy of `current` to end of `queue`; always
  `index++`. `current` getter = `queue[index]`. `remaining` getter.

- [ ] **Step 4: Run — PASS.**

- [ ] **Step 5: `ReviewView.vue`** — on mount `store.load()`. Card:
  big `latin`, small `cyrillic`; "Показать" reveals `ru` + `note`.
  Four buttons "Опять(1) / Трудно(2) / Хорошо(3) / Легко(4)" →
  `store.grade`. Keyboard: `Space` reveals, `1..4` grade (only when
  revealed). Progress `sessionCount` done, `remaining` left. Empty
  queue → "На сегодня всё 🎉" + link to `/`.

- [ ] **Step 6: Test + manual**
Run: `cd web && npm run test` then `make dev`, open `/review`.

- [ ] **Step 7: Commit**

```bash
git add -A && git commit -m "feat(web): SRS review screen with keyboard shortcuts"
```

---

## Task 11: Vocab + False Friends screens

**Files:**
- Modify: `web/src/views/VocabView.vue`, `web/src/views/FalseFriendsView.vue`

- [ ] **Step 1: `VocabView.vue`** — fetch `api.vocab()`. Controls: text
  search (debounced, refetch with `q`), lesson `<select>` (unique
  lessons), tag `<select>`, a latin/cyrillic toggle. Table: script
  column, ru, note, lesson. Row count.

- [ ] **Step 2: "Учить выборку"** — button that takes the currently
  filtered list into a local flashcard mode (no server SRS write):
  show script side → reveal ru → "дальше"/"не знаю" just cycles;
  purely a quick self-quiz. Keep it in-component, no store.

- [ ] **Step 3: `FalseFriendsView.vue`** — fetch `api.falseFriends()`.
  Group `<select>` (top/shop/small/all) + search. Table: `sr`, means,
  "а не" (`not`), "правильно" (`correct`). Color the `not` cell subtly.

- [ ] **Step 4: Manual check** — `make dev`, open `/vocab` and
  `/false-friends`, exercise filters.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat(web): vocab browser and false-friends table"
```

---

## Task 12: Dashboard screen

**Files:**
- Modify: `web/src/views/DashboardView.vue`

- [ ] **Step 1: Fetch `api.progress()` on mount.**

- [ ] **Step 2: Render cards:**
  - "Повторить слова" — `srs.due_today` due + `srs.new_available` new;
    big `RouterLink` button to `/review`; `srs.reviewed_today` today.
  - "Прогресс" — per phase a labelled bar `done/total`.
  - "Продолжить" — most recent `in_progress` lesson (from
    `recent_lessons`), link to it; else link to first `not_started`.
  - "Слабые места" — list `weak_exercises` (prompt, `wrong/total`),
    each linking to its lesson.
  - "Серия" — `streak_days` 🔥.

- [ ] **Step 3: Empty states** — no progress yet → friendly "начни с
  урока 01" panel.

- [ ] **Step 4: Manual check** — `make dev`, open `/`, verify numbers
  move after doing a review and completing a lesson.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat(web): progress dashboard"
```

---

## Task 13: Production build, embed, docs, final pass

**Files:**
- Modify: `Makefile`, `README.md`
- Verify: `server/web/embed.go`

- [ ] **Step 1: `make build`** — confirm it produces `web/dist`, copies
  to `server/web/dist`, and `go build -o serbian-app ./server`
  succeeds with the SPA embedded.

- [ ] **Step 2: Run the binary standalone**

```bash
cd /Users/grisha/plans/serbian-app && ./serbian-app -addr :8080
```
Open `http://localhost:8080` — the SPA loads from the binary, all five
screens work, SPA deep-links (`/lesson/01` refresh) resolve via
`spaHandler`.

- [ ] **Step 3: `make test`** — whole suite green
  (`go test ./server/...` + `vitest run`).

- [ ] **Step 4: README** — final content: what it is, `make dev`,
  `make build` + `./serbian-app`, where content lives, how to add a
  lesson (copy `exercises/NN.yaml` shape, add `file:` in `course.yaml`,
  drop the markdown in `content/lessons/`), link to the spec. Add a
  `content/exercises/_TEMPLATE.yaml` with every exercise type
  commented.

- [ ] **Step 5: `.gitignore` sanity** — `serbian-app` binary,
  `server/web/dist/` (except `.gitkeep`), `data/`, `node_modules/`,
  `web/dist/` all ignored; `git status` clean after a build.

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "chore: production build, embed SPA, README, exercise template"
```

---

## Self-Review

**1. Spec coverage**

| Spec section | Task |
|---|---|
| Content model (`course.yaml`, lessons, exercises, vocab, false-friends) | 2, 3 |
| Exercise types translate/fill_blank/fix_error/conjugate/free | 2 (types), 3 (data), 9 (UI) |
| Answer checker (Normalize, Check, diff, conjugate) | 4 |
| SRS SM-2 (`Schedule`, lazy card creation, daily queue) | 5, 6, 7 |
| SQLite schema (srs_cards, reviews, attempts, lesson_progress) | 6 |
| HTTP API (all 11 endpoints) | 7 |
| `accept` never leaves server except `/check` | 7 (dto.go, Step 2 + Step 5), test in Step 3 |
| Content hot-reload (fsnotify, atomic snapshot, stale flag) | 7 (watch.go) |
| `go:embed` single binary + SPA fallback | 1 (embed.go, spaHandler), 13 |
| Frontend screens: dashboard, course, lesson, review, vocab, false-friends | 8–12 |
| Exercise components | 9 |
| SRS trainer with keyboard shortcuts | 10 |
| Pinia stores (`useCourse`, `useReview`) | 9, 10 |
| Error handling (ApiError, inline banners, 404/503) | 7, 8 |
| Testing: checker, srs, content, api, api.ts, TextAnswer, useReview | 4,5,2,7,8,9,10 |
| `make dev` / `make build` / `make test` | 1, 13 |
| Migration of existing content + Obsidian README pointer | 3 |

No uncovered spec sections.

**2. Placeholder scan** — Task 2 Step 3 leaves three tests "stubbed"
but names the exact fixture/assertion for each; acceptable as they
mirror the two fully-written tests above them. Task 5 / Task 6 tests
similarly name concrete conditions. No "TBD"/"add error handling"/bare
"write tests" left.

**3. Type consistency**

- `checker.Chunk{Text string; OK bool}` used identically in Task 4 and
  Task 9 (`diff: [{ text, ok }]`) and `dto.go`.
- `srs.Grade` int with `Again=0..Easy=3` — matches API body
  `{"grade":2}` (Task 7) and `store.GradeCard(..., srs.Grade(...))`
  (Task 6/7) and frontend `grade(g: number)` (Task 10).
- `store.CardRow` embeds `srs.Card`; `reviewQueue` hydrates to
  `reviewCardDTO` with `card_id` — frontend `ReviewCard.card_id` used
  in Task 10 test. Consistent.
- `api.check` return type widened to `CheckResult | CheckResult[]` in
  Task 9 Step 6 — Task 8 defines it as `Promise<CheckResult>`; Task 9
  Step 6 explicitly updates the signature. Flagged inline, consistent.
- Lesson status strings `not_started` / `in_progress` / `done` — used
  in Task 6 (`SetLessonStatus`), Task 7 (`getCourse` default), Task 9
  (badges), Task 12 (`recent_lessons`). Consistent.
- The Task 9 note about `checkExercise` also setting `in_progress` is
  duplicated at the end of Task 9 so a Task 7 executor reading ahead
  catches it; if Task 7 is already done, Task 9 executor adds it.

Plan is internally consistent.
