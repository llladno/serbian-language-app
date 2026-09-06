package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/content"
	"github.com/grisha/serbian-app/server/internal/store"
)

var fixedNow = time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

func newTestAPI(t *testing.T) (http.Handler, *store.Store) {
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
	if _, err := st.EnsureUser("tester"); err != nil {
		t.Fatal(err)
	}
	h := Handler(Deps{
		Course: func() *content.Course { return c },
		Store:  st,
		Now:    func() time.Time { return fixedNow },
		Stale:  func() bool { return false },
	})
	return h, st
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
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	return rr
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
	if s, _ := st.User("tester").LessonStatus("01"); s != "in_progress" {
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

func TestUsersEndpoints(t *testing.T) {
	h, _ := newTestAPI(t)
	if rr := doAs(h, "", "POST", "/api/users", `{"name":"  Оля  "}`); rr.Code != 200 {
		t.Fatalf("create: %d %s", rr.Code, rr.Body)
	}
	rr := doAs(h, "", "GET", "/api/users", "")
	got := decodeBody[struct {
		Users []string `json:"users"`
	}](t, rr)
	if len(got.Users) != 2 || got.Users[1] != "Оля" {
		t.Errorf("users = %v", got.Users)
	}
	// the new account works and starts empty
	if rr := doAs(h, "Оля", "GET", "/api/progress", ""); rr.Code != 200 {
		t.Errorf("new account progress = %d", rr.Code)
	}
}

func TestAccountsAreIsolatedOverAPI(t *testing.T) {
	h, st := newTestAPI(t)
	st.EnsureUser("Оля")
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
	st.EnsureUser("Оля")
	do(h, "POST", "/api/lessons/01/complete", "")            // tester: 1 lesson
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
