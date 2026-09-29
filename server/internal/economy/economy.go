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
