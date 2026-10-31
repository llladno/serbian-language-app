package api

import (
	"hash/fnv"
	"log"
	"net/http"
	"time"

	"github.com/grisha/serbian-app/server/internal/content"
	"github.com/grisha/serbian-app/server/internal/repeat"
	"github.com/grisha/serbian-app/server/internal/store"
)

// «Повторить»: a batch of exercises the learner has already answered, worst first, one batch a
// day unless a year or a level paid for in rubles lifts the limit.
// See docs/superpowers/specs/2026-10-23-repeat-design.md.

type repeatSessionDTO struct {
	Size      int `json:"size"`
	Remaining int `json:"remaining"`
}

type repeatDTO struct {
	Total     int              `json:"total"`
	Items     []mistakeDTO     `json:"items"`
	Unlimited bool             `json:"unlimited"`
	Session   repeatSessionDTO `json:"session"`
}

// repeatItem is an exercise that can actually be opened: it still exists and its lesson is not locked.
type repeatItem struct {
	Lesson   string
	Exercise content.Exercise
}

type repeatPool struct {
	cands []repeat.Candidate
	byKey map[string]repeatItem
}

func repeatKey(lesson, exerciseID string) string { return lesson + "|" + exerciseID }

// levelOf maps each lesson to the number of its level: the position of its phase in course.yaml,
// counted from 1.
func (h handlers) levelOf() map[string]int {
	out := map[string]int{}
	for i, p := range h.Course().Phases {
		for _, id := range p.Lessons {
			out[id] = i + 1
		}
	}
	return out
}

// repeatPool lists what could go into a batch: every exercise the learner has attempted that still
// exists in an open lesson, described for repeat.Select.
func (h handlers) repeatPool(us *store.UserStore, now time.Time) (repeatPool, error) {
	rows, err := us.RepeatCandidates()
	if err != nil {
		return repeatPool{}, err
	}
	locked, err := h.lockedLessonIDs(us)
	if err != nil {
		return repeatPool{}, err
	}
	levels := h.levelOf()
	loc := us.Location()
	today := now.In(loc).Format("2006-01-02")

	pool := repeatPool{byKey: map[string]repeatItem{}}
	for _, r := range rows {
		if locked[r.Lesson] {
			continue
		}
		ex, _, found := h.findExercise(r.Lesson, r.ExerciseID)
		if !found {
			continue
		}
		key := repeatKey(r.Lesson, r.ExerciseID)
		pool.byKey[key] = repeatItem{Lesson: r.Lesson, Exercise: ex}
		pool.cands = append(pool.cands, repeat.Candidate{
			Key: key, Level: levels[r.Lesson],
			LastCorrect: r.LastCorrect, LastTypo: r.LastTypo, Wrong: r.Wrong, Total: r.Total,
			LastAt:        r.LastAt,
			DaysSince:     max(0, int(now.Sub(r.LastAt).Hours()/24)),
			AnsweredToday: r.LastAt.In(loc).Format("2006-01-02") == today,
		})
	}
	return pool, nil
}

// seedFor makes a learner's batch the same for the whole day, so reloading the page does not shuffle it.
func seedFor(userID, day string) int64 {
	f := fnv.New64a()
	f.Write([]byte(userID + "|" + day))
	return int64(f.Sum64())
}

// nextMidnight is the start of the learner's next calendar day, in their timezone.
func nextMidnight(us *store.UserStore, now time.Time) string {
	loc := us.Location()
	t := now.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, loc).Format(time.RFC3339)
}

// getRepeat serves today's batch. A learner without a year or a paid level gets one batch a day:
// the stored one until they finish it, then 402 repeat_limit until their next midnight.
func (h handlers) getRepeat(w http.ResponseWriter, r *http.Request) {
	us, ok := h.user(w, r)
	if !ok {
		return
	}
	ac, _ := authFrom(r)
	audioOff := r.URL.Query().Get("audio") == "off"
	now := h.Now()
	day := us.Today(now)
	// Recorded for everyone: a learner with unlimited repeats has no stored batch to show it, and
	// the Telegram nudge must not write to someone who is using the button.
	if err := us.NoteRepeatOpened(now); err != nil {
		log.Printf("repeat: note open for %s: %v", ac.UserID, err)
	}

	unlimited, err := h.Store.HasUnlimitedRepeat(ac.UserID, now)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	pool, err := h.repeatPool(us, now)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}

	var keys []string // what to show
	size := 0         // how many the batch had when it started
	switch {
	case unlimited:
		keys = repeat.Select(pool.cands, repeat.Size, now.UnixNano())
		size = len(keys)
	default:
		sess, err := us.RepeatSession(day)
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		if sess == nil {
			picked := repeat.Select(pool.cands, repeat.Size, seedFor(ac.UserID, day))
			if len(picked) > 0 { // an empty pool must not spend the day
				if sess, err = us.StartRepeatSession(day, picked, now); err != nil {
					fail(w, 500, err.Error())
					return
				}
			}
		}
		if sess != nil {
			pending, err := us.RepeatPending(*sess)
			if err != nil {
				fail(w, 500, err.Error())
				return
			}
			if len(pending) == 0 {
				writeJSON(w, http.StatusPaymentRequired, map[string]string{
					"error": "repeat_limit", "next_available": nextMidnight(us, now),
				})
				return
			}
			keys, size = pending, len(sess.Keys)
		}
	}

	out := repeatDTO{Items: []mistakeDTO{}, Unlimited: unlimited}
	for _, k := range keys {
		if it, ok := pool.byKey[k]; ok {
			out.Items = append(out.Items, mistakeDTO{Lesson: it.Lesson, Exercise: toExerciseDTO(it.Exercise, audioOff)})
		}
	}
	out.Total = len(out.Items)
	out.Session = repeatSessionDTO{Size: size, Remaining: out.Total}
	writeJSON(w, http.StatusOK, out)
}
