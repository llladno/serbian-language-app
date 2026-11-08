// Package campaign holds the rules of the «return» campaign: a one-off message to learners who never reached lesson 02, sent in
// three waves with two button styles and a group that is never written to
// (docs/superpowers/specs/2026-11-08-return-campaign-design.md). It decides who is held out, how each wave splits between the
// styles and which style leads; storage is internal/store, sending is internal/api.
package campaign

import (
	"fmt"
	"hash/fnv"
)

// Name is the campaign's key in campaign_sends.
const Name = "return"

// The two button styles: the plain call to action, and the low-threshold «just have a look».
const (
	StyleDirect = 0
	StyleGlance = 1
	NumStyles   = 2
)

// Waves is how many waves the campaign has.
const Waves = 3

// holdoutPercent of the audience is never written to, so the people who got a message can be compared with those who did not.
const holdoutPercent = 15

func hash(s string) uint32 {
	f := fnv.New32a()
	f.Write([]byte(s))
	return f.Sum32()
}

// Holdout reports whether this learner is in the group that is never written to. Stable for a given learner.
func Holdout(userID string) bool {
	return hash("return|holdout|"+userID)%100 < holdoutPercent
}

// Ord is the learner's place in the random order the waves take people in. Stable for a given learner.
func Ord(userID string) int64 {
	return int64(hash("return|order|" + userID))
}

// WaveSize is how many people of one channel a wave takes: 50 in the first two waves, everybody left in the third (-1).
func WaveSize(wave int) int {
	if wave >= Waves {
		return -1
	}
	return 50
}

// leaderShare is the share of the leading style in a wave, in thousandths. The first wave has no leader yet and splits evenly.
func leaderShare(wave int) int {
	switch wave {
	case 1:
		return 500
	case 2:
		return 550
	}
	return 600
}

// Shares is how a wave splits its people between the styles, in thousandths: the leader gets its share, the other style the rest.
func Shares(wave, leader int) [NumStyles]int {
	var s [NumStyles]int
	s[leader] = leaderShare(wave)
	s[1-leader] = 1000 - s[leader]
	return s
}

// LeaderShare is the leader's share in a wave, in thousandths, for the record kept with the wave.
func LeaderShare(wave int) int { return leaderShare(wave) }

// StyleFor gives a learner their style in a wave by hashing them into the shares: the same learner in the same wave always gets the
// same style, and over many learners the shares come out as asked.
func StyleFor(userID string, wave int, shares [NumStyles]int) int {
	bucket := int(hash(fmt.Sprintf("return|style|%d|%s", wave, userID)) % 1000)
	sum := 0
	for style, share := range shares {
		sum += share
		if bucket < sum {
			return style
		}
	}
	return NumStyles - 1
}

// Stat is how a style has done so far.
type Stat struct{ Sent, Clicked int }

// Leader is the style that has done better so far: the one with the higher (clicked+1)/(sent+20). The +1 and +20 pull a style with
// few sends toward a 5% click rate, so a style with no sends cannot lead on no evidence. The plain style wins a tie, so a campaign
// with no clicks yet carries on with the text that was approved first.
func Leader(stats [NumStyles]Stat) int {
	best := StyleDirect
	for style := 1; style < NumStyles; style++ {
		// compare (c_s+1)/(n_s+20) > (c_b+1)/(n_b+20) without dividing
		if (stats[style].Clicked+1)*(stats[best].Sent+20) > (stats[best].Clicked+1)*(stats[style].Sent+20) {
			best = style
		}
	}
	return best
}
