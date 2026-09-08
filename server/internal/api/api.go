// Package api exposes the JSON HTTP API over the course content and store.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
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
	mux.HandleFunc("GET /api/users", h.listUsers)
	mux.HandleFunc("POST /api/users", h.createUser)
	mux.HandleFunc("GET /api/course", h.getCourse)
	mux.HandleFunc("GET /api/lessons/{id}", h.getLesson)
	mux.HandleFunc("GET /api/lessons/{id}/exercises", h.getExercises)
	mux.HandleFunc("POST /api/lessons/{id}/exercises/{exId}/check", h.checkExercise)
	mux.HandleFunc("GET /api/lessons/{id}/attempts", h.lessonAttempts)
	mux.HandleFunc("POST /api/lessons/{id}/steps/{step}", h.setStepStatus)
	mux.HandleFunc("POST /api/lessons/{id}/complete", h.completeLesson)
	mux.HandleFunc("POST /api/lessons/{id}/reset", h.resetLesson)
	mux.HandleFunc("POST /api/reset-exercises", h.resetExercises)
	mux.HandleFunc("GET /api/vocab", h.getVocab)
	mux.HandleFunc("GET /api/lookup", h.lookup)
	mux.HandleFunc("GET /api/false-friends", h.getFalseFriends)
	mux.HandleFunc("GET /api/review/queue", h.reviewQueue)
	mux.HandleFunc("POST /api/review/grade", h.reviewGrade)
	mux.HandleFunc("POST /api/review/add", h.reviewAdd)
	mux.HandleFunc("GET /api/progress", h.getProgress)
	mux.HandleFunc("GET /api/leaderboard", h.getLeaderboard)
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

// user resolves the account from the X-User header. On failure it writes a
// 401/500 response and returns ok=false.
func (h handlers) user(w http.ResponseWriter, r *http.Request) (*store.UserStore, bool) {
	raw := r.Header.Get("X-User")
	if dec, err := url.PathUnescape(raw); err == nil {
		raw = dec
	}
	name := store.NormalizeName(raw)
	if name == "" {
		fail(w, http.StatusUnauthorized, "no account")
		return nil, false
	}
	exists, err := h.Store.UserExists(name)
	if err != nil {
		fail(w, 500, err.Error())
		return nil, false
	}
	if !exists {
		fail(w, http.StatusUnauthorized, "unknown account")
		return nil, false
	}
	return h.Store.User(name), true
}

// ---- handlers ----

func (h handlers) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "content_stale": h.Stale()})
}

func (h handlers) listUsers(w http.ResponseWriter, r *http.Request) {
	names, err := h.Store.ListUsers()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if names == nil {
		names = []string{}
	}
	writeJSON(w, 200, map[string]any{"users": names})
}

func (h handlers) createUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, 400, "bad request body")
		return
	}
	name, err := h.Store.EnsureUser(req.Name)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"name": name})
}

func (h handlers) getCourse(w http.ResponseWriter, r *http.Request) {
	us, ok := h.user(w, r)
	if !ok {
		return
	}
	c := h.Course()
	statuses, err := us.LessonStatuses()
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
	us, ok := h.user(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	l := h.Course().Lessons[id]
	if l == nil {
		fail(w, 404, "unknown lesson")
		return
	}
	st, err := us.LessonStatus(id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if st == "" {
		st = "not_started"
	}
	stepSt, err := us.StepStatuses(id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
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
	writeJSON(w, 200, lessonDTO{
		ID: l.ID, Title: l.Title, Subtitle: l.Subtitle,
		Planned: l.Planned, Markdown: l.Markdown, Status: st,
		Reading: l.Reading, ReadingRU: l.ReadingRU, Steps: steps,
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
				ID: e.ID, Type: e.Type, Prompt: e.Prompt, Forms: e.Forms, Meta: e.Meta, Audio: e.Audio,
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
	us, ok := h.user(w, r)
	if !ok {
		return
	}
	lesson := r.PathValue("id")
	exID := r.PathValue("exId")
	ex, block, found := h.findExercise(lesson, exID)
	if !found {
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

	_ = us.AddAttempt(store.Attempt{
		ExerciseID: exID, Lesson: lesson, Block: block, Answer: recordAnswer, Correct: correct,
	}, now)
	if st, _ := us.LessonStatus(lesson); st != "done" {
		_ = us.SetLessonStatus(lesson, "in_progress", now)
	}

	writeJSON(w, 200, resp)
}

func (h handlers) lessonAttempts(w http.ResponseWriter, r *http.Request) {
	us, ok := h.user(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if h.Course().Lessons[id] == nil {
		fail(w, 404, "unknown lesson")
		return
	}
	m, err := us.LessonAttempts(id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	out := map[string]attemptDTO{}
	for exID, a := range m {
		out[exID] = attemptDTO{Answer: a.Answer, Correct: a.Correct}
	}
	writeJSON(w, 200, out)
}

func (h handlers) resetLesson(w http.ResponseWriter, r *http.Request) {
	us, ok := h.user(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if h.Course().Lessons[id] == nil {
		fail(w, 404, "unknown lesson")
		return
	}
	if err := us.ResetLesson(id); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "reset"})
}

func (h handlers) resetExercises(w http.ResponseWriter, r *http.Request) {
	us, ok := h.user(w, r)
	if !ok {
		return
	}
	if err := us.ResetExercises(); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "reset"})
}

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
	var req struct {
		Status string `json:"status"`
	}
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

func (h handlers) completeLesson(w http.ResponseWriter, r *http.Request) {
	us, ok := h.user(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if h.Course().Lessons[id] == nil {
		fail(w, 404, "unknown lesson")
		return
	}
	if err := us.SetLessonStatus(id, "done", h.Now()); err != nil {
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
			Emoji: v.Emoji, Image: v.Image, Audio: v.Audio,
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
			Emoji: f.Emoji, Image: f.Image,
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
	us, ok := h.user(w, r)
	if !ok {
		return
	}
	if err := us.EnsureCards(h.cardSeeds()); err != nil {
		fail(w, 500, err.Error())
		return
	}
	rows, err := us.DueQueue(h.Now(), newPerDay)
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
			d.Emoji, d.Image, d.Audio = v.Emoji, v.Image, v.Audio
		case "ff":
			f, ok := ff[row.RefID]
			if !ok {
				continue
			}
			d.Front, d.Back = f.SR, f.Means
			d.Emoji, d.Image = f.Emoji, f.Image
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
	us, ok := h.user(w, r)
	if !ok {
		return
	}
	var req gradeRequest
	if err := decode(r, &req); err != nil {
		fail(w, 400, "bad request body")
		return
	}
	if req.Grade < 0 || req.Grade > 3 {
		fail(w, 400, "grade must be 0..3")
		return
	}
	card, err := us.GradeCard(req.CardID, srs.Grade(req.Grade), h.Now())
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

// reviewAdd pulls a dictionary word into the learner's review queue — used by
// the "＋ в повторение" button on a word card in a reading text or exercise.
func (h handlers) reviewAdd(w http.ResponseWriter, r *http.Request) {
	us, ok := h.user(w, r)
	if !ok {
		return
	}
	var req struct {
		VocabID string `json:"vocab_id"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, 400, "bad request body")
		return
	}
	var found *content.Vocab
	for i, v := range h.Course().Vocab {
		if v.ID == req.VocabID {
			found = &h.Course().Vocab[i]
			break
		}
	}
	if found == nil {
		fail(w, 404, "unknown word")
		return
	}
	activated, err := us.ActivateCard(
		store.CardSeed{CardID: "vocab:" + found.ID, Kind: "vocab", RefID: found.ID}, h.Now())
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	status := "already"
	if activated {
		status = "added"
	}
	writeJSON(w, 200, map[string]string{"status": status})
}

func (h handlers) getProgress(w http.ResponseWriter, r *http.Request) {
	us, ok := h.user(w, r)
	if !ok {
		return
	}
	c := h.Course()
	now := h.Now()
	_ = us.EnsureCards(h.cardSeeds()) // so "new" counts are accurate before first review
	statuses, err := us.LessonStatuses()
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

	dueToday, _ := us.DueCount(now)
	newCount, _ := us.NewCount()
	reviewed, _ := us.ReviewedToday(now)
	total, known, _ := us.CardStats()
	newAvail := newCount
	if newAvail > newPerDay {
		newAvail = newPerDay
	}
	out.SRS = srsProgressDTO{
		DueToday: dueToday, NewAvailable: newAvail, ReviewedToday: reviewed,
		TotalCards: total, Known: known,
	}

	streak, _ := us.StreakDays(now)
	out.StreakDays = streak

	weak, _ := us.WeakExercises(10)
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
	if acts, err := us.ActivityByDay(now.AddDate(0, 0, -97)); err == nil {
		for _, a := range acts {
			out.Activity = append(out.Activity, dayActivityDTO{Date: a.Date, Count: a.Count})
		}
	}

	writeJSON(w, 200, out)
}

func (h handlers) getLeaderboard(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.user(w, r); !ok {
		return
	}
	rows, err := h.Store.AllUsersProgress(h.Now())
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	totalLessons := 0
	for _, p := range h.Course().Phases {
		totalLessons += len(p.Lessons)
	}
	out := make([]leaderRowDTO, 0, len(rows))
	for _, p := range rows {
		out = append(out, leaderRowDTO{
			Name: p.Name, LessonsDone: p.LessonsDone, LessonsTotal: totalLessons,
			CardsKnown: p.CardsKnown, TotalCards: p.TotalCards, StreakDays: p.StreakDays,
			ReviewedToday: p.ReviewedToday, LastActive: p.LastActive,
		})
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
