// Package srs implements an SM-2-style spaced-repetition scheduler.
package srs

import (
	"math"
	"time"
)

// Grade is the learner's self-assessment after seeing a card.
type Grade int

const (
	Again Grade = 0
	Hard  Grade = 1
	Good  Grade = 2
	Easy  Grade = 3
)

// State is a card's lifecycle stage.
type State string

const (
	New      State = "new"
	Learning State = "learning"
	Review   State = "review"
)

const (
	minEase     = 1.3
	startEase   = 2.5
	maxInterval = 365
)

// Card is the mutable scheduling state for one item.
type Card struct {
	Ease         float64
	IntervalDays int
	Reps         int
	Lapses       int
	State        State
	Due          time.Time // zero for a brand-new card
}

// NewCard returns a fresh, unseen card.
func NewCard() Card {
	return Card{Ease: startEase, State: New}
}

// Schedule returns the updated card after a review at time now.
func Schedule(c Card, g Grade, now time.Time) Card {
	if c.Ease == 0 {
		c.Ease = startEase
	}

	if c.State == New || c.State == Learning {
		switch g {
		case Again:
			c.State = Learning
			c.IntervalDays = 0
			c.Due = now
		case Hard, Good:
			c.State = Review
			c.IntervalDays = 1
			c.Reps++
			c.Due = addDays(now, 1)
		case Easy:
			c.State = Review
			c.IntervalDays = 4
			c.Reps++
			c.Due = addDays(now, 4)
		}
		return c
	}

	// Review card.
	switch g {
	case Again:
		c.Lapses++
		c.Ease = clampEase(c.Ease - 0.2)
		c.IntervalDays = 1
		c.State = Learning
	case Hard:
		c.Ease = clampEase(c.Ease - 0.15)
		c.IntervalDays = capInterval(roundAtLeast1(float64(c.IntervalDays) * 1.2))
		c.Reps++
	case Good:
		c.IntervalDays = capInterval(roundAtLeast1(float64(c.IntervalDays) * c.Ease))
		c.Reps++
	case Easy:
		c.Ease += 0.15
		c.IntervalDays = capInterval(roundAtLeast1(float64(c.IntervalDays) * c.Ease * 1.3))
		c.Reps++
	}
	c.Due = addDays(now, c.IntervalDays)
	return c
}

// Preview returns the next interval (in days) for each grade if the learner
// were to pick it now, without changing the card. Again on a new/learning
// card is reported as 0 (same day).
func Preview(c Card, now time.Time) map[Grade]int {
	out := make(map[Grade]int, 4)
	for _, g := range []Grade{Again, Hard, Good, Easy} {
		out[g] = Schedule(c, g, now).IntervalDays
	}
	return out
}

func clampEase(e float64) float64 {
	if e < minEase {
		return minEase
	}
	return e
}

func capInterval(n int) int {
	if n > maxInterval {
		return maxInterval
	}
	return n
}

func roundAtLeast1(x float64) int {
	n := int(math.Round(x))
	if n < 1 {
		return 1
	}
	return n
}

// addDays returns midnight, n days after now, in now's location.
func addDays(now time.Time, n int) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d+n, 0, 0, 0, 0, now.Location())
}
