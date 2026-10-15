package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
)

func TestBotPrefsEndpointRequiresASession(t *testing.T) {
	h, _, _ := newTelegramBotAPI(t)
	if rr := anon(h, "GET", "/api/me/bot-prefs", ""); rr.Code != http.StatusUnauthorized {
		t.Fatalf("GET without a session = %d, want 401", rr.Code)
	}
	if rr := anon(h, "PUT", "/api/me/bot-prefs", `{"news":true}`); rr.Code != http.StatusUnauthorized {
		t.Fatalf("PUT without a session = %d, want 401", rr.Code)
	}
}

func TestBotPrefsEndpointDefaultsAndPartialUpdate(t *testing.T) {
	h, st, _ := newTelegramBotAPI(t)
	uid := registerAndVerify(t, h, st, "prefs@example.com", "secret1234", "Prefs")
	cookie := authed(t, st, uid)

	rr := doCookie(h, cookie, "GET", "/api/me/bot-prefs", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET = %d %s", rr.Code, rr.Body)
	}
	got := decodeBody[botPrefsDTO](t, rr)
	if got != (botPrefsDTO{Learning: true, Billing: true, News: false}) {
		t.Fatalf("defaults = %+v: study and tariff reminders on, news off until agreed", got)
	}

	// One field in, the other two untouched.
	rr = doCookie(h, cookie, "PUT", "/api/me/bot-prefs", `{"learning":false}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("PUT = %d %s", rr.Code, rr.Body)
	}
	got = decodeBody[botPrefsDTO](t, rr)
	if got != (botPrefsDTO{Learning: false, Billing: true, News: false}) {
		t.Fatalf("after PUT learning=false: %+v", got)
	}

	rr = doCookie(h, cookie, "PUT", "/api/me/bot-prefs", `{"news":true}`)
	got = decodeBody[botPrefsDTO](t, rr)
	if got != (botPrefsDTO{Learning: false, Billing: true, News: true}) {
		t.Fatalf("a second PUT must keep the first: %+v", got)
	}

	stored, _ := st.BotPrefs(uid)
	if stored != (store.BotPrefs{Learning: false, Billing: true, News: true}) {
		t.Fatalf("stored = %+v", stored)
	}
}

// /stop is the one command that works for anybody: it turns every optional
// category off, says so, and promises that service messages still arrive.
func TestBotStopSwitchesEverythingOffAndSaysSo(t *testing.T) {
	h, st, sink := newTelegramBotAPI(t)
	uid := linkForChannel(t, h, st, `{}`, 7010)
	before := len(sink.all())

	postWebhook(h, tgWebhookSecret, webhookUpdate(7010, 7010, "chan", "Chan", "/stop"))

	got, _ := st.BotPrefs(uid)
	if got != (store.BotPrefs{}) {
		t.Fatalf("prefs after /stop = %+v, want all off", got)
	}
	sent := sink.all()[before:]
	if len(sent) != 1 || !strings.Contains(sent[0].Text, "больше не пишу") || !strings.Contains(sent[0].Text, "/settings") {
		t.Fatalf("reply = %+v, want a confirmation that points to /settings", sent)
	}
}

func TestBotStopFromAnUnlinkedTelegramChangesNothing(t *testing.T) {
	h, st, sink := newTelegramBotAPI(t)
	uid := linkForChannel(t, h, st, `{}`, 7010)
	before := len(sink.all())

	postWebhook(h, tgWebhookSecret, webhookUpdate(7999, 7999, "stranger", "Stranger", "/stop"))

	if got, _ := st.BotPrefs(uid); got != store.DefaultBotPrefs {
		t.Fatalf("someone else's /stop changed an account: %+v", got)
	}
	sent := sink.all()[before:]
	if len(sent) != 1 || !strings.Contains(sent[0].Text, "не привязан") {
		t.Fatalf("reply = %+v, want the 'not linked' explanation", sent)
	}
}

func TestBotSettingsShowsTheCurrentState(t *testing.T) {
	h, st, sink := newTelegramBotAPI(t)
	uid := linkForChannel(t, h, st, `{}`, 7010)
	if err := st.SetBotPrefs(uid, store.BotPrefs{Learning: false, Billing: true, News: true}, fixedNow); err != nil {
		t.Fatal(err)
	}
	before := len(sink.all())

	postWebhook(h, tgWebhookSecret, webhookUpdate(7010, 7010, "chan", "Chan", "/settings@"+tgBotUsername))

	sent := sink.all()[before:]
	if len(sent) != 1 || sent[0].Button == nil || sent[0].Button.Prefs == nil {
		t.Fatalf("reply = %+v, want one message with the settings keyboard", sent)
	}
	if *sent[0].Button.Prefs != (telegram.PrefsState{Learning: false, Billing: true, News: true}) {
		t.Errorf("keyboard state = %+v, want what the account has", *sent[0].Button.Prefs)
	}
}

func TestPrefToggleSavesAndRedrawsTheKeyboard(t *testing.T) {
	h, st, _ := newTelegramBotAPI(t)
	bot := newFakeBotAPI(t, "left")
	uid := linkForChannel(t, h, st, `{}`, 7010)

	postWebhook(h, tgWebhookSecret, callbackOnMessage(7010, "pref:news:1", 321))

	got, _ := st.BotPrefs(uid)
	if !got.News || !got.Learning || !got.Billing {
		t.Fatalf("prefs = %+v, want news switched on and the rest untouched", got)
	}
	if a := bot.lastAnswer(); !strings.HasPrefix(a, "Включено") {
		t.Errorf("notice = %q, want it to say it was switched on", a)
	}
	bot.mu.Lock()
	defer bot.mu.Unlock()
	if len(bot.edits) != 1 || !strings.Contains(bot.edits[0], `"pref:news:0"`) {
		t.Fatalf("keyboard edits = %v, want the news row redrawn so that pressing it again switches it off", bot.edits)
	}
}

func TestStopButtonUnderAReminderSwitchesStudyRemindersOffAndRemovesItself(t *testing.T) {
	h, st, _ := newTelegramBotAPI(t)
	bot := newFakeBotAPI(t, "left")
	uid := linkForChannel(t, h, st, `{}`, 7010)

	postWebhook(h, tgWebhookSecret, callbackOnMessage(7010, "prefoff:learning", 555))

	got, _ := st.BotPrefs(uid)
	if got.Learning || !got.Billing {
		t.Fatalf("prefs = %+v, want only study reminders off", got)
	}
	if a := bot.lastAnswer(); !strings.HasPrefix(a, "Выключено") {
		t.Errorf("notice = %q", a)
	}
	bot.mu.Lock()
	defer bot.mu.Unlock()
	if len(bot.edits) != 1 || !strings.Contains(bot.edits[0], `"inline_keyboard":[]`) {
		t.Fatalf("keyboard edits = %v, want the button taken off the message", bot.edits)
	}
}

func TestPrefButtonFromAnUnlinkedTelegramIsRefused(t *testing.T) {
	h, st, _ := newTelegramBotAPI(t)
	bot := newFakeBotAPI(t, "left")
	uid := linkForChannel(t, h, st, `{}`, 7010)

	postWebhook(h, tgWebhookSecret, callbackOnMessage(7999, "pref:learning:0", 1))

	if got, _ := st.BotPrefs(uid); got != store.DefaultBotPrefs {
		t.Fatalf("an unlinked Telegram changed an account: %+v", got)
	}
	if a := bot.lastAnswer(); !strings.Contains(a, "привяжи Telegram") {
		t.Errorf("notice = %q, want to be told to link Telegram", a)
	}
}

// The point of the whole feature: a person who said stop hears nothing.
func TestReminderSweepSendsNothingToSomeoneWhoOptedOut(t *testing.T) {
	d, st, sink := newReminderDeps(t)
	uid := linkTelegram(t, st, 4242, "grisha")
	gradeAllGood(t, st.User(uid), fixedNow)

	if err := st.SetBotPrefs(uid, store.BotPrefs{Learning: false, Billing: true, News: false}, fixedNow); err != nil {
		t.Fatal(err)
	}
	// Empty queue today (would earn the congrats) AND a long silence (would
	// earn the nudge): both must stay quiet.
	if err := RunReminderSweep(d, fixedNow.Add(72*time.Hour)); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if got := sink.all(); len(got) != 0 {
		t.Fatalf("sent %+v to an account that switched study reminders off", got)
	}
}

func TestReminderCarriesAStopButton(t *testing.T) {
	d, st, sink := newReminderDeps(t)
	uid := linkTelegram(t, st, 4242, "grisha")
	gradeAllGood(t, st.User(uid), fixedNow)

	if err := RunReminderSweep(d, fixedNow); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	got := sink.all()
	if len(got) != 1 {
		t.Fatalf("sent %+v, want the one congrats", got)
	}
	if b := got[0].Button; b == nil || b.PrefOff != telegram.PrefLearning || b.Label != telegram.PrefOffLabel {
		t.Fatalf("button = %+v, want the study-reminders stop button: leaving must cost one tap", b)
	}
}
