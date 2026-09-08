package content

import (
	"fmt"
	"strings"

	"github.com/grisha/serbian-app/server/internal/checker"
)

// ranks order exercise types from easiest (recognition) to hardest
// (free production).
var ranks = map[string]int{
	"choice": 1,
	"match":  2, "fill_blank": 2,
	"word_bank": 3, "fix_error": 3,
	"conjugate": 4, "listen": 4,
	"translate": 5,
	"free":      6,
}

func exerciseRank(t string) int {
	if r, ok := ranks[t]; ok {
		return r
	}
	return 99
}

// checkDifficultyOrder enforces non-decreasing exercise difficulty within a
// practice step. checkpoint steps and steps flagged `mixed` are exempt.
func checkDifficultyOrder(rel string, s Step) error {
	if s.Kind != "practice" || s.Mixed {
		return nil
	}
	prev, prevID := 0, ""
	for _, e := range s.Exercises {
		r := exerciseRank(e.Type)
		if r < prev {
			return fmt.Errorf("%s: step %s: %s (%s) is easier than the preceding %s — reorder or set `mixed: true`",
				rel, s.ID, e.ID, e.Type, prevID)
		}
		prev, prevID = r, e.ID
	}
	return nil
}

// validateWordBank checks every token of every accepted answer is present in
// the chip bank, so the exercise is actually solvable.
func validateWordBank(bank, accept []string) error {
	have := map[string]bool{}
	for _, b := range bank {
		for _, tok := range strings.Fields(checker.Normalize(b)) {
			have[tok] = true
		}
	}
	for _, a := range accept {
		for _, tok := range strings.Fields(checker.Normalize(a)) {
			if !have[tok] {
				return fmt.Errorf("bank is missing %q (needed for accepted answer %q)", tok, a)
			}
		}
	}
	return nil
}
