// Package economy holds the currency's pure arithmetic — the streak drip
// ladder and discount maths. It has no database and no HTTP, so the rules
// that decide how much a user pays or earns are testable on their own.
package economy

import (
	"encoding/json"
	"fmt"
)

// Ladder maps a streak length to a daily payout: each entry is
// [from_day, coins_per_day], ascending, starting at day 1. Stored in
// economy_settings.streak_drip as JSON so it is editable from the admin.
type Ladder [][2]int

// ParseLadder reads the JSON form and validates it: non-empty, first entry
// starts at day 1, days strictly ascending, payouts non-negative.
func ParseLadder(s string) (Ladder, error) {
	var l Ladder
	if err := json.Unmarshal([]byte(s), &l); err != nil {
		return nil, fmt.Errorf("parse drip ladder: %w", err)
	}
	if len(l) == 0 {
		return nil, fmt.Errorf("parse drip ladder: empty")
	}
	if l[0][0] != 1 {
		return nil, fmt.Errorf("parse drip ladder: first step must start at day 1, got %d", l[0][0])
	}
	for i, step := range l {
		if step[1] < 0 {
			return nil, fmt.Errorf("parse drip ladder: step %d has negative payout", i)
		}
		if i > 0 && step[0] <= l[i-1][0] {
			return nil, fmt.Errorf("parse drip ladder: step %d does not increase (%d after %d)",
				i, step[0], l[i-1][0])
		}
	}
	return l, nil
}

// DripFor returns the payout for a day that brought the streak to streakDays.
// A zero-length streak pays nothing.
func (l Ladder) DripFor(streakDays int) int64 {
	if streakDays <= 0 {
		return 0
	}
	var out int64
	for _, step := range l {
		if streakDays >= step[0] {
			out = int64(step[1])
		}
	}
	return out
}

// EffectivePrice applies the better of a sale and a promo code — they never
// stack — and rounds up, so a discount can never make something free by
// accident. applied names the winner ("sale", "promo" or "" for neither) so
// the UI can explain why a promo code changed nothing.
func EffectivePrice(base int64, salePercent, promoPercent int) (int64, string) {
	applied := ""
	best := 0
	if salePercent > 0 {
		best, applied = salePercent, "sale"
	}
	if promoPercent > best {
		best, applied = promoPercent, "promo"
	}
	if best <= 0 {
		return base, ""
	}
	if best >= 100 {
		return 0, applied
	}
	remaining := base * int64(100-best)
	price := remaining / 100
	if remaining%100 != 0 {
		price++
	}
	return price, applied
}

// Quest kinds. Every kind must be computable server-side from data the user
// cannot forge — that is the whole reason the admin panel creates instances
// of a fixed list rather than free-form conditions.
const (
	QuestLessonsCompleted   = "lessons_completed"
	QuestVocabLearned       = "vocab_learned"
	QuestReviewsDone        = "reviews_done"
	QuestStreakDays         = "streak_days"
	QuestCorrectInRow       = "correct_in_row"
	QuestPhaseCompleted     = "phase_completed"
	QuestTelegramSubscribed = "telegram_subscribed"
)

// Quest is one row of the quests table.
type Quest struct {
	ID          int64
	Kind        string
	Target      int
	Param       string
	Title       string
	Description string
	Reward      int64
	Active      bool
	SortOrder   int
}

// Counters is everything a quest can be measured against, gathered once per
// request. CompletedLessons is keyed by lesson id as it appears in
// course.yaml. TelegramSubscribed is filled by the caller, since it needs a
// Bot API round trip.
type Counters struct {
	LessonsCompleted   int
	VocabLearned       int
	ReviewsDone        int
	AnswerBest         int
	StreakDays         int
	CompletedLessons   map[string]bool
	TelegramSubscribed bool
}

// QuestValue is the user's current value for q. For phase_completed the value
// is a percentage (0..100), which is why such a quest's target is always 100;
// for telegram_subscribed it is 0 or 1. An unknown kind is worth 0 rather than
// an error: an admin can save a kind this binary has not learned yet, and a
// quest nobody can finish is better than a 500 on the profile screen.
func QuestValue(q Quest, c Counters, phaseLessons map[string][]string) int {
	switch q.Kind {
	case QuestLessonsCompleted:
		return c.LessonsCompleted
	case QuestVocabLearned:
		return c.VocabLearned
	case QuestReviewsDone:
		return c.ReviewsDone
	case QuestStreakDays:
		return c.StreakDays
	case QuestCorrectInRow:
		return c.AnswerBest
	case QuestPhaseCompleted:
		lessons := phaseLessons[q.Param]
		if len(lessons) == 0 {
			return 0
		}
		done := 0
		for _, id := range lessons {
			if c.CompletedLessons[id] {
				done++
			}
		}
		return done * 100 / len(lessons)
	case QuestTelegramSubscribed:
		if c.TelegramSubscribed {
			return 1
		}
		return 0
	default:
		return 0
	}
}

// QuestDone reports whether the quest's target has been reached. A target of
// zero or less is never done: QuestValue reports 0 for an unknown kind, so
// without this guard every kind this binary does not recognise would be
// instantly claimable by everyone.
func QuestDone(q Quest, c Counters, phaseLessons map[string][]string) bool {
	if q.Target <= 0 {
		return false
	}
	return QuestValue(q, c, phaseLessons) >= q.Target
}
