package repeat

import (
	"slices"
	"testing"
	"time"
)

func cand(key string, mod func(*Candidate)) Candidate {
	x := Candidate{Key: key, Level: 1, LastCorrect: true, DaysSince: 3}
	if mod != nil {
		mod(&x)
	}
	return x
}

func TestTiersAreFilledInOrder(t *testing.T) {
	cands := []Candidate{
		cand("ok", nil),
		cand("typo", func(x *Candidate) { x.LastTypo = true }),
		cand("bad-once", func(x *Candidate) { x.LastCorrect, x.Wrong, x.Total = false, 1, 1 }),
		cand("bad-often", func(x *Candidate) { x.LastCorrect, x.Wrong, x.Total = false, 3, 4 }),
	}
	got := Select(cands, Size, 1)
	want := []string{"bad-often", "bad-once", "typo", "ok"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTypoTierPutsTheFreshestFirst(t *testing.T) {
	t0 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	cands := []Candidate{
		cand("old", func(x *Candidate) { x.LastTypo, x.LastAt = true, t0 }),
		cand("new", func(x *Candidate) { x.LastTypo, x.LastAt = true, t0.Add(48*time.Hour) }),
	}
	if got := Select(cands, Size, 1); !slices.Equal(got, []string{"new", "old"}) {
		t.Fatalf("got %v", got)
	}
}

func TestSizeCapsTheBatchAndMistakesWin(t *testing.T) {
	var cands []Candidate
	for i := 0; i < 20; i++ {
		cands = append(cands, cand(string(rune('a'+i)), func(x *Candidate) { x.LastCorrect, x.Wrong, x.Total = false, 1, 1 }))
	}
	cands = append(cands, cand("fine", nil))
	got := Select(cands, Size, 1)
	if len(got) != Size || slices.Contains(got, "fine") {
		t.Fatalf("15 mistakes must fill the batch and leave the rest out: %v", got)
	}
}

func TestTodaysAnswersComeLastAndOnlyWhenThereIsRoom(t *testing.T) {
	// Not enough older exercises: what was answered today fills the batch, after them.
	cands := []Candidate{cand("today", func(x *Candidate) { x.AnsweredToday = true }), cand("older", nil)}
	if got := Select(cands, Size, 1); !slices.Equal(got, []string{"older", "today"}) {
		t.Fatalf("got %v, want the older one first and today's after it", got)
	}

	// Plenty of older ones: today's answers are not repeated right after they were given.
	var many []Candidate
	for i := 0; i < Size; i++ {
		many = append(many, cand(string(rune('a'+i)), nil))
	}
	many = append(many, cand("today", func(x *Candidate) { x.AnsweredToday = true }))
	if got := Select(many, Size, 1); len(got) != Size || slices.Contains(got, "today") {
		t.Fatalf("a full batch of older exercises must leave today's out: %v", got)
	}
}

// A learner who has just finished lessons has nothing older to repeat. The batch must still hold what
// they answered, or one stray mistake would be the whole day's «Повторить».
func TestEverythingAnsweredTodayStillMakesABatch(t *testing.T) {
	var cands []Candidate
	for i := 0; i < 20; i++ {
		cands = append(cands, cand(string(rune('a'+i)), func(x *Candidate) { x.AnsweredToday = true }))
	}
	cands[3].LastCorrect, cands[3].Wrong, cands[3].Total = false, 1, 2
	got := Select(cands, Size, 1)
	if len(got) != Size || got[0] != "d" {
		t.Fatalf("got %v, want 15 with the mistake first", got)
	}
}

func TestSameSeedGivesTheSameBatch(t *testing.T) {
	var cands []Candidate
	for i := 0; i < 40; i++ {
		lvl := 1 + i%6
		cands = append(cands, cand(string(rune('A'+i)), func(x *Candidate) { x.Level = lvl }))
	}
	a, b := Select(cands, Size, 42), Select(cands, Size, 42)
	if !slices.Equal(a, b) {
		t.Fatalf("same seed must not shuffle:\n%v\n%v", a, b)
	}
	if other := Select(cands, Size, 43); slices.Equal(a, other) {
		t.Fatal("another seed should give another draw")
	}
}

func TestHigherLevelsAreDrawnMoreOften(t *testing.T) {
	// 30 level-1 and 30 level-6 exercises, batches of 15 over many seeds: level 6 must dominate.
	var cands []Candidate
	for i := 0; i < 30; i++ {
		cands = append(cands, cand("L1-"+string(rune('a'+i)), nil), cand("L6-"+string(rune('a'+i)), func(x *Candidate) { x.Level = 6 }))
	}
	six, one := 0, 0
	for seed := int64(0); seed < 200; seed++ {
		for _, k := range Select(cands, Size, seed) {
			if k[1] == '6' {
				six++
			} else {
				one++
			}
		}
	}
	if six < 3*one {
		t.Fatalf("level 6 drawn %d times vs level 1 %d: the level weight is not applied", six, one)
	}
}

func TestNoCandidatesGivesNothing(t *testing.T) {
	if got := Select(nil, Size, 1); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}
