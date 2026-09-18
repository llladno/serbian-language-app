// Package api exposes the JSON HTTP API over the course content and store.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
	"github.com/grisha/serbian-app/server/internal/checker"
	"github.com/grisha/serbian-app/server/internal/config"
	"github.com/grisha/serbian-app/server/internal/content"
	"github.com/grisha/serbian-app/server/internal/ratelimit"
	"github.com/grisha/serbian-app/server/internal/srs"
	"github.com/grisha/serbian-app/server/internal/store"
)

// Deps are the API's collaborators.
type Deps struct {
	Course func() *content.Course
	Store  *store.Store
	Now    func() time.Time
	Stale  func() bool
	// Config is the resolved process configuration. The zero value works
	// (non-secure cookies, no cross-origin writes).
	Config config.Config
	// SendMail delivers a message. Production enqueues it; tests capture it.
	// A nil value is replaced by a no-op in Handler.
	SendMail func(to, subject, text, html string)
	// Async runs f. Production spawns a goroutine; tests run it inline for
	// deterministic assertions. A nil value defaults to `go f()` in Handler.
	Async func(func())
	// Login throttles login attempts per client IP (nil = no limit).
	Login *ratelimit.Limiter
	// LoginEmail throttles login attempts per target email (nil = no limit).
	LoginEmail *ratelimit.Limiter
	// Slow throttles register/resend/forgot per ip:/email: key (nil = no limit).
	Slow *ratelimit.Limiter
	// Fails is the soft account lock keyed by email (nil = never locks).
	Fails *ratelimit.FailCounter
	// TelegramPending tracks outstanding /start login and link tokens. A nil
	// value is replaced by a fresh auth.NewPendingStore() in Handler.
	TelegramPending *auth.PendingStore
	// TelegramBotUsername is the bot's own @username (no "@"), resolved once
	// at startup via telegram.GetMe. Empty disables /start login/link (the
	// t.me deep link cannot be built without it) — it does not affect the
	// still-independent Mini App initData path.
	TelegramBotUsername string
	// TelegramWebhookSecret is the value Telegram echoes back on every
	// webhook call (X-Telegram-Bot-Api-Secret-Token), set once at startup
	// via telegram.SetWebhook. Empty makes the webhook handler a no-op 200
	// for every request — never authenticate against an empty secret.
	TelegramWebhookSecret string
	// SendTelegramMessage sends a chat message from the bot. Production
	// calls the real Bot API; tests capture it. A nil value is replaced by a
	// no-op in Handler.
	SendTelegramMessage func(chatID int64, text string)
}

type handlers struct{ Deps }

// allow reports whether limiter l admits an event for key. A nil limiter (a
// feature left unconfigured, or a test) always admits.
func allow(l *ratelimit.Limiter, key string) bool { return l == nil || l.Allow(key) }

// locked reports whether fail counter f currently holds key in a soft lock. A
// nil counter never locks.
func locked(f *ratelimit.FailCounter, key string) bool { return f != nil && f.Locked(key) }

// noteFail records one failed auth attempt for key. A nil counter is a no-op.
func noteFail(f *ratelimit.FailCounter, key string) {
	if f != nil {
		f.Fail(key)
	}
}

// clearFails wipes the failed-attempt record for key after a success. A nil
// counter is a no-op.
func clearFails(f *ratelimit.FailCounter, key string) {
	if f != nil {
		f.Reset(key)
	}
}

// Handler builds the /api router. Exact auth paths are public; account-scoped
// paths (/api/me*, logout-all, session) sit behind requireSession (cookie
// only); every other /api/ route sits behind requireAuth (a session cookie, or
// the legacy X-User bridge). The whole tree is wrapped in the security-header
// and origin guards.
func Handler(deps Deps) http.Handler {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.Stale == nil {
		deps.Stale = func() bool { return false }
	}
	if deps.SendMail == nil {
		deps.SendMail = func(to, subject, text, html string) {}
	}
	if deps.Async == nil {
		deps.Async = func(f func()) { go f() }
	}
	if deps.TelegramPending == nil {
		deps.TelegramPending = auth.NewPendingStore()
	}
	if deps.SendTelegramMessage == nil {
		deps.SendTelegramMessage = func(chatID int64, text string) {}
	}
	h := handlers{deps}

	root := http.NewServeMux()
	root.HandleFunc("GET /api/health", h.health)

	// Auth endpoints are public: they are how a caller gets a session in the
	// first place, so they must not sit behind requireAuth.
	root.HandleFunc("POST /api/auth/register", h.register)
	root.HandleFunc("POST /api/auth/login", h.login)
	root.HandleFunc("POST /api/auth/logout", h.logout)
	root.HandleFunc("GET /api/auth/verify", h.verifyEmail)
	root.HandleFunc("POST /api/auth/resend-verification", h.resendVerification)
	root.HandleFunc("POST /api/auth/forgot", h.forgotPassword)
	root.HandleFunc("POST /api/auth/reset", h.resetPassword)
	root.HandleFunc("POST /api/auth/telegram", h.telegramLogin)
	root.HandleFunc("POST /api/auth/telegram/start", h.telegramLoginStart)
	root.HandleFunc("GET /api/auth/telegram/poll", h.telegramPoll)
	// The webhook is Telegram calling US, never a browser — no session guard
	// of any kind, authenticated instead by the shared secret header (see
	// telegramWebhook's own doc comment).
	root.HandleFunc("POST /api/telegram/webhook", h.telegramWebhook)

	// Account-scoped endpoints. These are registered on root as exact
	// method+path patterns (the same precedence trick as the public auth routes
	// above) so Go 1.22+ ServeMux picks them over the "/api/" catch-all, and
	// they sit behind requireSession — NOT plain requireAuth: the legacy X-User
	// bridge must never reach anything that reads or mutates the account, since
	// a display name is public knowledge (/api/leaderboard) and most of these
	// accounts are Telegram-only (no password to check).
	root.HandleFunc("POST /api/auth/logout-all", h.requireSession(h.logoutAll))
	// Session inspection needs a resolved caller: 401 (not an empty body)
	// when logged out, so the auth middleware must run first.
	root.HandleFunc("GET /api/auth/session", h.requireSession(h.currentSession))
	root.HandleFunc("GET /api/me", h.requireSession(h.getMe))
	root.HandleFunc("PATCH /api/me", h.requireSession(h.patchMe))
	root.HandleFunc("POST /api/me/password", h.requireSession(h.changePassword))
	root.HandleFunc("POST /api/me/link/telegram", h.requireSession(h.linkTelegram))
	root.HandleFunc("POST /api/me/telegram/start", h.requireSession(h.telegramLinkStart))
	root.HandleFunc("DELETE /api/me/telegram", h.requireSession(h.unlinkTelegram))
	root.HandleFunc("DELETE /api/me/sessions/{id}", h.requireSession(h.deleteSession))
	root.HandleFunc("DELETE /api/me", h.requireSession(h.deleteMe))

	// Everything else under /api/ requires a resolved caller, and only these
	// legacy content routes still honour the X-User bridge (the pre-session
	// front-end talks to them during the transition).
	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/course", h.getCourse)
	protected.HandleFunc("GET /api/lessons/{id}", h.getLesson)
	protected.HandleFunc("GET /api/lessons/{id}/exercises", h.getExercises)
	protected.HandleFunc("POST /api/lessons/{id}/exercises/{exId}/check", h.checkExercise)
	protected.HandleFunc("GET /api/lessons/{id}/attempts", h.lessonAttempts)
	protected.HandleFunc("POST /api/lessons/{id}/steps/{step}", h.setStepStatus)
	protected.HandleFunc("POST /api/lessons/{id}/complete", h.completeLesson)
	protected.HandleFunc("POST /api/lessons/{id}/reset", h.resetLesson)
	protected.HandleFunc("POST /api/reset-exercises", h.resetExercises)
	protected.HandleFunc("GET /api/vocab", h.getVocab)
	protected.HandleFunc("GET /api/lookup", h.lookup)
	protected.HandleFunc("GET /api/false-friends", h.getFalseFriends)
	protected.HandleFunc("GET /api/review/queue", h.reviewQueue)
	protected.HandleFunc("POST /api/review/grade", h.reviewGrade)
	protected.HandleFunc("POST /api/review/add", h.reviewAdd)
	protected.HandleFunc("GET /api/progress", h.getProgress)
	protected.HandleFunc("GET /api/leaderboard", h.getLeaderboard)

	root.Handle("/api/", h.requireAuth(protected))

	return SecurityHeaders(deps.Config, checkOrigin(deps.Config, root))
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

// user resolves the account attached to the request by requireAuth. The
// X-User / session parsing lives in requireAuth now; this only reads the
// context it left behind. When no auth context is present it writes 401 and
// returns ok=false.
func (h handlers) user(w http.ResponseWriter, r *http.Request) (*store.UserStore, bool) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return nil, false
	}
	return h.Store.User(ac.UserID), true
}

// ---- handlers ----

func (h handlers) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "ok",
		"content_stale":   h.Stale(),
		"telegram_bot_id": h.Config.TelegramBotID(),
	})
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
	attempts, err := us.LessonAttempts(id)
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
		d.Scene, d.Voice = s.Scene, s.Voice
		for _, t := range s.Turns {
			td := turnDTO{Who: t.Who}
			shown := t.Who == "npc"
			if t.Exercise != nil {
				td.ExerciseID = t.Exercise.ID
				_, shown = attempts[t.Exercise.ID]
			}
			if shown {
				td.SR, td.RU, td.Audio = t.SR, t.RU, t.Audio
			}
			d.Turns = append(d.Turns, td)
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
		out = append(out, bd)
	}
	writeJSON(w, 200, out)
}

type checkRequest struct {
	Answer  string            `json:"answer"`
	Answers []string          `json:"answers"`
	Pairs   map[string]string `json:"pairs"` // match: left -> picked right
	Self    *bool             `json:"self"`
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
	if t, ok := h.findTurn(lesson, exID); ok {
		resp.Line, resp.LineRU, resp.LineAudio = t.SR, t.RU, t.Audio
	}
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
	case "choice":
		res := checker.CheckChoice(req.Answer, ex.Answer)
		resp.OK = res.OK
		resp.Expected = res.Expected
		recordAnswer = req.Answer
		correct = res.OK
	case "match":
		ok, per := checker.CheckMatch(req.Pairs, ex.Pairs)
		resp.OK = ok
		resp.Match = per
		b, _ := json.Marshal(req.Pairs)
		recordAnswer = string(b)
		correct = ok
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
			ExampleSR: v.ExampleSR, ExampleRU: v.ExampleRU,
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
	// newPerDay caps new false-friend cards per queue fetch, and (loosely,
	// for the profile dashboard's "new available" stat) new cards overall.
	// Vocab words are no longer drawn by count — see nextNewVocabCardID —
	// they're gated one at a time, in lesson order.
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

// orderedVocab returns the course vocabulary sorted by lesson ("01", "02", …
// — zero-padded, so lexical order is numeric order) so word introduction can
// follow course order instead of file order. Ties (same lesson) keep their
// original relative order (stable sort).
func orderedVocab(vocab []content.Vocab) []content.Vocab {
	out := make([]content.Vocab, len(vocab))
	copy(out, vocab)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Lesson < out[j].Lesson })
	return out
}

// beginnerWordCount is how many vocab words make up the "just the basics"
// on-ramp: while the learner has passed fewer than this many, every new-card
// slot goes to vocab (in lesson order) and false friends are held back
// entirely, so a brand-new learner's first sessions are nothing but plain
// lesson-one words — not a random hard false friend like "trudna"
// (беременная) sitting next to "zdravo".
const beginnerWordCount = 15

// nextNewVocabCardIDs walks the course vocabulary in lesson order and
// returns up to limit card ids the learner hasn't yet "passed" (graded Good
// or Easy at least once) — always the *earliest* unpassed ones, so a harder
// word from a later lesson never jumps ahead of one the learner hasn't
// remembered yet, even though (unlike a single-word gate) several can be
// introduced in the same session. A word already graded Again/Hard (state
// moved to "learning") is due for review through the normal due-queue
// instead, not reintroduced here.
func nextNewVocabCardIDs(vocab []content.Vocab, passed map[string]bool, limit int) []string {
	out := make([]string, 0, limit)
	for _, v := range orderedVocab(vocab) {
		if len(out) >= limit {
			break
		}
		id := "vocab:" + v.ID
		if !passed[id] {
			out = append(out, id)
		}
	}
	return out
}

// buildOptions returns 4 shuffled candidate answers for a first-encounter
// recognition quiz: the correct one plus up to 3 distinct distractors drawn
// from the other items' Back text (of the same kind — vocab translations
// aren't mixed with false-friend ones).
func buildOptions(correct string, pool []string) []string {
	distractors := make([]string, 0, len(pool))
	seen := map[string]bool{correct: true}
	for _, p := range pool {
		if seen[p] {
			continue
		}
		seen[p] = true
		distractors = append(distractors, p)
	}
	rand.Shuffle(len(distractors), func(i, j int) { distractors[i], distractors[j] = distractors[j], distractors[i] })
	if len(distractors) > 3 {
		distractors = distractors[:3]
	}
	out := append([]string{correct}, distractors...)
	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// dueQueueRows computes today's review queue for us — due learning/review
// cards plus the gated batch of new vocab/false-friend cards — the same
// selection reviewQueue turns into response DTOs. Split out so the reminder
// sweep (RunReminderSweep) can ask "is there anything left to do today?"
// without duplicating the new-card budget logic.
func (h handlers) dueQueueRows(us *store.UserStore, now time.Time) ([]store.CardRow, error) {
	if err := us.EnsureCards(h.cardSeeds()); err != nil {
		return nil, err
	}
	c := h.Course()

	passedVocab, err := us.PassedCardIDs("vocab:")
	if err != nil {
		return nil, err
	}
	// Split the newPerDay budget between vocab and false friends. Below
	// beginnerWordCount passed words, vocab gets the whole budget (in
	// lesson order) and false friends are held back entirely — a pure
	// "just the basics" on-ramp. Past that, vocab keeps half the budget
	// (still lesson-ordered, so a later lesson still can't leapfrog an
	// earlier one) and false friends fill the rest — back to the usual mix.
	vocabBudget := newPerDay
	beginner := len(passedVocab) < beginnerWordCount
	if !beginner {
		vocabBudget = newPerDay / 2
	}
	allowedNewVocab := nextNewVocabCardIDs(c.Vocab, passedVocab, vocabBudget)

	ffNewLimit := 0
	if !beginner {
		ffNewLimit = newPerDay - len(allowedNewVocab)
	}
	return us.DueQueue(now, allowedNewVocab, ffNewLimit)
}

func (h handlers) reviewQueue(w http.ResponseWriter, r *http.Request) {
	us, ok := h.user(w, r)
	if !ok {
		return
	}
	rows, err := h.dueQueueRows(us, h.Now())
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	c := h.Course()
	vocab := map[string]content.Vocab{}
	vocabBacks := make([]string, 0, len(c.Vocab))
	for _, v := range c.Vocab {
		vocab[v.ID] = v
		vocabBacks = append(vocabBacks, v.RU)
	}
	ff := map[string]content.FalseFriend{}
	ffBacks := make([]string, 0, len(c.FalseFriends))
	for _, f := range c.FalseFriends {
		ff[f.ID] = f
		ffBacks = append(ffBacks, f.Means)
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
			d.ExampleSR, d.ExampleRU = v.ExampleSR, v.ExampleRU
			if row.State == srs.New {
				d.Options = buildOptions(v.RU, vocabBacks)
			}
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
			if row.State == srs.New {
				d.Options = buildOptions(f.Means, ffBacks)
			}
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
