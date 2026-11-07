package api

import (
	"errors"
	"log"
	"net/http"

	"github.com/grisha/serbian-app/server/internal/economy"
	"github.com/grisha/serbian-app/server/internal/store"
)

// onboardingQuest is the visible «Знакомство с приложением» quest and whether this learner has taken its reward. A
// missing quest (the admin switched it off) is a zero value: the tour then pays nothing but still works.
func (h handlers) onboardingQuest(userID string) (q economy.Quest, claimed bool, err error) {
	quests, err := h.visibleQuests()
	if err != nil {
		return q, false, err
	}
	for _, c := range quests {
		if c.Kind == economy.QuestOnboardingDone {
			q = c
			break
		}
	}
	if q.ID == 0 {
		return q, false, nil
	}
	done, err := h.Store.User(userID).ClaimedQuestIDs()
	if err != nil {
		return q, false, err
	}
	return q, done[q.ID], nil
}

// getOnboarding handles GET /api/me/onboarding: {due, reward}. due is true when the tour should start by itself (the
// learner has finished a lesson and has not answered it yet); reward is what finishing it pays right now, 0 once the
// seeds are taken. Nothing is recorded: only the answers below are.
func (h handlers) getOnboarding(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	due, err := h.Store.OnboardingDue(ac.UserID)
	if err != nil {
		log.Printf("onboarding due: %v", err)
		writeJSON(w, http.StatusOK, map[string]any{"due": false, "reward": 0})
		return
	}
	var reward int64
	if q, claimed, err := h.onboardingQuest(ac.UserID); err != nil {
		log.Printf("onboarding quest: %v", err)
	} else if !claimed {
		reward = q.Reward
	}
	writeJSON(w, http.StatusOK, map[string]any{"due": due, "reward": reward})
}

// answerOnboarding handles POST /api/me/onboarding {action: "postpone" | "done"}. "done" also pays the quest's reward
// once: the row is written first, so if paying fails the quest is simply claimable in «Задания» instead of lost.
func (h handlers) answerOnboarding(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	var body struct {
		Action string `json:"action"`
	}
	if err := decode(r, &body); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	switch body.Action {
	case "postpone":
		if err := h.Store.SetOnboarding(ac.UserID, store.OnboardingPostponed, h.Now()); err != nil {
			log.Printf("onboarding postpone: %v", err)
			fail(w, http.StatusInternalServerError, "could not save")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"reward": 0})
	case "done":
		if err := h.Store.SetOnboarding(ac.UserID, store.OnboardingDone, h.Now()); err != nil {
			log.Printf("onboarding done: %v", err)
			fail(w, http.StatusInternalServerError, "could not save")
			return
		}
		var paid int64
		q, claimed, err := h.onboardingQuest(ac.UserID)
		if err != nil {
			log.Printf("onboarding quest: %v", err)
		} else if q.ID != 0 && !claimed {
			switch err := h.Store.User(ac.UserID).ClaimQuest(q, h.Now()); {
			case err == nil:
				paid = q.Reward
			case errors.Is(err, store.ErrAlreadyClaimed):
			default:
				log.Printf("onboarding claim: %v", err)
			}
		}
		bal, _ := h.Store.Balance(ac.UserID)
		writeJSON(w, http.StatusOK, map[string]any{"reward": paid, "balance": bal})
	default:
		fail(w, http.StatusBadRequest, "unknown action")
	}
}
