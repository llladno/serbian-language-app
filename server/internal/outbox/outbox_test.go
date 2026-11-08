package outbox

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestProcessNextEmptyQueue(t *testing.T) {
	st := newTestStore(t)
	ok, wait, err := ProcessNext(st, "tok", time.Now())
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	if ok || wait != 0 {
		t.Errorf("ProcessNext on empty queue = (%v, %v), want (false, 0)", ok, wait)
	}
}

func TestProcessNextSendsAndMarksSent(t *testing.T) {
	var gotChatID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotChatID = r.Form.Get("chat_id")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer srv.Close()
	restore := telegram.SetAPIBase(srv.URL)
	defer restore()

	st := newTestStore(t)
	if err := st.EnqueueBotMessage(555, "hi", store.OutboxButton{}, store.PriorityHigh, time.Now()); err != nil {
		t.Fatal(err)
	}

	ok, wait, err := ProcessNext(st, "tok", time.Now())
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	if !ok || wait != 0 {
		t.Fatalf("ProcessNext = (%v, %v), want (true, 0)", ok, wait)
	}
	if gotChatID != "555" {
		t.Errorf("chat_id sent to Telegram = %q, want 555", gotChatID)
	}
	if _, pending, _ := st.NextPendingOutboxMessage(); pending {
		t.Error("message still pending after a successful send, want it marked sent")
	}
}

func TestProcessNextLeavesRowPendingOn429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"ok":false,"error_code":429,"description":"Too Many Requests","parameters":{"retry_after":3}}`))
	}))
	defer srv.Close()
	restore := telegram.SetAPIBase(srv.URL)
	defer restore()

	st := newTestStore(t)
	if err := st.EnqueueBotMessage(555, "hi", store.OutboxButton{}, store.PriorityHigh, time.Now()); err != nil {
		t.Fatal(err)
	}

	ok, wait, err := ProcessNext(st, "tok", time.Now())
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	if !ok || wait != 3*time.Second {
		t.Fatalf("ProcessNext = (%v, %v), want (true, 3s)", ok, wait)
	}
	if _, pending, _ := st.NextPendingOutboxMessage(); !pending {
		t.Error("message no longer pending after a 429, want it left in the queue for a retry")
	}
}

func TestProcessNextMarksFailedOnOtherErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`))
	}))
	defer srv.Close()
	restore := telegram.SetAPIBase(srv.URL)
	defer restore()

	st := newTestStore(t)
	if err := st.EnqueueBotMessage(555, "hi", store.OutboxButton{}, store.PriorityHigh, time.Now()); err != nil {
		t.Fatal(err)
	}

	ok, wait, err := ProcessNext(st, "tok", time.Now())
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	if !ok || wait != 0 {
		t.Fatalf("ProcessNext = (%v, %v), want (true, 0)", ok, wait)
	}
	if _, pending, _ := st.NextPendingOutboxMessage(); pending {
		t.Error("message still pending after a permanent error, want it marked failed (removed from the pending queue)")
	}
}

func TestProcessNextSendsHighPriorityBeforeNormal(t *testing.T) {
	var order []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		order = append(order, r.Form.Get("text"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer srv.Close()
	restore := telegram.SetAPIBase(srv.URL)
	defer restore()

	st := newTestStore(t)
	now := time.Now()
	if err := st.EnqueueBotMessage(1, "normal", store.OutboxButton{}, store.PriorityNormal, now); err != nil {
		t.Fatal(err)
	}
	if err := st.EnqueueBotMessage(2, "high", store.OutboxButton{}, store.PriorityHigh, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ProcessNext(st, "tok", now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ProcessNext(st, "tok", now); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "high" || order[1] != "normal" {
		t.Errorf("send order = %v, want [high, normal] regardless of enqueue time", order)
	}
}

// The two-button channel keyboard travels through the one-button outbox as a
// preset: its type, with the channel's address as the target.
func TestProcessNextSendsTheChannelKeyboard(t *testing.T) {
	var markup string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		markup = r.Form.Get("reply_markup")
		w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer srv.Close()
	defer telegram.SetAPIBase(srv.URL)()

	st := newTestStore(t)
	if err := st.EnqueueBotMessage(555, "hi", store.OutboxButton{Type: "channel_check", Target: "https://t.me/ucimosrb"}, store.PriorityHigh, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ProcessNext(st, "tok", time.Now()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(markup, "https://t.me/ucimosrb") || !strings.Contains(markup, telegram.CallbackCheckChannel) {
		t.Errorf("reply_markup = %s, want the channel link and the check callback", markup)
	}
}

// Every keyboard has to survive being flattened into one (label, type, target)
// row and rebuilt, or a reminder's "stop" button would arrive as a dead link.
func TestButtonRowRoundTrip(t *testing.T) {
	prefs := telegram.PrefsState{Learning: true, News: true}
	cases := []struct {
		name string
		in   *telegram.InlineButton
		want store.OutboxButton
	}{
		{"none", nil, store.OutboxButton{}},
		{"web app", &telegram.InlineButton{Label: "Открыть", WebAppURL: "https://x/app"}, store.OutboxButton{Label: "Открыть", Type: "web_app", Target: "https://x/app"}},
		{"link", &telegram.InlineButton{Label: "Сайт", URL: "https://x"}, store.OutboxButton{Label: "Сайт", Type: "url", Target: "https://x"}},
		{"channel", &telegram.InlineButton{ChannelURL: "https://t.me/c"}, store.OutboxButton{Type: "channel_check", Target: "https://t.me/c"}},
		{"stop button", &telegram.InlineButton{Label: telegram.PrefOffLabel, PrefOff: telegram.PrefLearning}, store.OutboxButton{Label: telegram.PrefOffLabel, Type: "pref_off", Target: "learning"}},
		{"settings", &telegram.InlineButton{Prefs: &prefs}, store.OutboxButton{Type: "prefs", Target: "101"}},
		{"repeat nudge", &telegram.InlineButton{Label: "Повторить", RepeatNudgeURL: "https://x/repeat"}, store.OutboxButton{Label: "Повторить", Type: "repeat_nudge", Target: "https://x/repeat"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := ButtonRow(tc.in)
			if row != tc.want {
				t.Fatalf("ButtonRow = %+v, want %+v", row, tc.want)
			}
			back := buttonFromRow(row)
			if tc.in == nil {
				if back != nil {
					t.Fatalf("buttonFromRow(empty) = %+v, want nil", back)
				}
				return
			}
			if back.Label != tc.in.Label || back.WebAppURL != tc.in.WebAppURL || back.URL != tc.in.URL ||
				back.ChannelURL != tc.in.ChannelURL || back.PrefOff != tc.in.PrefOff || back.RepeatNudgeURL != tc.in.RepeatNudgeURL {
				t.Errorf("round trip = %+v, want %+v", back, tc.in)
			}
			if (back.Prefs == nil) != (tc.in.Prefs == nil) || (back.Prefs != nil && *back.Prefs != *tc.in.Prefs) {
				t.Errorf("round trip prefs = %+v, want %+v", back.Prefs, tc.in.Prefs)
			}
		})
	}
}

// The button reaches Telegram as callback_data the webhook understands.
func TestProcessNextSendsTheStopButton(t *testing.T) {
	var markup string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		markup = r.Form.Get("reply_markup")
		w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer srv.Close()
	defer telegram.SetAPIBase(srv.URL)()

	st := newTestStore(t)
	row := ButtonRow(&telegram.InlineButton{Label: telegram.PrefOffLabel, PrefOff: telegram.PrefLearning})
	if err := st.EnqueueBotMessage(555, "hi", row, store.PriorityNormal, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ProcessNext(st, "tok", time.Now()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(markup, `"callback_data":"prefoff:learning"`) {
		t.Errorf("reply_markup = %s, want a prefoff:learning callback", markup)
	}
}

// A queued photo goes out by its address the first time and by the file_id Telegram returned every time after.
func TestProcessNextSendsPhotoByAddressThenByFileID(t *testing.T) {
	type call struct{ method, photo, caption, markup string }
	var calls []call
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		calls = append(calls, call{r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:], r.Form.Get("photo"), r.Form.Get("caption"), r.Form.Get("reply_markup")})
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":{"photo":[{"file_id":"small"},{"file_id":"FILE-ID-1"}]}}`))
	}))
	defer srv.Close()
	defer telegram.SetAPIBase(srv.URL)()

	st := newTestStore(t)
	button := store.OutboxButton{Label: "Загляну на минутку", Type: "campaign", Target: "https://ucimo.ru/lesson/00?c=abc"}
	for i := 0; i < 2; i++ {
		if err := st.EnqueueBotPhoto(555, "https://img.test/a.png", "Подпись", button, store.PriorityNormal, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if ok, _, err := ProcessNext(st, "tok", time.Now()); err != nil || !ok {
			t.Fatalf("ProcessNext %d = %v, %v", i, ok, err)
		}
	}
	if len(calls) != 2 || calls[0].method != "sendPhoto" || calls[1].method != "sendPhoto" {
		t.Fatalf("calls = %+v, want two sendPhoto", calls)
	}
	if calls[0].photo != "https://img.test/a.png" {
		t.Errorf("first send photo = %q, want the address", calls[0].photo)
	}
	if calls[1].photo != "FILE-ID-1" {
		t.Errorf("second send photo = %q, want the cached file_id", calls[1].photo)
	}
	if calls[0].caption != "Подпись" {
		t.Errorf("caption = %q", calls[0].caption)
	}
	// The keyboard is the Mini App button over the switch-off button.
	if !strings.Contains(calls[0].markup, `"web_app":{"url":"https://ucimo.ru/lesson/00?c=abc"}`) || !strings.Contains(calls[0].markup, `"callback_data":"prefoff:learning"`) {
		t.Errorf("reply_markup = %s", calls[0].markup)
	}
	if st2, _, _ := st.NextPendingOutboxMessage(); st2.ID != 0 {
		t.Errorf("a message is still pending: %+v", st2)
	}
}

func TestCampaignButtonSurvivesTheOutbox(t *testing.T) {
	b := &telegram.InlineButton{Label: "Послушаю č и ć", CampaignURL: "https://ucimo.ru/lesson/01?c=abc"}
	got := buttonFromRow(ButtonRow(b))
	if got == nil || got.Label != b.Label || got.CampaignURL != b.CampaignURL {
		t.Errorf("round trip = %+v, want %+v", got, b)
	}
}

// Two workers can read the same pending row (the old and the new container during a deploy): only one may send it.
func TestProcessNextSendsAMessageOnlyOnceWhenWorkersRace(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		time.Sleep(20 * time.Millisecond) // a slow send, so the others have time to read the same row
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer srv.Close()
	defer telegram.SetAPIBase(srv.URL)()

	st := newTestStore(t)
	if err := st.EnqueueBotMessage(555, "hi", store.OutboxButton{}, store.PriorityNormal, time.Now()); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := ProcessNext(st, "tok", time.Now()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if calls != 1 {
		t.Errorf("Telegram was called %d times for one message, want 1", calls)
	}
}

// A send Telegram asks to retry later (429) is released back to the queue, not lost and not repeated while it waits.
func TestProcessNextReleasesAMessageOnRateLimit(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		if hits == 1 {
			w.WriteHeader(429)
			w.Write([]byte(`{"ok":false,"error_code":429,"description":"Too Many Requests: retry after 1","parameters":{"retry_after":1}}`))
			return
		}
		w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer srv.Close()
	defer telegram.SetAPIBase(srv.URL)()

	st := newTestStore(t)
	if err := st.EnqueueBotMessage(555, "hi", store.OutboxButton{}, store.PriorityNormal, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, wait, err := ProcessNext(st, "tok", time.Now()); err != nil || wait == 0 {
		t.Fatalf("first send: wait %v err %v, want a rate-limit wait", wait, err)
	}
	if ok, _, err := ProcessNext(st, "tok", time.Now()); err != nil || !ok {
		t.Fatalf("retry: %v %v", ok, err)
	}
	if hits != 2 {
		t.Errorf("Telegram was called %d times, want 2 (the limited one and the retry)", hits)
	}
	if _, found, _ := st.NextPendingOutboxMessage(); found {
		t.Error("the message is still pending after it was sent")
	}
}
