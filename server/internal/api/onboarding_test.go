package api

import (
	"net/http"
	"testing"
)

type onboardingState struct {
	Due    bool  `json:"due"`
	Reward int64 `json:"reward"`
}

func TestOnboardingDueOnlyAfterFirstLesson(t *testing.T) {
	h, st, id, c := economyAPI(t)
	seedQuestRow(t, st, "onboarding_done", 1, 5)
	get := func() onboardingState {
		res := doCookie(h, c, "GET", "/api/me/onboarding", "")
		if res.Code != 200 {
			t.Fatalf("GET onboarding = %d %s", res.Code, res.Body)
		}
		return decodeBody[onboardingState](t, res)
	}
	if o := get(); o.Due {
		t.Fatalf("due before any lesson: %+v", o)
	}
	if err := st.User(id).SetLessonStatus("00", "done", fixedNow); err != nil {
		t.Fatal(err)
	}
	if o := get(); !o.Due || o.Reward != 5 {
		t.Fatalf("after first lesson: %+v, want due with reward 5", o)
	}
	// Asking does not answer: a closed tab brings the tour back.
	if o := get(); !o.Due {
		t.Fatalf("second ask: %+v, want still due", o)
	}
}

func TestOnboardingPostponeStopsAutoStartButKeepsReward(t *testing.T) {
	h, st, id, c := economyAPI(t)
	seedQuestRow(t, st, "onboarding_done", 1, 5)
	if err := st.User(id).SetLessonStatus("00", "done", fixedNow); err != nil {
		t.Fatal(err)
	}
	if res := doCookie(h, c, "POST", "/api/me/onboarding", `{"action":"postpone"}`); res.Code != 200 {
		t.Fatalf("postpone = %d %s", res.Code, res.Body)
	}
	res := doCookie(h, c, "GET", "/api/me/onboarding", "")
	if o := decodeBody[onboardingState](t, res); o.Due || o.Reward != 5 {
		t.Fatalf("after postpone: %+v, want not due, reward still 5", o)
	}
	if bal, _ := st.Balance(id); bal != 0 {
		t.Fatalf("balance = %d, want 0", bal)
	}
}

func TestOnboardingDonePaysOnce(t *testing.T) {
	h, st, id, c := economyAPI(t)
	seedQuestRow(t, st, "onboarding_done", 1, 5)
	if err := st.User(id).SetLessonStatus("00", "done", fixedNow); err != nil {
		t.Fatal(err)
	}
	done := func() int64 {
		res := doCookie(h, c, "POST", "/api/me/onboarding", `{"action":"done"}`)
		if res.Code != 200 {
			t.Fatalf("done = %d %s", res.Code, res.Body)
		}
		return decodeBody[struct {
			Reward int64 `json:"reward"`
		}](t, res).Reward
	}
	if got := done(); got != 5 {
		t.Fatalf("first run paid %d, want 5", got)
	}
	if got := done(); got != 0 {
		t.Fatalf("replay paid %d, want 0", got)
	}
	if bal, _ := st.Balance(id); bal != 5 {
		t.Fatalf("balance = %d, want 5", bal)
	}
	// A postpone during a replay must not undo «done», or the quest would un-finish.
	if res := doCookie(h, c, "POST", "/api/me/onboarding", `{"action":"postpone"}`); res.Code != 200 {
		t.Fatalf("postpone after done = %d", res.Code)
	}
	if state, err := st.User(id).QuestCounters(fixedNow); err != nil || !state.OnboardingDone {
		t.Fatalf("OnboardingDone = %v, %v; want true", state.OnboardingDone, err)
	}
	res := doCookie(h, c, "GET", "/api/me/onboarding", "")
	if o := decodeBody[onboardingState](t, res); o.Due || o.Reward != 0 {
		t.Fatalf("after done: %+v, want not due, no reward", o)
	}
}

func TestOnboardingRejectsUnknownAction(t *testing.T) {
	h, _, _, c := economyAPI(t)
	if res := doCookie(h, c, "POST", "/api/me/onboarding", `{"action":"nope"}`); res.Code != http.StatusBadRequest {
		t.Fatalf("unknown action = %d, want 400", res.Code)
	}
}

func TestOnboardingQuestShowsInQuestList(t *testing.T) {
	h, st, _, c := economyAPI(t)
	seedQuestRow(t, st, "onboarding_done", 1, 5)
	res := doCookie(h, c, "GET", "/api/me/quests", "")
	if res.Code != 200 {
		t.Fatalf("quests = %d %s", res.Code, res.Body)
	}
	body := decodeBody[struct {
		Quests []struct {
			Kind   string `json:"kind"`
			Reward int64  `json:"reward"`
			Done   bool   `json:"done"`
		} `json:"quests"`
	}](t, res)
	for _, q := range body.Quests {
		if q.Kind == "onboarding_done" {
			if q.Reward != 5 || q.Done {
				t.Fatalf("quest = %+v, want reward 5, not done", q)
			}
			return
		}
	}
	t.Fatal("onboarding_done quest missing from the list")
}
