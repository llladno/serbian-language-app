package store

import (
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/content"
	"github.com/grisha/serbian-app/server/internal/economy"
)

var seedNow = time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)

// seedPhases stands in for the course when a test only cares about the fixed
// catalogue. Phase "1" has four lessons, enough for one of them to land on the
// every-fourth bump; phase 5 pays nothing for lessons, same as the real
// course.
var seedPhases = []economy.PhaseLessons{
	{ID: "1", Lessons: []string{"00", "01", "02", "03"}},
	{ID: "5", Lessons: []string{"60", "61"}},
}

// realPhases is the course as the seed sees it, or a skip when the content
// tree is not next to the test.
func realPhases(t *testing.T) []economy.PhaseLessons {
	t.Helper()
	c, err := content.Load("../../../content")
	if err != nil {
		t.Skipf("real content tree not available: %v", err)
	}
	var out []economy.PhaseLessons
	for _, ph := range c.Phases {
		out = append(out, economy.PhaseLessons{ID: ph.ID, Lessons: ph.Lessons})
	}
	return out
}

// The pool is checked against the real course, not a fixture: most of what a
// level pays now arrives lesson by lesson, so the total depends on how many
// lessons each level actually has.
func TestSeededEconomyMatchesTheDesign(t *testing.T) {
	s := newStore(t)
	if err := s.SeedEconomyDefaults(realPhases(t), seedNow); err != nil {
		t.Fatalf("seed: %v", err)
	}

	quests, err := s.ListQuests(true)
	if err != nil {
		t.Fatalf("list quests: %v", err)
	}
	var pool int64
	for _, q := range quests {
		pool += q.Reward
	}
	if pool != 853 {
		t.Fatalf("quest pool = %d, want 853 (see the design doc's economy table)", pool)
	}

	products, err := s.ListProducts(true)
	if err != nil {
		t.Fatalf("list products: %v", err)
	}
	want := map[string]int64{
		"phase_unlock:4":           230,
		"phase_unlock:5":           375,
		"consumable:streak_repair": 25,
	}
	got := map[string]int64{}
	for _, p := range products {
		got[p.Kind+":"+p.Ref] = p.Price
	}
	if len(got) != len(want) {
		t.Fatalf("seeded products = %v, want exactly %v", got, want)
	}
	for key, price := range want {
		if got[key] != price {
			t.Errorf("%s price = %d, want %d", key, got[key], price)
		}
	}

	// The economy's shape as an assertion: both prices are derived from the
	// rewards, so editing one without the other fails here rather than being
	// discovered by users.
	//   Level 4 = what the free levels pay + the Telegram quest.
	//   Level 5 = Level 4's price + what Level 4 pays + the 100-words and
	//             100-reviews quests.
	reward := func(kind string, target int, param string) int64 {
		for _, q := range quests {
			if q.Kind == kind && q.Target == target && q.Param == param {
				return q.Reward
			}
		}
		t.Fatalf("no seeded %s quest with target %d / param %q", kind, target, param)
		return 0
	}
	phasePay := map[string]int64{}
	lessonPhase := map[string]string{}
	for _, ph := range realPhases(t) {
		for _, id := range ph.Lessons {
			lessonPhase[id] = ph.ID
		}
	}
	for _, q := range quests {
		switch q.Kind {
		case economy.QuestPhaseCompleted:
			phasePay[q.Param] += q.Reward
		case economy.QuestLessonCompleted:
			phasePay[lessonPhase[q.Param]] += q.Reward
		}
	}
	telegram := reward(economy.QuestTelegramSubscribed, 1, "")
	wantLevel4 := phasePay["1"] + phasePay["2"] + phasePay["3"] + telegram
	if got["phase_unlock:4"] != wantLevel4 {
		t.Errorf("Level 4 costs %d, want free levels (%d) + Telegram (%d) = %d",
			got["phase_unlock:4"], phasePay["1"]+phasePay["2"]+phasePay["3"], telegram, wantLevel4)
	}
	wantLevel5 := got["phase_unlock:4"] + phasePay["4"] +
		reward(economy.QuestVocabLearned, 100, "") + reward(economy.QuestReviewsDone, 100, "")
	if got["phase_unlock:5"] != wantLevel5 {
		t.Errorf("Level 5 costs %d, want Level 4 price + Level 4 pay (%d) + 100 words + 100 reviews = %d",
			got["phase_unlock:5"], phasePay["4"], wantLevel5)
	}
}

// Splitting a level's reward across its lessons must not change what the level
// is worth: the design doc's 50 / 70 / 90 are what the rest of the economy —
// including the price of Level 4 — was calculated against.
func TestLessonRewardsMatchPhaseTotals(t *testing.T) {
	s := newStore(t)
	phases := realPhases(t)
	if err := s.SeedEconomyDefaults(phases, seedNow); err != nil {
		t.Fatalf("seed: %v", err)
	}
	quests, err := s.ListQuests(true)
	if err != nil {
		t.Fatalf("list quests: %v", err)
	}

	lessonPhase := map[string]string{}
	for _, ph := range phases {
		for _, id := range ph.Lessons {
			lessonPhase[id] = ph.ID
		}
	}
	perPhase := map[string]int64{}
	lessonRows := map[string]int{}
	for _, q := range quests {
		switch q.Kind {
		case economy.QuestPhaseCompleted:
			perPhase[q.Param] += q.Reward
		case economy.QuestLessonCompleted:
			phase, ok := lessonPhase[q.Param]
			if !ok {
				t.Errorf("lesson reward for %q, which no phase of the course contains", q.Param)
				continue
			}
			perPhase[phase] += q.Reward
			lessonRows[phase]++
		}
	}

	for _, want := range []struct {
		phase string
		total int64
	}{{"1", 50}, {"2", 70}, {"3", 90}, {"4", 110}} {
		if perPhase[want.phase] != want.total {
			t.Errorf("phase %s pays %d in total, want %d", want.phase, perPhase[want.phase], want.total)
		}
		for _, ph := range phases {
			if ph.ID == want.phase && lessonRows[want.phase] != len(ph.Lessons) {
				t.Errorf("phase %s has %d lessons but %d of them pay", want.phase, len(ph.Lessons), lessonRows[want.phase])
			}
		}
	}
	// Level 5 is the last one for sale: nothing follows it to discount.
	if perPhase["5"] != 0 {
		t.Errorf("phase 5 pays %d, want 0", perPhase["5"])
	}
}

func TestSeedIsIdempotent(t *testing.T) {
	s := newStore(t)
	for i := 0; i < 3; i++ {
		if err := s.SeedEconomyDefaults(seedPhases, seedNow); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	quests, _ := s.ListQuests(false)
	want := len(defaultQuests) + 4 // the catalogue plus one row per phase-1 lesson
	if len(quests) != want {
		t.Fatalf("%d quests after three seed runs, want %d", len(quests), want)
	}
}

// The one that matters operationally: once the owner curates the quest list in
// the admin panel, a restart must not undo their work.
func TestSeedDoesNotResurrectDeletedQuests(t *testing.T) {
	s := newStore(t)
	if err := s.SeedEconomyDefaults(seedPhases, seedNow); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.db.Exec(`DELETE FROM quests`); err != nil {
		t.Fatalf("delete quests: %v", err)
	}
	if err := s.SeedEconomyDefaults(seedPhases, seedNow); err != nil {
		t.Fatalf("re-seed: %v", err)
	}
	quests, _ := s.ListQuests(false)
	if len(quests) != 0 {
		t.Fatalf("%d quests came back after being deleted in the admin; the seed guard did not hold", len(quests))
	}
}

// The seeds name phases by course.yaml's ids. Renaming a phase would silently
// break both halves of the economy — a phase_completed quest nobody can finish,
// because QuestValue reports 0 for an unknown phase, and a phase that quietly
// stops locking because no product matches it. Neither failure is visible in
// any other test, since every one of them runs on the fixture course.
func TestSeedMatchesTheRealCourse(t *testing.T) {
	c, err := content.Load("../../../content")
	if err != nil {
		t.Skipf("real content tree not available: %v", err)
	}
	real := map[string]bool{}
	for _, p := range c.Phases {
		real[p.ID] = true
	}

	paid := map[string]bool{}
	for _, p := range defaultProducts {
		if p.kind != "phase_unlock" {
			continue
		}
		if !real[p.ref] {
			t.Errorf("seeded phase_unlock for phase %q, which course.yaml does not define", p.ref)
		}
		paid[p.ref] = true
	}
	for _, q := range defaultQuests {
		if q.kind != "phase_completed" {
			continue
		}
		if !real[q.param] {
			t.Errorf("seeded phase_completed quest for phase %q, which course.yaml does not define", q.param)
		}
	}

	// Phases 1..3 are the free course and must stay free: nothing may price
	// them. This is what keeps a deploy from locking out every existing user.
	for _, id := range []string{"1", "2", "3"} {
		if !real[id] {
			t.Fatalf("course.yaml no longer defines phase %q; the free/paid split needs rethinking", id)
		}
		if paid[id] {
			t.Errorf("phase %q is priced by a seeded product, but it is part of the free course", id)
		}
	}
	// And the paid ones have to exist, or nothing is for sale.
	for _, id := range []string{"4", "5"} {
		if !paid[id] {
			t.Errorf("phase %q has no seeded product, so it can never be unlocked", id)
		}
	}
}
