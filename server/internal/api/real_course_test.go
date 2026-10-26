package api

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/checker"
	"github.com/grisha/serbian-app/server/internal/config"
	"github.com/grisha/serbian-app/server/internal/content"
	"github.com/grisha/serbian-app/server/internal/economy"
	"github.com/grisha/serbian-app/server/internal/store"
)

// TestRealCourseAcceptsItsOwnAnswers walks every exercise of the real course
// through the API: the exercise has to be served, and its canonical answer has to
// be accepted by the checker. It catches what the content guards cannot: an
// accepted answer that the normaliser rejects (punctuation, diacritics), a match
// whose pairs do not round-trip, a dialogue line out of step with its exercise.
func TestRealCourseAcceptsItsOwnAnswers(t *testing.T) {
	c, err := content.Load("../../../content")
	if err != nil {
		t.Skipf("real content tree not available: %v", err)
	}
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if _, err := st.EnsureUserByName("tester"); err != nil {
		t.Fatal(err)
	}
	h := Handler(Deps{
		Course: func() *content.Course { return c },
		Store:  st,
		Now:    func() time.Time { return fixedNow },
		Stale:  func() bool { return false },
		Config: config.Config{AppBaseURL: testBaseURL},
	})

	checked := 0
	for _, ph := range c.Phases {
		for _, lid := range ph.Lessons {
			l := c.Lessons[lid]
			if l == nil || l.Planned {
				continue
			}
			rr := do(h, "GET", "/api/lessons/"+lid+"/exercises", "")
			if rr.Code != 200 {
				t.Errorf("lesson %s: exercises = %d", lid, rr.Code)
				continue
			}
			var blocks []exerciseBlockDTO
			if err := json.Unmarshal(rr.Body.Bytes(), &blocks); err != nil {
				t.Errorf("lesson %s: %v", lid, err)
				continue
			}
			served := map[string]bool{}
			for _, b := range blocks {
				for _, e := range b.Exercises {
					served[e.ID] = true
				}
			}
			for _, b := range c.Exercises[lid] {
				for _, e := range b.Exercises {
					if !served[e.ID] {
						t.Errorf("lesson %s: exercise %s is not served", lid, e.ID)
						continue
					}
					body := canonicalAnswer(e)
					res := decodeBody[checkResultDTO](t, do(h, "POST", "/api/lessons/"+lid+"/exercises/"+e.ID+"/check", body))
					if !res.OK {
						t.Errorf("lesson %s: %s (%s) refused its own answer %s: %+v", lid, e.ID, e.Type, body, res)
					}
					checked++
				}
			}
		}
	}
	if checked < 500 {
		t.Errorf("only %d exercises were walked, the course has more", checked)
	}
}

// canonicalAnswer is the request body that answers e correctly.
func canonicalAnswer(e content.Exercise) string {
	j := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	switch e.Type {
	case "choice":
		return j(map[string]any{"answer": e.Answer})
	case "match":
		pairs := map[string]string{}
		for _, p := range e.Pairs {
			pairs[p[0]] = p[1]
		}
		return j(map[string]any{"pairs": pairs})
	case "conjugate":
		var rows []string
		for _, r := range e.AcceptForms {
			rows = append(rows, r[0])
		}
		return j(map[string]any{"answers": rows})
	default:
		return j(map[string]any{"answer": e.Accept[0]})
	}
}

// TestRealCourseSecondLevelIsThePaywall walks the real course through the economy
// the way a learner meets it: Level 1 is open, every level after it is locked at
// its price, and buying Level 2 with the зёрнышки that Level 1 pays opens it and
// nothing else. The product refs, the course map and the lock live in three
// places; this is the test that notices when they stop agreeing.
func TestRealCourseSecondLevelIsThePaywall(t *testing.T) {
	c, err := content.Load("../../../content")
	if err != nil {
		t.Skipf("real content tree not available: %v", err)
	}
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	var phases []economy.PhaseLessons
	for _, ph := range c.Phases {
		phases = append(phases, economy.PhaseLessons{ID: ph.ID, Lessons: ph.Lessons})
	}
	if err := st.SeedEconomyDefaults(phases, fixedNow); err != nil {
		t.Fatal(err)
	}
	row, err := st.EnsureUserByName("tester")
	if err != nil {
		t.Fatal(err)
	}
	h := Handler(Deps{
		Course: func() *content.Course { return c },
		Store:  st,
		Now:    func() time.Time { return fixedNow },
		Stale:  func() bool { return false },
		Config: config.Config{AppBaseURL: testBaseURL},
	})
	cookie := authed(t, st, row.ID)

	type phaseInfo struct {
		ID     string `json:"id"`
		Locked bool   `json:"locked"`
		Price  int64  `json:"price"`
	}
	course := func() map[string]phaseInfo {
		t.Helper()
		res := doCookie(h, cookie, "GET", "/api/course", "")
		if res.Code != 200 {
			t.Fatalf("GET /api/course = %d", res.Code)
		}
		var out struct {
			Phases []phaseInfo `json:"phases"`
		}
		if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		m := map[string]phaseInfo{}
		for _, p := range out.Phases {
			m[p.ID] = p
		}
		return m
	}
	status := func(lesson string) int {
		return doCookie(h, cookie, "GET", "/api/lessons/"+lesson, "").Code
	}

	m := course()
	if m["1"].Locked || m["1"].Price != 0 {
		t.Errorf("Level 1 = %+v, want open and free", m["1"])
	}
	for id, price := range map[string]int64{"1b": 40, "2": 150, "3": 260, "4": 370, "5": 480} {
		if !m[id].Locked || m[id].Price != price {
			t.Errorf("phase %s = %+v, want locked at %d", id, m[id], price)
		}
	}
	if status("00") != 200 || status("14") != 200 {
		t.Errorf("Level 1 lessons are not open: 00 = %d, 14 = %d", status("00"), status("14"))
	}
	for _, lesson := range []string{"06", "10c", "15", "16", "30", "42", "60"} {
		if got := status(lesson); got != 403 {
			t.Errorf("GET /api/lessons/%s = %d before buying, want 403", lesson, got)
		}
	}

	// Level 2 is for sale at less than the lessons of Level 1 pay.
	products, err := st.ListProducts(true)
	if err != nil {
		t.Fatal(err)
	}
	var level2 economy.Product
	for _, p := range products {
		if p.Kind == economy.ProductPhaseUnlock && p.Ref == "1b" {
			level2 = p
		}
	}
	body := `{"product_id":` + strconv.FormatInt(level2.ID, 10) + `}`
	if res := doCookie(h, cookie, "POST", "/api/me/purchases", body); res.Code != 409 {
		t.Fatalf("buying Level 2 with nothing earned = %d, want 409", res.Code)
	}
	creditUser(t, st, row.ID, 40, "paywall:level1")
	if res := doCookie(h, cookie, "POST", "/api/me/purchases", body); res.Code != 200 {
		t.Fatalf("buying Level 2 with 40 = %d: %s", res.Code, res.Body.String())
	}

	m = course()
	if m["1b"].Locked {
		t.Error("Level 2 is still locked after buying it")
	}
	if status("06") != 200 || status("15") != 200 {
		t.Errorf("Level 2 lessons are not open after buying it: 06 = %d, 15 = %d", status("06"), status("15"))
	}
	for _, id := range []string{"2", "3", "4", "5"} {
		if !m[id].Locked {
			t.Errorf("phase %s opened by buying Level 2", id)
		}
	}
	if got := status("16"); got != 403 {
		t.Errorf("GET /api/lessons/16 = %d after buying Level 2, want 403", got)
	}
}

// A fix_error exercise starts with the wrong sentence already in the box. Handing it in unchanged
// must never pass: since one letter of difference is forgiven (checker/tolerance.go), `Imam brat.`
// would otherwise count as a typo of `Imam brata.` and the exercise would teach nothing.
func TestRealCourseFixErrorDoesNotPassUnchanged(t *testing.T) {
	c, err := content.Load("../../../content")
	if err != nil {
		t.Skipf("real content tree not available: %v", err)
	}
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if _, err := st.EnsureUserByName("tester"); err != nil {
		t.Fatal(err)
	}
	h := Handler(Deps{
		Course: func() *content.Course { return c },
		Store:  st,
		Now:    func() time.Time { return fixedNow },
		Stale:  func() bool { return false },
		Config: config.Config{AppBaseURL: testBaseURL},
	})

	checked := 0
	for lesson, blocks := range c.Exercises {
		for _, b := range blocks {
			for _, e := range b.Exercises {
				// With Russian text in the prompt the box starts empty, so there is nothing to hand in.
				if e.Type != "fix_error" || regexp.MustCompile(`[А-Яа-яЁё]`).MatchString(e.Prompt) {
					continue
				}
				body, _ := json.Marshal(map[string]string{"answer": e.Prompt})
				rr := do(h, "POST", "/api/lessons/"+lesson+"/exercises/"+e.ID+"/check", string(body))
				var res struct {
					OK bool `json:"ok"`
				}
				if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
					t.Fatalf("%s: %v (%s)", e.ID, err, rr.Body)
				}
				if res.OK {
					t.Errorf("%s: the unchanged wrong sentence %q passes", e.ID, e.Prompt)
				}
				checked++
			}
		}
	}
	if checked == 0 {
		t.Fatal("no fix_error exercises found: the test checks nothing")
	}
}

// A word_bank answer is built from tiles: swapping one tile for a lookalike (`sin` for `sina`) is a wrong
// choice, not a typo, so none of the bank's one-letter distractors may pass.
func TestRealCourseWordBankLookalikeTilesStayWrong(t *testing.T) {
	c, err := content.Load("../../../content")
	if err != nil {
		t.Skipf("real content tree not available: %v", err)
	}
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if _, err := st.EnsureUserByName("tester"); err != nil {
		t.Fatal(err)
	}
	h := Handler(Deps{
		Course: func() *content.Course { return c },
		Store:  st,
		Now:    func() time.Time { return fixedNow },
		Stale:  func() bool { return false },
		Config: config.Config{AppBaseURL: testBaseURL},
	})

	oneLetter := func(a, b string) bool { // a and b differ by exactly one substituted, added or dropped letter
		x, y := []rune(strings.ToLower(a)), []rune(strings.ToLower(b))
		if len(x) > len(y) {
			x, y = y, x
		}
		if len(y)-len(x) > 1 || len(y) < 4 {
			return false
		}
		if len(x) == len(y) {
			d := 0
			for i := range x {
				if x[i] != y[i] {
					d++
				}
			}
			return d == 1
		}
		for i := range y {
			if string(append(append([]rune{}, y[:i]...), y[i+1:]...)) == string(x) {
				return true
			}
		}
		return false
	}

	checked := 0
	for lesson, blocks := range c.Exercises {
		for _, b := range blocks {
			for _, e := range b.Exercises {
				if e.Type != "word_bank" {
					continue
				}
				accepted := map[string]bool{}
				for _, a := range e.Accept {
					accepted[checker.Normalize(a)] = true
				}
				for _, a := range e.Accept {
					words := strings.Fields(a)
					for i, w := range words {
						for _, tile := range e.Bank {
							if tile == w || !oneLetter(tile, w) {
								continue
							}
							swapped := append(append([]string{}, words[:i]...), append([]string{tile}, words[i+1:]...)...)
							ans := strings.Join(swapped, " ")
							if accepted[checker.Normalize(ans)] {
								continue // another accepted variant
							}
							body, _ := json.Marshal(map[string]string{"answer": ans})
							rr := do(h, "POST", "/api/lessons/"+lesson+"/exercises/"+e.ID+"/check", string(body))
							var res struct {
								OK bool `json:"ok"`
							}
							if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
								t.Fatalf("%s: %v", e.ID, err)
							}
							if res.OK {
								t.Errorf("%s: %q passes with the lookalike tile %q for %q", e.ID, ans, tile, w)
							}
							checked++
						}
					}
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no lookalike tiles found: the test checks nothing")
	}
	t.Logf("%d lookalike-tile answers stayed wrong", checked)
}
