package outbox

import (
	"net/http"
	"net/http/httptest"
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
	restore := telegram.SetAPIBaseForTesting(srv.URL)
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
	restore := telegram.SetAPIBaseForTesting(srv.URL)
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
	restore := telegram.SetAPIBaseForTesting(srv.URL)
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
	restore := telegram.SetAPIBaseForTesting(srv.URL)
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
