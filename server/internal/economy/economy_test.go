package economy

import "testing"

func TestParseLadderAndDrip(t *testing.T) {
	l, err := ParseLadder(`[[1,1],[30,2],[100,3]]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	cases := []struct {
		streak int
		want   int64
	}{
		{0, 0}, {1, 1}, {29, 1}, {30, 2}, {99, 2}, {100, 3}, {500, 3},
	}
	for _, c := range cases {
		if got := l.DripFor(c.streak); got != c.want {
			t.Errorf("DripFor(%d) = %d, want %d", c.streak, got, c.want)
		}
	}
}

func TestParseLadderRejectsGarbage(t *testing.T) {
	for _, in := range []string{``, `[]`, `[[0,1]]`, `[[2,1],[1,1]]`, `nope`} {
		if _, err := ParseLadder(in); err == nil {
			t.Errorf("ParseLadder(%q) succeeded; want error", in)
		}
	}
}

func TestEffectivePriceTakesTheBetterDiscount(t *testing.T) {
	cases := []struct {
		name        string
		base        int64
		sale, promo int
		wantPrice   int64
		wantApplied string
	}{
		{"no discounts", 500, 0, 0, 500, ""},
		{"sale only", 500, 20, 0, 400, "sale"},
		{"promo only", 500, 0, 10, 450, "promo"},
		{"sale wins", 500, 20, 10, 400, "sale"},
		{"promo wins", 500, 10, 20, 400, "promo"},
		{"tie prefers sale", 500, 20, 20, 400, "sale"},
		{"rounds up, never free by accident", 25, 99, 0, 1, "sale"},
		{"full discount still costs nothing less than zero", 25, 100, 0, 0, "sale"},
	}
	for _, c := range cases {
		price, applied := EffectivePrice(c.base, c.sale, c.promo)
		if price != c.wantPrice || applied != c.wantApplied {
			t.Errorf("%s: EffectivePrice(%d,%d,%d) = %d,%q; want %d,%q",
				c.name, c.base, c.sale, c.promo, price, applied, c.wantPrice, c.wantApplied)
		}
	}
}

func TestQuestValue(t *testing.T) {
	phases := map[string][]string{
		"1": {"00", "01", "02", "03"},
		"2": {"16", "17"},
	}
	c := Counters{
		LessonsCompleted: 12,
		VocabLearned:     140,
		ReviewsDone:      320,
		AnswerBest:       11,
		StreakDays:       9,
		CompletedLessons: map[string]bool{"00": true, "01": true, "02": true, "16": true},
	}
	cases := []struct {
		kind, param string
		want        int
	}{
		{"lessons_completed", "", 12},
		{"vocab_learned", "", 140},
		{"reviews_done", "", 320},
		{"correct_in_row", "", 11},
		{"streak_days", "", 9},
		{"phase_completed", "1", 75}, // 3 of 4
		{"phase_completed", "2", 50}, // 1 of 2
		{"phase_completed", "9", 0},  // unknown phase
		{"telegram_subscribed", "", 0},
		{"nonsense", "", 0},
	}
	for _, tc := range cases {
		q := Quest{Kind: tc.kind, Param: tc.param, Target: 100}
		if got := QuestValue(q, c, phases); got != tc.want {
			t.Errorf("QuestValue(%s,%q) = %d, want %d", tc.kind, tc.param, got, tc.want)
		}
	}
}

func TestQuestDone(t *testing.T) {
	phases := map[string][]string{"1": {"00", "01"}}
	c := Counters{LessonsCompleted: 10, CompletedLessons: map[string]bool{"00": true, "01": true}}

	if !QuestDone(Quest{Kind: "lessons_completed", Target: 10}, c, phases) {
		t.Error("exactly at target must count as done")
	}
	if QuestDone(Quest{Kind: "lessons_completed", Target: 11}, c, phases) {
		t.Error("one short must not count as done")
	}
	if !QuestDone(Quest{Kind: "phase_completed", Param: "1", Target: 100}, c, phases) {
		t.Error("a fully completed phase must count as done")
	}

	// telegram_subscribed is a flag, not a count.
	sub := c
	sub.TelegramSubscribed = true
	if !QuestDone(Quest{Kind: "telegram_subscribed", Target: 1}, sub, phases) {
		t.Error("a subscribed user must complete the telegram quest")
	}
	if QuestDone(Quest{Kind: "telegram_subscribed", Target: 1}, c, phases) {
		t.Error("an unsubscribed user must not complete the telegram quest")
	}

	// A target of zero never completes. An admin saving a quest with no target,
	// or this binary meeting a kind it does not know, must not hand out a
	// reward to everyone the moment the profile screen loads.
	if QuestDone(Quest{Kind: "lessons_completed", Target: 0}, c, phases) {
		t.Error("a zero target must never be done")
	}
	if QuestDone(Quest{Kind: "kind_from_the_future", Target: 1}, c, phases) {
		t.Error("an unknown kind must never be done")
	}
}
