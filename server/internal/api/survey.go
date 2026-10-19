package api

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/grisha/serbian-app/server/internal/store"
)

// Option codes the survey accepts. The app owns the wording; the server only
// keeps the codes, so the admin panel can label them without parsing prose.
var (
	surveyLikes = map[string]bool{
		"lessons": true, "audio": true, "repeat": true, "seeds": true, "design": true, "other": true,
	}
	surveyPains = map[string]bool{
		"hard": true, "practice": true, "unclear": true, "bugs": true, "price": true, "time": true,
		"nothing": true, "other": true,
	}
	surveyWishes = map[string]bool{
		"speaking": true, "listening": true, "grammar": true, "topics": true, "reminders": true,
		"people": true, "nothing": true, "other": true,
	}
	surveyPurposes = map[string]bool{
		"move": true, "partner": true, "trips": true, "interest": true, "other": true,
	}
)

// getSurvey handles GET /api/me/survey. It answers {offer: true, reward: N}
// exactly once per learner — the first call after their third finished lesson —
// and records that the offer was made, so a closed tab never brings it back.
func (h handlers) getSurvey(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	offer, err := h.Store.SurveyOffer(ac.UserID, h.Now())
	if err != nil {
		log.Printf("survey offer: %v", err)
		writeJSON(w, http.StatusOK, map[string]any{"offer": false})
		return
	}
	reward := 0
	if offer {
		if set, err := h.Store.EconomySettings(); err == nil {
			reward = set.SurveyReward
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"offer": offer, "reward": reward})
}

// answerSurvey handles POST /api/me/survey. Every question must be answered
// for the reward to be paid; the free-text note is the only optional part.
func (h handlers) answerSurvey(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	var a store.SurveyAnswers
	if err := decode(r, &a); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	a.PainText = strings.TrimSpace(a.PainText)
	if len([]rune(a.PainText)) > 1000 {
		a.PainText = string([]rune(a.PainText)[:1000])
	}
	if a.Rating < 1 || a.Rating > 5 || !allIn(a.Likes, surveyLikes) || !allIn(a.Pains, surveyPains) || !surveyPurposes[a.Purpose] || !surveyWishes[a.Wish] {
		fail(w, http.StatusBadRequest, "answer every question")
		return
	}
	set, err := h.Store.EconomySettings()
	if err != nil {
		fail(w, http.StatusInternalServerError, "settings")
		return
	}
	reward := int64(set.SurveyReward)
	if err := h.Store.SaveSurvey(ac.UserID, a, reward, h.Now()); err != nil {
		if errors.Is(err, store.ErrSurveyClosed) {
			fail(w, http.StatusConflict, "already answered")
			return
		}
		log.Printf("save survey: %v", err)
		fail(w, http.StatusInternalServerError, "could not save")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "reward": reward})
}

// dismissSurvey handles POST /api/me/survey/dismiss: closed without answers,
// no reward, never shown again.
func (h handlers) dismissSurvey(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	if err := h.Store.DismissSurvey(ac.UserID, h.Now()); err != nil {
		log.Printf("dismiss survey: %v", err)
		fail(w, http.StatusInternalServerError, "could not save")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// allIn reports whether vals is non-empty and every entry is a known code.
func allIn(vals []string, known map[string]bool) bool {
	if len(vals) == 0 {
		return false
	}
	for _, v := range vals {
		if !known[v] {
			return false
		}
	}
	return true
}
