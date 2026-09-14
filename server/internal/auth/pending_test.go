package auth

import (
	"testing"
	"time"
)

func TestPendingStoreLoginRoundTrip(t *testing.T) {
	s := NewPendingStore()
	raw, err := s.Create("")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	userID, ok := s.Lookup(raw)
	if !ok {
		t.Fatal("Lookup: want ok=true right after Create")
	}
	if userID != "" {
		t.Errorf("Lookup userID = %q, want empty for a login token", userID)
	}

	if got := s.Poll(raw); got.Status != PendingWaiting {
		t.Errorf("Poll before resolve = %+v, want status=pending", got)
	}

	s.Resolve(raw, "usr_abc")

	got := s.Poll(raw)
	if got.Status != PendingDone || got.ResolvedUserID != "usr_abc" {
		t.Errorf("Poll after resolve = %+v, want done/usr_abc", got)
	}
}

func TestPendingStoreLinkCarriesCallerUserID(t *testing.T) {
	s := NewPendingStore()
	raw, err := s.Create("usr_caller")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	userID, ok := s.Lookup(raw)
	if !ok || userID != "usr_caller" {
		t.Errorf("Lookup = (%q, %v), want (usr_caller, true)", userID, ok)
	}

	s.Resolve(raw, "usr_caller") // linking echoes the same id back
	got := s.Poll(raw)
	if got.Status != PendingDone || got.CallerUserID != "usr_caller" {
		t.Errorf("Poll after link resolve = %+v, want done with CallerUserID=usr_caller", got)
	}
}

func TestPendingStorePollIsOneShot(t *testing.T) {
	s := NewPendingStore()
	raw, _ := s.Create("")
	s.Resolve(raw, "usr_abc")

	first := s.Poll(raw)
	if first.Status != PendingDone {
		t.Fatalf("first Poll = %+v, want done", first)
	}
	second := s.Poll(raw)
	if second.Status != PendingError {
		t.Errorf("second Poll = %+v, want error (token consumed, not replayable)", second)
	}
}

func TestPendingStoreFail(t *testing.T) {
	s := NewPendingStore()
	raw, _ := s.Create("usr_caller")
	s.Fail(raw, "telegram_taken")

	got := s.Poll(raw)
	if got.Status != PendingError || got.Error != "telegram_taken" {
		t.Errorf("Poll after Fail = %+v, want error/telegram_taken", got)
	}
}

func TestPendingStoreUnknownTokenIsError(t *testing.T) {
	s := NewPendingStore()
	got := s.Poll("never-issued")
	if got.Status != PendingError {
		t.Errorf("Poll unknown token = %+v, want error", got)
	}
}

func TestPendingStoreExpiry(t *testing.T) {
	s := NewPendingStore()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s.SetNow(func() time.Time { return now })

	raw, _ := s.Create("")
	now = now.Add(pendingTTL + time.Second)

	if _, ok := s.Lookup(raw); ok {
		t.Error("Lookup after expiry: want ok=false")
	}
	if got := s.Poll(raw); got.Status != PendingError {
		t.Errorf("Poll after expiry = %+v, want error", got)
	}
}

func TestPendingStoreResolveUnknownTokenIsNoop(t *testing.T) {
	s := NewPendingStore()
	s.Resolve("never-issued", "usr_x") // must not panic
	s.Fail("never-issued", "oops")     // must not panic
}
