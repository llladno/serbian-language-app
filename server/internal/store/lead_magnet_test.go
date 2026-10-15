package store

import (
	"testing"
	"time"
)

func TestLeadMagnetFunnel(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	at := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

	// 1 was already a subscriber, got the file; 2 was not, subscribed and got it; 3 is waiting.
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(s.LeadMagnetStart("top100", 1, "tgpost", true, at))
	must(s.LeadMagnetDelivered("top100", 1, at))
	must(s.LeadMagnetStart("top100", 2, "tgpost", false, at))
	must(s.LeadMagnetDelivered("top100", 2, at))
	must(s.LeadMagnetStart("top100", 3, "ig", false, at))
	// a repeat visit changes nothing; a second delivery only raises the count
	must(s.LeadMagnetStart("top100", 2, "ig", true, at))
	must(s.LeadMagnetDelivered("top100", 2, at.Add(time.Hour)))
	// another magnet does not leak in
	must(s.LeadMagnetStart("other", 9, "x", true, at))

	got, err := s.LeadMagnetStats("top100")
	if err != nil {
		t.Fatal(err)
	}
	if got.Started != 3 || got.AlreadyMember != 1 || got.SubscribedForIt != 1 || got.Received != 2 || got.Waiting != 1 {
		t.Errorf("stats = %+v", got)
	}
	if len(got.BySource) != 2 || got.BySource[0].Source != "tgpost" || got.BySource[0].Started != 2 ||
		got.BySource[0].Received != 2 || got.BySource[0].SubscribedForIt != 1 || got.BySource[1].Source != "ig" {
		t.Errorf("by source = %+v", got.BySource)
	}

	empty, err := s.LeadMagnetStats("nothing")
	if err != nil || empty.Started != 0 || empty.Received != 0 {
		t.Errorf("empty stats = %+v, %v", empty, err)
	}
}

func TestLeadMagnetFileIDCache(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	at := time.Now()
	if _, ok, _ := s.LeadMagnetFileID("top100", "sha-a"); ok {
		t.Fatal("empty cache reported a hit")
	}
	if err := s.SaveLeadMagnetFileID("top100", "sha-a", "FILE-1", at); err != nil {
		t.Fatal(err)
	}
	if id, ok, _ := s.LeadMagnetFileID("top100", "sha-a"); !ok || id != "FILE-1" {
		t.Errorf("got %q,%v", id, ok)
	}
	if _, ok, _ := s.LeadMagnetFileID("top100", "sha-b"); ok {
		t.Error("a changed file must not reuse the old file_id")
	}
	if err := s.SaveLeadMagnetFileID("top100", "sha-b", "FILE-2", at); err != nil {
		t.Fatal(err)
	}
	if id, ok, _ := s.LeadMagnetFileID("top100", "sha-b"); !ok || id != "FILE-2" {
		t.Errorf("after update got %q,%v", id, ok)
	}
}
