package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grisha/serbian-app/server/internal/store"
)

const storiesToken = "story-token"

// storyBody is a valid story document for the service endpoint.
func storyBody(tier string, demo bool, status string) string {
	doc := map[string]any{
		"title_sr": "Drago mi je", "title_ru": "Приятно познакомиться", "level": "1", "tier": tier,
		"is_demo": demo, "status": status, "sort_order": 1,
		"content": map[string]any{
			"fast_rate": 1.25,
			"chapters": []any{map[string]any{
				"scenes": []any{map[string]any{"id": 1, "image": "/story-media/x/f1.webp"}},
				"sentences": []any{
					map[string]any{"n": 1, "who": "Marko", "scene": 1, "sr": "Zdravo!", "ru": "Привет!",
						"sr_audio": "/story-media/x/s01_sr.mp3", "sr_ms": 760, "sr_words": []any{map[string]any{"w": "Zdravo", "start_ms": 100, "end_ms": 700}},
						"ru_audio": "/story-media/x/s01_ru.mp3", "ru_ms": 970, "ru_words": []any{map[string]any{"w": "Привет", "start_ms": 100, "end_ms": 900}}},
					map[string]any{"n": 2, "who": "Ana", "scene": 1, "sr": "Ćao!", "ru": "Пока!",
						"sr_audio": "/story-media/x/s02_sr.mp3", "sr_ms": 600, "sr_words": []any{},
						"ru_audio": "/story-media/x/s02_ru.mp3", "ru_ms": 850, "ru_words": []any{}},
				},
			}},
			"quiz_inline": []any{map[string]any{"after": 1, "hint": "привет", "options": []string{"zdravo", "hvala"}, "answer": "zdravo"}},
			"quiz_final":  []any{map[string]any{"kind": "plot", "prompt": "Кто поздоровался?", "options": []string{"Марко", "Петар"}, "answer": "Марко"}},
		},
	}
	b, _ := json.Marshal(doc)
	return string(b)
}

func storiesAPI(t *testing.T) (http.Handler, *store.Store, string, *http.Cookie) {
	t.Helper()
	h, st := newTestAPIWith(t, func(d *Deps) { d.Config.StoriesServiceToken = storiesToken })
	id := testUserID(t, st)
	return h, st, id, authed(t, st, id)
}

func putStory(h http.Handler, slug, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("PUT", "/api/service/stories/"+slug, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestStoryServiceNeedsToken(t *testing.T) {
	h, _, _, _ := storiesAPI(t)
	if rr := putStory(h, "s", storyBody("short", false, "published"), ""); rr.Code != http.StatusUnauthorized {
		t.Fatalf("no token = %d, want 401", rr.Code)
	}
	if rr := putStory(h, "s", storyBody("short", false, "published"), "wrong"); rr.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token = %d, want 401", rr.Code)
	}
	if rr := putStory(h, "s", storyBody("short", false, "published"), storiesToken); rr.Code != 200 {
		t.Fatalf("good token = %d %s", rr.Code, rr.Body)
	}
	off, _ := newTestAPIWith(t, nil)
	if rr := putStory(off, "s", storyBody("short", false, "published"), storiesToken); rr.Code != http.StatusNotFound {
		t.Fatalf("endpoint with no token configured = %d, want 404", rr.Code)
	}
}

func TestStoryServiceRefusesBrokenDocuments(t *testing.T) {
	h, _, _, _ := storiesAPI(t)
	for name, mutate := range map[string]func(string) string{
		"unknown tier":      func(b string) string { return strings.Replace(b, `"tier":"short"`, `"tier":"huge"`, 1) },
		"answer not option": func(b string) string { return strings.Replace(b, `"answer":"zdravo"`, `"answer":"nema"`, 1) },
		"missing scene": func(b string) string {
			return strings.Replace(b, `"scene":1,"sr":"Zdravo!"`, `"scene":9,"sr":"Zdravo!"`, 1)
		},
		"quiz after nothing": func(b string) string {
			return strings.Replace(b, `"after":1`, `"after":7`, 1)
		},
		"first pass slowed too much": func(b string) string {
			return strings.Replace(b, `"fast_rate":1.25`, `"fast_rate":1.25,"slow_rate":0.3`, 1)
		},
		"first pass sped up": func(b string) string {
			return strings.Replace(b, `"fast_rate":1.25`, `"fast_rate":1.25,"slow_rate":1.4`, 1)
		},
	} {
		body := mutate(storyBody("short", false, "published"))
		if rr := putStory(h, "s", body, storiesToken); rr.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400 (%s)", name, rr.Code, rr.Body)
		}
	}
}

func TestFirstPassRateIsKeptAndDefaultsToNormalSpeed(t *testing.T) {
	h, _, _, c := storiesAPI(t)
	slowed := strings.Replace(storyBody("short", true, "published"), `"fast_rate":1.25`, `"fast_rate":1.25,"slow_rate":0.85`, 1)
	if rr := putStory(h, "slow", slowed, storiesToken); rr.Code != 200 {
		t.Fatalf("put slow = %d %s", rr.Code, rr.Body)
	}
	putStory(h, "plain", storyBody("short", true, "published"), storiesToken)
	// The third pass may be played at normal speed too: fast_rate 1 is the edge of the allowed range.
	level := strings.Replace(storyBody("short", true, "published"), `"fast_rate":1.25`, `"fast_rate":1,"slow_rate":0.9`, 1)
	if rr := putStory(h, "level", level, storiesToken); rr.Code != 200 {
		t.Fatalf("put fast_rate 1 = %d %s", rr.Code, rr.Body)
	}
	for slug, want := range map[string]string{"slow": `"slow_rate":0.85`, "plain": `"slow_rate":1`, "level": `"fast_rate":1,`} {
		res := doCookie(h, c, "GET", "/api/stories/"+slug, "")
		if res.Code != 200 || !strings.Contains(res.Body.String(), want) {
			t.Fatalf("%s: %d, want %s in %s", slug, res.Code, want, res.Body)
		}
	}
}

func TestStoriesListShowsDemoOpenAndPaidLockedHidesDrafts(t *testing.T) {
	h, _, _, c := storiesAPI(t)
	putStory(h, "demo", storyBody("short", true, "published"), storiesToken)
	putStory(h, "paid", storyBody("medium", false, "published"), storiesToken)
	putStory(h, "draft", storyBody("short", false, "draft"), storiesToken)

	res := doCookie(h, c, "GET", "/api/stories", "")
	if res.Code != 200 {
		t.Fatalf("list = %d %s", res.Code, res.Body)
	}
	out := decodeBody[struct{ Items []storyListItemDTO }](t, res)
	if len(out.Items) != 2 {
		t.Fatalf("items = %d, want 2 (draft hidden): %+v", len(out.Items), out.Items)
	}
	demo, paid := out.Items[0], out.Items[1]
	if demo.Slug != "demo" || !demo.Owned || demo.Price != 0 || demo.Reward != 0 {
		t.Errorf("demo = %+v", demo)
	}
	if demo.Phrases == 0 {
		t.Errorf("demo lists no phrases: %+v", demo)
	}
	if paid.Slug != "paid" || paid.Owned || paid.Price != store.StoryPrice["medium"] {
		t.Errorf("paid = %+v", paid)
	}
}

func TestPaidStoryIsLockedUntilBoughtAndAnswersStayHidden(t *testing.T) {
	h, st, id, c := storiesAPI(t)
	putStory(h, "paid", storyBody("short", false, "published"), storiesToken)

	res := doCookie(h, c, "GET", "/api/stories/paid", "")
	if res.Code != http.StatusForbidden || !strings.Contains(res.Body.String(), `"price":10`) {
		t.Fatalf("locked story = %d %s, want 403 with the price", res.Code, res.Body)
	}
	if res := doCookie(h, c, "POST", "/api/stories/paid/buy", ""); res.Code != http.StatusConflict {
		t.Fatalf("buy without money = %d, want 409", res.Code)
	}
	creditUser(t, st, id, 25, "story-test")
	if res := doCookie(h, c, "POST", "/api/stories/paid/buy", ""); res.Code != 200 {
		t.Fatalf("buy = %d %s", res.Code, res.Body)
	}
	if bal, _ := st.Balance(id); bal != 15 {
		t.Fatalf("balance after buying = %d, want 15", bal)
	}
	if res := doCookie(h, c, "POST", "/api/stories/paid/buy", ""); res.Code != http.StatusConflict || !strings.Contains(res.Body.String(), "already_owned") {
		t.Fatalf("second buy = %d %s, want already_owned", res.Code, res.Body)
	}
	res = doCookie(h, c, "GET", "/api/stories/paid", "")
	if res.Code != 200 {
		t.Fatalf("owned story = %d %s", res.Code, res.Body)
	}
	if strings.Contains(res.Body.String(), `"answer"`) {
		t.Fatalf("answers leaked to the client: %s", res.Body)
	}
}

// A story that sets its own price is sold for it, whatever its tier says (decision of 2026-11-03): the list, the locked
// answer and the shop product (what buying charges) all agree.
func TestStoryOwnPriceBeatsTheTierPrice(t *testing.T) {
	h, st, id, c := storiesAPI(t)
	var doc map[string]any
	if err := json.Unmarshal([]byte(storyBody("medium", false, "published")), &doc); err != nil {
		t.Fatal(err)
	}
	doc["content"].(map[string]any)["price"] = 50
	b, _ := json.Marshal(doc)
	putStory(h, "pricey", string(b), storiesToken)

	out := decodeBody[struct{ Items []storyListItemDTO }](t, doCookie(h, c, "GET", "/api/stories", ""))
	if len(out.Items) != 1 || out.Items[0].Price != 50 {
		t.Fatalf("list = %+v, want the story's own price 50", out.Items)
	}
	if res := doCookie(h, c, "GET", "/api/stories/pricey", ""); res.Code != http.StatusForbidden || !strings.Contains(res.Body.String(), `"price":50`) {
		t.Fatalf("locked story = %d %s, want 403 with price 50", res.Code, res.Body)
	}
	creditUser(t, st, id, 40, "story-test")
	if res := doCookie(h, c, "POST", "/api/stories/pricey/buy", ""); res.Code != http.StatusConflict {
		t.Fatalf("buy with 40 = %d, want 409 (it costs 50, not the tier's 20)", res.Code)
	}
	creditUser(t, st, id, 20, "story-test-2")
	if res := doCookie(h, c, "POST", "/api/stories/pricey/buy", ""); res.Code != 200 {
		t.Fatalf("buy with 60 = %d %s", res.Code, res.Body)
	}
	if bal, _ := st.Balance(id); bal != 10 {
		t.Fatalf("balance after buying = %d, want 10", bal)
	}
}

func TestQuickQuestionMaskDefaultsToTheAnswerAndMustBeInTheSentence(t *testing.T) {
	h, _, _, c := storiesAPI(t)
	putStory(h, "demo", storyBody("short", true, "published"), storiesToken)
	res := doCookie(h, c, "GET", "/api/stories/demo", "")
	if !strings.Contains(res.Body.String(), `"mask":"zdravo"`) {
		t.Fatalf("mask not sent to the player: %s", res.Body)
	}
	bad := strings.Replace(storyBody("short", true, "published"), `"mask":"zdravo"`, `"mask":"hvala"`, 1)
	bad = strings.Replace(bad, `"answer":"zdravo"`, `"answer":"zdravo","mask":"hvala"`, 1)
	if rr := putStory(h, "demo", bad, storiesToken); rr.Code != http.StatusBadRequest {
		t.Fatalf("mask outside the sentence = %d, want 400", rr.Code)
	}
}

func TestStoryAnswerChecksOnTheServer(t *testing.T) {
	h, _, _, c := storiesAPI(t)
	putStory(h, "demo", storyBody("short", true, "published"), storiesToken)
	type verdict struct {
		Correct bool   `json:"correct"`
		Answer  string `json:"answer"`
	}
	post := func(body string) *httptest.ResponseRecorder {
		return doCookie(h, c, "POST", "/api/stories/demo/answer", body)
	}
	if v := decodeBody[verdict](t, post(`{"kind":"inline","index":0,"choice":"zdravo"}`)); !v.Correct || v.Answer != "zdravo" {
		t.Errorf("right answer = %+v", v)
	}
	if v := decodeBody[verdict](t, post(`{"kind":"final","index":0,"choice":"Петар"}`)); v.Correct || v.Answer != "Марко" {
		t.Errorf("wrong answer = %+v", v)
	}
	if res := post(`{"kind":"final","index":5,"choice":"x"}`); res.Code != http.StatusBadRequest {
		t.Errorf("no such question = %d, want 400", res.Code)
	}
}

func TestStoryPaysNothing(t *testing.T) {
	h, st, id, c := storiesAPI(t)
	putStory(h, "demo", storyBody("short", true, "published"), storiesToken)
	finish := func() int64 {
		res := doCookie(h, c, "POST", "/api/stories/demo/progress", `{"pass":3,"completed":true,"score":3,"total":4}`)
		if res.Code != 200 {
			t.Fatalf("progress = %d %s", res.Code, res.Body)
		}
		return decodeBody[struct{ Reward int64 }](t, res).Reward
	}
	for i := 1; i <= 2; i++ {
		if got := finish(); got != 0 {
			t.Fatalf("finish %d paid %d, want 0", i, got)
		}
	}
	if bal, _ := st.Balance(id); bal != 0 {
		t.Fatalf("balance = %d, want 0", bal)
	}
	list := decodeBody[struct{ Items []storyListItemDTO }](t, doCookie(h, c, "GET", "/api/stories", ""))
	if p := list.Items[0]; !p.Completed || p.PassReached != 3 || p.QuizScore != 3 || p.QuizTotal != 4 {
		t.Errorf("progress in list = %+v", p)
	}
	if res := doCookie(h, c, "POST", "/api/stories/demo/progress", `{"pass":9}`); res.Code != http.StatusBadRequest {
		t.Errorf("pass 9 = %d, want 400", res.Code)
	}
}

func TestStoryEventsAndShopSeparation(t *testing.T) {
	h, _, _, c := storiesAPI(t)
	putStory(h, "demo", storyBody("short", true, "published"), storiesToken)
	putStory(h, "paid", storyBody("short", false, "published"), storiesToken)
	if res := doCookie(h, c, "POST", "/api/stories/demo/events", `{"kind":"pass_end","detail":"1"}`); res.Code != http.StatusNoContent {
		t.Errorf("event = %d, want 204", res.Code)
	}
	if res := doCookie(h, c, "POST", "/api/stories/demo/events", `{"kind":"hack"}`); res.Code != http.StatusBadRequest {
		t.Errorf("unknown event = %d, want 400", res.Code)
	}
	shop := doCookie(h, c, "GET", "/api/shop", "")
	if strings.Contains(shop.Body.String(), "story_unlock") {
		t.Errorf("story products leaked into the shop: %s", shop.Body)
	}
}

// allKindsBody is a story with one question of every final kind, over the two sentences of the fixture.
func allKindsBody() string {
	var doc map[string]any
	_ = json.Unmarshal([]byte(storyBody("short", true, "published")), &doc)
	doc["content"].(map[string]any)["quiz_final"] = []any{
		map[string]any{"kind": "plot", "prompt": "Кто поздоровался?", "options": []string{"Марко", "Петар"}, "answer": "Марко"},
		map[string]any{"kind": "listen", "prompt": "Что вы услышали?", "sentence": 1, "options": []string{"Привет!", "Пока!"}, "answer": "Привет!"},
		map[string]any{"kind": "who", "prompt": "Кто это сказал?", "sentence": 2, "options": []string{"Ана", "Марко"}, "answer": "Ана"},
		map[string]any{"kind": "blank", "sentence": 1, "mask": "Zdravo", "options": []string{"Zdravo", "Hvala"}},
		map[string]any{"kind": "bank", "sentence": 1, "extra": []string{"ja", "sam"}},
		map[string]any{"kind": "match", "pairs": [][2]string{{"Zdravo", "Привет"}, {"Ćao", "Пока"}, {"Hvala", "Спасибо"}}},
	}
	b, _ := json.Marshal(doc)
	return string(b)
}

func TestFinalQuestionKindsAreBuiltForThePlayerWithoutAnswers(t *testing.T) {
	h, _, _, c := storiesAPI(t)
	if rr := putStory(h, "demo", allKindsBody(), storiesToken); rr.Code != 200 {
		t.Fatalf("put = %d %s", rr.Code, rr.Body)
	}
	res := doCookie(h, c, "GET", "/api/stories/demo", "")
	var out struct {
		Content struct {
			QuizFinal []store.StoryFinalQuiz `json:"quiz_final"`
		} `json:"content"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	q := out.Content.QuizFinal
	if len(q) != 6 {
		t.Fatalf("questions = %d, want 6", len(q))
	}
	for i, x := range q {
		if x.Answer != "" || x.Mask != "" || len(x.Pairs) != 0 || len(x.Extra) != 0 {
			t.Errorf("question %d leaks its answer: %+v", i, x)
		}
	}
	if q[1].Audio != "/story-media/x/s01_sr.mp3" || q[2].Audio != "/story-media/x/s02_sr.mp3" {
		t.Errorf("listening clips = %q, %q", q[1].Audio, q[2].Audio)
	}
	// A question about a sentence says that sentence after the answer; one about the story or a match does not.
	if q[1].Say != "/story-media/x/s01_sr.mp3" || q[1].SayMS != 760 || q[3].Say == "" || q[4].Say == "" {
		t.Errorf("say clips = %+v / %+v / %+v", q[1], q[3], q[4])
	}
	if q[0].Say != "" || q[5].Say != "" {
		t.Errorf("plot and match say nothing: %q, %q", q[0].Say, q[5].Say)
	}
	if q[3].Prompt != "_____!" {
		t.Errorf("the gap prompt is built from the sentence: %q", q[3].Prompt)
	}
	if got := strings.Join(q[4].Bank, ","); got != "ja,sam,Zdravo" {
		t.Errorf("tiles = %q, want the word and the two extras, sorted", got)
	}
	if q[4].Prompt != "Соберите по-сербски: «Привет!»" {
		t.Errorf("bank prompt = %q", q[4].Prompt)
	}
	if len(q[5].Left) != 3 || len(q[5].Right) != 3 || strings.Join(q[5].Right, ",") != "Пока,Привет,Спасибо" {
		t.Errorf("match columns = %v | %v", q[5].Left, q[5].Right)
	}
}

func TestFinalQuestionKindsAreChecked(t *testing.T) {
	h, _, _, c := storiesAPI(t)
	putStory(h, "demo", allKindsBody(), storiesToken)
	type verdict struct {
		Correct bool       `json:"correct"`
		Answer  string     `json:"answer"`
		Pairs   [][]string `json:"pairs"`
	}
	ask := func(body string) verdict {
		res := doCookie(h, c, "POST", "/api/stories/demo/answer", body)
		if res.Code != 200 {
			t.Fatalf("answer %s = %d %s", body, res.Code, res.Body)
		}
		return decodeBody[verdict](t, res)
	}
	if v := ask(`{"kind":"final","index":1,"choice":"Привет!"}`); !v.Correct {
		t.Errorf("listen right = %+v", v)
	}
	if v := ask(`{"kind":"final","index":2,"choice":"Марко"}`); v.Correct || v.Answer != "Ана" {
		t.Errorf("who wrong = %+v", v)
	}
	if v := ask(`{"kind":"final","index":3,"choice":"Zdravo"}`); !v.Correct {
		t.Errorf("blank right = %+v", v)
	}
	// the built sentence is judged without caring for case and punctuation
	if v := ask(`{"kind":"final","index":4,"choice":"zdravo"}`); !v.Correct {
		t.Errorf("bank right = %+v", v)
	}
	if v := ask(`{"kind":"final","index":4,"choice":"ja sam"}`); v.Correct || v.Answer != "Zdravo!" {
		t.Errorf("bank wrong = %+v", v)
	}
	if v := ask(`{"kind":"final","index":5,"pairs":{"Zdravo":"Привет","Ćao":"Пока","Hvala":"Спасибо"}}`); !v.Correct {
		t.Errorf("match right = %+v", v)
	}
	if v := ask(`{"kind":"final","index":5,"pairs":{"Zdravo":"Пока","Ćao":"Привет","Hvala":"Спасибо"}}`); v.Correct || len(v.Pairs) != 3 {
		t.Errorf("match wrong = %+v", v)
	}
	if v := ask(`{"kind":"final","index":5,"pairs":{"Zdravo":"Привет"}}`); v.Correct {
		t.Errorf("an unfinished match is not right: %+v", v)
	}
}

func TestBrokenFinalQuestionsAreRefused(t *testing.T) {
	h, _, _, _ := storiesAPI(t)
	broken := func(mutate func(q []any)) string {
		var doc map[string]any
		_ = json.Unmarshal([]byte(allKindsBody()), &doc)
		mutate(doc["content"].(map[string]any)["quiz_final"].([]any))
		b, _ := json.Marshal(doc)
		return string(b)
	}
	for name, mutate := range map[string]func(q []any){
		"listen about a missing sentence": func(q []any) { q[1].(map[string]any)["sentence"] = 9 },
		"blank word not in the sentence":  func(q []any) { q[3].(map[string]any)["mask"] = "nema" },
		"match with a repeated side": func(q []any) {
			q[5].(map[string]any)["pairs"] = [][2]string{{"Zdravo", "Привет"}, {"Zdravo", "Пока"}}
		},
		"match with one pair":       func(q []any) { q[5].(map[string]any)["pairs"] = [][2]string{{"Zdravo", "Привет"}} },
		"unknown kind":              func(q []any) { q[2].(map[string]any)["kind"] = "riddle" },
		"choice without its answer": func(q []any) { q[0].(map[string]any)["answer"] = "Милан" },
		"who with one option":       func(q []any) { q[2].(map[string]any)["options"] = []string{"Ана"} },
	} {
		if rr := putStory(h, "demo", broken(mutate), storiesToken); rr.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400 (%s)", name, rr.Code, rr.Body)
		}
	}
}
