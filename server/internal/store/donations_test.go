package store

import (
	"testing"
	"time"
)

func TestCreateAndListDonations(t *testing.T) {
	s, u := newUser(t)

	if err := s.CreateDonation(Donation{
		UserID: u.user, TelegramUserID: "555", TelegramUsername: "alice",
		AmountMinorUnits: 10000, Currency: "RUB", EventType: "newDonation",
		TributeEventID: "evt_1", RawPayload: `{"name":"newDonation"}`,
	}, day0); err != nil {
		t.Fatalf("CreateDonation: %v", err)
	}
	if err := s.CreateDonation(Donation{
		TelegramUserID: "999", AmountMinorUnits: 50000, Currency: "RUB", EventType: "newDonation",
		TributeEventID: "evt_2", RawPayload: `{"name":"newDonation"}`,
	}, day0.Add(time.Hour)); err != nil {
		t.Fatalf("CreateDonation (2nd): %v", err)
	}

	got, err := s.ListDonations()
	if err != nil {
		t.Fatalf("ListDonations: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListDonations returned %d rows, want 2: %+v", len(got), got)
	}
	// newest first
	if got[0].TelegramUserID != "999" || got[0].UserID != "" {
		t.Errorf("got[0] = %+v, want the unlinked donor's donation", got[0])
	}
	if got[1].UserID != u.user || got[1].AmountMinorUnits != 10000 {
		t.Errorf("got[1] = %+v, want %s's linked donation", got[1], u.user)
	}
}

func TestCreateDonationIdempotentOnEventID(t *testing.T) {
	s, _ := newUser(t)
	d := Donation{
		TelegramUserID: "555", AmountMinorUnits: 10000, Currency: "RUB",
		EventType: "newDonation", TributeEventID: "evt_dup", RawPayload: `{}`,
	}
	if err := s.CreateDonation(d, day0); err != nil {
		t.Fatalf("CreateDonation (1st): %v", err)
	}
	if err := s.CreateDonation(d, day0.Add(time.Minute)); err != nil {
		t.Fatalf("CreateDonation (retry): %v", err)
	}
	got, err := s.ListDonations()
	if err != nil {
		t.Fatalf("ListDonations: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListDonations = %d rows after a retried delivery, want 1 (idempotent)", len(got))
	}
}

func TestCreateDonationRejectsEmptyTelegramUserID(t *testing.T) {
	s, _ := newUser(t)
	err := s.CreateDonation(Donation{
		TributeEventID: "evt_x", AmountMinorUnits: 100, Currency: "RUB",
		EventType: "newDonation", RawPayload: `{}`,
	}, day0)
	if err == nil {
		t.Fatal("CreateDonation with empty telegram_user_id = nil error, want error")
	}
}
