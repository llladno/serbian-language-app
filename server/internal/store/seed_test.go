package store

import (
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/content"
)

var seedNow = time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)

func TestSeededEconomyMatchesTheDesign(t *testing.T) {
	s := newStore(t)
	if err := s.SeedEconomyDefaults(seedNow); err != nil {
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
	if pool != 638 {
		t.Fatalf("quest pool = %d, want 638 (see the design doc's economy table)", pool)
	}

	products, err := s.ListProducts(true)
	if err != nil {
		t.Fatalf("list products: %v", err)
	}
	want := map[string]int64{
		"phase_unlock:4":           500,
		"phase_unlock:5":           1000,
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

	// The economy's shape as an assertion. Level 4 must be reachable on quests
	// alone — that is the whole promise of "100% + все задания" — and Level 5
	// must not be, or the daily streak drip has no purpose. Anyone editing the
	// seeds and breaking that relationship fails here rather than finding out
	// from users.
	if got["phase_unlock:4"] > pool {
		t.Fatalf("Level 4 costs %d but the entire quest pool is %d", got["phase_unlock:4"], pool)
	}
	if got["phase_unlock:5"] <= pool {
		t.Fatalf("Level 5 costs %d, which the quest pool (%d) already covers", got["phase_unlock:5"], pool)
	}
}

func TestSeedIsIdempotent(t *testing.T) {
	s := newStore(t)
	for i := 0; i < 3; i++ {
		if err := s.SeedEconomyDefaults(seedNow); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	quests, _ := s.ListQuests(false)
	if len(quests) != len(defaultQuests) {
		t.Fatalf("%d quests after three seed runs, want %d", len(quests), len(defaultQuests))
	}
}

// The one that matters operationally: once the owner curates the quest list in
// the admin panel, a restart must not undo their work.
func TestSeedDoesNotResurrectDeletedQuests(t *testing.T) {
	s := newStore(t)
	if err := s.SeedEconomyDefaults(seedNow); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.db.Exec(`DELETE FROM quests`); err != nil {
		t.Fatalf("delete quests: %v", err)
	}
	if err := s.SeedEconomyDefaults(seedNow); err != nil {
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
