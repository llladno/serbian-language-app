package store

import "testing"

func TestEnqueueAndDequeueOutboxMessage(t *testing.T) {
	s := newStore(t)
	button := OutboxButton{Label: "Открыть", Type: "web_app", Target: "https://ucimo.ru/profile"}
	if err := s.EnqueueBotMessage(555, "hi", button, PriorityHigh, day0); err != nil {
		t.Fatalf("EnqueueBotMessage: %v", err)
	}

	msg, ok, err := s.NextPendingOutboxMessage()
	if err != nil {
		t.Fatalf("NextPendingOutboxMessage: %v", err)
	}
	if !ok {
		t.Fatal("NextPendingOutboxMessage: ok = false, want a queued row")
	}
	if msg.ChatID != 555 || msg.Text != "hi" || msg.Button != button {
		t.Errorf("dequeued = %+v, want ChatID=555 Text=hi Button=%+v", msg, button)
	}
}

func TestNextPendingOutboxMessageEmptyQueue(t *testing.T) {
	s := newStore(t)
	_, ok, err := s.NextPendingOutboxMessage()
	if err != nil {
		t.Fatalf("NextPendingOutboxMessage: %v", err)
	}
	if ok {
		t.Error("NextPendingOutboxMessage: ok = true on an empty queue, want false")
	}
}

func TestOutboxPriorityOrdersBeforeAge(t *testing.T) {
	s := newStore(t)
	// Enqueue normal first, then high — high must still come out first.
	if err := s.EnqueueBotMessage(1, "normal", OutboxButton{}, PriorityNormal, day0); err != nil {
		t.Fatal(err)
	}
	if err := s.EnqueueBotMessage(2, "high", OutboxButton{}, PriorityHigh, day0.Add(1000)); err != nil {
		t.Fatal(err)
	}
	msg, _, err := s.NextPendingOutboxMessage()
	if err != nil {
		t.Fatal(err)
	}
	if msg.Text != "high" {
		t.Errorf("first dequeued = %q, want the high-priority row even though it was enqueued later", msg.Text)
	}
}

func TestMarkOutboxSentRemovesFromPendingQueue(t *testing.T) {
	s := newStore(t)
	if err := s.EnqueueBotMessage(1, "hi", OutboxButton{}, PriorityHigh, day0); err != nil {
		t.Fatal(err)
	}
	msg, _, err := s.NextPendingOutboxMessage()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkOutboxSent(msg.ID, day0); err != nil {
		t.Fatalf("MarkOutboxSent: %v", err)
	}
	_, ok, err := s.NextPendingOutboxMessage()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("NextPendingOutboxMessage after MarkOutboxSent: ok = true, want the row gone from the pending queue")
	}
}

func TestMarkOutboxFailedRemovesFromPendingQueue(t *testing.T) {
	s := newStore(t)
	if err := s.EnqueueBotMessage(1, "hi", OutboxButton{}, PriorityHigh, day0); err != nil {
		t.Fatal(err)
	}
	msg, _, err := s.NextPendingOutboxMessage()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkOutboxFailed(msg.ID, "bot was blocked"); err != nil {
		t.Fatalf("MarkOutboxFailed: %v", err)
	}
	_, ok, err := s.NextPendingOutboxMessage()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("NextPendingOutboxMessage after MarkOutboxFailed: ok = true, want the row gone from the pending queue")
	}
}
