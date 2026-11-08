package store

import (
	"fmt"
	"testing"
	"time"
)

var campaignNow = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

// campaignUser makes an account registered at the given time, with no identity yet.
func campaignUser(t *testing.T, s *Store, name string, registered time.Time) string {
	t.Helper()
	id, err := s.CreateUser(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE users SET created_at = ? WHERE id = ?`, stamp(registered), id); err != nil {
		t.Fatal(err)
	}
	return id
}

func campaignTelegram(t *testing.T, s *Store, id string, chat int64) {
	t.Helper()
	if err := s.CreateIdentity(Identity{ID: "tg_" + id, UserID: id, Provider: "telegram", ProviderUID: fmt.Sprint(chat)}); err != nil {
		t.Fatal(err)
	}
}

func campaignEmail(t *testing.T, s *Store, id, email string, verified bool) {
	t.Helper()
	in := Identity{ID: "pw_" + id, UserID: id, Provider: "password", ProviderUID: email, Email: email, PasswordHash: "x"}
	if verified {
		in.EmailVerifiedAt = stamp(campaignNow.Add(-72 * time.Hour))
	}
	if err := s.CreateIdentity(in); err != nil {
		t.Fatal(err)
	}
}

func campaignLesson(t *testing.T, s *Store, id, lesson, status string, at time.Time) {
	t.Helper()
	if err := s.User(id).SetLessonStatus(lesson, status, at); err != nil {
		t.Fatal(err)
	}
}

func audienceByUser(t *testing.T, s *Store) map[string]CampaignCandidate {
	t.Helper()
	got, err := s.CampaignAudience("test", campaignNow)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]CampaignCandidate{}
	for _, c := range got {
		m[c.UserID] = c
	}
	return m
}

func TestCampaignAudienceSegmentsAndChannels(t *testing.T) {
	s := newStore(t)
	old := campaignNow.Add(-10 * 24 * time.Hour)

	nothing := campaignUser(t, s, "nothing", old)
	campaignTelegram(t, s, nothing, 111)
	started := campaignUser(t, s, "started", old)
	campaignTelegram(t, s, started, 222)
	campaignLesson(t, s, started, "00", "in_progress", old)
	done00 := campaignUser(t, s, "done00", old)
	campaignTelegram(t, s, done00, 333)
	campaignLesson(t, s, done00, "00", "done", old)
	done01 := campaignUser(t, s, "done01", old)
	campaignEmail(t, s, done01, "d01@example.com", true)
	campaignLesson(t, s, done01, "00", "done", old)
	campaignLesson(t, s, done01, "01", "done", old)
	reached02 := campaignUser(t, s, "reached02", old)
	campaignTelegram(t, s, reached02, 444)
	campaignLesson(t, s, reached02, "02", "in_progress", old)
	fresh := campaignUser(t, s, "fresh", campaignNow.Add(-5*time.Hour))
	campaignTelegram(t, s, fresh, 555)
	unverified := campaignUser(t, s, "unverified", old)
	campaignEmail(t, s, unverified, "u@example.com", false)
	nowhere := campaignUser(t, s, "nowhere", old)
	_ = nowhere

	got := audienceByUser(t, s)
	want := map[string]struct {
		segment int
		channel string
	}{
		nothing: {1, ChannelTelegram}, started: {2, ChannelTelegram}, done00: {3, ChannelTelegram}, done01: {4, ChannelEmail},
	}
	if len(got) != len(want) {
		t.Errorf("audience = %+v, want exactly %d people", got, len(want))
	}
	for id, w := range want {
		c, ok := got[id]
		if !ok {
			t.Errorf("%s missing from the audience", id)
			continue
		}
		if c.Segment != w.segment || c.Channel != w.channel {
			t.Errorf("%s: segment %d channel %s, want %d %s", id, c.Segment, c.Channel, w.segment, w.channel)
		}
	}
	if c := got[nothing]; c.ChatID != 111 {
		t.Errorf("telegram chat = %d, want 111", c.ChatID)
	}
	if c := got[done01]; c.Email != "d01@example.com" {
		t.Errorf("email = %q", c.Email)
	}
	for _, id := range []string{reached02, fresh, unverified, nowhere} {
		if _, ok := got[id]; ok {
			t.Errorf("%s should not be in the audience", id)
		}
	}
}

func TestCampaignAudienceLeavesOutThoseWhoAreStillAround(t *testing.T) {
	s := newStore(t)
	old := campaignNow.Add(-10 * 24 * time.Hour)
	recent := campaignUser(t, s, "recent", old)
	campaignTelegram(t, s, recent, 1)
	if err := s.User(recent).AddAttempt(Attempt{ExerciseID: "e", Lesson: "00", Block: "practice", Answer: "x", Correct: true}, campaignNow.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	gone := campaignUser(t, s, "gone", old)
	campaignTelegram(t, s, gone, 2)
	if err := s.User(gone).AddAttempt(Attempt{ExerciseID: "e", Lesson: "00", Block: "practice", Answer: "x", Correct: true}, campaignNow.Add(-72*time.Hour)); err != nil {
		t.Fatal(err)
	}
	got := audienceByUser(t, s)
	if _, ok := got[recent]; ok {
		t.Error("someone who studied yesterday should not be called back")
	}
	if _, ok := got[gone]; !ok {
		t.Error("someone quiet for three days should be in the audience")
	}
}

func TestCampaignAudienceRespectsOptOutsAndBlockedChats(t *testing.T) {
	s := newStore(t)
	old := campaignNow.Add(-10 * 24 * time.Hour)

	off := campaignUser(t, s, "off", old) // reminders off in the bot; no e-mail: not reachable at all
	campaignTelegram(t, s, off, 10)
	if err := s.SetBotPrefs(off, BotPrefs{Learning: false}, old); err != nil {
		t.Fatal(err)
	}
	offWithMail := campaignUser(t, s, "offwithmail", old) // reminders off, but a verified e-mail: reached by e-mail
	campaignTelegram(t, s, offWithMail, 11)
	campaignEmail(t, s, offWithMail, "ow@example.com", true)
	if err := s.SetBotPrefs(offWithMail, BotPrefs{Learning: false}, old); err != nil {
		t.Fatal(err)
	}
	blocked := campaignUser(t, s, "blocked", old)
	campaignTelegram(t, s, blocked, 12)
	if err := s.EnqueueBotMessage(12, "x", OutboxButton{}, PriorityNormal, old); err != nil {
		t.Fatal(err)
	}
	pending, found, err := s.NextPendingOutboxMessage()
	if err != nil || !found {
		t.Fatalf("queued message: %v %v", found, err)
	}
	if err := s.MarkOutboxFailed(pending.ID, "telegram: 403 Forbidden: bot was blocked by the user"); err != nil {
		t.Fatal(err)
	}
	unsub := campaignUser(t, s, "unsub", old)
	campaignEmail(t, s, unsub, "un@example.com", true)
	if err := s.SetEmailOptout(unsub, old); err != nil {
		t.Fatal(err)
	}

	got := audienceByUser(t, s)
	if _, ok := got[off]; ok {
		t.Error("reminders switched off and no e-mail: should be left out")
	}
	if c := got[offWithMail]; c.Channel != ChannelEmail {
		t.Errorf("reminders off but e-mail verified: channel = %q, want email", c.Channel)
	}
	if _, ok := got[blocked]; ok {
		t.Error("a chat that blocked the bot should be left out")
	}
	if _, ok := got[unsub]; ok {
		t.Error("an unsubscribed e-mail should be left out")
	}
}

func freezeAll(t *testing.T, s *Store, n int, holdoutEvery int) []string {
	t.Helper()
	var ids []string
	var rows []CampaignRow
	for i := 0; i < n; i++ {
		id := campaignUser(t, s, fmt.Sprintf("p%02d", i), campaignNow.Add(-10*24*time.Hour))
		ids = append(ids, id)
		kind := CampaignSent
		if holdoutEvery > 0 && i%holdoutEvery == 0 {
			kind = CampaignHoldout
		}
		rows = append(rows, CampaignRow{UserID: id, Channel: ChannelTelegram, Segment: 1 + i%4, Kind: kind, Ord: int64(i)})
	}
	if err := s.CampaignFreeze("test", rows, campaignNow); err != nil {
		t.Fatal(err)
	}
	return ids
}

func TestCampaignFreezeTakeAndRecordWaves(t *testing.T) {
	s := newStore(t)
	ids := freezeAll(t, s, 10, 5) // p00 and p05 are held out
	if frozen, _ := s.CampaignFrozen("test"); !frozen {
		t.Fatal("campaign should be frozen")
	}
	// Freezing again changes nothing.
	if err := s.CampaignFreeze("test", []CampaignRow{{UserID: ids[0], Channel: ChannelTelegram, Segment: 1, Kind: CampaignSent, Ord: 99}}, campaignNow); err != nil {
		t.Fatal(err)
	}

	// A dry run says who would be taken and writes nothing.
	dry, err := s.CampaignTake("test", ChannelTelegram, 1, 3, func(string) int { return 1 }, false)
	if err != nil || len(dry) != 3 {
		t.Fatalf("dry take = %v, %v", dry, err)
	}
	if due, _ := s.CampaignDue("test"); len(due) != 0 {
		t.Errorf("a dry run left %d rows due", len(due))
	}

	taken, err := s.CampaignTake("test", ChannelTelegram, 1, 3, func(id string) int {
		if id == ids[1] {
			return 0
		}
		return 1
	}, true)
	if err != nil || len(taken) != 3 {
		t.Fatalf("take = %v, %v", taken, err)
	}
	// Order is the frozen order, and held-out people (p00, p05) are never taken.
	if taken[0].UserID != ids[1] || taken[1].UserID != ids[2] || taken[2].UserID != ids[3] {
		t.Errorf("taken = %+v, want p01, p02, p03 in order", taken)
	}
	if taken[0].Style != 0 || taken[1].Style != 1 {
		t.Errorf("styles = %d %d, want 0 1", taken[0].Style, taken[1].Style)
	}
	if err := s.CampaignRecordWave("test", 1, ChannelTelegram, 0, 500, campaignNow); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.CampaignWavesOpened("test"); n != 1 {
		t.Errorf("waves opened = %d, want 1", n)
	}

	due, err := s.CampaignDue("test")
	if err != nil || len(due) != 3 {
		t.Fatalf("due = %v, %v", due, err)
	}
	for _, d := range due {
		if len(d.Token) != 16 {
			t.Errorf("token %q should be 16 hex characters", d.Token)
		}
	}
	if n, _ := s.CampaignUnsent("test"); n != 3 {
		t.Errorf("unsent = %d, want 3", n)
	}

	// The next wave takes everyone left, never the people already taken.
	rest, err := s.CampaignTake("test", ChannelTelegram, 2, -1, func(string) int { return 0 }, true)
	if err != nil || len(rest) != 5 { // 10 - 2 held out - 3 taken
		t.Fatalf("rest = %d, %v, want 5", len(rest), err)
	}
	for _, r := range rest {
		if r.UserID == ids[0] || r.UserID == ids[5] {
			t.Errorf("held-out %s was taken", r.UserID)
		}
	}
}

func TestCampaignClicksStatsAndSkips(t *testing.T) {
	s := newStore(t)
	ids := freezeAll(t, s, 6, 0)
	if _, err := s.CampaignTake("test", ChannelTelegram, 1, -1, func(id string) int {
		if id == ids[0] || id == ids[1] || id == ids[2] {
			return 0
		}
		return 1
	}, true); err != nil {
		t.Fatal(err)
	}
	due, _ := s.CampaignDue("test")
	if len(due) != 6 {
		t.Fatalf("due = %d, want 6", len(due))
	}
	byUser := map[string]CampaignDueRow{}
	for _, d := range due {
		byUser[d.UserID] = d
	}
	for _, id := range ids[:5] {
		if err := s.CampaignMarkSent(byUser[id].ID, campaignNow); err != nil {
			t.Fatal(err)
		}
	}
	// One person can no longer be reached: skipped, so the waves do not wait for them.
	if err := s.CampaignSkip(byUser[ids[5]].ID); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.CampaignUnsent("test"); n != 0 {
		t.Errorf("unsent = %d, want 0 once everyone is sent or skipped", n)
	}

	// Style 0 got p00..p02, style 1 got p03..p04 (p05 skipped). One click for each of p00 and p03, a second press is not counted.
	if first, err := s.NoteCampaignClick(byUser[ids[0]].Token, campaignNow); err != nil || !first {
		t.Fatalf("first click = %v, %v", first, err)
	}
	if again, _ := s.NoteCampaignClick(byUser[ids[0]].Token, campaignNow); again {
		t.Error("a second press must not count")
	}
	if first, _ := s.NoteCampaignClick(byUser[ids[3]].Token, campaignNow); !first {
		t.Error("p03's click was not counted")
	}
	if ok, _ := s.NoteCampaignClick("nosuchtoken", campaignNow); ok {
		t.Error("an unknown token must be ignored")
	}
	stats, err := s.CampaignStyleStats("test", ChannelTelegram)
	if err != nil {
		t.Fatal(err)
	}
	if stats[0] != (CampaignStyleStat{Sent: 3, Clicked: 1}) || stats[1] != (CampaignStyleStat{Sent: 2, Clicked: 1}) {
		t.Errorf("stats = %+v, want style 0: 3 sent 1 clicked; style 1: 2 sent 1 clicked", stats)
	}
	if id, ok, _ := s.CampaignUserByToken(byUser[ids[2]].Token); !ok || id != ids[2] {
		t.Errorf("user by token = %q %v", id, ok)
	}
}

func TestResetSystemOnboardingOnlyTouchesWhatTheMigrationPut(t *testing.T) {
	s := newStore(t)
	insert := func(id, status, at string) {
		if _, err := s.db.Exec(`INSERT INTO onboarding (user_id, status, updated_at) VALUES (?, ?, ?)`, id, status, at); err != nil {
			t.Fatal(err)
		}
	}
	insert("system", "postponed", systemOnboardingStamp)
	insert("chose", "postponed", "2026-11-06T10:00:00Z")
	insert("done", "done", systemOnboardingStamp)
	for _, id := range []string{"system", "chose", "done"} {
		if err := s.ResetSystemOnboarding(id); err != nil {
			t.Fatal(err)
		}
	}
	var left []string
	if err := s.collectStrings(`SELECT user_id FROM onboarding ORDER BY user_id`, func(v string) { left = append(left, v) }); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(left) != "[chose done]" {
		t.Errorf("rows left = %v, want only the one the learner postponed and the one that was finished", left)
	}
}

func TestCampaignResultsCompareSentWithHoldout(t *testing.T) {
	s := newStore(t)
	frozen := campaignNow
	rows := []CampaignRow{
		{"a", ChannelTelegram, 1, CampaignSent, 1}, {"b", ChannelTelegram, 1, CampaignSent, 2},
		{"c", ChannelTelegram, 1, CampaignHoldout, 3}, {"d", ChannelTelegram, 1, CampaignHoldout, 4},
		{"e", ChannelTelegram, 4, CampaignSent, 5},
	}
	if err := s.CampaignFreeze("test", rows, frozen); err != nil {
		t.Fatal(err)
	}
	taken, err := s.CampaignTake("test", ChannelTelegram, 1, -1, func(string) int { return 0 }, true)
	if err != nil || len(taken) != 3 {
		t.Fatalf("take: %v %v", taken, err)
	}
	due, _ := s.CampaignDue("test")
	for _, d := range due {
		if err := s.CampaignMarkSent(d.ID, frozen.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	// a finishes lesson 00 after the freeze (reached), b did it before (does not count), c (held out) finishes it, d does not.
	// e (segment 4) opens lesson 02 after the freeze.
	for id, at := range map[string]time.Time{"a": frozen.Add(2 * time.Hour), "b": frozen.Add(-2 * time.Hour), "c": frozen.Add(5 * time.Hour)} {
		if _, err := s.db.Exec(`INSERT INTO users (id, name, created_at) VALUES (?, ?, ?)`, id, "n"+id, stamp(frozen.Add(-48*time.Hour))); err != nil {
			t.Fatal(err)
		}
		campaignLesson(t, s, id, "00", "done", at)
	}
	if _, err := s.db.Exec(`INSERT INTO users (id, name, created_at) VALUES ('e', 'ne', ?)`, stamp(frozen.Add(-48*time.Hour))); err != nil {
		t.Fatal(err)
	}
	campaignLesson(t, s, "e", "02", "in_progress", frozen.Add(3*time.Hour))

	res, err := s.CampaignResults("test")
	if err != nil {
		t.Fatal(err)
	}
	groups := map[string]CampaignGroup{}
	for _, g := range res.Groups {
		groups[fmt.Sprintf("%d/%s", g.Segment, g.Kind)] = g
	}
	if g := groups["1/sent"]; g.People != 2 || g.Reached != 1 {
		t.Errorf("segment 1 sent = %+v, want 2 people, 1 reached", g)
	}
	if g := groups["1/holdout"]; g.People != 2 || g.Reached != 1 {
		t.Errorf("segment 1 holdout = %+v, want 2 people, 1 reached", g)
	}
	if g := groups["4/sent"]; g.People != 1 || g.Reached != 1 {
		t.Errorf("segment 4 sent = %+v, want 1 person, 1 reached (opening lesson 02 counts)", g)
	}
	if len(res.Cells) == 0 {
		t.Error("results have no cells")
	}
}
