package repeat

import (
	"hash/fnv"
	"time"
)

// The Telegram nudge for a learner who studies but has stopped using «Повторить». It writes at
// most once per quiet spell: after a message, silence until the learner repeats on their own.

const (
	// NudgeAfter is how long without a repeat before the nudge is due.
	NudgeAfter = 48 * time.Hour
	// nudgeActiveWithin: the learner must have studied this recently. Someone who has left
	// altogether already gets the «come back» reminder, and two messages in two days to a person
	// who ignored the first is spam.
	nudgeActiveWithin = 48 * time.Hour
	// The send window, in the learner's own timezone (the quiet hours of docs/TELEGRAM_MESSAGES.md).
	nudgeFromHour = 10
	nudgeToHour   = 20
)

// NudgeInput describes one learner at the moment of a reminder sweep. Now must already be in the
// learner's timezone: the send window is read from its clock.
type NudgeInput struct {
	Now          time.Time
	LastRepeat   time.Time // latest time they opened «Повторить»; zero = never
	LastActivity time.Time // latest review or exercise attempt; zero = none
	LastNudge    time.Time // latest nudge written for them, a real one or a control-group one; zero = none
	OptedOut     bool      // pressed «Не про повторения»
	// OtherLearningToday: another study message (the «all done» congrats, the «come back» nudge)
	// already went out today. One study message a day.
	OtherLearningToday bool
}

// NudgeDue reports whether the learner should get the nudge now.
func NudgeDue(in NudgeInput) bool {
	switch {
	case in.OptedOut, in.OtherLearningToday:
		return false
	case in.LastRepeat.IsZero():
		// Only people who already know the button: for the rest this would be an advert for a
		// feature they have never seen. Widen this once the first results are in.
		return false
	case in.Now.Sub(in.LastRepeat) < NudgeAfter:
		return false
	case in.LastActivity.IsZero() || in.Now.Sub(in.LastActivity) >= nudgeActiveWithin:
		return false
	case !in.LastNudge.IsZero() && !in.LastRepeat.After(in.LastNudge):
		return false // already nudged since their last repeat
	}
	h := in.Now.Hour()
	return h >= nudgeFromHour && h < nudgeToHour
}

func hash(s string) uint32 {
	f := fnv.New32a()
	f.Write([]byte(s))
	return f.Sum32()
}

// NudgeVariant picks which of n texts a learner gets. Texts go round in order for each learner,
// and each learner starts at a different one, so every text is somebody's first.
func NudgeVariant(userID string, sentBefore, n int) int {
	return int((hash("nudge-variant|"+userID) + uint32(sentBefore)) % uint32(n))
}

// InControl: this learner is in the half that is never written to, so the nudge can be judged
// by comparing the two halves. Stable for a given learner.
func InControl(userID string) bool {
	return hash("nudge-control|"+userID)%2 == 0
}
