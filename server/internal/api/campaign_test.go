package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/campaign"
	"github.com/grisha/serbian-app/server/internal/config"
	"github.com/grisha/serbian-app/server/internal/content"
	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
)

const campaignToken = "campaign-secret"

type sentPhoto struct {
	ChatID  int64
	Photo   string
	Caption string
	Button  *telegram.InlineButton
}

type campaignMail struct{ To, Subject, Text, HTML string }

type campaignSinks struct {
	mu     sync.Mutex
	photos []sentPhoto
	mails  []campaignMail
}

func (s *campaignSinks) photo(chatID int64, photo, caption string, button *telegram.InlineButton, _ int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.photos = append(s.photos, sentPhoto{chatID, photo, caption, button})
}

func (s *campaignSinks) mail(to, subject, text, html string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mails = append(s.mails, campaignMail{to, subject, text, html})
}

// campaignNowFor is a noon (UTC) far enough ahead of the accounts the tests create (registered "now") for them to count as quiet.
func campaignNowFor() time.Time {
	d := time.Now().UTC().Add(5 * 24 * time.Hour)
	return time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, time.UTC)
}

// belgradeAt is the given clock time in Belgrade on the day of d.Now() plus dayOffset days.
func belgradeAt(d Deps, dayOffset, hour, min int) time.Time {
	l := d.Now().In(campaignZone).AddDate(0, 0, dayOffset)
	return time.Date(l.Year(), l.Month(), l.Day(), hour, min, 0, 0, campaignZone)
}

// sweepDay runs the reminder sweeps of one whole working day, every 20 minutes from 10:00 to 19:40 Belgrade time, as the ticker would,
// and returns how many messages each sweep sent (Telegram photos plus e-mails).
func sweepDay(t *testing.T, d Deps, dayOffset int) []int {
	t.Helper()
	sinks := campaignSinksOf(d)
	var perSweep []int
	start := belgradeAt(d, dayOffset, 10, 0)
	for i := 0; i < 30; i++ {
		before := sinks.total()
		if err := RunCampaignSweep(d, start.Add(time.Duration(i)*campaignSweepEvery)); err != nil {
			t.Fatal(err)
		}
		perSweep = append(perSweep, sinks.total()-before)
	}
	return perSweep
}

func (s *campaignSinks) total() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.photos) + len(s.mails)
}

// campaignSinksOf finds the capture sinks behind a Deps built by newCampaignDeps.
func campaignSinksOf(d Deps) *campaignSinks { return campaignSinkByDeps[fmt.Sprintf("%p", d.Store)] }

var campaignSinkByDeps = map[string]*campaignSinks{}

func newCampaignDeps(t *testing.T) (Deps, *store.Store, *campaignSinks) {
	t.Helper()
	c, err := content.Load("../content/testdata/content")
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	sinks := &campaignSinks{}
	campaignSinkByDeps[fmt.Sprintf("%p", st)] = sinks
	now := campaignNowFor()
	d := Deps{
		Course: func() *content.Course { return c }, Store: st, Stale: func() bool { return false },
		Now: func() time.Time { return now },
		Config: config.Config{
			AppBaseURL: testBaseURL, Campaign: true, CampaignServiceToken: campaignToken, CampaignPhotoBase: "https://img.test/photos",
		},
		EnqueueTelegramPhoto: sinks.photo, SendMail: sinks.mail, Async: func(f func()) { f() },
	}
	return d, st, sinks
}

func campaignEmailUser(t *testing.T, st *store.Store, name string) string {
	t.Helper()
	uid, err := st.CreateUser(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateIdentity(store.Identity{
		ID: "pw_" + name, UserID: uid, Provider: "password", ProviderUID: name + "@example.com", Email: name + "@example.com",
		PasswordHash: "x", EmailVerifiedAt: "2026-09-01T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	return uid
}

func campaignCall(h http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func openWave(t *testing.T, h http.Handler, wave int, dry bool) map[string]any {
	t.Helper()
	rr := campaignCall(h, "POST", "/api/service/campaigns/return/wave", fmt.Sprintf(`{"wave":%d,"dry_run":%v}`, wave, dry), campaignToken)
	if rr.Code != 200 {
		t.Fatalf("open wave %d (dry=%v) = %d %s", wave, dry, rr.Code, rr.Body)
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// seedAudience makes n Telegram learners spread over the four segments and m e-mail learners, and returns their user ids by chat id /
// address so the tests can tell who got what.
func seedAudience(t *testing.T, st *store.Store, n, m int) (tg map[int64]string, mails map[string]string) {
	t.Helper()
	tg, mails = map[int64]string{}, map[string]string{}
	at := campaignNowFor().Add(-10 * 24 * time.Hour)
	for i := 0; i < n; i++ {
		uid := linkTelegram(t, st, int64(1000+i), fmt.Sprintf("tg%02d", i))
		tg[int64(1000+i)] = uid
		switch i % 4 {
		case 1:
			if err := st.User(uid).SetLessonStatus("00", "in_progress", at); err != nil {
				t.Fatal(err)
			}
		case 2:
			if err := st.User(uid).SetLessonStatus("00", "done", at); err != nil {
				t.Fatal(err)
			}
		case 3:
			for _, l := range []string{"00", "01"} {
				if err := st.User(uid).SetLessonStatus(l, "done", at); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	for i := 0; i < m; i++ {
		name := fmt.Sprintf("mail%02d", i)
		mails[name+"@example.com"] = campaignEmailUser(t, st, name)
	}
	return tg, mails
}

func TestCampaignServiceNeedsTheToken(t *testing.T) {
	d, _, _ := newCampaignDeps(t)
	h := Handler(d)
	body := `{"wave":1,"dry_run":true}`
	if rr := campaignCall(h, "POST", "/api/service/campaigns/return/wave", body, ""); rr.Code != http.StatusUnauthorized {
		t.Errorf("no token = %d, want 401", rr.Code)
	}
	if rr := campaignCall(h, "POST", "/api/service/campaigns/return/wave", body, "wrong"); rr.Code != http.StatusUnauthorized {
		t.Errorf("wrong token = %d, want 401", rr.Code)
	}
	if rr := campaignCall(h, "GET", "/api/service/campaigns/return/results", "", campaignToken); rr.Code != 200 {
		t.Errorf("results with the token = %d, want 200", rr.Code)
	}
	d.Config.CampaignServiceToken = ""
	if rr := campaignCall(Handler(d), "POST", "/api/service/campaigns/return/wave", body, campaignToken); rr.Code != http.StatusNotFound {
		t.Errorf("no token configured = %d, want 404", rr.Code)
	}
}

func TestCampaignDryRunChangesNothingAndWavesOpenInOrder(t *testing.T) {
	d, st, sinks := newCampaignDeps(t)
	h := Handler(d)
	seedAudience(t, st, 8, 3)

	dry := openWave(t, h, 1, true)
	if dry["dry_run"] != true || dry["audience_total"].(float64) != 11 {
		t.Errorf("dry run = %+v, want 11 people in the audience", dry)
	}
	if frozen, _ := st.CampaignFrozen(campaign.Name); frozen {
		t.Error("a dry run froze the audience")
	}
	if n, _ := st.CampaignWavesOpened(campaign.Name); n != 0 {
		t.Errorf("a dry run opened %d waves", n)
	}
	if err := RunCampaignSweep(d, d.Now()); err != nil || len(sinks.photos)+len(sinks.mails) != 0 {
		t.Errorf("after a dry run %d messages were sent (err %v)", len(sinks.photos)+len(sinks.mails), err)
	}

	if rr := campaignCall(h, "POST", "/api/service/campaigns/return/wave", `{"wave":2}`, campaignToken); rr.Code != http.StatusConflict {
		t.Errorf("opening wave 2 first = %d, want 409", rr.Code)
	}
	openWave(t, h, 1, false)
	if rr := campaignCall(h, "POST", "/api/service/campaigns/return/wave", `{"wave":1}`, campaignToken); rr.Code != http.StatusConflict {
		t.Errorf("opening wave 1 twice = %d, want 409", rr.Code)
	}
	if rr := campaignCall(h, "POST", "/api/service/campaigns/return/wave", `{"wave":0}`, campaignToken); rr.Code != http.StatusBadRequest {
		t.Errorf("wave 0 = %d, want 400", rr.Code)
	}
	// Messages of wave 1 have not gone out, so wave 2 waits, unless forced.
	if rr := campaignCall(h, "POST", "/api/service/campaigns/return/wave", `{"wave":2}`, campaignToken); rr.Code != http.StatusConflict {
		t.Errorf("wave 2 while wave 1 is unsent = %d, want 409", rr.Code)
	}
}

func TestCampaignWaveOneSendsToEveryoneButTheHoldout(t *testing.T) {
	d, st, sinks := newCampaignDeps(t)
	h := Handler(d)
	tg, mails := seedAudience(t, st, 12, 4)
	plan := openWave(t, h, 1, false)
	if plan["audience_frozen_now"] != true {
		t.Errorf("wave 1 should freeze the audience: %+v", plan)
	}
	sweepDay(t, d, 0)

	wantTG, wantMail := 0, 0
	for _, uid := range tg {
		if !campaign.Holdout(uid) {
			wantTG++
		}
	}
	for _, uid := range mails {
		if !campaign.Holdout(uid) {
			wantMail++
		}
	}
	if len(sinks.photos) != wantTG || len(sinks.mails) != wantMail {
		t.Fatalf("sent %d photos and %d e-mails, want %d and %d (everyone but the holdout)", len(sinks.photos), len(sinks.mails), wantTG, wantMail)
	}
	seen := map[int64]bool{}
	for _, p := range sinks.photos {
		uid := tg[p.ChatID]
		if uid == "" || campaign.Holdout(uid) || seen[p.ChatID] {
			t.Errorf("photo to chat %d: unknown, held out or sent twice", p.ChatID)
		}
		seen[p.ChatID] = true
		if !strings.HasPrefix(p.Photo, "https://img.test/photos/") || !strings.HasSuffix(p.Photo, ".jpg") {
			t.Errorf("photo address = %q", p.Photo)
		}
		if p.Button == nil || p.Button.CampaignURL == "" || p.Button.Label == "" {
			t.Fatalf("button = %+v", p.Button)
		}
		if !strings.HasPrefix(p.Button.CampaignURL, testBaseURL+"/lesson/") || !strings.Contains(p.Button.CampaignURL, "?c=") {
			t.Errorf("button address = %q", p.Button.CampaignURL)
		}
		known := false
		for seg := 1; seg <= campaign.Segments; seg++ {
			if p.Caption == campaign.TelegramHTML(campaign.For(seg).Telegram) {
				known = true
			}
		}
		if !known {
			t.Errorf("caption %q is not one of the approved texts", p.Caption)
		}
	}
	for _, m := range sinks.mails {
		if uid := mails[m.To]; uid == "" || campaign.Holdout(uid) {
			t.Errorf("e-mail to %s: unknown or held out", m.To)
		}
		if !strings.Contains(m.HTML, testBaseURL+"/api/unsubscribe?t=") || !strings.Contains(m.Text, "/api/unsubscribe?t=") {
			t.Errorf("e-mail to %s has no unsubscribe link", m.To)
		}
		if !strings.Contains(m.HTML, testBaseURL+"/lesson/") || !strings.Contains(m.HTML, testBaseURL+"/campaign/") {
			t.Errorf("e-mail to %s lacks the lesson link or the picture", m.To)
		}
	}
	if n, _ := st.CampaignUnsent(campaign.Name); n != 0 {
		t.Errorf("%d messages are still unsent", n)
	}
	// Each person is written to once, however many sweeps run.
	sweepDay(t, d, 0)
	if len(sinks.photos) != wantTG || len(sinks.mails) != wantMail {
		t.Errorf("a second sweep sent more: %d photos, %d e-mails", len(sinks.photos), len(sinks.mails))
	}

	// Wave 2 opens now that wave 1 is out, and takes nobody twice.
	openWave(t, h, 2, false)
	sweepDay(t, d, 0)
	if len(sinks.photos) != wantTG || len(sinks.mails) != wantMail {
		t.Errorf("wave 2 wrote to people who already had their message")
	}
}

func TestCampaignWavesAreFiftyThenFiftyThenTheRest(t *testing.T) {
	d, st, sinks := newCampaignDeps(t)
	h := Handler(d)
	seedAudience(t, st, 160, 0)
	for wave := 1; wave <= 2; wave++ {
		plan := openWave(t, h, wave, false)
		taken := int(plan["channels"].([]any)[0].(map[string]any)["taken"].(float64))
		if taken != 50 {
			t.Errorf("wave %d took %d Telegram people, want 50", wave, taken)
		}
		sweepDay(t, d, 0)
	}
	if len(sinks.photos) != 100 {
		t.Fatalf("after two waves %d photos went out, want 100", len(sinks.photos))
	}
	plan := openWave(t, h, 3, false)
	left := int(plan["channels"].([]any)[0].(map[string]any)["taken"].(float64))
	if left <= 0 {
		t.Fatalf("wave 3 should take everyone left, took %d", left)
	}
	sweepDay(t, d, 0)
	// Everyone who is not held out has had exactly one message: 100 from the first two waves and the rest from the third.
	if len(sinks.photos) != 100+left {
		t.Errorf("%d photos went out, want 100 + %d from wave 3", len(sinks.photos), left)
	}
	seen := map[int64]bool{}
	for _, p := range sinks.photos {
		if seen[p.ChatID] {
			t.Errorf("chat %d was written to twice", p.ChatID)
		}
		seen[p.ChatID] = true
	}
}

func TestCampaignWaveTwoLeansToTheBetterButton(t *testing.T) {
	d, st, sinks := newCampaignDeps(t)
	h := Handler(d)
	seedAudience(t, st, 160, 0)
	openWave(t, h, 1, false)
	sweepDay(t, d, 0)
	// Only the glance style's buttons get pressed.
	pressed := 0
	for _, p := range sinks.photos {
		if p.Button.Label != campaignButtonLabel(p.Caption, campaign.StyleGlance) {
			continue
		}
		token := p.Button.CampaignURL[strings.Index(p.Button.CampaignURL, "?c=")+3:]
		if _, err := st.NoteCampaignClick(token, d.Now()); err != nil {
			t.Fatal(err)
		}
		pressed++
	}
	if pressed == 0 {
		t.Fatal("no glance-style message went out in wave 1")
	}

	plan := openWave(t, h, 2, false)
	ch := plan["channels"].([]any)[0].(map[string]any)
	if int(ch["leader"].(float64)) != campaign.StyleGlance || int(ch["leader_share_permille"].(float64)) != 550 {
		t.Errorf("wave 2 leader = %v share %v, want the glance style at 550", ch["leader"], ch["leader_share_permille"])
	}
	// The split itself is a hash draw over 50 random accounts, so its exact numbers vary from run to run (campaign.StyleFor is tested on
	// thousands of ids); here it is enough that all 50 were given one of the two styles.
	styles := ch["styles"].(map[string]any)
	if total := int(styles["1"].(float64)) + int(styles["0"].(float64)); total != 50 {
		t.Errorf("wave 2 styles add up to %d, want 50", total)
	}
}

// campaignButtonLabel is the label of the given style in the segment whose Telegram text is caption.
func campaignButtonLabel(caption string, style int) string {
	for seg := 1; seg <= campaign.Segments; seg++ {
		if campaign.TelegramHTML(campaign.For(seg).Telegram) == caption {
			return campaign.For(seg).Buttons[style]
		}
	}
	return ""
}

func TestCampaignSweepStaysInsideTheBelgradeWindow(t *testing.T) {
	d, st, sinks := newCampaignDeps(t)
	h := Handler(d)
	seedAudience(t, st, 12, 4)
	openWave(t, h, 1, false)

	// Before 10:00 and from 20:00 on, in Belgrade time, nothing goes out, whatever the learner's own timezone.
	for _, at := range []time.Time{belgradeAt(d, 0, 3, 0), belgradeAt(d, 0, 9, 59), belgradeAt(d, 0, 20, 0), belgradeAt(d, 0, 23, 0)} {
		if err := RunCampaignSweep(d, at); err != nil || sinks.total() != 0 {
			t.Fatalf("a sweep at %s Belgrade time sent %d messages (err %v), want none", at.In(campaignZone).Format("15:04"), sinks.total(), err)
		}
	}
	if err := RunCampaignSweep(d, belgradeAt(d, 0, 10, 0)); err != nil {
		t.Fatal(err)
	}
	if n, _ := st.CampaignUnsent(campaign.Name); n > 0 && sinks.total() == 0 {
		t.Error("the sweep at 10:00 sent nothing although messages were due")
	}
}

// A wave must leave as a trickle through the day with the weight on the evening, not as a burst.
func TestCampaignWaveIsSpreadOverTheDayAndHeavierInTheEvening(t *testing.T) {
	d, st, sinks := newCampaignDeps(t)
	h := Handler(d)
	seedAudience(t, st, 80, 80)
	openWave(t, h, 1, false) // 50 Telegram messages and 50 e-mails

	perSweep := sweepDay(t, d, 0) // 30 sweeps: 10:00 ... 19:40
	t.Logf("messages per 20-minute sweep from 10:00: %v", perSweep)
	total, morning, afternoon, evening, most := 0, 0, 0, 0, 0
	for i, n := range perSweep {
		total += n
		switch {
		case i < 12: // 10:00-14:00
			morning += n
		case i < 21: // 14:00-17:00
			afternoon += n
		default: // 17:00-20:00
			evening += n
		}
		if n > most {
			most = n
		}
	}
	if total != 100 || sinks.total() != 100 {
		t.Fatalf("the day sent %d messages, want the 100 of the wave", total)
	}
	if !(morning < afternoon && afternoon < evening) {
		t.Errorf("morning %d, afternoon %d, evening %d: each part of the day should carry more than the one before", morning, afternoon, evening)
	}
	// About a fifth in the morning, a third in the afternoon and nearly half in the evening, never a burst.
	if morning > 35 || evening < 40 {
		t.Errorf("morning %d and evening %d of 100: the weight is not where it should be", morning, evening)
	}
	if most > 8 {
		t.Errorf("one sweep sent %d messages: the wave is not spread (per sweep: %v)", most, perSweep)
	}
}

func TestCampaignWeights(t *testing.T) {
	d, _, _ := newCampaignDeps(t)
	for _, tc := range []struct {
		h, m, want int
	}{{9, 59, 0}, {10, 0, 1}, {13, 40, 1}, {14, 0, 2}, {16, 40, 2}, {17, 0, 3}, {19, 40, 3}, {20, 0, 0}, {23, 0, 0}, {3, 0, 0}} {
		if got := campaignWeight(belgradeAt(d, 0, tc.h, tc.m)); got != tc.want {
			t.Errorf("weight at %02d:%02d = %d, want %d", tc.h, tc.m, got, tc.want)
		}
	}
	// The whole day adds up to 12*1 + 9*2 + 9*3 = 57, and a sweep's share is its weight over what is left.
	if this, total := campaignWeightsFrom(belgradeAt(d, 0, 10, 0)); this != 1 || total != 57 {
		t.Errorf("at 10:00 weight %d of %d, want 1 of 57", this, total)
	}
	if this, total := campaignWeightsFrom(belgradeAt(d, 0, 19, 40)); this != 3 || total != 3 {
		t.Errorf("at 19:40 weight %d of %d, want the last sweep to take everything (3 of 3)", this, total)
	}
	if this, total := campaignWeightsFrom(belgradeAt(d, 0, 20, 0)); this != 0 || total != 0 {
		t.Errorf("at 20:00 weight %d of %d, want 0 of 0", this, total)
	}
}

func TestCampaignSweepWaitsForOtherStudyMessages(t *testing.T) {
	d, st, sinks := newCampaignDeps(t)
	h := Handler(d)
	tg, _ := seedAudience(t, st, 12, 0)
	openWave(t, h, 1, false)

	// Somebody who already had a study message today waits for the next day.
	var busy string
	for _, uid := range tg {
		if !campaign.Holdout(uid) {
			busy = uid
			break
		}
	}
	if busy == "" {
		t.Skip("every account of this run was held out")
	}
	today := belgradeAt(d, 0, 10, 0).UTC().Format("2006-01-02")
	if err := st.SetBotReminderAllDoneDate(busy, today); err != nil {
		t.Fatal(err)
	}
	sweepDay(t, d, 0)
	for _, p := range sinks.photos {
		if tg[p.ChatID] == busy {
			t.Error("the person with a study message today was written to as well")
		}
	}
	before := len(sinks.photos)
	sweepDay(t, d, 1)
	if len(sinks.photos) != before+1 {
		t.Errorf("the postponed message did not go out the next day: %d -> %d photos", before, len(sinks.photos))
	}
}

func TestCampaignSweepSkipsWhoCannotBeReachedAnymore(t *testing.T) {
	d, st, sinks := newCampaignDeps(t)
	h := Handler(d)
	tg, mails := seedAudience(t, st, 12, 12)
	openWave(t, h, 1, false)
	// Between the wave opening and the message: one person switches the reminders off, another unsubscribes.
	var offTG, offMail string
	for _, uid := range tg {
		if !campaign.Holdout(uid) {
			offTG = uid
			break
		}
	}
	for _, uid := range mails {
		if !campaign.Holdout(uid) {
			offMail = uid
			break
		}
	}
	if offTG == "" || offMail == "" {
		t.Skip("every account of one channel was held out in this run")
	}
	if err := st.SetBotPrefs(offTG, store.BotPrefs{Learning: false}, d.Now()); err != nil {
		t.Fatal(err)
	}
	if err := st.SetEmailOptout(offMail, d.Now()); err != nil {
		t.Fatal(err)
	}
	sweepDay(t, d, 0)
	for _, p := range sinks.photos {
		if tg[p.ChatID] == offTG {
			t.Error("a person who switched the reminders off got the message")
		}
	}
	for _, m := range sinks.mails {
		if mails[m.To] == offMail {
			t.Error("a person who unsubscribed got the e-mail")
		}
	}
	if n, _ := st.CampaignUnsent(campaign.Name); n != 0 {
		t.Errorf("%d messages still unsent: a skipped person must not hold the next wave", n)
	}
}

func TestCampaignSweepBringsTheTourBack(t *testing.T) {
	d, st, sinks := newCampaignDeps(t)
	h := Handler(d)
	// The hold-out is decided by hashing the random user id: draw accounts until one is not held out.
	var uid string
	for i := 0; uid == "" && i < 100; i++ {
		if c := linkTelegram(t, st, int64(7000+i), fmt.Sprintf("tourist%03d", i)); !campaign.Holdout(c) {
			uid = c
		}
	}
	if uid == "" {
		t.Skip("every drawn account was held out")
	}
	if err := st.User(uid).SetLessonStatus("00", "done", d.Now().Add(-10*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	// The tour was put off for everybody by the migration, with its stamp.
	if err := st.SetOnboarding(uid, "postponed", time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if due, _ := st.OnboardingDue(uid); due {
		t.Fatal("tour should not be due before the message")
	}
	openWave(t, h, 1, false)
	sweepDay(t, d, 0)
	if len(sinks.photos) != 1 {
		t.Fatalf("%d photos went out, want 1", len(sinks.photos))
	}
	if due, _ := st.OnboardingDue(uid); !due {
		t.Error("the swallow's tour should be due again after the message that promises it")
	}
}

func TestCampaignSweepIsOffWithoutTheFlag(t *testing.T) {
	d, st, sinks := newCampaignDeps(t)
	h := Handler(d)
	seedAudience(t, st, 12, 4)
	openWave(t, h, 1, false)
	d.Config.Campaign = false
	if err := RunCampaignSweep(d, d.Now()); err != nil || len(sinks.photos)+len(sinks.mails) != 0 {
		t.Errorf("with CAMPAIGN_RETURN off %d messages went out (err %v)", len(sinks.photos)+len(sinks.mails), err)
	}
}

func TestCampaignClickIsCountedWhenTheLessonOpens(t *testing.T) {
	d, st, sinks := newCampaignDeps(t)
	h := Handler(d)
	seedAudience(t, st, 12, 0)
	openWave(t, h, 1, false)
	sweepDay(t, d, 0)
	if len(sinks.photos) == 0 {
		t.Skip("every account of this run was held out")
	}
	url := sinks.photos[0].Button.CampaignURL
	path := url[len(testBaseURL):]
	served := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { served = true })
	rr := httptest.NewRecorder()
	RepeatClickTracker(d, next).ServeHTTP(rr, httptest.NewRequest("GET", path, nil))
	if !served {
		t.Error("the page was not served")
	}
	var clicked int
	for _, c := range func() []store.CampaignCell { r, _ := st.CampaignResults(campaign.Name); return r.Cells }() {
		clicked += c.Clicked
	}
	if clicked != 1 {
		t.Errorf("clicks = %d, want 1", clicked)
	}
	// A lesson opened without a token, or with a made-up one, counts nothing.
	RepeatClickTracker(d, next).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/lesson/00", nil))
	RepeatClickTracker(d, next).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/lesson/00?c=0123456789abcdef", nil))
}

func TestUnsubscribeAsksFirstAndThenSwitchesTheRemindersOff(t *testing.T) {
	d, st, sinks := newCampaignDeps(t)
	h := Handler(d)
	_, mails := seedAudience(t, st, 0, 12)
	openWave(t, h, 1, false)
	sweepDay(t, d, 0)
	if len(sinks.mails) == 0 {
		t.Skip("every account of this run was held out")
	}
	m := sinks.mails[0]
	uid := mails[m.To]
	i := strings.Index(m.Text, "/api/unsubscribe?t=")
	link := strings.TrimSpace(m.Text[i:])
	if nl := strings.IndexAny(link, "\r\n "); nl >= 0 {
		link = link[:nl]
	}

	// A GET (a mail scanner opening the link) changes nothing and shows the confirmation.
	rr := campaignCall(h, "GET", link, "", "")
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), `<form method="post"`) {
		t.Fatalf("GET = %d %s", rr.Code, rr.Body)
	}
	if out, _ := st.EmailOptedOut(uid); out {
		t.Fatal("a GET unsubscribed the person")
	}
	// The confirmation page posts from the site itself, so the browser sends its own origin.
	req := httptest.NewRequest("POST", link, nil)
	req.Header.Set("Origin", testBaseURL)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), "Вы отписались") {
		t.Fatalf("POST = %d %s", rr.Code, rr.Body)
	}
	if out, _ := st.EmailOptedOut(uid); !out {
		t.Error("the e-mail opt-out was not recorded")
	}
	if p, _ := st.BotPrefs(uid); p.Learning {
		t.Error("the study reminders in Telegram should be off too")
	}
	rr = campaignCall(h, "GET", "/api/unsubscribe?t=0000000000000000", "", "")
	if !strings.Contains(rr.Body.String(), "Ссылка устарела") {
		t.Errorf("an unknown token page = %s", rr.Body)
	}
	req = httptest.NewRequest("POST", "/api/unsubscribe?t=0000000000000000", nil)
	req.Header.Set("Origin", testBaseURL)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), "Ссылка устарела") {
		t.Errorf("POST with an unknown token = %d %s", rr.Code, rr.Body)
	}
}

// Two sweeps can run at once (the old and the new container for a minute during a deploy) and both read the same due rows: each person
// must still get exactly one message.
func TestCampaignConcurrentSweepsSendEachPersonOnce(t *testing.T) {
	d, st, sinks := newCampaignDeps(t)
	h := Handler(d)
	tg, mails := seedAudience(t, st, 40, 20)
	openWave(t, h, 1, false)

	// At 19:40 a sweep takes everything still due, so every sweep below tries to send every row.
	at := belgradeAt(d, 0, 19, 40)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := RunCampaignSweep(d, at); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	wantTG, wantMail := 0, 0
	for _, uid := range tg {
		if !campaign.Holdout(uid) {
			wantTG++
		}
	}
	for _, uid := range mails {
		if !campaign.Holdout(uid) {
			wantMail++
		}
	}
	chats := map[int64]int{}
	for _, p := range sinks.photos {
		chats[p.ChatID]++
	}
	to := map[string]int{}
	for _, m := range sinks.mails {
		to[m.To]++
	}
	for chat, n := range chats {
		if n != 1 {
			t.Errorf("chat %d got %d messages, want 1", chat, n)
		}
	}
	for addr, n := range to {
		if n != 1 {
			t.Errorf("%s got %d e-mails, want 1", addr, n)
		}
	}
	if len(chats) != wantTG || len(to) != wantMail {
		t.Errorf("%d chats and %d addresses were written to, want %d and %d", len(chats), len(to), wantTG, wantMail)
	}
}

// Nobody gets both channels, and nobody gets a message in a later wave if they had one already.
func TestCampaignWritesToOnePersonOnceAcrossChannelsAndWaves(t *testing.T) {
	d, st, sinks := newCampaignDeps(t)
	h := Handler(d)
	// One account reachable both ways: Telegram wins and the e-mail is never sent.
	uid := linkTelegram(t, st, 9001, "both")
	if err := st.CreateIdentity(store.Identity{ID: "pw_both", UserID: uid, Provider: "password", ProviderUID: "both@example.com",
		Email: "both@example.com", PasswordHash: "x", EmailVerifiedAt: "2026-09-01T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	seedAudience(t, st, 30, 0)
	for wave := 1; wave <= campaign.Waves; wave++ {
		openWave(t, h, wave, false)
		sweepDay(t, d, wave)
	}
	for _, m := range sinks.mails {
		if m.To == "both@example.com" {
			t.Error("a person reachable on Telegram was also e-mailed")
		}
	}
	seen := map[int64]bool{}
	for _, p := range sinks.photos {
		if seen[p.ChatID] {
			t.Errorf("chat %d was written to twice across the waves", p.ChatID)
		}
		seen[p.ChatID] = true
	}
}
