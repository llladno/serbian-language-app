package repeat

import (
	"testing"
	"time"
)

// A Tuesday, 14:00 in the learner's own timezone: inside the send window.
var nudgeNow = time.Date(2026, 11, 3, 14, 0, 0, 0, time.UTC)

// dueInput is a learner who is due a nudge: repeated four days ago, studied yesterday, never nudged.
func dueInput() NudgeInput {
	return NudgeInput{
		Now:          nudgeNow,
		LastRepeat:   nudgeNow.Add(-4 * 24 * time.Hour),
		LastActivity: nudgeNow.Add(-20 * time.Hour),
	}
}

func TestNudgeDueWhenRepeatedBeforeAndStillStudying(t *testing.T) {
	if !NudgeDue(dueInput()) {
		t.Fatal("a learner who repeated 4 days ago and studied yesterday should get a nudge")
	}
}

func TestNudgeNeedsAPriorRepeat(t *testing.T) {
	in := dueInput()
	in.LastRepeat = time.Time{}
	if NudgeDue(in) {
		t.Fatal("someone who never opened «Повторить» must not be nudged: the message would advertise a feature they never saw")
	}
}

func TestNudgeWaitsForTwoDays(t *testing.T) {
	in := dueInput()
	in.LastRepeat = nudgeNow.Add(-47 * time.Hour)
	if NudgeDue(in) {
		t.Fatal("47 hours since the last repeat is too early")
	}
	in.LastRepeat = nudgeNow.Add(-48 * time.Hour)
	if !NudgeDue(in) {
		t.Fatal("exactly 48 hours is due")
	}
}

func TestNudgeOnlyForLearnersWhoAreStillStudying(t *testing.T) {
	in := dueInput()
	in.LastActivity = nudgeNow.Add(-3 * 24 * time.Hour)
	if NudgeDue(in) {
		t.Fatal("someone gone for 3 days gets the «come back» nudge, not this one")
	}
	in.LastActivity = time.Time{}
	if NudgeDue(in) {
		t.Fatal("no activity at all is not a learner who skips repeating")
	}
}

func TestNudgeOncePerQuietSpell(t *testing.T) {
	in := dueInput()
	in.LastNudge = nudgeNow.Add(-5 * time.Hour) // after the last repeat
	if NudgeDue(in) {
		t.Fatal("already nudged since the last repeat: stay silent until they repeat again")
	}
	// They repeated after that nudge and then went quiet again: a new spell.
	in.LastNudge = nudgeNow.Add(-10 * 24 * time.Hour)
	if !NudgeDue(in) {
		t.Fatal("a repeat after the last nudge starts a new spell, so the next nudge is allowed")
	}
}

func TestNudgeOnlyInDaytimeInTheLearnersZone(t *testing.T) {
	cases := []struct {
		hour int
		want bool
	}{{9, false}, {10, true}, {19, true}, {20, false}, {23, false}, {3, false}}
	for _, c := range cases {
		in := dueInput()
		in.Now = time.Date(2026, 11, 3, c.hour, 30, 0, 0, time.UTC)
		in.LastRepeat = in.Now.Add(-4 * 24 * time.Hour)
		in.LastActivity = in.Now.Add(-20 * time.Hour)
		if got := NudgeDue(in); got != c.want {
			t.Errorf("hour %d: NudgeDue = %v, want %v", c.hour, got, c.want)
		}
	}
}

func TestNudgeHonoursOptOutAndTheDailyCap(t *testing.T) {
	in := dueInput()
	in.OptedOut = true
	if NudgeDue(in) {
		t.Fatal("someone who switched this nudge off hears nothing")
	}
	in = dueInput()
	in.OtherLearningToday = true
	if NudgeDue(in) {
		t.Fatal("only one study message a day: another one already went out today")
	}
}

func TestNudgeVariantRotatesAndDiffersBetweenLearners(t *testing.T) {
	seen := map[int]bool{}
	for sent := 0; sent < 3; sent++ {
		v := NudgeVariant("u1", sent, 3)
		if v < 0 || v > 2 {
			t.Fatalf("variant %d out of range", v)
		}
		seen[v] = true
	}
	if len(seen) != 3 {
		t.Fatalf("three nudges in a row used variants %v, want all three", seen)
	}
	first := map[int]int{}
	for i := 0; i < 300; i++ {
		first[NudgeVariant("user-"+string(rune('a'+i%26))+string(rune('a'+i/26%26)), 0, 3)]++
	}
	for v := 0; v < 3; v++ {
		if first[v] == 0 {
			t.Fatalf("variant %d is never anyone's first: the experiment would be lopsided (%v)", v, first)
		}
	}
}

func TestControlGroupIsStableAndRoughlyHalf(t *testing.T) {
	if InControl("u1") != InControl("u1") {
		t.Fatal("a learner must stay in the same group")
	}
	n := 0
	for i := 0; i < 1000; i++ {
		if InControl("user-" + string(rune('a'+i%26)) + string(rune('a'+i/26%26)) + string(rune('a'+i/676))) {
			n++
		}
	}
	if n < 400 || n > 600 {
		t.Fatalf("control group holds %d of 1000, want about half", n)
	}
}

// introInput is a learner who studies but has never opened «Повторить»: the one-off introduction is for them.
func introInput() IntroInput {
	return IntroInput{Now: nudgeNow, LastActivity: nudgeNow.Add(-5 * 24 * time.Hour)}
}

func TestIntroDueForAStudentWhoNeverOpenedRepeat(t *testing.T) {
	if !IntroDue(introInput()) {
		t.Fatal("a learner who studied and never opened «Повторить» should get the introduction, however long ago they studied")
	}
}

func TestIntroSkipsWhoOpenedRepeatWhoNeverStudiedWhoGotItAndWhoHeardTodayAndNight(t *testing.T) {
	cases := map[string]func(*IntroInput){
		"opened repeat": func(in *IntroInput) { in.LastRepeat = nudgeNow.Add(-time.Hour) },
		"never studied": func(in *IntroInput) { in.LastActivity = time.Time{} },
		"already sent":  func(in *IntroInput) { in.AlreadySent = true },
		"other message": func(in *IntroInput) { in.OtherLearningToday = true },
		"before 10":     func(in *IntroInput) { in.Now = time.Date(2026, 11, 3, 9, 59, 0, 0, time.UTC) },
		"from 20":       func(in *IntroInput) { in.Now = time.Date(2026, 11, 3, 20, 0, 0, 0, time.UTC) },
	}
	for name, change := range cases {
		in := introInput()
		change(&in)
		if IntroDue(in) {
			t.Errorf("%s: introduction due, want not", name)
		}
	}
}
