package api

import (
	"strconv"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/config"
	"github.com/grisha/serbian-app/server/internal/content"
	"github.com/grisha/serbian-app/server/internal/srs"
	"github.com/grisha/serbian-app/server/internal/store"
)

// newReminderDeps builds Deps around a fresh in-memory store and the small
// testdata course fixture (4 vocab words, no false friends reachable at
// beginner level) — enough to drive a review queue to empty without pages of
// setup. Returns the Deps, the store (for direct assertions/seeding) and a
// capturing EnqueueTelegramMessage sink.
func newReminderDeps(t *testing.T) (Deps, *store.Store, *tgSink) {
	t.Helper()
	c, err := content.Load("../content/testdata/content")
	if err != nil {
		t.Fatalf("load fixture content: %v", err)
	}
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	sink := &tgSink{}
	d := Deps{
		Course:                 func() *content.Course { return c },
		Store:                  st,
		Now:                    func() time.Time { return fixedNow },
		Stale:                  func() bool { return false },
		Config:                 config.Config{AppBaseURL: testBaseURL},
		EnqueueTelegramMessage: sink.enqueue,
	}
	return d, st, sink
}

// linkTelegram creates an account with a linked (non-pending) Telegram chat
// and returns its user id.
func linkTelegram(t *testing.T, st *store.Store, chatID int64, name string) string {
	t.Helper()
	uid, err := st.CreateUser(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateIdentity(store.Identity{
		ID: "idn_" + name, UserID: uid, Provider: "telegram",
		ProviderUID: strconv.FormatInt(chatID, 10), TgUsername: name,
	}); err != nil {
		t.Fatal(err)
	}
	return uid
}

var allTestVocabCards = []store.CardSeed{
	{CardID: "vocab:zdravo", Kind: "vocab", RefID: "zdravo"},
	{CardID: "vocab:svet", Kind: "vocab", RefID: "svet"},
	{CardID: "vocab:dobar-dan", Kind: "vocab", RefID: "dobar-dan"},
	{CardID: "vocab:raditi", Kind: "vocab", RefID: "raditi"},
	// The fixture's one grammar.yaml point — it rides the same
	// beginner-level "new" budget as vocab (see gramPerDay), so it must be
	// graded too for the queue to actually go empty.
	{CardID: "gram:prezent-am", Kind: "gram", RefID: "prezent-am"},
}

// gradeAllGood seeds and grades every fixture vocab/grammar card Good, which
// is enough to empty the review queue at beginner level (false friends never
// enter it with only 4 vocab words — see beginnerWordCount).
func gradeAllGood(t *testing.T, us *store.UserStore, at time.Time) {
	t.Helper()
	if err := us.EnsureCards(allTestVocabCards); err != nil {
		t.Fatalf("EnsureCards: %v", err)
	}
	for _, seed := range allTestVocabCards {
		if _, err := us.GradeCard(seed.CardID, srs.Good, at); err != nil {
			t.Fatalf("GradeCard %s: %v", seed.CardID, err)
		}
	}
}

func TestRunReminderSweepAllDoneCongratsOnce(t *testing.T) {
	d, st, sink := newReminderDeps(t)
	uid := linkTelegram(t, st, 4242, "grisha")
	now := fixedNow

	// Fresh account: the queue still has new words to introduce, so no
	// congrats yet.
	if err := RunReminderSweep(d, now); err != nil {
		t.Fatalf("sweep 1: %v", err)
	}
	if got := sink.all(); len(got) != 0 {
		t.Fatalf("sweep 1 sent %+v, want none (queue not empty yet)", got)
	}

	gradeAllGood(t, st.User(uid), now)

	if err := RunReminderSweep(d, now); err != nil {
		t.Fatalf("sweep 2: %v", err)
	}
	if got := sink.all(); len(got) != 1 || got[0].ChatID != 4242 {
		t.Fatalf("sweep 2 sent %+v, want exactly one message to chat 4242", got)
	}
	state, err := st.BotReminderState(uid)
	if err != nil {
		t.Fatalf("BotReminderState: %v", err)
	}
	if want := now.UTC().Format("2006-01-02"); state.AllDoneDate != want {
		t.Fatalf("AllDoneDate = %q, want %q", state.AllDoneDate, want)
	}

	// Same day, queue still empty: must not send a second congrats.
	if err := RunReminderSweep(d, now.Add(time.Hour)); err != nil {
		t.Fatalf("sweep 3: %v", err)
	}
	if got := sink.all(); len(got) != 1 {
		t.Fatalf("sweep 3 sent %d messages total, want still 1 (same-day dedup)", len(got))
	}
}

func TestRunReminderSweepInactivityNudgeOncePerGap(t *testing.T) {
	d, st, sink := newReminderDeps(t)
	uid := linkTelegram(t, st, 777, "olya")

	t0 := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	// One attempt establishes "last activity" without clearing the queue
	// (so the all-done congrats never fires and can't muddy the count).
	if err := st.User(uid).AddAttempt(store.Attempt{
		ExerciseID: "e1", Lesson: "01", Block: "practice", Answer: "x", Correct: true,
	}, t0); err != nil {
		t.Fatalf("AddAttempt: %v", err)
	}

	if err := RunReminderSweep(d, t0.Add(10*time.Hour)); err != nil {
		t.Fatalf("sweep @10h: %v", err)
	}
	if got := sink.all(); len(got) != 0 {
		t.Fatalf("sweep @10h sent %+v, want none (under 24h)", got)
	}

	if err := RunReminderSweep(d, t0.Add(25*time.Hour)); err != nil {
		t.Fatalf("sweep @25h: %v", err)
	}
	if got := sink.all(); len(got) != 1 || got[0].ChatID != 777 {
		t.Fatalf("sweep @25h sent %+v, want exactly one nudge to chat 777", got)
	}

	// Still quiet: must not repeat the nudge.
	if err := RunReminderSweep(d, t0.Add(26*time.Hour)); err != nil {
		t.Fatalf("sweep @26h: %v", err)
	}
	if got := sink.all(); len(got) != 1 {
		t.Fatalf("sweep @26h sent %d messages total, want still 1 (no repeat while quiet)", len(got))
	}

	// User returns...
	returnAt := t0.Add(30 * time.Hour)
	if err := st.User(uid).AddAttempt(store.Attempt{
		ExerciseID: "e2", Lesson: "01", Block: "practice", Answer: "x", Correct: true,
	}, returnAt); err != nil {
		t.Fatalf("AddAttempt (return): %v", err)
	}
	// ...briefly, then goes quiet again: no nudge until 24h have passed since
	// this new activity.
	if err := RunReminderSweep(d, returnAt.Add(time.Minute)); err != nil {
		t.Fatalf("sweep just after return: %v", err)
	}
	if got := sink.all(); len(got) != 1 {
		t.Fatalf("sweep just after return sent %d total, want still 1", len(got))
	}

	if err := RunReminderSweep(d, returnAt.Add(25*time.Hour)); err != nil {
		t.Fatalf("sweep 25h after return: %v", err)
	}
	if got := sink.all(); len(got) != 2 {
		t.Fatalf("sweep 25h after return sent %d total, want 2 (a fresh quiet spell earns a fresh nudge)", len(got))
	}
}

func TestRunReminderSweepSkipsNeverActiveAccount(t *testing.T) {
	d, st, sink := newReminderDeps(t)
	linkTelegram(t, st, 999, "new-user")

	if err := RunReminderSweep(d, fixedNow.Add(30*24*time.Hour)); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if got := sink.all(); len(got) != 0 {
		t.Fatalf("sent %+v for an account with no activity ever, want none", got)
	}
}
