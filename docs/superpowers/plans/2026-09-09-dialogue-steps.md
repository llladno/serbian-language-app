# Диалоговые шаги — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить в уроки шаг `dialogue` — линейный разговор, где собеседник
говорит, а ученик отвечает выбором фразы, набором фразы или вставкой слова,
с переводом слова и реплики и озвучкой двумя голосами.

**Architecture:** Упражнения из реплик кладутся и в `Step.Exercises`, поэтому
проверка ответов, попытки, прогресс шагов и SRS работают без правок. Реплики
ходов `me` не уходят клиенту, пока по ним нет попытки. Фронт — лента пузырей,
внутри активного пузыря переиспользуются существующие `ChoiceAnswer`
и `TextAnswer`.

**Tech Stack:** Go 1.25 (net/http, yaml.v3), Vue 3 + TS + Vitest, edge-tts.

**Spec:** [docs/superpowers/specs/2026-09-09-dialogue-steps-design.md](../specs/2026-09-09-dialogue-steps-design.md)

## Global Constraints

- Модуль `github.com/grisha/serbian-app`, Go 1.25. Тесты: `make test`
  (это `go test ./server/...` + `cd web && npm run test`).
- Типы заданий, допустимые в диалоге: `choice`, `translate`, `fill_blank`.
- Имена аудио реплик: `<id шага>-t<N>.mp3`, нумерация ходов с 1 (`05.9-t1.mp3`).
- Голоса: `f` → `sr-RS-SophieNeural`, `m` → `sr-RS-NicholasNeural`.
  Собеседник — `voice` шага (по умолчанию `f`), ходы `me` — противоположный.
- Весь сербский контент авторится латиницей; в кириллицу переводит
  `sr_lat_to_cyr()` внутри `scripts/tts.py`.
- Клиенту не отдаются: `accept`, `answer`, `pairs`, `say` и `sr`/`ru`/`audio`
  ходов `me` без попытки.
- Гвардия лексики: сербские токены только из накопительного словаря урока,
  его `teaches`, `also_ok` шага и `content/allow-words.yaml`.
- Коммиты заканчивать строкой `Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>`.
- После `make build` восстанавливать `git checkout server/web/dist/.gitkeep`.

---

## Task 1: Модель и парсер диалога

**Files:**
- Modify: `server/internal/content/types.go`
- Modify: `server/internal/content/manifest.go`
- Test: `server/internal/content/manifest_test.go`

**Interfaces:**
- Consumes: `decodeExercise(dir, rel string, e exerciseYAML) (Exercise, error)`,
  `checker.Normalize(string) string`.
- Produces: тип `Turn{Who, SR, RU, Audio string; Exercise *Exercise}`;
  поля `Step.Scene`, `Step.Voice`, `Step.Turns []Turn`;
  функция `decodeDialogue(dir, rel string, s stepYAML, step *Step) error`.

- [ ] **Step 1: Написать падающий тест**

В `server/internal/content/manifest_test.go` (рядом с существующими тестами
манифеста; смотри, как они создают временный контент-каталог, и повтори этот
приём):

```go
func TestParseDialogueStep(t *testing.T) {
	dir := t.TempDir()
	writeDialogueFixture(t, dir)

	l, err := parseManifest(dir, "lessons/05.yaml")
	if err != nil {
		t.Fatal(err)
	}
	s := l.Steps[0]
	if s.Kind != "dialogue" || s.Scene == "" || s.Voice != "f" {
		t.Fatalf("bad step: kind=%q scene=%q voice=%q", s.Kind, s.Scene, s.Voice)
	}
	if len(s.Turns) != 3 {
		t.Fatalf("want 3 turns, got %d", len(s.Turns))
	}
	if s.Turns[0].Who != "npc" || s.Turns[0].Exercise != nil {
		t.Error("turn 1 should be a plain npc line")
	}
	if s.Turns[1].Exercise == nil || s.Turns[1].Exercise.ID != "05.9.1" {
		t.Fatal("turn 2 should carry exercise 05.9.1")
	}
	// упражнения диалога видны и как обычные упражнения шага
	if len(s.Exercises) != 1 || s.Exercises[0].ID != "05.9.1" {
		t.Fatalf("want exercise 05.9.1 in Step.Exercises, got %+v", s.Exercises)
	}
	if s.Turns[1].Exercise != &s.Exercises[0] {
		t.Error("Turn.Exercise must point into Step.Exercises")
	}
}
```

Фикстура — рядом, в том же файле:

```go
func writeDialogueFixture(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "lessons"), 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `lesson: "05"
title: "Куповина"
steps:
  - id: "05.9"
    kind: dialogue
    title: "У пекари"
    scene: "Ты зашёл в пекару."
    turns:
      - who: npc
        sr: "Izvolite?"
        ru: "Слушаю вас?"
      - who: me
        sr: "Jedan hleb, molim."
        ru: "Один хлеб, пожалуйста."
        exercise:
          id: "05.9.1"
          type: choice
          prompt: "Попроси хлеб"
          options: ["Jedan hleb, molim.", "Jedan hleb, hvala."]
          answer: "Jedan hleb, molim."
      - who: npc
        sr: "Hvala."
        ru: "Спасибо."
`
	if err := os.WriteFile(filepath.Join(dir, "lessons", "05.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./server/internal/content/ -run TestParseDialogueStep -v`
Expected: FAIL — `unknown kind "dialogue"`.

- [ ] **Step 3: Добавить типы**

В `server/internal/content/types.go`, рядом с `Step`:

```go
// Turn is one line of a dialogue step. A "me" turn carries the exercise that
// produces the line; SR is the canonical line shown in the chat afterwards.
type Turn struct {
	Who      string // npc | me
	SR       string
	RU       string
	Audio    string // filename under content/audio/, "" when the clip is absent
	Exercise *Exercise
}
```

В структуру `Step` дописать:

```go
	Scene string // dialogue: Russian setup line ("Ты зашёл в пекару")
	Voice string // dialogue: f | m — the other speaker's voice
	Turns []Turn // dialogue: the ordered conversation
```

- [ ] **Step 4: Вынести структуру шага в именованный тип**

В `server/internal/content/manifest.go` анонимная структура внутри
`manifestFile.Steps` мешает передавать шаг в функцию. Вынести как есть,
добавив три поля:

```go
type stepYAML struct {
	ID        string         `yaml:"id"`
	Kind      string         `yaml:"kind"`
	Title     string         `yaml:"title"`
	MD        string         `yaml:"md"`
	AlsoOK    []string       `yaml:"also_ok"`
	Mixed     bool           `yaml:"mixed"`
	Exercises []exerciseYAML `yaml:"exercises"`
	Scene     string         `yaml:"scene"`
	Voice     string         `yaml:"voice"`
	Turns     []turnYAML     `yaml:"turns"`
}

type turnYAML struct {
	Who      string        `yaml:"who"`
	SR       string        `yaml:"sr"`
	RU       string        `yaml:"ru"`
	Exercise *exerciseYAML `yaml:"exercise"`
}
```

и заменить поле на `Steps []stepYAML \`yaml:"steps"\``.

- [ ] **Step 5: Реализовать разбор диалога**

В `manifest.go` добавить `"dialogue": true` в `stepKinds` и функцию:

```go
// dialogueTypes are the exercise types a dialogue turn may use.
var dialogueTypes = map[string]bool{"choice": true, "translate": true, "fill_blank": true}

// decodeDialogue fills step.Turns and step.Exercises from a dialogue step.
func decodeDialogue(dir, rel string, s stepYAML, step *Step) error {
	if s.Scene == "" {
		return fmt.Errorf("%s: step %s: dialogue needs a scene", rel, s.ID)
	}
	switch s.Voice {
	case "", "f":
		step.Voice = "f"
	case "m":
		step.Voice = "m"
	default:
		return fmt.Errorf("%s: step %s: voice must be f or m, got %q", rel, s.ID, s.Voice)
	}
	if len(s.Turns) == 0 {
		return fmt.Errorf("%s: step %s: dialogue needs turns", rel, s.ID)
	}
	step.Scene = s.Scene

	me := 0
	for i, t := range s.Turns {
		n := i + 1
		if t.SR == "" {
			return fmt.Errorf("%s: step %s: turn %d needs sr", rel, s.ID, n)
		}
		turn := Turn{Who: t.Who, SR: t.SR, RU: t.RU}
		switch t.Who {
		case "npc":
			if t.Exercise != nil {
				return fmt.Errorf("%s: step %s: turn %d: npc turn cannot have an exercise", rel, s.ID, n)
			}
		case "me":
			if t.Exercise == nil {
				return fmt.Errorf("%s: step %s: turn %d: me turn needs an exercise", rel, s.ID, n)
			}
			if !dialogueTypes[t.Exercise.Type] {
				return fmt.Errorf("%s: step %s: turn %d: type %q is not allowed in a dialogue (choice, translate, fill_blank)", rel, s.ID, n, t.Exercise.Type)
			}
			ex, err := decodeExercise(dir, rel, *t.Exercise)
			if err != nil {
				return err
			}
			if err := checkTurnLine(rel, s.ID, n, t.SR, ex); err != nil {
				return err
			}
			step.Exercises = append(step.Exercises, ex)
			me++
		default:
			return fmt.Errorf("%s: step %s: turn %d: who must be npc or me, got %q", rel, s.ID, n, t.Who)
		}
		step.Turns = append(step.Turns, turn)
	}
	if me == 0 {
		return fmt.Errorf("%s: step %s: dialogue needs at least one me turn", rel, s.ID)
	}

	// Pointers are bound only now: append above may reallocate step.Exercises,
	// and a pointer taken mid-loop would dangle into the old array.
	k := 0
	for i := range step.Turns {
		if step.Turns[i].Who == "me" {
			step.Turns[i].Exercise = &step.Exercises[k]
			k++
		}
	}
	return nil
}

// checkTurnLine keeps the canonical line in sync with what the checker accepts.
func checkTurnLine(rel, stepID string, n int, sr string, ex Exercise) error {
	switch ex.Type {
	case "choice":
		if checker.Normalize(sr) != checker.Normalize(ex.Answer) {
			return fmt.Errorf("%s: step %s: turn %d: sr %q must equal the choice answer %q", rel, stepID, n, sr, ex.Answer)
		}
	case "translate":
		for _, a := range ex.Accept {
			if checker.Normalize(a) == checker.Normalize(sr) {
				return nil
			}
		}
		return fmt.Errorf("%s: step %s: turn %d: sr %q is not among accept", rel, stepID, n, sr)
	case "fill_blank":
		have := map[string]bool{}
		for _, tok := range strings.Fields(checker.Normalize(sr)) {
			have[tok] = true
		}
		for _, a := range ex.Accept {
			for _, tok := range strings.Fields(checker.Normalize(a)) {
				if !have[tok] {
					return fmt.Errorf("%s: step %s: turn %d: accept %q is not part of sr %q", rel, stepID, n, a, sr)
				}
			}
		}
	}
	return nil
}
```

Добавить импорт `"github.com/grisha/serbian-app/server/internal/checker"` —
пакет `content` уже импортирует его в `lexicon.go`, цикла нет.

- [ ] **Step 6: Подключить разбор в основной цикл**

В `parseManifest`, внутри `switch s.Kind`, добавить ветку до общих проверок:

```go
		case "dialogue":
			if err := decodeDialogue(dir, rel, s, &step); err != nil {
				return nil, err
			}
```

и пропустить для диалога проверку порядка сложности: заменить безусловный
вызов на

```go
		if s.Kind != "dialogue" {
			if err := checkDifficultyOrder(rel, step); err != nil {
				return nil, err
			}
		}
```

- [ ] **Step 7: Убедиться, что тест проходит**

Run: `go test ./server/internal/content/ -run TestParseDialogueStep -v`
Expected: PASS

- [ ] **Step 8: Написать тесты валидации**

```go
func TestDialogueValidation(t *testing.T) {
	cases := []struct {
		name, turns, want string
	}{
		{"npc with exercise", `
      - who: npc
        sr: "Izvolite?"
        exercise:
          id: "05.9.1"
          type: choice
          prompt: "x"
          options: ["a", "b"]
          answer: "a"`, "cannot have an exercise"},
		{"me without exercise", `
      - who: me
        sr: "Hvala."`, "needs an exercise"},
		{"unknown who", `
      - who: waiter
        sr: "Izvolite?"`, "who must be npc or me"},
		{"forbidden type", `
      - who: me
        sr: "Hvala."
        exercise:
          id: "05.9.1"
          type: free
          prompt: "x"`, "not allowed in a dialogue"},
		{"line differs from answer", `
      - who: me
        sr: "Dobar dan."
        exercise:
          id: "05.9.1"
          type: choice
          prompt: "x"
          options: ["Hvala.", "Molim."]
          answer: "Hvala."`, "must equal the choice answer"},
		{"no me turn", `
      - who: npc
        sr: "Izvolite?"`, "at least one me turn"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeDialogueYAML(t, dir, tc.turns)
			_, err := parseManifest(dir, "lessons/05.yaml")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func writeDialogueYAML(t *testing.T, dir, turns string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "lessons"), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := `lesson: "05"
title: "Куповина"
steps:
  - id: "05.9"
    kind: dialogue
    title: "У пекари"
    scene: "Ты зашёл в пекару."
    turns:` + turns + "\n"
	if err := os.WriteFile(filepath.Join(dir, "lessons", "05.yaml"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 9: Прогнать тесты пакета**

Run: `go test ./server/internal/content/ -v`
Expected: PASS, включая существующие тесты манифеста.

- [ ] **Step 10: Коммит**

```bash
git add server/internal/content/types.go server/internal/content/manifest.go server/internal/content/manifest_test.go
git commit -m "feat(content): шаг dialogue — модель и парсер

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 2: Аудио реплик в загрузчике

**Files:**
- Modify: `server/internal/content/manifest.go`
- Test: `server/internal/content/manifest_test.go`

**Interfaces:**
- Consumes: `decodeDialogue` из Task 1.
- Produces: заполненный `Turn.Audio` вида `05.9-t1.mp3`.

- [ ] **Step 1: Написать падающий тест**

```go
func TestDialogueTurnAudio(t *testing.T) {
	dir := t.TempDir()
	writeDialogueFixture(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, "audio"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "audio", "05.9-t1.mp3"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	l, err := parseManifest(dir, "lessons/05.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got := l.Steps[0].Turns[0].Audio; got != "05.9-t1.mp3" {
		t.Errorf("turn 1 audio = %q, want 05.9-t1.mp3", got)
	}
	if got := l.Steps[0].Turns[1].Audio; got != "" {
		t.Errorf("turn 2 has no clip on disk, got audio %q", got)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./server/internal/content/ -run TestDialogueTurnAudio -v`
Expected: FAIL — `turn 1 audio = "", want 05.9-t1.mp3`.

- [ ] **Step 3: Реализовать**

В `decodeDialogue`, сразу после создания `turn` в цикле:

```go
		clip := fmt.Sprintf("%s-t%d.mp3", s.ID, n)
		if _, err := os.Stat(filepath.Join(dir, "audio", clip)); err == nil {
			turn.Audio = clip
		}
```

- [ ] **Step 4: Проверить**

Run: `go test ./server/internal/content/ -v`
Expected: PASS

- [ ] **Step 5: Коммит**

```bash
git add server/internal/content/manifest.go server/internal/content/manifest_test.go
git commit -m "feat(content): аудио реплик диалога в загрузчике

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 3: Гвардия лексики на репликах

**Files:**
- Modify: `server/internal/content/lexicon.go`
- Test: `server/internal/content/lexicon_test.go`

**Interfaces:**
- Consumes: `unknownTokens(text string, known map[string]bool, alsoOK []string) []string`,
  `strictStrings(e Exercise) []string`.
- Produces: `turnStrings(s Step) []string` — сербские строки реплик шага.

- [ ] **Step 1: Написать падающий тест**

```go
func TestTurnStrings(t *testing.T) {
	s := Step{Kind: "dialogue", Turns: []Turn{
		{Who: "npc", SR: "Izvolite?", RU: "Слушаю вас?"},
		{Who: "me", SR: "Jedan hleb, molim.", RU: "Один хлеб, пожалуйста."},
	}}
	got := turnStrings(s)
	if len(got) != 2 || got[0] != "Izvolite?" || got[1] != "Jedan hleb, molim." {
		t.Fatalf("turnStrings = %v", got)
	}
	for _, g := range got {
		if isCyrillic(g) {
			t.Errorf("russian text leaked into the guard: %q", g)
		}
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./server/internal/content/ -run TestTurnStrings -v`
Expected: FAIL — `undefined: turnStrings`.

- [ ] **Step 3: Реализовать**

В конец `server/internal/content/lexicon.go`:

```go
// turnStrings collects the Serbian lines of a dialogue step. The Russian
// fields (RU, Scene) and the exercise prompts are not checked — they are
// deliberately Russian.
func turnStrings(s Step) []string {
	out := make([]string, 0, len(s.Turns))
	for _, t := range s.Turns {
		if t.SR != "" {
			out = append(out, t.SR)
		}
	}
	return out
}
```

- [ ] **Step 4: Подключить в общий обход**

В `lexicon_test.go`, в `TestLexiconGuardRealContent`, внутри цикла по шагам,
перед циклом по упражнениям:

```go
				for _, txt := range turnStrings(s) {
					for _, u := range unknownTokens(txt, known, s.AlsoOK) {
						msg := "lesson %s step %s: unknown word %q in a dialogue line " +
							"(add it to the step's also_ok or the lesson's teaches)"
						if l.Manifest {
							t.Errorf(msg, id, s.ID, u)
						} else {
							t.Logf("[legacy] "+msg, id, s.ID, u)
						}
					}
				}
```

- [ ] **Step 5: Проверить**

Run: `go test ./server/internal/content/ -v`
Expected: PASS (реального диалога в контенте ещё нет, обход просто пустой).

- [ ] **Step 6: Коммит**

```bash
git add server/internal/content/lexicon.go server/internal/content/lexicon_test.go
git commit -m "feat(content): гвардия лексики проверяет реплики диалога

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 4: DTO и раскрытие реплик по попыткам

**Files:**
- Modify: `server/internal/api/dto.go`
- Modify: `server/internal/api/api.go:178-217` (`getLesson`)
- Test: `server/internal/api/api_test.go`

**Interfaces:**
- Consumes: `us.LessonAttempts(lesson string) (map[string]store.AttemptSummary, error)`.
- Produces: `turnDTO`; поля `stepDTO.Scene`, `stepDTO.Voice`, `stepDTO.Turns`.

- [ ] **Step 1: Написать падающий тест**

В `server/internal/api/api_test.go` (повтори приём существующих тестов —
как они поднимают handler с тестовым контентом и пользователем):

```go
func TestDialogueHidesUnansweredLines(t *testing.T) {
	h, cleanup := newTestHandlerWithDialogue(t)
	defer cleanup()

	var before lessonDTO
	getJSON(t, h, "GET", "/api/lessons/05", nil, &before)
	turns := before.Steps[0].Turns
	if turns[0].SR == "" {
		t.Error("npc line must always be visible")
	}
	if turns[1].SR != "" || turns[1].RU != "" {
		t.Errorf("unanswered me line leaked: sr=%q ru=%q", turns[1].SR, turns[1].RU)
	}
	if turns[1].ExerciseID != "05.9.1" {
		t.Errorf("me turn must expose its exercise id, got %q", turns[1].ExerciseID)
	}

	postJSON(t, h, "/api/lessons/05/exercises/05.9.1/check",
		map[string]string{"answer": "Jedan hleb, molim."})

	var after lessonDTO
	getJSON(t, h, "GET", "/api/lessons/05", nil, &after)
	if got := after.Steps[0].Turns[1].SR; got != "Jedan hleb, molim." {
		t.Errorf("answered me line should be revealed, got %q", got)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./server/internal/api/ -run TestDialogueHidesUnansweredLines -v`
Expected: FAIL — у `stepDTO` нет поля `Turns`.

- [ ] **Step 3: Добавить DTO**

В `server/internal/api/dto.go`:

```go
// turnDTO is one dialogue line. For a "me" turn the line itself (sr/ru/audio)
// is sent only once the user has an attempt on its exercise — otherwise the
// correct answer would be readable in the page source.
type turnDTO struct {
	Who        string `json:"who"`
	SR         string `json:"sr,omitempty"`
	RU         string `json:"ru,omitempty"`
	Audio      string `json:"audio,omitempty"`
	ExerciseID string `json:"exercise_id,omitempty"`
}
```

В `stepDTO` дописать:

```go
	Scene string    `json:"scene,omitempty"`
	Voice string    `json:"voice,omitempty"`
	Turns []turnDTO `json:"turns,omitempty"`
```

- [ ] **Step 4: Реализовать раскрытие в `getLesson`**

После получения `stepSt`:

```go
	attempts, err := us.LessonAttempts(id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
```

В цикле по шагам, после заполнения `ExerciseIDs`:

```go
		d.Scene, d.Voice = s.Scene, s.Voice
		for _, t := range s.Turns {
			td := turnDTO{Who: t.Who}
			answered := t.Who == "npc"
			if t.Exercise != nil {
				td.ExerciseID = t.Exercise.ID
				_, answered = attempts[t.Exercise.ID]
			}
			if answered {
				td.SR, td.RU, td.Audio = t.SR, t.RU, t.Audio
			}
			d.Turns = append(d.Turns, td)
		}
```

- [ ] **Step 5: Проверить**

Run: `go test ./server/internal/api/ -v`
Expected: PASS

- [ ] **Step 6: Коммит**

```bash
git add server/internal/api/dto.go server/internal/api/api.go server/internal/api/api_test.go
git commit -m "feat(api): реплики диалога в DTO урока, скрытые до ответа

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 5: Каноничная реплика в ответе на проверку

**Files:**
- Modify: `server/internal/api/dto.go`
- Modify: `server/internal/api/api.go:256-345` (`findExercise`, `checkExercise`)
- Test: `server/internal/api/api_test.go`

**Interfaces:**
- Produces: `findTurn(lesson, exID string) (content.Turn, bool)`;
  поля `checkResultDTO.Line`, `.LineRU`, `.LineAudio`.

- [ ] **Step 1: Написать падающий тест**

```go
func TestCheckReturnsDialogueLine(t *testing.T) {
	h, cleanup := newTestHandlerWithDialogue(t)
	defer cleanup()

	var res checkResultDTO
	postJSONInto(t, h, "/api/lessons/05/exercises/05.9.1/check",
		map[string]string{"answer": "Jedan hleb, molim."}, &res)

	if !res.OK {
		t.Fatal("answer should be accepted")
	}
	if res.Line != "Jedan hleb, molim." || res.LineRU != "Один хлеб, пожалуйста." {
		t.Errorf("line=%q line_ru=%q", res.Line, res.LineRU)
	}
}

func TestCheckReturnsLineAfterWrongAnswer(t *testing.T) {
	h, cleanup := newTestHandlerWithDialogue(t)
	defer cleanup()

	var res checkResultDTO
	postJSONInto(t, h, "/api/lessons/05/exercises/05.9.1/check",
		map[string]string{"answer": "Jedan hleb, hvala."}, &res)

	if res.OK {
		t.Fatal("answer should be rejected")
	}
	// лента показывает правильную реплику даже после ошибки — иначе разговор рвётся
	if res.Line != "Jedan hleb, molim." {
		t.Errorf("line после ошибки = %q", res.Line)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./server/internal/api/ -run TestCheckReturns -v`
Expected: FAIL — у `checkResultDTO` нет поля `Line`.

- [ ] **Step 3: Добавить поля DTO**

В `checkResultDTO`:

```go
	Line      string `json:"line,omitempty"`       // dialogue: canonical line for the chat
	LineRU    string `json:"line_ru,omitempty"`
	LineAudio string `json:"line_audio,omitempty"`
```

- [ ] **Step 4: Реализовать поиск реплики**

Рядом с `findExercise` в `api.go`:

```go
// findTurn locates the dialogue turn an exercise belongs to, if any.
func (h handlers) findTurn(lesson, exID string) (content.Turn, bool) {
	l := h.Course().Lessons[lesson]
	if l == nil {
		return content.Turn{}, false
	}
	for _, s := range l.Steps {
		for _, t := range s.Turns {
			if t.Exercise != nil && t.Exercise.ID == exID {
				return t, true
			}
		}
	}
	return content.Turn{}, false
}
```

В `checkExercise`, после `resp := checkResultDTO{Explain: ex.Explain}`:

```go
	if t, ok := h.findTurn(lesson, exID); ok {
		resp.Line, resp.LineRU, resp.LineAudio = t.SR, t.RU, t.Audio
	}
```

- [ ] **Step 5: Проверить**

Run: `go test ./server/... `
Expected: PASS

- [ ] **Step 6: Коммит**

```bash
git add server/internal/api/dto.go server/internal/api/api.go server/internal/api/api_test.go
git commit -m "feat(api): проверка ответа возвращает реплику для ленты

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 6: Озвучка реплик в tts.py

**Files:**
- Modify: `scripts/tts.py:77-112` (`listen_entries`), `:135-165` (`run`)

**Interfaces:**
- Produces: `dialogue_entries() -> list[dict]` — строки
  `{"id": "05.9-t1", "cyrillic": ..., "voice": "sr-RS-SophieNeural"}`.

- [ ] **Step 1: Добавить сбор реплик**

После `listen_entries()` в `scripts/tts.py`:

```python
VOICE_F = "sr-RS-SophieNeural"
VOICE_M = "sr-RS-NicholasNeural"


def dialogue_entries() -> list[dict]:
    """Collect {id, cyrillic, voice} rows for every dialogue turn, keyed
    <step id>-t<N> (05.9-t1.mp3). The other speaker uses the step's `voice`
    (f by default), the learner's own lines always take the opposite one, so
    the chat has two distinct voices."""
    out = []
    for path in sorted(glob.glob(LESSONS_GLOB)):
        if os.path.basename(path) == "_TEMPLATE.yaml":
            continue
        doc = yaml.safe_load(open(path, encoding="utf-8")) or {}
        for step in doc.get("steps") or []:
            if step.get("kind") != "dialogue":
                continue
            npc = VOICE_M if step.get("voice") == "m" else VOICE_F
            me = VOICE_F if npc == VOICE_M else VOICE_M
            for i, turn in enumerate(step.get("turns") or [], start=1):
                if not turn.get("sr"):
                    continue
                out.append({
                    "id": f"{step['id']}-t{i}",
                    "cyrillic": sr_lat_to_cyr(turn["sr"]),
                    "voice": npc if turn.get("who") == "npc" else me,
                })
    return out
```

- [ ] **Step 2: Разрешить голос на строку**

В `run()` заменить постановку задачи:

```python
        tasks.append(synth(sem, e.get("voice") or voice, text, dest))
```

и строку отчёта — на нейтральную, раз голоса теперь разные:

```python
    print(f"generating {len(tasks)} file(s)\n")
```

В `main()`, после `entries += listen_entries()`:

```python
    entries += dialogue_entries()
```

- [ ] **Step 3: Проверить сбор без синтеза**

Run: `python3 -c "import sys; sys.path.insert(0,'scripts'); import tts; print(tts.dialogue_entries())"`
Expected: `[]` — диалогов в контенте ещё нет, ошибок нет.

- [ ] **Step 4: Коммит**

```bash
git add scripts/tts.py
git commit -m "feat(tts): озвучка реплик диалога двумя голосами

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 7: Фронт — типы и пузырь реплики

**Files:**
- Modify: `web/src/types.ts`
- Create: `web/src/components/DialogueBubble.vue`
- Test: `web/src/components/DialogueBubble.test.ts`

**Interfaces:**
- Consumes: `GlossedText.vue` (props `{ text: string }`),
  `SpeakButton.vue` (props `{ src?: string; size?: number }`).
- Produces: типы `Turn`, поля `Step.scene/voice/turns`,
  `CheckResult.line/line_ru/line_audio`; компонент `DialogueBubble`
  с props `{ who: 'npc' | 'me'; sr: string; ru?: string; audio?: string; showTranslation?: boolean }`.

- [ ] **Step 1: Расширить типы**

В `web/src/types.ts`:

```ts
export type StepKind = 'teach' | 'practice' | 'reading' | 'checkpoint' | 'dialogue'

export interface Turn {
  who: 'npc' | 'me'
  sr?: string
  ru?: string
  audio?: string
  exercise_id?: string
}
```

В `Step` дописать `scene?: string`, `voice?: 'f' | 'm'`, `turns?: Turn[]`.
В `CheckResult` дописать `line?: string`, `line_ru?: string`, `line_audio?: string`.

- [ ] **Step 2: Написать падающий тест**

`web/src/components/DialogueBubble.test.ts`:

```ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DialogueBubble from './DialogueBubble.vue'

const props = { who: 'npc' as const, sr: 'Izvolite?', ru: 'Слушаю вас?' }

describe('DialogueBubble', () => {
  it('shows the Serbian line and hides the translation until asked', async () => {
    const w = mount(DialogueBubble, { props })
    expect(w.text()).toContain('Izvolite?')
    expect(w.text()).not.toContain('Слушаю вас?')

    await w.find('[data-test="translate"]').trigger('click')
    expect(w.text()).toContain('Слушаю вас?')
  })

  it('shows the translation upfront when showTranslation is set', () => {
    const w = mount(DialogueBubble, { props: { ...props, showTranslation: true } })
    expect(w.text()).toContain('Слушаю вас?')
  })

  it('renders no speaker button without a clip', () => {
    const w = mount(DialogueBubble, { props })
    expect(w.find('button[aria-label="озвучить"]').exists()).toBe(false)
  })
})
```

- [ ] **Step 3: Убедиться, что тест падает**

Run: `cd web && npx vitest run src/components/DialogueBubble.test.ts`
Expected: FAIL — файла компонента нет.

- [ ] **Step 4: Написать компонент**

`web/src/components/DialogueBubble.vue`:

```vue
<script setup lang="ts">
// One line of a dialogue. The Serbian text is word-clickable (GlossedText);
// the whole-line translation lives behind its own button, so it never fights
// with the per-word lookup for the same tap.
import { computed, ref } from 'vue'
import { Languages } from 'lucide-vue-next'
import GlossedText from './GlossedText.vue'
import SpeakButton from './SpeakButton.vue'

const props = defineProps<{
  who: 'npc' | 'me'
  sr: string
  ru?: string
  audio?: string
  showTranslation?: boolean
}>()

const open = ref(false)
const translated = computed(() => props.showTranslation || open.value)
</script>

<template>
  <div class="flex" :class="who === 'me' ? 'justify-end' : 'justify-start'">
    <div
      class="max-w-[85%] rounded-2xl px-3.5 py-2.5"
      :class="
        who === 'me'
          ? 'rounded-br-sm bg-[var(--accent-soft)]'
          : 'rounded-bl-sm bg-[var(--bg-soft)]'
      "
    >
      <div class="flex items-start gap-2">
        <p class="serbian leading-7"><GlossedText :text="sr" /></p>
        <SpeakButton v-if="audio" :src="audio" :size="26" class="mt-0.5" />
        <button
          v-if="ru"
          type="button"
          data-test="translate"
          class="mt-0.5 inline-grid h-[26px] w-[26px] shrink-0 place-items-center rounded-full border border-[var(--border)] text-[var(--muted)] transition hover:border-[var(--accent)] hover:text-[var(--accent)]"
          :class="{ 'border-[var(--accent)] text-[var(--accent)]': translated }"
          :aria-label="translated ? 'скрыть перевод' : 'перевести реплику'"
          @click="open = !open"
        >
          <Languages :size="13" :stroke-width="2.25" />
        </button>
      </div>

      <p v-if="translated && ru" class="mt-1.5 text-sm text-[var(--muted)]">{{ ru }}</p>
    </div>
  </div>
</template>
```

- [ ] **Step 5: Проверить**

Run: `cd web && npx vitest run src/components/DialogueBubble.test.ts`
Expected: PASS

- [ ] **Step 6: Коммит**

```bash
git add web/src/types.ts web/src/components/DialogueBubble.vue web/src/components/DialogueBubble.test.ts
git commit -m "feat(web): пузырь реплики диалога

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 8: Фронт — событие graded с результатом

**Files:**
- Modify: `web/src/components/exercises/ChoiceAnswer.vue:13,29`
- Modify: `web/src/components/exercises/TextAnswer.vue:24,37`
- Test: `web/src/components/exercises/ChoiceAnswer.test.ts`

**Interfaces:**
- Produces: событие `graded: [ok: boolean, result: CheckResult]` у обоих
  компонентов. `ExerciseItem.vue` второй аргумент игнорирует — правки не требует.

- [ ] **Step 1: Написать падающий тест**

Дописать в `web/src/components/exercises/ChoiceAnswer.test.ts`:

```ts
  it('passes the check result along with the verdict', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, line: 'Hvala', line_ru: 'Спасибо' })
    const w = mount(ChoiceAnswer, { props })
    await w.findAll('button').find((b) => b.text() === 'Hvala')!.trigger('click')
    await flushPromises()

    expect(w.emitted('graded')?.[0]).toEqual([true, { ok: true, line: 'Hvala', line_ru: 'Спасибо' }])
  })
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `cd web && npx vitest run src/components/exercises/ChoiceAnswer.test.ts`
Expected: FAIL — событие несёт только `[true]`.

- [ ] **Step 3: Расширить эмиты**

В `ChoiceAnswer.vue`:

```ts
const emit = defineEmits<{ graded: [ok: boolean, result: CheckResult] }>()
```

и в `choose`:

```ts
    result.value = await api.check(props.lesson, props.exerciseId, { answer: opt })
    emit('graded', !!result.value.ok, result.value)
```

В `TextAnswer.vue` — то же самое: тип события и

```ts
    emit('graded', !!result.value.ok, result.value)
```

- [ ] **Step 4: Проверить**

Run: `cd web && npm run test`
Expected: PASS — существующие проверки `toEqual([true])` могут потребовать
обновления до `toEqual([true, ...])`; обнови их, если упадут.

- [ ] **Step 5: Коммит**

```bash
git add web/src/components/exercises/
git commit -m "feat(web): graded отдаёт результат проверки

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 9: Фронт — лента диалога

**Files:**
- Create: `web/src/components/DialogueStep.vue`
- Test: `web/src/components/DialogueStep.test.ts`

**Interfaces:**
- Consumes: `DialogueBubble` (Task 7), `ChoiceAnswer`/`TextAnswer` с событием
  `graded: [ok, result]` (Task 8), типы `Turn`, `Exercise`, `LessonAttempt`.
- Produces: компонент с props
  `{ lesson: string; step: Step; exercises: Exercise[]; priors: Record<string, LessonAttempt> }`
  и событием `graded: [exerciseId: string, ok: boolean]`.

- [ ] **Step 1: Написать падающий тест**

`web/src/components/DialogueStep.test.ts`:

```ts
import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import DialogueStep from './DialogueStep.vue'
import { api } from '../api'
import type { Step, Exercise } from '../types'

afterEach(() => vi.restoreAllMocks())

const step: Step = {
  id: '05.9',
  kind: 'dialogue',
  title: 'У пекари',
  scene: 'Ты зашёл в пекару.',
  status: 'not_started',
  turns: [
    { who: 'npc', sr: 'Izvolite?', ru: 'Слушаю вас?' },
    { who: 'me', exercise_id: '05.9.1' },
    { who: 'npc', sr: 'Hvala.', ru: 'Спасибо.' },
  ],
}

const exercises: Exercise[] = [
  {
    id: '05.9.1',
    type: 'choice',
    prompt: 'Попроси хлеб',
    options: ['Jedan hleb, molim.', 'Jedan hleb, hvala.'],
  },
]

const props = { lesson: '05', step, exercises, priors: {} }

describe('DialogueStep', () => {
  it('shows the scene and only the lines up to the active turn', () => {
    const w = mount(DialogueStep, { props })
    expect(w.text()).toContain('Ты зашёл в пекару.')
    expect(w.text()).toContain('Izvolite?')
    expect(w.text()).toContain('Попроси хлеб')
    expect(w.text()).not.toContain('Hvala.')
  })

  it('turns a correct answer into a bubble and advances the conversation', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({
      ok: true,
      line: 'Jedan hleb, molim.',
      line_ru: 'Один хлеб, пожалуйста.',
    })
    const w = mount(DialogueStep, { props })
    await w.findAll('button').find((b) => b.text() === 'Jedan hleb, molim.')!.trigger('click')
    await flushPromises()

    expect(w.emitted('graded')?.[0]).toEqual(['05.9.1', true])
    expect(w.text()).toContain('Hvala.')
  })

  it('keeps the correct line after a wrong answer so the context holds', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({
      ok: false,
      line: 'Jedan hleb, molim.',
      line_ru: 'Один хлеб, пожалуйста.',
      expected: 'Jedan hleb, molim.',
    })
    const w = mount(DialogueStep, { props })
    await w.findAll('button').find((b) => b.text() === 'Jedan hleb, hvala.')!.trigger('click')
    await flushPromises()

    expect(w.emitted('graded')?.[0]).toEqual(['05.9.1', false])
    expect(w.text()).toContain('Jedan hleb, molim.')
    expect(w.text()).toContain('Hvala.')
  })

  it('shows every translation at once when the toggle is on', async () => {
    const w = mount(DialogueStep, { props })
    await w.find('[data-test="toggle-translations"]').trigger('click')
    expect(w.text()).toContain('Слушаю вас?')
  })

  it('remembers the autoplay toggle between dialogues', async () => {
    localStorage.removeItem('dialogue.autoplay')
    const w = mount(DialogueStep, { props })
    await w.find('[data-test="toggle-autoplay"]').trigger('click')
    expect(localStorage.getItem('dialogue.autoplay')).toBe('1')

    const again = mount(DialogueStep, { props })
    expect(again.find('[data-test="toggle-autoplay"]').classes().join(' ')).toContain('accent')
  })
})
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `cd web && npx vitest run src/components/DialogueStep.test.ts`
Expected: FAIL — файла компонента нет.

- [ ] **Step 3: Написать компонент**

`web/src/components/DialogueStep.vue`:

```vue
<script setup lang="ts">
// A dialogue step: the conversation grows downwards, one turn at a time. A
// "me" turn shows its exercise; once answered — right or wrong — the canonical
// line takes its place as a bubble, so the thread of the conversation never
// breaks.
import { computed, reactive, ref } from 'vue'
import type { CheckResult, Exercise, LessonAttempt, Step } from '../types'
import DialogueBubble from './DialogueBubble.vue'
import ChoiceAnswer from './exercises/ChoiceAnswer.vue'
import TextAnswer from './exercises/TextAnswer.vue'

const props = defineProps<{
  lesson: string
  step: Step
  exercises: Exercise[]
  priors: Record<string, LessonAttempt>
}>()
const emit = defineEmits<{ graded: [exerciseId: string, ok: boolean] }>()

const showTranslations = ref(localStorage.getItem('dialogue.translations') === '1')
function toggleTranslations() {
  showTranslations.value = !showTranslations.value
  localStorage.setItem('dialogue.translations', showTranslations.value ? '1' : '0')
}

const autoplay = ref(localStorage.getItem('dialogue.autoplay') === '1')
function toggleAutoplay() {
  autoplay.value = !autoplay.value
  localStorage.setItem('dialogue.autoplay', autoplay.value ? '1' : '0')
}

// Autoplay is opt-in: mobile browsers mute audio without a user gesture, and
// an unexpected voice in a quiet room is worse than a missing one.
function played(clip?: string) {
  if (!autoplay.value || !clip) return
  new Audio(`/audio/${clip}`).play().catch(() => {})
}

const turns = computed(() => props.step.turns ?? [])
const byID = computed(() => Object.fromEntries(props.exercises.map((e) => [e.id, e])))

// Lines learned during this session, keyed by exercise id: the server sends
// them with the check result. Turns answered in an earlier session already
// carry their sr/ru from the lesson payload.
const answered = reactive<Record<string, CheckResult>>({})

function lineOf(exerciseId: string) {
  return answered[exerciseId]
}

function isDone(exerciseId: string) {
  return !!answered[exerciseId] || exerciseId in props.priors
}

// The conversation is revealed up to and including the first unanswered turn.
const visible = computed(() => {
  const out: { index: number; active: boolean }[] = []
  for (let i = 0; i < turns.value.length; i++) {
    const t = turns.value[i]
    const pending = t.who === 'me' && t.exercise_id && !isDone(t.exercise_id)
    out.push({ index: i, active: !!pending })
    if (pending) break
  }
  return out
})

function onGraded(exerciseId: string, ok: boolean, result: CheckResult) {
  answered[exerciseId] = result
  emit('graded', exerciseId, ok)
}
</script>

<template>
  <section class="space-y-4">
    <header class="flex items-start justify-between gap-3">
      <p class="rounded-xl bg-[var(--bg-soft)] px-4 py-2.5 text-sm text-[var(--muted)]">
        {{ step.scene }}
      </p>
      <button
        type="button"
        data-test="toggle-translations"
        class="shrink-0 rounded-full border border-[var(--border)] px-3 py-1.5 text-xs text-[var(--muted)] transition hover:border-[var(--accent)] hover:text-[var(--accent)]"
        :class="{ 'border-[var(--accent)] text-[var(--accent)]': showTranslations }"
        @click="toggleTranslations"
      >
        переводы
      </button>
      <button
        type="button"
        data-test="toggle-autoplay"
        class="shrink-0 rounded-full border border-[var(--border)] px-3 py-1.5 text-xs text-[var(--muted)] transition hover:border-[var(--accent)] hover:text-[var(--accent)]"
        :class="{ 'border-[var(--accent)] text-[var(--accent)]': autoplay }"
        @click="toggleAutoplay"
      >
        звук
      </button>
    </header>

    <div class="space-y-3">
      <template v-for="v in visible" :key="v.index">
        <DialogueBubble
          v-if="turns[v.index].who === 'npc'"
          who="npc"
          :sr="turns[v.index].sr ?? ''"
          :ru="turns[v.index].ru"
          :audio="turns[v.index].audio"
          :show-translation="showTranslations"
          @vue:mounted="played(turns[v.index].audio)"
        />

        <template v-else>
          <DialogueBubble
            v-if="!v.active"
            who="me"
            :sr="lineOf(turns[v.index].exercise_id!)?.line ?? turns[v.index].sr ?? ''"
            :ru="lineOf(turns[v.index].exercise_id!)?.line_ru ?? turns[v.index].ru"
            :audio="lineOf(turns[v.index].exercise_id!)?.line_audio ?? turns[v.index].audio"
            :show-translation="showTranslations"
          />

          <ChoiceAnswer
            v-else-if="byID[turns[v.index].exercise_id!]?.type === 'choice'"
            :lesson="lesson"
            :exercise-id="turns[v.index].exercise_id!"
            :prompt="byID[turns[v.index].exercise_id!].prompt"
            :options="byID[turns[v.index].exercise_id!].options ?? []"
            @graded="(ok, res) => onGraded(turns[v.index].exercise_id!, ok, res)"
          />

          <TextAnswer
            v-else-if="byID[turns[v.index].exercise_id!]"
            :lesson="lesson"
            :exercise-id="turns[v.index].exercise_id!"
            :type="byID[turns[v.index].exercise_id!].type as 'translate' | 'fill_blank'"
            :prompt="byID[turns[v.index].exercise_id!].prompt"
            @graded="(ok, res) => onGraded(turns[v.index].exercise_id!, ok, res)"
          />
        </template>
      </template>
    </div>
  </section>
</template>
```

- [ ] **Step 4: Проверить**

Run: `cd web && npx vitest run src/components/DialogueStep.test.ts`
Expected: PASS

- [ ] **Step 5: Коммит**

```bash
git add web/src/components/DialogueStep.vue web/src/components/DialogueStep.test.ts
git commit -m "feat(web): лента диалога

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 10: Встроить диалог в урок

**Files:**
- Modify: `web/src/views/LessonView.vue:44-52` (`canAdvance`), `:198-220` (шаблон)
- Test: `web/src/views/LessonView.test.ts`

**Interfaces:**
- Consumes: `DialogueStep` (Task 9), существующий `onGraded(exerciseId, ok)`
  в `LessonView`.

- [ ] **Step 1: Написать падающий тест**

Дописать в `web/src/views/LessonView.test.ts` тест в стиле уже имеющихся:
урок с одним шагом `kind: 'dialogue'` рендерится через `DialogueStep`,
а кнопка «дальше» заблокирована, пока не отвечен ход:

```ts
  it('renders a dialogue step and gates advancing on its turns', async () => {
    // замокай api.lesson/api.exercises так же, как в соседних тестах файла,
    // вернув урок с одним шагом kind: 'dialogue', turns и exercise_ids: ['05.9.1']
    const w = await mountLessonWithDialogue()
    expect(w.text()).toContain('Ты зашёл в пекару.')
    expect(w.find('[data-test="next-step"]').attributes('disabled')).toBeDefined()
  })
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `cd web && npx vitest run src/views/LessonView.test.ts`
Expected: FAIL — сцена не отрисована.

- [ ] **Step 3: Подключить компонент**

Импорт в `<script setup>`:

```ts
import DialogueStep from '../components/DialogueStep.vue'
```

В шаблоне, до ветки `v-else-if="curBlock"`:

```vue
          <DialogueStep
            v-else-if="cur.kind === 'dialogue'"
            :lesson="lesson.id"
            :step="cur"
            :exercises="curBlock?.exercises ?? []"
            :priors="priors"
            @graded="onGraded"
          />
```

`DialogueStep` эмитит `graded: [exerciseId, ok]` — ровно ту же пару, что
`ExerciseBlock`, а `onGraded(exId: string, ok: boolean)` в этом файле её и
ждёт, поэтому обработчик передаётся по имени, без обёртки.

`canAdvance` править не требуется: диалог отдаёт `exercise_ids`, и общее
правило «все упражнения шага отвечены» работает как есть. Убедись в этом,
прочитав `canAdvance`, и не трогай его, если правило совпадает.

- [ ] **Step 4: Проверить**

Run: `cd web && npm run test`
Expected: PASS

- [ ] **Step 5: Коммит**

```bash
git add web/src/views/LessonView.vue web/src/views/LessonView.test.ts
git commit -m "feat(web): шаг диалога в уроке

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 11: Сцена «У пекари» — урок 05

Завершает вертикальный срез: первая живая сцена от YAML до озвученной ленты.

**Files:**
- Modify: `content/vocab.yaml`
- Modify: `content/lessons/05.yaml`
- Modify: `content/lessons/_TEMPLATE.yaml`
- Create: `content/audio/05.*-t*.mp3` (генерируются)

- [ ] **Step 1: Добавить слова урока**

В `content/vocab.yaml` добавить записи в стиле соседних (`id`, `latin`,
`cyrillic`, `ru`, `lesson: "05"`, `pos`, при необходимости `note`):
`cena` (цена), `dajte` (дайте), `još` (ещё). Их id дописать в `teaches`
урока 05 в `content/lessons/05.yaml`.

- [ ] **Step 2: Написать шаг диалога**

В конец `steps` в `content/lessons/05.yaml` добавить шаг `kind: dialogue`
с `id: "05.<следующий номер по порядку>"`. Каркас — дописать до 8 ходов,
из них 3–4 хода `me`, сюжет: зашёл, попросил, спросил цену, расплатился:

```yaml
  - id: "05.10"
    kind: dialogue
    title: "У пекари"
    scene: "Ты зашёл в пекару. За прилавком — продавщица."
    voice: f
    turns:
      - who: npc
        sr: "Dobar dan, izvolite?"
        ru: "Добрый день, слушаю вас?"

      - who: me
        sr: "Dobar dan. Jedan hleb, molim."
        ru: "Добрый день. Один хлеб, пожалуйста."
        exercise:
          id: "05.10.1"
          type: choice
          prompt: "Поздоровайся и попроси один хлеб"
          options:
            - "Dobar dan. Jedan hleb, molim."
            - "Dobar dan. Jedan hleb, hvala."
            - "Laku noć. Jedan hleb, molim."
          answer: "Dobar dan. Jedan hleb, molim."
          explain: "«molim» — просьба; «hvala» говорят уже после."

      - who: npc
        sr: "Izvolite. Još nešto?"
        ru: "Пожалуйста. Что-нибудь ещё?"

      - who: me
        sr: "Koliko košta?"
        ru: "Сколько стоит?"
        exercise:
          id: "05.10.2"
          type: translate
          prompt: "Спроси, сколько это стоит"
          accept: ["Koliko košta?", "Koliko to košta?"]
```

Правила, которые проверит парсер и гвардия: только лексика, накопленная
к уроку 05; типы заданий `choice`, `translate`, `fill_blank`; поле `sr`
хода `me` совпадает с `answer` (для `choice`) или входит в `accept`
(для `translate`).

- [ ] **Step 3: Прогнать гвардию и парсер**

Run: `go test ./server/internal/content/ -v`
Expected: PASS. При ошибке «unknown word» — либо добавить слово в `teaches`
урока, либо в `also_ok` шага, если оно декоративное.

- [ ] **Step 4: Сгенерировать озвучку**

Run: `python3 scripts/tts.py`
Expected: создаются файлы `content/audio/05.<N>-t1.mp3` … по числу реплик.

- [ ] **Step 5: Дополнить шаблон урока**

В `content/lessons/_TEMPLATE.yaml` добавить закомментированный пример шага
`dialogue` со всеми полями (`scene`, `voice`, `turns`, ход `npc`, ход `me`
с `sr`/`ru`/`exercise`), в стиле соседних блоков шаблона.

- [ ] **Step 6: Проверить целиком**

Run: `make test`
Expected: PASS

- [ ] **Step 7: Посмотреть глазами**

Run: `make dev`, открыть урок 05, дойти до диалога. Проверить: реплики
появляются по одной, тап по слову даёт карточку, кнопка перевода —
перевод реплики, динамик играет, после неверного ответа разговор
продолжается правильной репликой.

- [ ] **Step 8: Коммит**

```bash
git add content/vocab.yaml content/lessons/05.yaml content/lessons/_TEMPLATE.yaml content/audio/
git commit -m "content: диалог «У пекари» в уроке 05

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 12: Сцены уроков 02, 09, 10

Три коротких диалога по 8–10 ходов. Для каждого повторяется цикл из Task 11:
слова → шаг → `go test ./server/internal/content/` → `python3 scripts/tts.py`.

- [ ] **Step 1: Урок 02 — сосед в подъезде**

Слова в `teaches` урока 02 и `content/vocab.yaml`: `ne razumem` (не понимаю),
`ponovite` (повторите), `polako` (медленно, не спеша). Сцена: сосед
здоровается, спрашивает, откуда ты, кем работаешь, говоришь ли по-сербски.
8 ходов, 4 из них `me`. Один ход обязательно отрабатывает фразу выживания
(«Ne razumem» / «Polako, molim»).

- [ ] **Step 2: Урок 09 — кафич**

Слово `račun` (счёт) в `teaches` урока 09 и в `content/vocab.yaml`. Сцена:
официант принимает заказ, уточняет, ты просишь счёт. 10 ходов, 4–5 `me`.
`voice: m` — официант мужской.

- [ ] **Step 3: Урок 10 — дорога к автобусной станции**

Новых слов не требуется. Сцена: спрашиваешь прохожего, где автобусная
станция, уточняешь, далеко ли, благодаришь. 8 ходов, 3–4 `me`.

- [ ] **Step 4: Проверить**

Run: `go test ./server/internal/content/ -v && python3 scripts/tts.py && make test`
Expected: PASS

- [ ] **Step 5: Коммит**

```bash
git add content/
git commit -m "content: диалоги в уроках 02, 09, 10

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 13: Сцены уроков 08 и 06

Два длинных диалога — осмотр квартиры и заселение.

- [ ] **Step 1: Урок 08 — осмотр квартиры**

Слово `kirija` (аренда, плата за жильё) в `teaches` урока 08 и в
`content/vocab.yaml`. Сцена: смотришь квартиру — сколько комнат, где кухня,
какая аренда. 10 ходов, 4–5 `me`.

- [ ] **Step 2: Урок 06 — заселение**

Урок 06 — проверочный, его `teaches` пуст по замыслу и таким остаётся.
Слова сцены `vlasnik` (хозяин) и `ključ` (ключ) идут в `also_ok` шага,
в словарь не добавляются. Сцена: знакомство с хозяином квартиры, ключи,
деньги. 12 ходов, 5–6 `me`, лексика только из уроков 01–05.

- [ ] **Step 3: Проверить**

Run: `go test ./server/internal/content/ -v && python3 scripts/tts.py && make test`
Expected: PASS

- [ ] **Step 4: Пройти все шесть диалогов**

Run: `make dev` — пройти диалоги в уроках 02, 05, 06, 08, 09, 10 подряд,
на десктопе и в мобильной ширине.

- [ ] **Step 5: Коммит**

```bash
git add content/
git commit -m "content: диалоги в уроках 08 и 06

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```
