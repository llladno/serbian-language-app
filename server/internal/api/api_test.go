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
	h := Handler(Deps{
		Course: func() *content.Course { return c },
		Store:  st,
		Now:    func() time.Time { return fixedNow },
		Stale:  func() bool { return false },
	})
	return h, st
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
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
	if st, _ := st.LessonStatus("01"); st != "in_progress" {
		t.Errorf("lesson status = %q, want in_progress", st)
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
