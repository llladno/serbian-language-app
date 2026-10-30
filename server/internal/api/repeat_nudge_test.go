package api

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/repeat"
	"github.com/grisha/serbian-app/server/internal/store"
)

// nudgeLearner links a Telegram chat to a fresh account that is (or is not) in the control group.
// Account ids are random, so it makes accounts until one falls in the wanted half.
func nudgeLearner(t *testing.T, st *store.Store, chatID int64, control bool) string {
	t.Helper()
	for i := 0; i < 200; i++ {
		uid, err := st.CreateUser("Оля")
		if err != nil {
			t.Fatal(err)
		}
		if repeat.InControl(uid) != control {
			continue
		}
		if err := st.CreateIdentity(store.Identity{
			ID: "idn_" + uid, UserID: uid, Provider: "telegram", ProviderUID: strconv.FormatInt(chatID, 10), TgUsername: "olya",
		}); err != nil {
			t.Fatal(err)
		}
		return uid
	}
	t.Fatal("no account fell in the wanted group")
	return ""
}

// studyingButNotRepeating makes uid a learner who repeated four days before now and studied ten
// hours before it: the learner the nudge is for.
func studyingButNotRepeating(t *testing.T, st *store.Store, uid string, now time.Time) {
	t.Helper()
	if err := st.User(uid).NoteRepeatOpened(now.Add(-4 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	studyAt(t, st, uid, now.Add(-10*time.Hour))
}

func studyAt(t *testing.T, st *store.Store, uid string, at time.Time) {
	t.Helper()
	if err := st.User(uid).AddAttempt(store.Attempt{
		ExerciseID: "e1", Lesson: "01", Block: "practice", Answer: "x", Correct: true,
	}, at); err != nil {
		t.Fatalf("AddAttempt: %v", err)
	}
}

func nudgeMessages(sink *tgSink) []sentTgMessage {
	var out []sentTgMessage
	for _, m := range sink.all() {
		if m.Button != nil && m.Button.RepeatNudgeURL != "" {
			out = append(out, m)
		}
	}
	return out
}

func TestSweepNudgesALearnerWhoStudiesButStoppedRepeating(t *testing.T) {
	d, st, sink := newReminderDeps(t)
	uid := nudgeLearner(t, st, 4242, false)
	studyingButNotRepeating(t, st, uid, fixedNow)

	if err := RunReminderSweep(d, fixedNow); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	got := nudgeMessages(sink)
	if len(got) != 1 || got[0].ChatID != 4242 {
		t.Fatalf("nudges = %+v, want exactly one to chat 4242", got)
	}
	if got[0].Button.RepeatNudgeURL != testBaseURL+"/repeat" || got[0].Button.Label != "Повторить" {
		t.Errorf("button = %+v, want «Повторить» opening the repeat screen", got[0].Button)
	}
	info, _ := st.RepeatNudgeInfo(uid)
	if info.Sent != 1 || !info.LastNudge.Equal(fixedNow) {
		t.Errorf("recorded = %+v, want one sent nudge at the sweep time", info)
	}
}

func TestSweepWritesOnceThenWaitsForTheLearnerToRepeat(t *testing.T) {
	d, st, sink := newReminderDeps(t)
	uid := nudgeLearner(t, st, 4242, false)
	studyingButNotRepeating(t, st, uid, fixedNow)

	_ = RunReminderSweep(d, fixedNow)
	studyAt(t, st, uid, fixedNow.Add(2*time.Hour))
	_ = RunReminderSweep(d, fixedNow.Add(3*time.Hour))
	_ = RunReminderSweep(d, fixedNow.Add(26*time.Hour))
	if got := nudgeMessages(sink); len(got) != 1 {
		t.Fatalf("%d nudges, want 1: ignoring the message means silence", len(got))
	}

	// They repeat on their own, then go quiet again while still studying: a new spell.
	_ = st.User(uid).NoteRepeatOpened(fixedNow.Add(30 * time.Hour))
	studyAt(t, st, uid, fixedNow.Add(88*time.Hour))
	_ = RunReminderSweep(d, fixedNow.Add(98*time.Hour)) // 12:00 + 98h = 14:00 UTC, 16:00 in Belgrade
	got := nudgeMessages(sink)
	if len(got) != 2 {
		t.Fatalf("%d nudges, want 2 after they repeated and lapsed again", len(got))
	}
	if got[0].Text == got[1].Text {
		t.Errorf("the same text twice in a row: %q", got[0].Text)
	}
}

func TestSweepNeverWritesToTheControlGroupButMarksTheSpell(t *testing.T) {
	d, st, sink := newReminderDeps(t)
	uid := nudgeLearner(t, st, 4242, true)
	studyingButNotRepeating(t, st, uid, fixedNow)

	_ = RunReminderSweep(d, fixedNow)
	_ = RunReminderSweep(d, fixedNow.Add(time.Hour))
	if got := nudgeMessages(sink); len(got) != 0 {
		t.Fatalf("control learner got %+v", got)
	}
	stats, _ := st.RepeatNudgeStats()
	n := 0
	for _, g := range stats.Groups {
		if g.Kind == store.NudgeControl {
			n += g.Count
		}
	}
	if n != 1 {
		t.Fatalf("%d control rows, want 1: one per spell, not one per sweep", n)
	}
}

func TestSweepSkipsWhoNeverRepeatedWhoOptedOutAndWhoSwitchedStudyMessagesOff(t *testing.T) {
	d, st, sink := newReminderDeps(t)
	never := nudgeLearner(t, st, 1, false)
	studyAt(t, st, never, fixedNow.Add(-10*time.Hour))

	off := nudgeLearner(t, st, 2, false)
	studyingButNotRepeating(t, st, off, fixedNow)
	if err := st.SetRepeatNudgeOff(off); err != nil {
		t.Fatal(err)
	}

	quiet := nudgeLearner(t, st, 3, false)
	studyingButNotRepeating(t, st, quiet, fixedNow)
	if err := st.SetBotPrefs(quiet, store.BotPrefs{Learning: false, Billing: true}, fixedNow); err != nil {
		t.Fatal(err)
	}

	_ = RunReminderSweep(d, fixedNow)
	if got := nudgeMessages(sink); len(got) != 0 {
		t.Fatalf("wrote to people who should hear nothing: %+v", got)
	}
}

func TestSweepNudgesOnlyInTheLearnersDaytime(t *testing.T) {
	d, st, sink := newReminderDeps(t)
	uid := nudgeLearner(t, st, 4242, false)
	night := time.Date(2026, 9, 6, 21, 0, 0, 0, time.UTC) // 23:00 in Belgrade
	studyingButNotRepeating(t, st, uid, night)

	_ = RunReminderSweep(d, night)
	if got := nudgeMessages(sink); len(got) != 0 {
		t.Fatalf("wrote at 23:00 local: %+v", got)
	}
	morning := time.Date(2026, 9, 7, 8, 0, 0, 0, time.UTC) // 10:00 in Belgrade
	studyAt(t, st, uid, morning.Add(-3*time.Hour))
	_ = RunReminderSweep(d, morning)
	if got := nudgeMessages(sink); len(got) != 1 {
		t.Fatalf("%d nudges at 10:00 local, want 1", len(got))
	}
}

func TestSweepSendsOneStudyMessageADay(t *testing.T) {
	d, st, sink := newReminderDeps(t)
	uid := nudgeLearner(t, st, 4242, false)
	studyingButNotRepeating(t, st, uid, fixedNow)
	if err := st.SetBotReminderAllDoneDate(uid, fixedNow.UTC().Format("2006-01-02")); err != nil {
		t.Fatal(err)
	}
	_ = RunReminderSweep(d, fixedNow)
	if got := nudgeMessages(sink); len(got) != 0 {
		t.Fatalf("a second study message went out the same day: %+v", got)
	}
}

func TestNudgeTextsAreWarmShortAndHaveOneSerbianWord(t *testing.T) {
	serbian := regexp.MustCompile(`\b(Hajde|bravo|Polako|polako)\b`)
	emoji := regexp.MustCompile(`[\x{1F300}-\x{1FAFF}\x{2600}-\x{27BF}]`)
	for v := 0; v < repeatNudgeVariants; v++ {
		text := repeatNudgeText(v, "Оля", 3)
		if n := len(serbian.FindAllString(text, -1)); n > 1 {
			t.Errorf("variant %d has %d Serbian words, want at most one: %q", v, n, text)
		}
		if n := len(emoji.FindAllString(text, -1)); n < 1 || n > 2 {
			t.Errorf("variant %d has %d emoji, want 1-2: %q", v, n, text)
		}
		if strings.Contains(text, "{") || strings.Contains(text, "%!") {
			t.Errorf("variant %d has an unfilled placeholder: %q", v, text)
		}
		if !strings.Contains(strings.ToLower(text), "пятнадцать") {
			t.Errorf("variant %d does not say how many exercises", v)
		}
	}
	if repeat.Size != 15 {
		t.Errorf("repeat.Size = %d: the nudge texts say «пятнадцать», update them", repeat.Size)
	}
}

func TestNudgeTextLeadsWithWhatGrabsAndFillsTheDays(t *testing.T) {
	if got := repeatNudgeText(0, "Оля", 2); !strings.HasPrefix(got, "Оля, 2 дня без повторения") {
		t.Errorf("variant 1 starts %q", got)
	}
	if got := repeatNudgeText(0, "", 5); !strings.HasPrefix(got, "5 дней без повторения") {
		t.Errorf("no name: starts %q, want it to drop the name and its comma", got)
	}
	if got := repeatNudgeText(1, "Оля", 3); !strings.HasPrefix(got, "Давай повторим сербский язык") {
		t.Errorf("variant 2 starts %q, want the challenge", got)
	}
	if got := repeatNudgeText(2, "Оля", 21); !strings.Contains(got, "21 день без повторения") {
		t.Errorf("variant 3 = %q, want «21 день»", got)
	}
}

func TestOpeningRepeatIsRecordedForFreeAndUnlimitedLearners(t *testing.T) {
	h, st, now := movableAPI(t)
	u, err := st.UserByName("tester")
	if err != nil {
		t.Fatal(err)
	}
	do(h, "POST", "/api/lessons/01/exercises/01-A-2/check", `{"answer":"x"}`)
	*now = now.Add(24 * time.Hour)
	if rr := do(h, "GET", "/api/repeat", ""); rr.Code != 200 {
		t.Fatalf("GET /api/repeat = %d %s", rr.Code, rr.Body)
	}
	info, _ := st.RepeatNudgeInfo(u.ID)
	if !info.LastRepeat.Equal(*now) {
		t.Fatalf("LastRepeat = %v, want %v", info.LastRepeat, *now)
	}
}

func TestOptOutButtonSwitchesOffOnlyTheRepeatNudge(t *testing.T) {
	h, st, _ := newTelegramBotAPI(t)
	bot := newFakeBotAPI(t, "left")
	uid := linkForChannel(t, h, st, `{}`, 7010)

	postWebhook(h, tgWebhookSecret, callbackOnMessage(7010, "nudgeoff:repeat", 777))

	info, _ := st.RepeatNudgeInfo(uid)
	if !info.OptedOut {
		t.Fatal("the nudge is still on")
	}
	if got, _ := st.BotPrefs(uid); got != store.DefaultBotPrefs {
		t.Errorf("prefs = %+v: the other study messages must stay on", got)
	}
	if a := bot.lastAnswer(); !strings.Contains(a, "повторен") {
		t.Errorf("notice = %q, want it to say what was switched off", a)
	}
	bot.mu.Lock()
	defer bot.mu.Unlock()
	if len(bot.edits) != 1 || !strings.Contains(bot.edits[0], `"inline_keyboard":[]`) {
		t.Fatalf("keyboard edits = %v, want the buttons taken off", bot.edits)
	}
}

func TestOptOutButtonFromAnUnlinkedTelegramIsRefused(t *testing.T) {
	h, st, _ := newTelegramBotAPI(t)
	bot := newFakeBotAPI(t, "left")
	uid := linkForChannel(t, h, st, `{}`, 7010)

	postWebhook(h, tgWebhookSecret, callbackOnMessage(7999, "nudgeoff:repeat", 1))

	if info, _ := st.RepeatNudgeInfo(uid); info.OptedOut {
		t.Fatal("an unlinked Telegram switched off someone's nudge")
	}
	if a := bot.lastAnswer(); !strings.Contains(a, "привяжи Telegram") {
		t.Errorf("notice = %q", a)
	}
}
