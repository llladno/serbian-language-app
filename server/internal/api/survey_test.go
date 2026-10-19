package api

import (
	"net/http"
	"strings"
	"testing"
)

const goodSurvey = `{"rating":4,"likes":["lessons","audio"],"pains":["hard"],"pain_text":" слишком быстро ","purpose":"move","wish":"speaking"}`

func TestSurveyOfferedOnceAfterThirdLesson(t *testing.T) {
	h, st, id, c := economyAPI(t)
	type offer struct {
		Offer  bool `json:"offer"`
		Reward int  `json:"reward"`
	}
	get := func() offer {
		res := doCookie(h, c, "GET", "/api/me/survey", "")
		if res.Code != 200 {
			t.Fatalf("GET survey = %d %s", res.Code, res.Body)
		}
		return decodeBody[offer](t, res)
	}
	for _, l := range []string{"00", "01"} {
		if err := st.User(id).SetLessonStatus(l, "done", fixedNow); err != nil {
			t.Fatal(err)
		}
	}
	if get().Offer {
		t.Fatal("offered after only two lessons")
	}
	if err := st.User(id).SetLessonStatus("02", "done", fixedNow); err != nil {
		t.Fatal(err)
	}
	if o := get(); !o.Offer || o.Reward != 10 {
		t.Fatalf("after third lesson: %+v, want offer with reward 10", o)
	}
	if get().Offer {
		t.Fatal("offered a second time")
	}
}

func TestSurveyAnswerPaysOnce(t *testing.T) {
	h, st, id, c := economyAPI(t)
	res := doCookie(h, c, "POST", "/api/me/survey", goodSurvey)
	if res.Code != 200 {
		t.Fatalf("answer = %d %s", res.Code, res.Body)
	}
	if bal, _ := st.Balance(id); bal != 10 {
		t.Fatalf("balance = %d, want 10", bal)
	}
	if res := doCookie(h, c, "POST", "/api/me/survey", goodSurvey); res.Code != http.StatusConflict {
		t.Fatalf("second answer = %d, want 409", res.Code)
	}
	if bal, _ := st.Balance(id); bal != 10 {
		t.Fatalf("balance after repeat = %d, want 10", bal)
	}
}

func TestSurveyRejectsIncomplete(t *testing.T) {
	h, st, id, c := economyAPI(t)
	for _, body := range []string{
		`{"rating":0,"likes":["lessons"],"pains":["hard"],"purpose":"move","wish":"speaking"}`,
		`{"rating":4,"likes":[],"pains":["hard"],"purpose":"move","wish":"speaking"}`,
		`{"rating":4,"likes":["lessons"],"pains":["bogus"],"purpose":"move","wish":"speaking"}`,
		`{"rating":4,"likes":["lessons"],"pains":["hard"],"purpose":"","wish":"speaking"}`,
		`{"rating":4,"likes":["lessons"],"pains":["hard"],"purpose":"move","wish":"bogus"}`,
	} {
		if res := doCookie(h, c, "POST", "/api/me/survey", body); res.Code != 400 {
			t.Errorf("%s = %d, want 400", strings.TrimSpace(body), res.Code)
		}
	}
	if bal, _ := st.Balance(id); bal != 0 {
		t.Fatalf("balance = %d, want 0", bal)
	}
}

func TestSurveyDismissPaysNothingAndStopsOffers(t *testing.T) {
	h, st, id, c := economyAPI(t)
	for _, l := range []string{"00", "01", "02"} {
		if err := st.User(id).SetLessonStatus(l, "done", fixedNow); err != nil {
			t.Fatal(err)
		}
	}
	if res := doCookie(h, c, "POST", "/api/me/survey/dismiss", ""); res.Code != 200 {
		t.Fatalf("dismiss = %d", res.Code)
	}
	res := doCookie(h, c, "GET", "/api/me/survey", "")
	if !strings.Contains(res.Body.String(), `"offer":false`) {
		t.Fatalf("offer after dismiss: %s", res.Body)
	}
	if bal, _ := st.Balance(id); bal != 0 {
		t.Fatalf("balance = %d, want 0", bal)
	}
}
