package store

import (
	"testing"
	"time"
)

func TestSetUserAttributionStoresFirstTouchOnly(t *testing.T) {
	s, u := newUser(t)

	if err := s.SetUserAttribution(u.user, Attribution{
		UtmSource: "vk", UtmMedium: "social", UtmCampaign: "launch",
	}); err != nil {
		t.Fatalf("SetUserAttribution: %v", err)
	}

	// A second call (e.g. a later visit through a different link) must not
	// overwrite the first touch.
	if err := s.SetUserAttribution(u.user, Attribution{
		UtmSource: "instagram", UtmMedium: "story", UtmCampaign: "later",
	}); err != nil {
		t.Fatalf("SetUserAttribution (2nd): %v", err)
	}

	row, err := s.UserByID(u.user)
	if err != nil {
		t.Fatalf("UserByID: %v", err)
	}
	if row.UtmSource != "vk" || row.UtmMedium != "social" || row.UtmCampaign != "launch" {
		t.Errorf("attribution = %+v, want the first-touch vk/social/launch", row)
	}
}

func TestSetUserAttributionEmptyIsNoop(t *testing.T) {
	s, u := newUser(t)
	if err := s.SetUserAttribution(u.user, Attribution{}); err != nil {
		t.Fatalf("SetUserAttribution(empty): %v", err)
	}
	row, err := s.UserByID(u.user)
	if err != nil {
		t.Fatalf("UserByID: %v", err)
	}
	if row.UtmSource != "" {
		t.Errorf("UtmSource = %q, want empty", row.UtmSource)
	}
}

func TestRecordLinkVisit(t *testing.T) {
	s, _ := newUser(t)

	if err := s.RecordLinkVisit(Attribution{
		UtmSource: "vk", UtmMedium: "social", UtmCampaign: "launch",
	}, day0); err != nil {
		t.Fatalf("RecordLinkVisit: %v", err)
	}

	rows, err := s.db.Query(`SELECT utm_source, utm_medium, utm_campaign, created_at FROM link_visits`)
	if err != nil {
		t.Fatalf("query link_visits: %v", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var src, medium, campaign, createdAt string
		if err := rows.Scan(&src, &medium, &campaign, &createdAt); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if src != "vk" || medium != "social" || campaign != "launch" {
			t.Errorf("row = %s/%s/%s, want vk/social/launch", src, medium, campaign)
		}
		count++
	}
	if count != 1 {
		t.Fatalf("link_visits has %d rows, want 1", count)
	}
}

func TestRecordLinkVisitEmptyIsNoop(t *testing.T) {
	s, _ := newUser(t)
	if err := s.RecordLinkVisit(Attribution{}, day0); err != nil {
		t.Fatalf("RecordLinkVisit(empty): %v", err)
	}
	rows, err := s.db.Query(`SELECT COUNT(*) FROM link_visits`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	rows.Next()
	var count int
	if err := rows.Scan(&count); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if count != 0 {
		t.Fatalf("link_visits has %d rows, want 0", count)
	}
}

func TestRecordLinkVisitTruncatesLongFields(t *testing.T) {
	s, _ := newUser(t)
	long := ""
	for i := 0; i < 200; i++ {
		long += "a"
	}
	if err := s.RecordLinkVisit(Attribution{UtmSource: long, UtmMedium: "x", UtmCampaign: "y"}, day0); err != nil {
		t.Fatalf("RecordLinkVisit: %v", err)
	}
	rows, err := s.db.Query(`SELECT utm_source FROM link_visits`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	rows.Next()
	var src string
	if err := rows.Scan(&src); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len([]rune(src)) != maxUTMLen {
		t.Errorf("utm_source length = %d, want %d", len([]rune(src)), maxUTMLen)
	}
	_ = time.Now
}
