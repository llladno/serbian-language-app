package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/config"
	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
)

// tgCall is one Bot API request the fake Telegram saw.
type tgCall struct {
	Method   string
	Form     url.Values
	Upload   bool   // sendDocument carried the file body (multipart), not a file_id
	Filename string // name of the uploaded file
}

// fakeTG stands in for api.telegram.org: it answers getChatMember from a membership map,
// hands out a file_id on sendDocument, and records everything.
type fakeTG struct {
	mu        sync.Mutex
	members   map[int64]bool
	calls     []tgCall
	rateLimit int  // the next N sendDocument calls answer 429
	failURL   bool // sendDocument given a URL answers 400, as Telegram does when it cannot download it
}

func newFakeTG(t *testing.T) *fakeTG {
	t.Helper()
	f := &fakeTG{members: map[int64]bool{}}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	restore := telegram.SetAPIBase(srv.URL)
	t.Cleanup(func() { restore(); srv.Close() })
	return f
}

func (f *fakeTG) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	call := tgCall{Method: method}
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		if err := r.ParseMultipartForm(32 << 20); err == nil {
			call.Form = url.Values(r.MultipartForm.Value)
			if fh := r.MultipartForm.File["document"]; len(fh) > 0 {
				call.Upload, call.Filename = true, fh[0].Filename
			}
		}
	} else {
		_ = r.ParseForm()
		call.Form = r.PostForm
	}
	f.calls = append(f.calls, call)

	reply := func(result any) { _ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": result}) }
	switch method {
	case "getChatMember":
		id, _ := strconv.ParseInt(call.Form.Get("user_id"), 10, 64)
		status := "left"
		if f.members[id] {
			status = "member"
		}
		reply(map[string]any{"status": status})
	case "sendDocument":
		if f.rateLimit > 0 {
			f.rateLimit--
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": 429, "description": "Too Many Requests",
				"parameters": map[string]any{"retry_after": 1}})
			return
		}
		if f.failURL && strings.HasPrefix(call.Form.Get("document"), "http") {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": 400,
				"description": "Bad Request: failed to get HTTP URL content"})
			return
		}
		reply(map[string]any{"document": map[string]any{"file_id": "FILE-ID-1"}})
	default:
		reply(true)
	}
}

func (f *fakeTG) named(method string) []tgCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []tgCall
	for _, c := range f.calls {
		if c.Method == method {
			out = append(out, c)
		}
	}
	return out
}

func (f *fakeTG) setMember(id int64, member bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.members[id] = member
}

// leadAPI is a bot-enabled API with the channel configured and user 999 as admin.
func leadAPI(t *testing.T, channel string) (http.Handler, *store.Store, *fakeTG, *tgSink) {
	t.Helper()
	leadSleep = func(time.Duration) {}
	t.Cleanup(func() { leadSleep = time.Sleep })
	tg := newFakeTG(t)
	sink := &tgSink{}
	h, st, _ := newAuthAPI(t, func(d *Deps) {
		d.Config.TelegramBotToken = tgTestToken
		d.Config.TelegramAdminIDs = []int64{999}
		d.TelegramBotUsername = tgBotUsername
		d.TelegramWebhookSecret = tgWebhookSecret
		d.EnqueueTelegramMessage = sink.enqueue
	})
	// A fresh database already carries the production default channel (migration 019), so an
	// empty argument has to clear it explicitly.
	if err := st.SetEconomySetting("telegram_channel", channel, time.Now()); err != nil {
		t.Fatal(err)
	}
	return h, st, tg, sink
}

func leadCallbackUpdate(fromID int64, data string) string {
	return fmt.Sprintf(`{"callback_query":{"id":"cb1","data":%q,"from":{"id":%d,"username":"u","first_name":"Аня"}}}`, data, fromID)
}

func TestParseLeadPayload(t *testing.T) {
	for _, tc := range []struct {
		in           string
		slug, source string
		ok           bool
	}{
		{"lm_top100", "top100", "direct", true},
		{"lm_top100_tgpost", "top100", "tgpost", true},
		{"lm_top100_ig_story", "top100", "ig_story", true},
		{"lm_", "", "", false},
		{"abc123", "", "", false},
		{"lm_top100_" + strings.Repeat("x", 60), "top100", strings.Repeat("x", leadMaxSource), true},
	} {
		slug, source, ok := parseLeadPayload(tc.in)
		if slug != tc.slug || source != tc.source || ok != tc.ok {
			t.Errorf("parseLeadPayload(%q) = %q,%q,%v want %q,%q,%v", tc.in, slug, source, ok, tc.slug, tc.source, tc.ok)
		}
	}
	if s, src, ok := parseLeadCallback(leadCheckData("top100", "tgpost")); !ok || s != "top100" || src != "tgpost" {
		t.Errorf("callback round trip = %q,%q,%v", s, src, ok)
	}
	if len(leadCheckData("top100", strings.Repeat("x", leadMaxSource))) > 64 {
		t.Error("callback_data over Telegram's 64 byte limit")
	}
}

func TestLeadMagnetSubscriberGetsFileAtOnce(t *testing.T) {
	h, st, tg, _ := leadAPI(t, "@ucimosrb")
	tg.setMember(501, true)

	postWebhook(h, tgWebhookSecret, webhookUpdate(501, 501, "anya", "Аня", "/start lm_top100_tgpost"))

	docs := tg.named("sendDocument")
	if len(docs) != 1 || docs[0].Upload {
		t.Fatalf("want one document sent by link (not uploaded through the relay), got %+v", docs)
	}
	if got := docs[0].Form.Get("document"); !strings.HasPrefix(got, testBaseURL+"/api/lead-magnet/top100/") {
		t.Errorf("document = %q, want a signed link on the app's own address", got)
	}
	if got := docs[0].Form.Get("caption"); !strings.Contains(got, "Аня") {
		t.Errorf("caption %q lacks the name", got)
	}
	if !strings.Contains(docs[0].Form.Get("reply_markup"), "web_app") {
		t.Errorf("no app button under the file: %q", docs[0].Form.Get("reply_markup"))
	}
	if len(tg.named("sendMessage")) != 0 {
		t.Error("a subscriber must get the file without a subscribe prompt")
	}
	s, _ := st.LeadMagnetStats("top100")
	if s.Started != 1 || s.AlreadyMember != 1 || s.Received != 1 || s.SubscribedForIt != 0 || s.Waiting != 0 {
		t.Errorf("stats = %+v", s)
	}
}

func TestLeadMagnetSecondSendUsesCachedFileID(t *testing.T) {
	h, _, tg, _ := leadAPI(t, "@ucimosrb")
	tg.setMember(501, true)
	tg.setMember(502, true)

	postWebhook(h, tgWebhookSecret, webhookUpdate(501, 501, "a", "А", "/start lm_top100"))
	postWebhook(h, tgWebhookSecret, webhookUpdate(502, 502, "b", "Б", "/start lm_top100"))

	docs := tg.named("sendDocument")
	if len(docs) != 2 {
		t.Fatalf("documents sent = %d, want 2", len(docs))
	}
	if !strings.HasPrefix(docs[0].Form.Get("document"), "http") || docs[1].Upload {
		t.Errorf("first send goes by link, second must reuse the file_id: %+v", docs)
	}
	if docs[1].Form.Get("document") != "FILE-ID-1" {
		t.Errorf("second send document = %q, want the cached file_id", docs[1].Form.Get("document"))
	}
}

func TestLeadMagnetNonSubscriberIsAskedThenGetsFile(t *testing.T) {
	h, st, tg, _ := leadAPI(t, "@ucimosrb")

	postWebhook(h, tgWebhookSecret, webhookUpdate(601, 601, "v", "Вика", "/start lm_top100_tgpost"))
	if len(tg.named("sendDocument")) != 0 {
		t.Fatal("a non-subscriber must not get the file")
	}
	msgs := tg.named("sendMessage")
	if len(msgs) != 1 || !strings.Contains(msgs[0].Form.Get("text"), "Вика") {
		t.Fatalf("want one prompt addressed by name, got %+v", msgs)
	}
	markup := msgs[0].Form.Get("reply_markup")
	if !strings.Contains(markup, "https://t.me/ucimosrb") || !strings.Contains(markup, "lm:top100:tgpost") {
		t.Errorf("prompt keyboard = %s, want the channel link and this magnet's check button", markup)
	}

	// presses "check" without subscribing
	postWebhook(h, tgWebhookSecret, leadCallbackUpdate(601, "lm:top100:tgpost"))
	if len(tg.named("sendDocument")) != 0 {
		t.Fatal("still not subscribed, still no file")
	}
	if ans := tg.named("answerCallbackQuery"); len(ans) != 1 || !strings.Contains(ans[0].Form.Get("text"), "не видно") {
		t.Errorf("callback answers = %+v", ans)
	}
	if got := len(tg.named("sendMessage")); got != 2 {
		t.Errorf("messages = %d, want the prompt and the 'not yet' reminder", got)
	}

	// subscribes, presses "check" again
	tg.setMember(601, true)
	postWebhook(h, tgWebhookSecret, leadCallbackUpdate(601, "lm:top100:tgpost"))
	if len(tg.named("sendDocument")) != 1 {
		t.Fatal("subscribed: the file must arrive")
	}
	s, _ := st.LeadMagnetStats("top100")
	if s.Started != 1 || s.AlreadyMember != 0 || s.SubscribedForIt != 1 || s.Received != 1 || s.Waiting != 0 {
		t.Errorf("stats = %+v", s)
	}
	if len(s.BySource) != 1 || s.BySource[0].Source != "tgpost" || s.BySource[0].SubscribedForIt != 1 {
		t.Errorf("by source = %+v", s.BySource)
	}
}

func TestLeadMagnetWaitingAreCountedAndRepeatStartKeepsFirstFacts(t *testing.T) {
	h, st, tg, _ := leadAPI(t, "@ucimosrb")
	postWebhook(h, tgWebhookSecret, webhookUpdate(701, 701, "w", "Олег", "/start lm_top100_ig"))
	// now subscribes on their own and opens the link again from another source
	tg.setMember(701, true)
	postWebhook(h, tgWebhookSecret, webhookUpdate(701, 701, "w", "Олег", "/start lm_top100_tgpost"))

	s, _ := st.LeadMagnetStats("top100")
	if s.Started != 1 || s.AlreadyMember != 0 || s.SubscribedForIt != 1 || s.Received != 1 {
		t.Errorf("stats = %+v: the first visit's facts (not subscribed, source ig) must stick", s)
	}
	if len(s.BySource) != 1 || s.BySource[0].Source != "ig" {
		t.Errorf("by source = %+v, want ig", s.BySource)
	}
}

func TestLeadMagnetUnknownSlugFallsBackToGreeting(t *testing.T) {
	h, _, tg, sink := leadAPI(t, "@ucimosrb")
	postWebhook(h, tgWebhookSecret, webhookUpdate(801, 801, "x", "Ира", "/start lm_nosuchmagnet"))
	if len(tg.named("sendDocument")) != 0 {
		t.Error("unknown magnet must not send a file")
	}
	if msgs := sink.all(); len(msgs) != 1 || !strings.Contains(msgs[0].Text, "Ира") {
		t.Errorf("want the ordinary greeting through the outbox, got %+v", msgs)
	}
}

func TestLeadMagnetWithoutChannelDeliversWithoutGate(t *testing.T) {
	h, _, tg, _ := leadAPI(t, "")
	postWebhook(h, tgWebhookSecret, webhookUpdate(901, 901, "n", "Нина", "/start lm_top100"))
	if len(tg.named("sendDocument")) != 1 || len(tg.named("getChatMember")) != 0 {
		t.Errorf("no channel configured: deliver straight away, no membership call (docs=%d)", len(tg.named("sendDocument")))
	}
}

func TestLeadMagnetRetriesAfterRateLimit(t *testing.T) {
	h, st, tg, _ := leadAPI(t, "@ucimosrb")
	tg.setMember(1001, true)
	tg.mu.Lock()
	tg.rateLimit = 1
	tg.mu.Unlock()
	postWebhook(h, tgWebhookSecret, webhookUpdate(1001, 1001, "r", "Рита", "/start lm_top100"))
	if got := len(tg.named("sendDocument")); got != 2 {
		t.Errorf("sendDocument calls = %d, want 429 then a retry", got)
	}
	if s, _ := st.LeadMagnetStats("top100"); s.Received != 1 {
		t.Errorf("stats = %+v, want the file counted as received", s)
	}
}

func TestBotCommandsIDAndStats(t *testing.T) {
	h, _, tg, _ := leadAPI(t, "@ucimosrb")
	tg.setMember(1101, true)
	postWebhook(h, tgWebhookSecret, webhookUpdate(1101, 1101, "a", "А", "/start lm_top100_tgpost"))
	before := len(tg.named("sendMessage"))

	postWebhook(h, tgWebhookSecret, webhookUpdate(555, 555, "z", "Зина", "/id"))
	msgs := tg.named("sendMessage")
	if len(msgs) != before+1 || !strings.Contains(msgs[len(msgs)-1].Form.Get("text"), "555") {
		t.Fatalf("/id must answer with the caller's id, got %+v", msgs)
	}

	// a stranger's /stats is ignored
	postWebhook(h, tgWebhookSecret, webhookUpdate(555, 555, "z", "Зина", "/stats"))
	if got := len(tg.named("sendMessage")); got != before+1 {
		t.Errorf("/stats from a non-admin got an answer")
	}

	// the admin's /stats shows the funnel
	postWebhook(h, tgWebhookSecret, webhookUpdate(999, 999, "me", "Гриша", "/stats"))
	msgs = tg.named("sendMessage")
	text := msgs[len(msgs)-1].Form.Get("text")
	for _, want := range []string{"Перешли по ссылке: 1", "уже были подписаны: 1", "Получили файл: 1 (100%)", "tgpost"} {
		if !strings.Contains(text, want) {
			t.Errorf("stats text lacks %q:\n%s", want, text)
		}
	}
}

func TestLeadMagnetFallsBackToUploadWhenTelegramCannotFetchTheLink(t *testing.T) {
	h, st, tg, _ := leadAPI(t, "@ucimosrb")
	tg.mu.Lock()
	tg.failURL = true
	tg.mu.Unlock()
	tg.setMember(1201, true)
	tg.setMember(1202, true)

	postWebhook(h, tgWebhookSecret, webhookUpdate(1201, 1201, "a", "А", "/start lm_top100"))
	docs := tg.named("sendDocument")
	if len(docs) != 2 || strings.HasPrefix(docs[0].Form.Get("document"), "http") == false || !docs[1].Upload {
		t.Fatalf("want a failed link attempt then an upload, got %+v", docs)
	}
	if s, _ := st.LeadMagnetStats("top100"); s.Received != 1 {
		t.Errorf("stats = %+v, want the file counted as received", s)
	}

	// The uploaded id is not cached (the upload path is the unreliable one), so the next person tries the link again.
	tg.mu.Lock()
	tg.failURL = false
	tg.mu.Unlock()
	postWebhook(h, tgWebhookSecret, webhookUpdate(1202, 1202, "b", "Б", "/start lm_top100"))
	docs = tg.named("sendDocument")
	last := docs[len(docs)-1]
	if last.Upload || !strings.HasPrefix(last.Form.Get("document"), "http") {
		t.Errorf("second person must get the link attempt, not a cached upload: %+v", last)
	}
}

func TestLeadMagnetFileLinks(t *testing.T) {
	h, _, tg, _ := leadAPI(t, "@ucimosrb")
	tg.setMember(1301, true)
	postWebhook(h, tgWebhookSecret, webhookUpdate(1301, 1301, "a", "А", "/start lm_top100"))
	link := tg.named("sendDocument")[0].Form.Get("document")
	path := strings.TrimPrefix(link, testBaseURL)

	get := func(p string) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest("GET", p, nil))
		return rr
	}
	ok := get(path)
	if ok.Code != http.StatusOK || ok.Header().Get("Content-Type") != "application/pdf" ||
		!strings.HasPrefix(ok.Body.String(), "%PDF") || !strings.Contains(ok.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("valid link = %d %q, body starts %q", ok.Code, ok.Header().Get("Content-Type"), ok.Body.String()[:8])
	}

	if !strings.HasSuffix(path, "/ucimo-top100-slov.pdf") {
		t.Errorf("link %q must end in a .pdf file name, Telegram names the file after it", path)
	}
	parts := strings.Split(path, "/") // "", api, lead-magnet, slug, exp, sig, name
	tampered := strings.Join([]string{"", "api", "lead-magnet", "top100", parts[4], strings.Repeat("0", len(parts[5])), parts[6]}, "/")
	if rr := get(tampered); rr.Code != http.StatusForbidden {
		t.Errorf("forged signature = %d, want 403", rr.Code)
	}
	if rr := get("/api/lead-magnet/top100/9999999999/" + parts[5] + "/x.pdf"); rr.Code != http.StatusForbidden {
		t.Errorf("signature for another expiry = %d, want 403", rr.Code)
	}
	if rr := get("/api/lead-magnet/nosuch/" + parts[4] + "/" + parts[5] + "/x.pdf"); rr.Code != http.StatusNotFound {
		t.Errorf("unknown magnet = %d, want 404", rr.Code)
	}

	// an expired link, even one with a correct signature
	hh := handlers{Deps: Deps{Config: configWithSecret(), TelegramWebhookSecret: tgWebhookSecret}}
	past := fixedNow.Add(-time.Minute).Unix()
	expired := fmt.Sprintf("/api/lead-magnet/top100/%d/%s/x.pdf", past, hh.leadSign("top100", past))
	if rr := get(expired); rr.Code != http.StatusForbidden {
		t.Errorf("expired link = %d, want 403", rr.Code)
	}
}

func configWithSecret() config.Config { return config.Config{TelegramBotToken: tgTestToken} }
