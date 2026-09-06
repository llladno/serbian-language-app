// Package api exposes the JSON HTTP API over the course content and store.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/grisha/serbian-app/server/internal/checker"
	"github.com/grisha/serbian-app/server/internal/content"
	"github.com/grisha/serbian-app/server/internal/srs"
	"github.com/grisha/serbian-app/server/internal/store"
)

// Deps are the API's collaborators.
type Deps struct {
	Course func() *content.Course
	Store  *store.Store
	Now    func() time.Time
	Stale  func() bool
}

type handlers struct{ Deps }

// Handler builds the /api router.
func Handler(deps Deps) http.Handler {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.Stale == nil {
		deps.Stale = func() bool { return false }
	}
	h := handlers{deps}
	mux := http.NewServeMux()
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
	return mux
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	if r.Body == nil {
		return nil
	}
	if err := json.NewDecoder(r.Body).Decode(v); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func contains(hay, needle string) bool {
	return strings.Contains(strings.ToLower(hay), strings.ToLower(needle))
}

// ---- handlers ----

func (h handlers) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "content_stale": h.Stale()})
}

func (h handlers) getCourse(w http.ResponseWriter, r *http.Request) {
	c := h.Course()
	statuses, err := h.Store.LessonStatuses()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	out := courseDTO{Title: c.Title, Phases: []phaseDTO{}, Lessons: []lessonRefDTO{}}
	for _, p := range c.Phases {
		out.Phases = append(out.Phases, phaseDTO{ID: p.ID, Title: p.Title, Lessons: p.Lessons})
		for _, id := range p.Lessons {
			l := c.Lessons[id]
			if l == nil {
				continue
			}
			st := statuses[id]
			if st == "" {
				st = "not_started"
			}
			out.Lessons = append(out.Lessons, lessonRefDTO{
				ID: id, Title: l.Title, Subtitle: l.Subtitle, Planned: l.Planned, Status: st,
			})
		}
	}
	writeJSON(w, 200, out)
}

func (h handlers) getLesson(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	l := h.Course().Lessons[id]
	if l == nil {
		fail(w, 404, "unknown lesson")
		return
	}
	st, err := h.Store.LessonStatus(id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if st == "" {
		st = "not_started"
	}
	writeJSON(w, 200, lessonDTO{
		ID: l.ID, Title: l.Title, Subtitle: l.Subtitle,
		Planned: l.Planned, Markdown: l.Markdown, Status: st,
	})
}

func (h handlers) getExercises(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if h.Course().Lessons[id] == nil {
		fail(w, 404, "unknown lesson")
		return
	}
	out := []exerciseBlockDTO{}
	for _, b := range h.Course().Exercises[id] {
		bd := exerciseBlockDTO{ID: b.ID, Title: b.Title, Instruction: b.Instruction}
		for _, e := range b.Exercises {
			bd.Exercises = append(bd.Exercises, exerciseDTO{
				ID: e.ID, Type: e.Type, Prompt: e.Prompt, Forms: e.Forms, Meta: e.Meta,
			})
		}
		out = append(out, bd)
	}
	writeJSON(w, 200, out)
}

type checkRequest struct {
	Answer  string   `json:"answer"`
	Answers []string `json:"answers"`
	Self    *bool    `json:"self"`
}

func (h handlers) findExercise(lesson, exID string) (content.Exercise, string, bool) {
	for _, b := range h.Course().Exercises[lesson] {
		for _, e := range b.Exercises {
			if e.ID == exID {
				return e, b.ID, true
			}
		}
	}
	return content.Exercise{}, "", false
}

func (h handlers) checkExercise(w http.ResponseWriter, r *http.Request) {
	lesson := r.PathValue("id")
	exID := r.PathValue("exId")
	ex, block, ok := h.findExercise(lesson, exID)
	if !ok {
		fail(w, 404, "unknown exercise")
		return
	}
	var req checkRequest
	if err := decode(r, &req); err != nil {
		fail(w, 400, "bad request body")
		return
	}

	now := h.Now()
	resp := checkResultDTO{Explain: ex.Explain}
	var recordAnswer string
	var correct bool

	switch ex.Type {
	case "conjugate":
		results := checker.CheckForms(req.Answers, ex.AcceptForms)
		allOK := true
		for _, r := range results {
			if !r.OK {
				allOK = false
			}
		}
		resp.OK = allOK
		resp.Forms = results
		recordAnswer = strings.Join(req.Answers, " | ")
		correct = allOK
	case "free":
		self := true
		if req.Self != nil {
			self = *req.Self
		}
		resp.OK = self
		resp.Sample = ex.Sample
		recordAnswer = req.Answer
		correct = self
	default:
		res := checker.Check(req.Answer, ex.Accept)
		resp.OK = res.OK
		resp.Diff = res.Diff
		resp.Expected = res.Expected
		resp.NearMiss = res.NearMiss
		recordAnswer = req.Answer
		correct = res.OK
	}

	_ = h.Store.AddAttempt(store.Attempt{
		ExerciseID: exID, Lesson: lesson, Block: block, Answer: recordAnswer, Correct: correct,
	}, now)
	if st, _ := h.Store.LessonStatus(lesson); st != "done" {
		_ = h.Store.SetLessonStatus(lesson, "in_progress", now)
	}

	writeJSON(w, 200, resp)
}

func (h handlers) completeLesson(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if h.Course().Lessons[id] == nil {
		fail(w, 404, "unknown lesson")
		return
	}
	if err := h.Store.SetLessonStatus(id, "done", h.Now()); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "done"})
}

func (h handlers) getVocab(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	lesson, tag, term := q.Get("lesson"), q.Get("tag"), q.Get("q")
	out := []vocabDTO{}
	for _, v := range h.Course().Vocab {
		if lesson != "" && v.Lesson != lesson {
			continue
		}
		if tag != "" && !hasTag(v.Tags, tag) {
			continue
		}
		if term != "" && !(contains(v.Latin, term) || contains(v.Cyrillic, term) || contains(v.RU, term)) {
			continue
		}
		out = append(out, vocabDTO{
			ID: v.ID, Latin: v.Latin, Cyrillic: v.Cyrillic, RU: v.RU, Note: v.Note,
			Lesson: v.Lesson, POS: v.POS, Gender: v.Gender, Aspect: v.Aspect, Tags: v.Tags,
		})
	}
	writeJSON(w, 200, out)
}

func (h handlers) getFalseFriends(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	group, term := q.Get("group"), q.Get("q")
	out := []falseFriendDTO{}
	for _, f := range h.Course().FalseFriends {
		if group != "" && group != "all" && f.Group != group {
			continue
		}
		if term != "" && !(contains(f.SR, term) || contains(f.Means, term) || contains(f.Not, term)) {
			continue
		}
		out = append(out, falseFriendDTO{
			ID: f.ID, SR: f.SR, Means: f.Means, Not: f.Not, Correct: f.Correct, Group: f.Group,
		})
	}
	writeJSON(w, 200, out)
}

const (
	newPerDay = 15
	dailyGoal = 20 // reviews + attempts that count as "a day done"
)

func (h handlers) cardSeeds() []store.CardSeed {
	c := h.Course()
	seeds := make([]store.CardSeed, 0, len(c.Vocab)+len(c.FalseFriends))
	for _, v := range c.Vocab {
		seeds = append(seeds, store.CardSeed{CardID: "vocab:" + v.ID, Kind: "vocab", RefID: v.ID})
	}
	for _, f := range c.FalseFriends {
		seeds = append(seeds, store.CardSeed{CardID: "ff:" + f.ID, Kind: "ff", RefID: f.ID})
	}
	return seeds
}

func (h handlers) reviewQueue(w http.ResponseWriter, r *http.Request) {
	if err := h.Store.EnsureCards(h.cardSeeds()); err != nil {
		fail(w, 500, err.Error())
		return
	}
	rows, err := h.Store.DueQueue(h.Now(), newPerDay)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	c := h.Course()
	vocab := map[string]content.Vocab{}
	for _, v := range c.Vocab {
		vocab[v.ID] = v
	}
	ff := map[string]content.FalseFriend{}
	for _, f := range c.FalseFriends {
		ff[f.ID] = f
	}

	now := h.Now()
	gradeNames := map[srs.Grade]string{srs.Again: "again", srs.Hard: "hard", srs.Good: "good", srs.Easy: "easy"}
	out := make([]reviewCardDTO, 0, len(rows))
	for _, row := range rows {
		d := reviewCardDTO{CardID: row.CardID, Kind: row.Kind, State: string(row.State), Preview: map[string]int{}}
		for g, days := range srs.Preview(row.Card, now) {
			d.Preview[gradeNames[g]] = days
		}
		switch row.Kind {
		case "vocab":
			v, ok := vocab[row.RefID]
			if !ok {
				continue
			}
			d.Front, d.Cyrillic, d.Back, d.Note = v.Latin, v.Cyrillic, v.RU, v.Note
		case "ff":
			f, ok := ff[row.RefID]
			if !ok {
				continue
			}
			d.Front, d.Back = f.SR, f.Means
			note := ""
			if f.Not != "" {
				note = "не: " + f.Not
			}
			if f.Correct != "" {
				if note != "" {
					note += " · "
				}
				note += "верно: " + f.Correct
			}
			d.Note = note
		}
		out = append(out, d)
	}
	writeJSON(w, 200, out)
}

type gradeRequest struct {
	CardID string `json:"card_id"`
	Grade  int    `json:"grade"`
}

func (h handlers) reviewGrade(w http.ResponseWriter, r *http.Request) {
	var req gradeRequest
	if err := decode(r, &req); err != nil {
		fail(w, 400, "bad request body")
		return
	}
	if req.Grade < 0 || req.Grade > 3 {
		fail(w, 400, "grade must be 0..3")
		return
	}
	card, err := h.Store.GradeCard(req.CardID, srs.Grade(req.Grade), h.Now())
	if err != nil {
		fail(w, 404, "unknown card")
		return
	}
	due := ""
	if !card.Due.IsZero() {
		due = card.Due.Format("2006-01-02")
	}
	writeJSON(w, 200, gradeResultDTO{Due: due, IntervalDays: card.IntervalDays, State: string(card.State)})
}

func (h handlers) getProgress(w http.ResponseWriter, r *http.Request) {
	c := h.Course()
	now := h.Now()
	_ = h.Store.EnsureCards(h.cardSeeds()) // so "new" counts are accurate before first review
	statuses, err := h.Store.LessonStatuses()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}

	out := progressDTO{
		Phases:        []phaseProgressDTO{},
		WeakExercises: []weakExerciseDTO{},
		RecentLessons: []recentLessonDTO{},
	}
	for _, p := range c.Phases {
		pp := phaseProgressDTO{ID: p.ID, Title: p.Title, Total: len(p.Lessons)}
		for _, id := range p.Lessons {
			if statuses[id] == "done" {
				pp.Done++
			}
		}
		out.Phases = append(out.Phases, pp)
	}

	dueToday, _ := h.Store.DueCount(now)
	newCount, _ := h.Store.NewCount()
	reviewed, _ := h.Store.ReviewedToday(now)
	total, known, _ := h.Store.CardStats()
	newAvail := newCount
	if newAvail > newPerDay {
		newAvail = newPerDay
	}
	out.SRS = srsProgressDTO{
		DueToday: dueToday, NewAvailable: newAvail, ReviewedToday: reviewed,
		TotalCards: total, Known: known,
	}

	streak, _ := h.Store.StreakDays(now)
	out.StreakDays = streak

	weak, _ := h.Store.WeakExercises(10)
	promptByID := map[string]string{}
	for _, blocks := range c.Exercises {
		for _, b := range blocks {
			for _, e := range b.Exercises {
				promptByID[e.ID] = e.Prompt
			}
		}
	}
	for _, wk := range weak {
		out.WeakExercises = append(out.WeakExercises, weakExerciseDTO{
			ExerciseID: wk.ExerciseID, Lesson: wk.Lesson, Prompt: promptByID[wk.ExerciseID],
			Wrong: wk.Wrong, Total: wk.Total,
		})
	}

	recent := []recentLessonDTO{}
	for id, st := range statuses {
		if st == "in_progress" || st == "done" {
			title := ""
			if l := c.Lessons[id]; l != nil {
				title = l.Title
			}
			recent = append(recent, recentLessonDTO{Lesson: id, Title: title, Status: st})
		}
	}
	sort.Slice(recent, func(i, j int) bool { return recent[i].Lesson < recent[j].Lesson })
	out.RecentLessons = recent

	out.DailyGoal = dailyGoal
	out.Activity = []dayActivityDTO{}
	if acts, err := h.Store.ActivityByDay(now.AddDate(0, 0, -97)); err == nil {
		for _, a := range acts {
			out.Activity = append(out.Activity, dayActivityDTO{Date: a.Date, Count: a.Count})
		}
	}

	writeJSON(w, 200, out)
}

func hasTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}
