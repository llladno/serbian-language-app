package store

import (
	"testing"
	"time"
)

func TestRepeatNudgeInfoStartsEmpty(t *testing.T) {
	s, _ := newUser(t)
	uid, _ := s.CreateUser("Оля")
	info, err := s.RepeatNudgeInfo(uid)
	if err != nil {
		t.Fatalf("RepeatNudgeInfo: %v", err)
	}
	if !info.LastRepeat.IsZero() || !info.LastNudge.IsZero() || info.Sent != 0 || info.OptedOut {
		t.Fatalf("fresh account = %+v, want all empty", info)
	}
}

func TestOpeningRepeatIsRememberedEvenWithoutAStoredBatch(t *testing.T) {
	s, u := newUser(t)
	at := day0.Add(30 * time.Hour)
	// A learner with unlimited repeats never gets a repeat_sessions row, so the open must be
	// recorded on its own.
	if err := u.NoteRepeatOpened(at); err != nil {
		t.Fatalf("NoteRepeatOpened: %v", err)
	}
	info, _ := s.RepeatNudgeInfo(u.user)
	if !info.LastRepeat.Equal(at) {
		t.Fatalf("LastRepeat = %v, want %v", info.LastRepeat, at)
	}
	later := at.Add(5 * time.Hour)
	_ = u.NoteRepeatOpened(later)
	info, _ = s.RepeatNudgeInfo(u.user)
	if !info.LastRepeat.Equal(later) {
		t.Fatalf("LastRepeat = %v, want the newer %v", info.LastRepeat, later)
	}
}

func TestRepeatNudgeInfoCountsRepeatsMadeBeforeTheNudgeExisted(t *testing.T) {
	s, u := newUser(t)
	// Batches stored by the first release of «Повторить» are the only trace of those repeats.
	if _, err := u.StartRepeatSession("2026-10-27", []string{"01|01.1.1"}, day0); err != nil {
		t.Fatalf("StartRepeatSession: %v", err)
	}
	info, _ := s.RepeatNudgeInfo(u.user)
	if !info.LastRepeat.Equal(day0) {
		t.Fatalf("LastRepeat = %v, want %v from the stored batch", info.LastRepeat, day0)
	}
}

func TestRecordedNudgesCountAndRemember(t *testing.T) {
	s, u := newUser(t)
	if _, err := s.RecordRepeatNudge(u.user, 1, NudgeSent, day0, time.Time{}); err != nil {
		t.Fatalf("RecordRepeatNudge: %v", err)
	}
	// The learner repeated in between, so this is a new spell.
	if _, err := s.RecordRepeatNudge(u.user, 2, NudgeControl, day0.Add(time.Hour), day0.Add(30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	info, _ := s.RepeatNudgeInfo(u.user)
	if info.Sent != 1 {
		t.Fatalf("Sent = %d, want 1: a control-group row is not a message", info.Sent)
	}
	if !info.LastNudge.Equal(day0.Add(time.Hour)) {
		t.Fatalf("LastNudge = %v, want the control row too (it starts a spell)", info.LastNudge)
	}
}

func TestOptOutIsRemembered(t *testing.T) {
	s, u := newUser(t)
	if err := s.SetRepeatNudgeOff(u.user); err != nil {
		t.Fatalf("SetRepeatNudgeOff: %v", err)
	}
	// Opening «Повторить» afterwards must not switch it back on.
	_ = u.NoteRepeatOpened(day0)
	info, _ := s.RepeatNudgeInfo(u.user)
	if !info.OptedOut {
		t.Fatal("opt-out lost after the learner opened «Повторить»")
	}
}

func TestOpeningRepeatMarksTheLatestNudgeAsAnswered(t *testing.T) {
	s, u := newUser(t)
	_, _ = s.RecordRepeatNudge(u.user, 0, NudgeSent, day0, time.Time{})
	_ = u.NoteRepeatOpened(day0.Add(3 * time.Hour))
	// A second open must not move the answer time: the first one after the message counts.
	_ = u.NoteRepeatOpened(day0.Add(60 * time.Hour))

	stats, err := s.RepeatNudgeStats()
	if err != nil {
		t.Fatalf("RepeatNudgeStats: %v", err)
	}
	g := findNudgeGroup(stats, NudgeSent, 0)
	if g.Count != 1 || g.OpenedWithin24h != 1 {
		t.Fatalf("variant 0 = %+v, want 1 sent and 1 opened within a day", g)
	}
}

func TestStatsSeparateLateOpensAndControlGroup(t *testing.T) {
	s, _ := newUser(t)
	a, _ := s.CreateUser("А")
	b, _ := s.CreateUser("Б")
	c, _ := s.CreateUser("В")
	_, _ = s.RecordRepeatNudge(a, 1, NudgeSent, day0, time.Time{})
	_, _ = s.RecordRepeatNudge(b, 1, NudgeSent, day0, time.Time{})
	_, _ = s.RecordRepeatNudge(c, 0, NudgeControl, day0, time.Time{})
	_ = s.User(a).NoteRepeatOpened(day0.Add(2 * time.Hour))  // within a day
	_ = s.User(b).NoteRepeatOpened(day0.Add(50 * time.Hour)) // too late to be the message's doing
	_ = s.User(c).NoteRepeatOpened(day0.Add(5 * time.Hour))  // control learner repeats on their own
	_ = s.SetRepeatNudgeOff(b)

	stats, _ := s.RepeatNudgeStats()
	if g := findNudgeGroup(stats, NudgeSent, 1); g.Count != 2 || g.OpenedWithin24h != 1 {
		t.Fatalf("sent variant 1 = %+v, want 2 sent, 1 opened within 24h", g)
	}
	if g := findNudgeGroup(stats, NudgeControl, 0); g.Count != 1 || g.OpenedWithin24h != 1 {
		t.Fatalf("control = %+v, want 1 learner who repeated on their own", g)
	}
	if stats.OptedOut != 1 {
		t.Fatalf("OptedOut = %d, want 1", stats.OptedOut)
	}
}

// findNudgeGroup returns the stats row for one kind and variant, zero when there is none.
func findNudgeGroup(stats RepeatNudgeStats, kind string, variant int) RepeatNudgeGroup {
	for _, g := range stats.Groups {
		if g.Kind == kind && g.Variant == variant {
			return g
		}
	}
	return RepeatNudgeGroup{}
}

func TestRecordRepeatNudgeWritesOnlyOnceASpellEvenWhenTwoSweepsRace(t *testing.T) {
	s, u := newUser(t)
	lastRepeat := day0.Add(-72 * time.Hour)

	first, err := s.RecordRepeatNudge(u.user, 1, NudgeSent, day0, lastRepeat)
	if err != nil || !first {
		t.Fatalf("first = %v, %v; want it written", first, err)
	}
	// A second sweep (another container during a deploy) that read the state before the first
	// one wrote must not add a second row.
	second, err := s.RecordRepeatNudge(u.user, 2, NudgeSent, day0.Add(time.Minute), lastRepeat)
	if err != nil || second {
		t.Fatalf("second = %v, %v; want it refused", second, err)
	}
	if info, _ := s.RepeatNudgeInfo(u.user); info.Sent != 1 {
		t.Fatalf("Sent = %d, want 1", info.Sent)
	}

	// After the learner repeats again a new spell begins and a new row is allowed.
	third, err := s.RecordRepeatNudge(u.user, 2, NudgeSent, day0.Add(96*time.Hour), day0.Add(48*time.Hour))
	if err != nil || !third {
		t.Fatalf("third = %v, %v; want a new spell to be written", third, err)
	}
}
