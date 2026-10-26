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
	if pool != 925 {
		t.Fatalf("quest pool = %d, want 925 (see the design doc's economy table)", pool)
	}

	products, err := s.ListProducts(true)
	if err != nil {
		t.Fatalf("list products: %v", err)
	}
	want := map[string]int64{
		"phase_unlock:1b":          40,
		"phase_unlock:2":           150,
		"phase_unlock:3":           260,
		"phase_unlock:4":           370,
		"phase_unlock:5":           480,
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

	// The economy's shape as an assertion, so editing a reward without the price
	// (or the other way round) fails here rather than being discovered by users.
	//
	// It is tied to a reference learner: one lesson a day, a streak that is never
	// broken, subscribed to the channel, no invited friends. At the end of each
	// level they hold the level pays, the one-off quests that arrive by then and
	// the daily drip (income); take away the streak quests and the drip and what is
	// left is what lessons and ordinary quests alone have paid (floor).
	//   Level 2 costs at most four fifths of what finishing Level 1 pays: the
	//   lessons alone open it, with room to spare, no quests needed.
	//   Level 3 costs no more than the floor: lessons and ordinary quests pay for
	//   it, no streak needed.
	//   Level 4 costs more than the floor and no more than the income: it needs a
	//   streak, and a streak is enough.
	//   Levels 5 and 6 cost more than the reference learner has when they finish
	//   the level before: they are bought.
	// Prices never fall from one level to the next, or a later level could be
	// bought before an earlier one.
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
	ladder, err := economy.ParseLadder("[[1,1],[30,2],[100,3]]")
	if err != nil {
		t.Fatal(err)
	}
	drip := func(days int) int64 {
		var sum int64
		for d := 1; d <= days; d++ {
			sum += ladder.DripFor(d)
		}
		return sum
	}

	// Where the reference learner meets each one-off quest: the first level by
	// whose end they have done enough. Levels are numbered 1..5 (phases 1, 1b, 2,
	// 3, 4) and a day is a lesson. vocab_learned and reviews_done are assumptions
	// about the pace of the spaced repetition, the rest follow from the lesson and
	// day counts.
	levelPhases := []string{"1", "1b", "2", "3", "4"}
	lessonsBy := []int{17, 40, 54, 66, 84} // lessons (and days) done by the end of each level
	reachedAt := func(q economy.Quest) (level int, habit, ok bool) {
		first := func(done func(k int) bool) int {
			for k := 1; k <= 5; k++ {
				if done(k) {
					return k
				}
			}
			return 0
		}
		switch q.Kind {
		case economy.QuestTelegramSubscribed, economy.QuestCorrectInRow:
			return 1, false, true
		case economy.QuestLessonsCompleted:
			l := first(func(k int) bool { return lessonsBy[k-1] >= q.Target })
			return l, false, l > 0
		case economy.QuestStreakDays:
			l := first(func(k int) bool { return lessonsBy[k-1] >= q.Target })
			return l, true, l > 0
		case economy.QuestVocabLearned:
			l := map[int]int{30: 1, 100: 2, 300: 4}[q.Target]
			return l, false, l > 0
		case economy.QuestReviewsDone:
			l := map[int]int{100: 2, 500: 4}[q.Target]
			return l, false, l > 0
		}
		return 0, false, false
	}
	var income, floor [6]int64 // by level 1..5
	for k := 1; k <= 5; k++ {
		for j := 1; j <= k; j++ {
			income[k] += phasePay[levelPhases[j-1]]
			floor[k] += phasePay[levelPhases[j-1]]
		}
		for _, q := range quests {
			if l, habit, ok := reachedAt(q); ok && l <= k && q.Kind != economy.QuestPhaseCompleted && q.Kind != economy.QuestLessonCompleted {
				income[k] += q.Reward
				if !habit {
					floor[k] += q.Reward
				}
			}
		}
		income[k] += drip(lessonsBy[k-1])
	}
	order := []string{"1b", "2", "3", "4", "5"} // the levels for sale, 2..6
	var spent int64
	prev := int64(0)
	for i, ref := range order {
		price := got["phase_unlock:"+ref]
		spent += price
		if price < prev {
			t.Errorf("phase %s costs %d, less than the level before it (%d)", ref, price, prev)
		}
		prev = price
		bought := i + 1 // the learner has finished this many levels when they buy it
		switch ref {
		case "1b":
			if spent*5 > phasePay["1"]*4 {
				t.Errorf("Level 2 costs %d, more than four fifths of the %d that Level 1 pays: the lessons alone should open it with room to spare", spent, phasePay["1"])
			}
		case "2":
			if spent > floor[bought] {
				t.Errorf("phase %s brings the total to %d but lessons and ordinary quests pay %d by the end of level %d: it should need no streak", ref, spent, floor[bought], bought)
			}
		case "3":
			if spent > income[bought] {
				t.Errorf("phase %s brings the total to %d, but the reference learner has %d at the end of level %d: unreachable", ref, spent, income[bought], bought)
			}
			if spent <= floor[bought] {
				t.Errorf("phase %s brings the total to %d, but lessons and ordinary quests already pay %d by the end of level %d: it should need a streak", ref, spent, floor[bought], bought)
			}
		default:
			if spent <= income[bought] {
				t.Errorf("phase %s brings the total to %d, but the reference learner has %d at the end of level %d: it should be bought, not saved up", ref, spent, income[bought], bought)
			}
		}
	}
	t.Logf("reference learner by level: income %v, floor %v", income[1:], floor[1:])
}

// Splitting a level's reward across its lessons must not change what the level
// is worth: the totals are what the prices above were calculated against.
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
	}{{"1", 52}, {"1b", 70}, {"2", 70}, {"3", 90}, {"4", 110}} {
		if perPhase[want.phase] != want.total {
			t.Errorf("phase %s pays %d in total, want %d", want.phase, perPhase[want.phase], want.total)
		}
		// Every lesson pays, the short ones ("02b") included.
		for _, ph := range phases {
			if ph.ID == want.phase && lessonRows[want.phase] != len(ph.Lessons) {
				t.Errorf("phase %s has %d lessons but %d of them pay", want.phase, len(ph.Lessons), lessonRows[want.phase])
			}
		}
	}
	// Level 6 is the last one for sale: nothing follows it to discount.
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

	// Level 1 is the free course and must stay free: nothing may price it. This
	// is what keeps a deploy from locking out every new learner.
	if !real["1"] {
		t.Fatal(`course.yaml no longer defines phase "1"; the free/paid split needs rethinking`)
	}
	if paid["1"] {
		t.Error(`phase "1" is priced by a seeded product, but it is the free course`)
	}
	// And every other level has to be for sale, or it can never be unlocked.
	for _, id := range []string{"1b", "2", "3", "4", "5"} {
		if !real[id] {
			t.Fatalf("course.yaml no longer defines phase %q", id)
		}
		if !paid[id] {
			t.Errorf("phase %q has no seeded product, so it can never be unlocked", id)
		}
	}
}
