package api

import (
	"crypto/subtle"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/grisha/serbian-app/server/internal/store"
)

type storyListItemDTO struct {
	Slug       string `json:"slug"`
	TitleSR    string `json:"title_sr"`
	TitleRU    string `json:"title_ru"`
	Tier       string `json:"tier"`
	IsDemo     bool   `json:"is_demo"`
	Cover      string `json:"cover"`
	DurationMS int    `json:"duration_ms"`
	Phrases    int    `json:"phrases"`
	// RecommendedLesson is the lesson to have finished before the story ("" = no advice).
	RecommendedLesson string `json:"recommended_lesson"`
	// Difficulty groups the list: 1 easy, 2 medium, 3 harder.
	Difficulty  int   `json:"difficulty"`
	Owned       bool  `json:"owned"`
	Price       int64 `json:"price"`
	Reward      int64 `json:"reward"`
	PassReached int   `json:"pass_reached"`
	Completed   bool  `json:"completed"`
	QuizScore   int   `json:"quiz_score"`
	QuizTotal   int   `json:"quiz_total"`
}

// listStories handles GET /api/stories: the published stories with what the learner owns and has finished.
func (h handlers) listStories(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	rows, err := h.Store.ListStories(true)
	if err != nil {
		log.Printf("stories: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	us := h.Store.User(ac.UserID)
	prog, err := us.StoryProgressAll()
	if err != nil {
		log.Printf("stories: progress: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	out := struct {
		Items []storyListItemDTO `json:"items"`
	}{Items: []storyListItemDTO{}}
	for _, s := range rows {
		owned, err := us.StoryOwned(s)
		if err != nil {
			log.Printf("stories: owned: %v", err)
			fail(w, http.StatusInternalServerError, "internal error")
			return
		}
		p := prog[s.Slug]
		var price int64
		if !s.IsDemo {
			price = s.PriceZ()
		}
		out.Items = append(out.Items, storyListItemDTO{
			Slug: s.Slug, TitleSR: s.TitleSR, TitleRU: s.TitleRU, Tier: s.Tier, IsDemo: s.IsDemo,
			Cover: s.CoverURL, DurationMS: s.Content.FullMS(), Phrases: s.Content.SentenceCount(), RecommendedLesson: s.Content.RecommendedLesson, Difficulty: s.Content.Difficulty, Owned: owned, Price: price,
			Reward:      store.StoryReward(s.Tier, s.IsDemo),
			PassReached: p.PassReached, Completed: p.Completed, QuizScore: p.QuizScore, QuizTotal: p.QuizTotal,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// publishedStory loads the story named in the path. A draft or a missing story is a 404 for a learner.
func (h handlers) publishedStory(w http.ResponseWriter, r *http.Request) (store.StoryRow, bool) {
	s, err := h.Store.StoryBySlug(r.PathValue("slug"))
	if errors.Is(err, store.ErrStoryNotFound) || (err == nil && s.Status != store.StoryStatusPublished) {
		fail(w, http.StatusNotFound, "not found")
		return store.StoryRow{}, false
	}
	if err != nil {
		log.Printf("story: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return store.StoryRow{}, false
	}
	return s, true
}

// ownedStory is publishedStory plus the ownership check: a paid story the learner has not bought is a 403 that
// carries the price, so the client can offer the purchase.
func (h handlers) ownedStory(w http.ResponseWriter, r *http.Request) (store.StoryRow, string, bool) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return store.StoryRow{}, "", false
	}
	s, ok := h.publishedStory(w, r)
	if !ok {
		return store.StoryRow{}, "", false
	}
	owned, err := h.Store.User(ac.UserID).StoryOwned(s)
	if err != nil {
		log.Printf("story: owned: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return store.StoryRow{}, "", false
	}
	if !owned {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "locked", "price": s.PriceZ()})
		return store.StoryRow{}, "", false
	}
	return s, ac.UserID, true
}

// getStory handles GET /api/stories/{slug}: the whole document the player needs, answers removed.
func (h handlers) getStory(w http.ResponseWriter, r *http.Request) {
	s, _, ok := h.ownedStory(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"slug": s.Slug, "title_sr": s.TitleSR, "title_ru": s.TitleRU, "tier": s.Tier, "is_demo": s.IsDemo,
		"duration_ms": s.Content.FullMS(), "reward": store.StoryReward(s.Tier, s.IsDemo),
		"content": s.Content.Public(),
	})
}

// answerStory handles POST /api/stories/{slug}/answer: checks one question and says what the right answer was.
func (h handlers) answerStory(w http.ResponseWriter, r *http.Request) {
	s, _, ok := h.ownedStory(w, r)
	if !ok {
		return
	}
	var req struct {
		Kind   string            `json:"kind"`
		Index  int               `json:"index"`
		Choice string            `json:"choice"`
		Pairs  map[string]string `json:"pairs"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	v, found := s.Content.Check(req.Kind, req.Index, req.Choice, req.Pairs)
	if !found {
		fail(w, http.StatusBadRequest, "no such question")
		return
	}
	out := map[string]any{"correct": v.Correct, "answer": v.Answer}
	if v.Pairs != nil {
		out["pairs"] = v.Pairs
	}
	writeJSON(w, http.StatusOK, out)
}

// storyProgress handles POST /api/stories/{slug}/progress: a finished pass, or the whole story with the score.
func (h handlers) storyProgress(w http.ResponseWriter, r *http.Request) {
	s, userID, ok := h.ownedStory(w, r)
	if !ok {
		return
	}
	var req struct {
		Pass      int  `json:"pass"`
		Completed bool `json:"completed"`
		Score     int  `json:"score"`
		Total     int  `json:"total"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	if req.Pass < 0 || req.Pass > 3 || req.Score < 0 || req.Total < 0 || req.Score > req.Total {
		fail(w, http.StatusBadRequest, "bad progress")
		return
	}
	reward, err := h.Store.User(userID).SaveStoryProgress(s, req.Pass, req.Completed, req.Score, req.Total, h.Now())
	if err != nil {
		log.Printf("story progress: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	bal, _ := h.Store.Balance(userID)
	writeJSON(w, http.StatusOK, map[string]any{"reward": reward, "balance": bal})
}

// storyEvent handles POST /api/stories/{slug}/events: one raw event for the stories metrics. It never fails the
// player: an unknown event is a 400, a storage error is logged and answered with 204.
func (h handlers) storyEvent(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	s, ok := h.publishedStory(w, r)
	if !ok {
		return
	}
	var req struct {
		Kind   string `json:"kind"`
		Detail string `json:"detail"`
	}
	if err := decode(r, &req); err != nil || !store.StoryEventKinds[req.Kind] {
		fail(w, http.StatusBadRequest, "bad event")
		return
	}
	if err := h.Store.RecordStoryEvent(ac.UserID, s.Slug, req.Kind, req.Detail, h.Now()); err != nil {
		log.Printf("story event: %v", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

// buyStory handles POST /api/stories/{slug}/buy: pays for a paid story in зёрнышки through the shop's Purchase.
func (h handlers) buyStory(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	s, ok := h.publishedStory(w, r)
	if !ok {
		return
	}
	if s.IsDemo {
		fail(w, http.StatusBadRequest, "demo is free")
		return
	}
	pid, err := h.Store.StoryProductID(s.Slug)
	if err != nil {
		fail(w, http.StatusNotFound, "product_unavailable")
		return
	}
	res, err := h.Store.User(ac.UserID).Purchase(pid, "", r.Header.Get("Idempotency-Key"), h.Now())
	switch {
	case err == nil:
	case errors.Is(err, store.ErrInsufficientFunds):
		fail(w, http.StatusConflict, "insufficient_funds")
		return
	case errors.Is(err, store.ErrAlreadyOwned):
		fail(w, http.StatusConflict, "already_owned")
		return
	case errors.Is(err, store.ErrProductUnavailable):
		fail(w, http.StatusNotFound, "product_unavailable")
		return
	default:
		log.Printf("buy story: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	bal, _ := h.Store.Balance(ac.UserID)
	writeJSON(w, http.StatusOK, map[string]any{"paid": res.Paid, "balance": bal, "replayed": res.Replayed})
}

// putStoryService handles PUT /api/service/stories/{slug}: how the publishing scripts create or replace a story.
// It is guarded by a bearer token (STORIES_SERVICE_TOKEN), not by a session; without the token configured the
// endpoint does not exist.
func (h handlers) putStoryService(w http.ResponseWriter, r *http.Request) {
	token := h.Config.StoriesServiceToken
	if token == "" {
		fail(w, http.StatusNotFound, "not found")
		return
	}
	got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
		fail(w, http.StatusUnauthorized, "bad token")
		return
	}
	var req struct {
		TitleSR   string             `json:"title_sr"`
		TitleRU   string             `json:"title_ru"`
		Level     string             `json:"level"`
		Tier      string             `json:"tier"`
		IsDemo    bool               `json:"is_demo"`
		Status    string             `json:"status"`
		SortOrder int                `json:"sort_order"`
		Cover     string             `json:"cover_url"`
		Checks    string             `json:"checks"`
		Content   store.StoryContent `json:"content"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	err := h.Store.UpsertStory(store.StoryRow{
		Slug: r.PathValue("slug"), TitleSR: req.TitleSR, TitleRU: req.TitleRU, Level: req.Level, Tier: req.Tier,
		IsDemo: req.IsDemo, Status: req.Status, SortOrder: req.SortOrder, CoverURL: req.Cover,
		Checks: req.Checks, Content: req.Content,
	}, h.Now())
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case errors.Is(err, store.ErrStoryInvalid):
		fail(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("put story: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
	}
}
