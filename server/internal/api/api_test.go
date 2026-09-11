package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
	"github.com/grisha/serbian-app/server/internal/config"
	"github.com/grisha/serbian-app/server/internal/content"
	"github.com/grisha/serbian-app/server/internal/store"
)

var fixedNow = time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

// testBaseURL is the origin newTestAPI configures; non-GET helpers send it as
// the Origin header so checkOrigin lets the request through.
const testBaseURL = "http://localhost:8080"

func newTestAPI(t *testing.T) (http.Handler, *store.Store) {
	return newTestAPIWith(t, nil)
}

// newTestAPIWith builds the API with the default test wiring, then lets the
// caller mutate the Deps before Handler runs — used by the auth tests to
// inject a capturing SendMail, a synchronous Async, a fixed-clock function,
// and real limiters with tiny thresholds.
func newTestAPIWith(t *testing.T, tweak func(*Deps)) (http.Handler, *store.Store) {
	t.Helper()
	c, err := content.Load("../content/testdata/content")
	if err != nil {
		t.Fatalf("load fixture content: %v", err)
	}
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if _, err := st.EnsureUserByName("tester"); err != nil {
		t.Fatal(err)
	}
	d := Deps{
		Course: func() *content.Course { return c },
		Store:  st,
		Now:    func() time.Time { return fixedNow },
		Stale:  func() bool { return false },
		Config: config.Config{AppBaseURL: testBaseURL},
		// SendMail/Async and the limiters stay nil unless tweak sets them:
		// Handler's defaults plus the allow/locked helpers treat nil as
		// permissive.
	}
	if tweak != nil {
		tweak(&d)
	}
	return Handler(d), st
}

// do issues a request as account "tester".
func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	return doAs(h, "tester", method, path, body)
}

func doAs(h http.Handler, user, method, path, body string) *httptest.ResponseRecorder {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if user != "" {
		r.Header.Set("X-User", user)
	}
	setTestOrigin(r)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	return rr
}

// authed creates a live session for userID and returns the cookie that carries
// its raw token. The cookie name matches config.Config{AppBaseURL: http...}.CookieName().
func authed(t *testing.T, st *store.Store, userID string) *http.Cookie {
	t.Helper()
	raw, hash, err := auth.NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if err := st.CreateSession(hash, userID, "test-agent", fixedNow, fixedNow.Add(720*time.Hour)); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	return &http.Cookie{Name: "session", Value: raw}
}

// doCookie mirrors doAs but authenticates with a session cookie instead of the
// X-User bridge.
func doCookie(h http.Handler, c *http.Cookie, method, path, body string) *httptest.ResponseRecorder {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	r.AddCookie(c)
	setTestOrigin(r)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	return rr
}

// setTestOrigin stamps the Origin header on state-changing requests so
// checkOrigin does not 403 them; GET/HEAD/OPTIONS are exempt from that guard.
func setTestOrigin(r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
	default:
		r.Header.Set("Origin", testBaseURL)
	}
}

func decodeBody[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(rr.Body).Decode(&v); err != nil {
		b, _ := io.ReadAll(rr.Body)
		t.Fatalf("decode: %v (body: %s)", err, b)
	}
	return v
}

func TestHealth(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "GET", "/api/health", "")
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
}

func TestGetCourseHasStatuses(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "GET", "/api/course", "")
	if rr.Code != 200 {
		t.Fatal(rr.Code)
	}
	got := decodeBody[courseDTO](t, rr)
	if len(got.Phases) != 1 || len(got.Lessons) != 2 {
		t.Fatalf("course = %+v", got)
	}
	if got.Lessons[0].Status != "not_started" {
		t.Errorf("status = %q", got.Lessons[0].Status)
	}
}

func TestExercisesEndpointHidesAccept(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "GET", "/api/lessons/01/exercises", "")
	if rr.Code != 200 {
		t.Fatal(rr.Code)
	}
	if strings.Contains(rr.Body.String(), "Zdravo") || strings.Contains(rr.Body.String(), "govoriš") {
		t.Errorf("accept leaked to client: %s", rr.Body)
	}
}

func TestCheckRecordsAttemptAndReturnsResult(t *testing.T) {
	h, st := newTestAPI(t)
	rr := do(h, "POST", "/api/lessons/01/exercises/01-A-1/check", `{"answer":"Zdravo! Kako si?"}`)
	if rr.Code != 200 {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
	res := decodeBody[checkResultDTO](t, rr)
	if !res.OK {
		t.Errorf("expected ok, got %+v", res)
	}
	tester, _ := st.UserByName("tester")
	if s, _ := st.User(tester.ID).LessonStatus("01"); s != "in_progress" {
		t.Errorf("lesson status = %q, want in_progress", s)
	}
}

func TestStateEndpointsRequireAccount(t *testing.T) {
	h, _ := newTestAPI(t)
	for _, p := range []string{"/api/course", "/api/progress", "/api/review/queue", "/api/lessons/01"} {
		if rr := doAs(h, "", "GET", p, ""); rr.Code != 401 {
			t.Errorf("%s without account = %d, want 401", p, rr.Code)
		}
	}
	if rr := doAs(h, "ghost", "GET", "/api/course", ""); rr.Code != 401 {
		t.Errorf("unknown account = %d, want 401", rr.Code)
	}
}

// TestMiddlewareWiring pins the router precedence: exact auth paths reach
// their handler without requireAuth, while any other /api/ route is gated.
func TestMiddlewareWiring(t *testing.T) {
	h, st := newTestAPI(t)
	tester, _ := st.UserByName("tester")

	// Public auth path: no credentials -> reaches the handler (register 400s on
	// the empty body), not requireAuth (401).
	if rr := doAs(h, "", "POST", "/api/auth/register", `{}`); rr.Code != http.StatusBadRequest {
		t.Errorf("POST /api/auth/register unauthenticated = %d, want 400", rr.Code)
	}

	// Protected path: no credentials -> requireAuth 401 (never reaches the stub).
	if rr := doAs(h, "", "GET", "/api/progress", ""); rr.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/progress unauthenticated = %d, want 401", rr.Code)
	}
	if rr := doAs(h, "", "GET", "/api/me", ""); rr.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/me unauthenticated = %d, want 401", rr.Code)
	}
	// GET /api/auth/session moved behind requireAuth in Task 12: 401 when
	// logged out, not an empty session body.
	if rr := doAs(h, "", "GET", "/api/auth/session", ""); rr.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/auth/session unauthenticated = %d, want 401", rr.Code)
	}

	// A cookie session flows through securityHeaders -> checkOrigin -> requireAuth.
	c := authed(t, st, tester.ID)
	if rr := doCookie(h, c, "GET", "/api/progress", ""); rr.Code != http.StatusOK {
		t.Errorf("GET /api/progress with cookie = %d, want 200", rr.Code)
	}
	// requireAuth resolved the caller, so the protected route is now reachable
	// (Task 16 implemented /api/me; it no longer 501s).
	if rr := doCookie(h, c, "GET", "/api/me", ""); rr.Code != http.StatusOK {
		t.Errorf("GET /api/me with cookie = %d, want 200", rr.Code)
	}
	// ...and GET /api/auth/session, now protected, returns the account (200).
	if rr := doCookie(h, c, "GET", "/api/auth/session", ""); rr.Code != http.StatusOK {
		t.Errorf("GET /api/auth/session with cookie = %d, want 200", rr.Code)
	}

	// The origin guard still rejects a state-changing request with a foreign Origin.
	r := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{}`))
	r.Header.Set("Origin", "https://evil.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusForbidden {
		t.Errorf("cross-origin POST /api/auth/login = %d, want 403", rr.Code)
	}
}

func TestAccountsAreIsolatedOverAPI(t *testing.T) {
	h, st := newTestAPI(t)
	_, _ = st.EnsureUserByName("Оля")
	doAs(h, "tester", "POST", "/api/lessons/01/complete", "")

	a := decodeBody[progressDTO](t, doAs(h, "tester", "GET", "/api/progress", ""))
	b := decodeBody[progressDTO](t, doAs(h, "Оля", "GET", "/api/progress", ""))
	if a.Phases[0].Done != 1 || b.Phases[0].Done != 0 {
		t.Errorf("progress leaked: tester=%d оля=%d", a.Phases[0].Done, b.Phases[0].Done)
	}
}

func TestCheckWrongReturnsExpectedAndExplain(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "POST", "/api/lessons/01/exercises/01-A-1/check", `{"answer":"zdravo kako sti"}`)
	res := decodeBody[checkResultDTO](t, rr)
	if res.OK {
		t.Fatal("should be wrong")
	}
	if res.Expected == "" || res.Explain == "" {
		t.Errorf("want expected+explain, got %+v", res)
	}
}

func TestCheckConjugateFormByForm(t *testing.T) {
	h, _ := newTestAPI(t)
	body := `{"answers":["govorim","WRONG","govori","govorimo","govorite","govore"]}`
	rr := do(h, "POST", "/api/lessons/01/exercises/01-A-3/check", body)
	res := decodeBody[checkResultDTO](t, rr)
	if res.OK || len(res.Forms) != 6 || res.Forms[1].OK {
		t.Errorf("forms = %+v", res)
	}
}

func TestCheckFreeAlwaysOKAndSample(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"blah","self":false}`)
	res := decodeBody[checkResultDTO](t, rr)
	if res.OK {
		t.Error("self:false should record as not ok")
	}
	if res.Sample == "" {
		t.Error("want sample")
	}
}

func TestReviewQueueThenGrade(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "GET", "/api/review/queue", "")
	if rr.Code != 200 {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
	q := decodeBody[[]reviewCardDTO](t, rr)
	if len(q) == 0 {
		t.Fatal("empty queue")
	}
	rr2 := do(h, "POST", "/api/review/grade", `{"card_id":"vocab:zdravo","grade":2}`)
	if rr2.Code != 200 {
		t.Fatalf("grade: %d %s", rr2.Code, rr2.Body)
	}
	g := decodeBody[gradeResultDTO](t, rr2)
	if g.State != "review" || g.IntervalDays != 1 {
		t.Errorf("grade result = %+v", g)
	}
}

func TestReviewGradeUnknownCard404(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "POST", "/api/review/grade", `{"card_id":"vocab:nope","grade":2}`)
	if rr.Code != 404 {
		t.Errorf("code = %d, want 404", rr.Code)
	}
}

func TestLessonCompleteThenProgress(t *testing.T) {
	h, _ := newTestAPI(t)
	if rr := do(h, "POST", "/api/lessons/01/complete", ""); rr.Code != 200 {
		t.Fatalf("complete: %d", rr.Code)
	}
	rr := do(h, "GET", "/api/progress", "")
	p := decodeBody[progressDTO](t, rr)
	if p.Phases[0].Done != 1 {
		t.Errorf("phase done = %d, want 1", p.Phases[0].Done)
	}
}

func TestLessonAttemptsAndReset(t *testing.T) {
	h, _ := newTestAPI(t)
	do(h, "POST", "/api/lessons/01/exercises/01-A-1/check", `{"answer":"Zdravo! Kako si?"}`)
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"x","self":false}`)

	got := decodeBody[map[string]attemptDTO](t, do(h, "GET", "/api/lessons/01/attempts", ""))
	if len(got) != 2 || !got["01-A-1"].Correct || got["01-A-2"].Correct {
		t.Fatalf("attempts = %+v", got)
	}
	if got["01-A-1"].Answer != "Zdravo! Kako si?" {
		t.Errorf("answer not stored: %+v", got["01-A-1"])
	}

	if rr := do(h, "POST", "/api/lessons/01/reset", ""); rr.Code != 200 {
		t.Fatalf("reset: %d", rr.Code)
	}
	after := decodeBody[map[string]attemptDTO](t, do(h, "GET", "/api/lessons/01/attempts", ""))
	if len(after) != 0 {
		t.Errorf("attempts after reset = %+v", after)
	}
	p := decodeBody[progressDTO](t, do(h, "GET", "/api/progress", ""))
	if len(p.RecentLessons) != 0 {
		t.Errorf("lesson progress survived reset: %+v", p.RecentLessons)
	}
}

func TestResetExercisesKeepsSRS(t *testing.T) {
	h, _ := newTestAPI(t)
	do(h, "POST", "/api/lessons/01/exercises/01-A-1/check", `{"answer":"Zdravo! Kako si?"}`)
	do(h, "GET", "/api/review/queue", "") // seed cards
	do(h, "POST", "/api/review/grade", `{"card_id":"vocab:zdravo","grade":2}`)

	if rr := do(h, "POST", "/api/reset-exercises", ""); rr.Code != 200 {
		t.Fatalf("reset-exercises: %d", rr.Code)
	}
	p := decodeBody[progressDTO](t, do(h, "GET", "/api/progress", ""))
	if len(p.RecentLessons) != 0 {
		t.Errorf("lessons not reset: %+v", p.RecentLessons)
	}
	// SRS review still counts
	if p.SRS.ReviewedToday != 1 {
		t.Errorf("SRS wiped by exercise reset: reviewed_today=%d", p.SRS.ReviewedToday)
	}
}

func TestLeaderboard(t *testing.T) {
	h, st := newTestAPI(t)
	_, _ = st.EnsureUserByName("Оля")
	do(h, "POST", "/api/lessons/01/complete", "")                                      // tester: 1 lesson
	doAs(h, "Оля", "POST", "/api/lessons/01/exercises/01-A-1/check", `{"answer":"x"}`) // Оля: activity, 0 lessons

	rows := decodeBody[[]leaderRowDTO](t, do(h, "GET", "/api/leaderboard", ""))
	if len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	if rows[0].Name != "tester" || rows[0].LessonsDone != 1 {
		t.Errorf("leader = %+v", rows[0])
	}
	if rows[0].LessonsTotal != 2 { // fixture course has 2 lessons
		t.Errorf("lessons_total = %d", rows[0].LessonsTotal)
	}
}

func TestGetLessonCarriesReadingText(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "GET", "/api/lessons/01", "")
	if rr.Code != 200 {
		t.Fatal(rr.Code)
	}
	got := decodeBody[lessonDTO](t, rr)
	if got.Reading != "Zdravo, ja sam Milan." || got.ReadingRU != "Привет, я Милан." {
		t.Errorf("reading = %q / %q", got.Reading, got.ReadingRU)
	}
	if strings.Contains(got.Markdown, "Zdravo") {
		t.Errorf("reading leaked into markdown: %q", got.Markdown)
	}
}

func TestGetLessonIncludesSteps(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "GET", "/api/lessons/90", "")
	if rr.Code != 200 {
		t.Fatalf("code %d: %s", rr.Code, rr.Body)
	}
	got := decodeBody[lessonDTO](t, rr)
	if len(got.Steps) != 4 || got.Steps[0].Kind != "teach" {
		t.Fatalf("steps: %+v", got.Steps)
	}
	if got.Steps[0].Status != "not_started" {
		t.Errorf("fresh step status = %q", got.Steps[0].Status)
	}
	if len(got.Steps[1].ExerciseIDs) == 0 || got.Steps[1].ExerciseIDs[0] != "90.2.1" {
		t.Errorf("practice step exercise ids: %v", got.Steps[1].ExerciseIDs)
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
	rr := do(h, "GET", "/api/lessons/90", "")
	got := decodeBody[lessonDTO](t, rr)
	if got.Steps[0].Status != "done" {
		t.Errorf("step status after set = %q", got.Steps[0].Status)
	}
	if got.Status != "in_progress" {
		t.Errorf("lesson status = %q, want in_progress", got.Status)
	}
	if rr := do(h, "POST", "/api/lessons/90/steps/90.1", `{"status":"nope"}`); rr.Code != 400 {
		t.Fatalf("bad status: code %d", rr.Code)
	}
	if rr := do(h, "POST", "/api/lessons/nope/steps/x", `{"status":"done"}`); rr.Code != 404 {
		t.Fatalf("unknown lesson: code %d", rr.Code)
	}
}

func TestExercisesEndpointHidesNewAnswers(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "GET", "/api/lessons/90/exercises", "")
	body := rr.Body.String()
	for _, leak := range []string{`"answer"`, `"accept"`, `"pairs"`, `"say"`} {
		if strings.Contains(body, leak) {
			t.Errorf("exercises endpoint leaked %s: %s", leak, body)
		}
	}
	if !strings.Contains(body, `"options"`) {
		t.Error("choice options not delivered")
	}
}

func TestCheckChoiceEndpoint(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "POST", "/api/lessons/90/exercises/90.2.1/check", `{"answer":"Da"}`)
	res := decodeBody[checkResultDTO](t, rr)
	if !res.OK {
		t.Fatalf("right choice not OK: %s", rr.Body)
	}
	rr = do(h, "POST", "/api/lessons/90/exercises/90.2.1/check", `{"answer":"Ne"}`)
	res = decodeBody[checkResultDTO](t, rr)
	if res.OK || res.Expected != "Da" {
		t.Fatalf("wrong choice graded OK or missing expected: %+v", res)
	}
}

func TestListenExerciseExposesAudioNotAnswer(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "GET", "/api/lessons/01/exercises", "")
	body := rr.Body.String()
	if strings.Contains(body, "kako si") {
		t.Errorf("listen answer/say leaked: %s", body)
	}
	blocks := decodeBody[[]exerciseBlockDTO](t, rr)
	var listen *exerciseDTO
	for i := range blocks {
		for j := range blocks[i].Exercises {
			if blocks[i].Exercises[j].ID == "01-A-4" {
				listen = &blocks[i].Exercises[j]
			}
		}
	}
	if listen == nil {
		t.Fatal("listen exercise 01-A-4 not in response")
	}
	if listen.Type != "listen" || listen.Audio != "01-A-4.mp3" {
		t.Errorf("listen dto = %+v", *listen)
	}
}

func TestCheckListenGradesTypedTranscription(t *testing.T) {
	h, _ := newTestAPI(t)
	ok := do(h, "POST", "/api/lessons/01/exercises/01-A-4/check", `{"answer":"Zdravo, kako si?"}`)
	if r := decodeBody[checkResultDTO](t, ok); !r.OK {
		t.Errorf("correct transcription rejected: %+v", r)
	}
	bad := do(h, "POST", "/api/lessons/01/exercises/01-A-4/check", `{"answer":"Zdravo, kako ste?"}`)
	if r := decodeBody[checkResultDTO](t, bad); r.OK {
		t.Errorf("wrong transcription accepted: %+v", r)
	}
}

func TestReviewAddActivatesWordCard(t *testing.T) {
	h, _ := newTestAPI(t)

	rr := do(h, "POST", "/api/review/add", `{"vocab_id":"svet"}`)
	if rr.Code != 200 {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
	if got := decodeBody[struct {
		Status string `json:"status"`
	}](t, rr); got.Status != "added" {
		t.Errorf("status = %q, want added", got.Status)
	}

	// it now shows up in the review queue
	q := decodeBody[[]reviewCardDTO](t, do(h, "GET", "/api/review/queue", ""))
	found := false
	for _, c := range q {
		if c.CardID == "vocab:svet" {
			found = true
		}
	}
	if !found {
		t.Errorf("activated card not in review queue: %+v", q)
	}

	// second time: already there
	rr2 := do(h, "POST", "/api/review/add", `{"vocab_id":"svet"}`)
	if got := decodeBody[struct {
		Status string `json:"status"`
	}](t, rr2); got.Status != "already" {
		t.Errorf("status = %q, want already", got.Status)
	}
}

func TestReviewAddUnknownWord404(t *testing.T) {
	h, _ := newTestAPI(t)
	if rr := do(h, "POST", "/api/review/add", `{"vocab_id":"nonesuch"}`); rr.Code != 404 {
		t.Errorf("code = %d, want 404", rr.Code)
	}
}

func TestReviewAddRequiresAccount(t *testing.T) {
	h, _ := newTestAPI(t)
	if rr := doAs(h, "", "POST", "/api/review/add", `{"vocab_id":"svet"}`); rr.Code != 401 {
		t.Errorf("code = %d, want 401", rr.Code)
	}
}

func TestUnknownLesson404(t *testing.T) {
	h, _ := newTestAPI(t)
	if do(h, "GET", "/api/lessons/zz", "").Code != 404 {
		t.Error("want 404 for unknown lesson")
	}
}

func TestVocabFilter(t *testing.T) {
	h, _ := newTestAPI(t)
	rr := do(h, "GET", "/api/vocab?q=здраво", "")
	v := decodeBody[[]vocabDTO](t, rr)
	if len(v) != 1 || v[0].ID != "zdravo" {
		t.Errorf("vocab filter = %+v", v)
	}
}

func TestDialogueHidesUnansweredLines(t *testing.T) {
	h, _ := newTestAPI(t)

	before := decodeBody[lessonDTO](t, do(h, "GET", "/api/lessons/91", ""))
	if len(before.Steps) != 1 || before.Steps[0].Kind != "dialogue" {
		t.Fatalf("fixture lesson 91 should hold one dialogue step, got %+v", before.Steps)
	}
	s := before.Steps[0]
	if s.Scene == "" || s.Voice != "f" {
		t.Errorf("scene=%q voice=%q", s.Scene, s.Voice)
	}
	if len(s.Turns) != 3 {
		t.Fatalf("want 3 turns, got %d", len(s.Turns))
	}
	if s.Turns[0].SR != "Izvolite?" {
		t.Errorf("npc line must always be visible, got %q", s.Turns[0].SR)
	}
	if s.Turns[1].SR != "" || s.Turns[1].RU != "" {
		t.Errorf("unanswered me line leaked: sr=%q ru=%q", s.Turns[1].SR, s.Turns[1].RU)
	}
	if s.Turns[1].ExerciseID != "91.1.1" {
		t.Errorf("me turn must expose its exercise id, got %q", s.Turns[1].ExerciseID)
	}

	if rr := do(h, "POST", "/api/lessons/91/exercises/91.1.1/check", `{"answer":"Jedan hleb, molim."}`); rr.Code != 200 {
		t.Fatalf("check: status %d", rr.Code)
	}

	after := decodeBody[lessonDTO](t, do(h, "GET", "/api/lessons/91", ""))
	if got := after.Steps[0].Turns[1].SR; got != "Jedan hleb, molim." {
		t.Errorf("answered me line should be revealed, got %q", got)
	}
	if got := after.Steps[0].Turns[1].RU; got != "Один хлеб, пожалуйста." {
		t.Errorf("answered me line translation = %q", got)
	}
}

func TestCheckReturnsDialogueLine(t *testing.T) {
	h, _ := newTestAPI(t)

	res := decodeBody[checkResultDTO](t, do(h, "POST",
		"/api/lessons/91/exercises/91.1.1/check", `{"answer":"Jedan hleb, molim."}`))
	if !res.OK {
		t.Fatal("answer should be accepted")
	}
	if res.Line != "Jedan hleb, molim." || res.LineRU != "Один хлеб, пожалуйста." {
		t.Errorf("line=%q line_ru=%q", res.Line, res.LineRU)
	}
}

func TestCheckReturnsLineAfterWrongAnswer(t *testing.T) {
	h, _ := newTestAPI(t)

	res := decodeBody[checkResultDTO](t, do(h, "POST",
		"/api/lessons/91/exercises/91.1.1/check", `{"answer":"Jedan hleb, hvala."}`))
	if res.OK {
		t.Fatal("answer should be rejected")
	}
	// The chat shows the correct line even after a miss — otherwise the
	// conversation loses its thread.
	if res.Line != "Jedan hleb, molim." {
		t.Errorf("line после ошибки = %q", res.Line)
	}
}

func TestCheckOutsideDialogueHasNoLine(t *testing.T) {
	h, _ := newTestAPI(t)

	res := decodeBody[checkResultDTO](t, do(h, "POST",
		"/api/lessons/90/exercises/90.2.1/check", `{"answer":"Da"}`))
	if res.Line != "" || res.LineRU != "" {
		t.Errorf("plain exercise leaked a dialogue line: %q / %q", res.Line, res.LineRU)
	}
}
