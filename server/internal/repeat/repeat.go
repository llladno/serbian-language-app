// Package repeat picks the exercises of a «Повторить» batch. It knows nothing about HTTP or the
// database: the caller describes the candidates and gets keys back.
package repeat

import (
	"math"
	"math/rand"
	"sort"
	"time"
)

// Size is how many exercises one batch holds.
const Size = 15

// Candidate is one exercise the learner has already answered.
type Candidate struct {
	Key          string // "lesson|exerciseID"
	Level        int    // 1 for the first level … 6 for the last: harder material comes back more often
	LastCorrect  bool
	LastTypo     bool // the last answer was right only because a one-letter slip was forgiven
	Wrong, Total int
	LastAt       time.Time
	DaysSince    int // whole days since the last attempt
	// AnsweredToday: answered on the learner's calendar day already (not repeated as filler).
	AnsweredToday bool
}

// Select returns up to size keys: first every exercise whose last answer was wrong (most wrong
// first), then the almost-right ones (freshest first), then a weighted random draw from the
// rest. What was answered today is drawn only after the older exercises, and only if the batch is
// still not full: a learner who has just finished lessons would otherwise get nothing to repeat
// (or a batch of one stray mistake). seed makes the draw repeatable.
func Select(cands []Candidate, size int, seed int64) []string {
	var wrong, typo, rest, today []Candidate
	for _, c := range cands {
		switch {
		case !c.LastCorrect:
			wrong = append(wrong, c)
		case c.LastTypo:
			typo = append(typo, c)
		case c.AnsweredToday:
			today = append(today, c)
		default:
			rest = append(rest, c)
		}
	}
	sort.Slice(wrong, func(i, j int) bool {
		a, b := wrong[i], wrong[j]
		if a.Wrong != b.Wrong {
			return a.Wrong > b.Wrong
		}
		if a.Total != b.Total {
			return a.Total > b.Total
		}
		return a.Key < b.Key
	})
	sort.Slice(typo, func(i, j int) bool {
		a, b := typo[i], typo[j]
		if !a.LastAt.Equal(b.LastAt) {
			return a.LastAt.After(b.LastAt)
		}
		return a.Key < b.Key
	})

	var out []string
	for _, tier := range [][]Candidate{wrong, typo} {
		for _, c := range tier {
			if len(out) == size {
				return out
			}
			out = append(out, c.Key)
		}
	}
	out = append(out, draw(rest, size-len(out), seed)...)
	return append(out, draw(today, size-len(out), seed)...)
}

// draw picks n of cands without repeats, each with a probability proportional to its weight
// (Efraimidis–Spirakis: the n largest u^(1/w)). weight = level × (1 + min(days, 30) / 10).
func draw(cands []Candidate, n int, seed int64) []string {
	if n <= 0 || len(cands) == 0 {
		return nil
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].Key < cands[j].Key }) // a stable base for the seed
	rng := rand.New(rand.NewSource(seed))
	type scored struct {
		key string
		s   float64
	}
	all := make([]scored, len(cands))
	for i, c := range cands {
		w := float64(max(c.Level, 1)) * (1 + float64(min(c.DaysSince, 30))/10)
		all[i] = scored{c.Key, math.Pow(rng.Float64(), 1/w)}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].s != all[j].s {
			return all[i].s > all[j].s
		}
		return all[i].key < all[j].key
	})
	out := make([]string, 0, n)
	for i := 0; i < len(all) && i < n; i++ {
		out = append(out, all[i].key)
	}
	return out
}
