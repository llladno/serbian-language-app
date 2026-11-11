package store

import (
	"testing"
	"time"
)

// The places that sell the tariffs report each showing and each press on the way to them; the admin
// counts the rows. Here: what is written, and what is not.

func eventCount(t *testing.T, s *Store, where string) string {
	t.Helper()
	got, err := s.DebugQuery("SELECT COUNT(*) FROM tariff_events WHERE " + where)
	if err != nil || len(got) != 1 {
		t.Fatalf("count %q: %v %v", where, got, err)
	}
	return got[0]
}

func TestATariffEventIsRecordedWithItsKindSourceAndTime(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	for _, c := range []struct{ kind, source string }{
		{TariffEventView, TariffSourceProfileBanner},
		{TariffEventClick, TariffSourceProfileBanner},
		{TariffEventView, TariffSourceRepeatLimit},
		{TariffEventView, TariffSourceUnlockModal},
		{TariffEventClick, TariffSourceStoryModal},
		{TariffEventDismiss, TariffSourceProfileBanner},
	} {
		recorded, err := s.RecordTariffEvent(uid, c.kind, c.source, day0)
		if err != nil || !recorded {
			t.Fatalf("%s/%s = (%v, %v), want recorded", c.kind, c.source, recorded, err)
		}
	}
	if got := eventCount(t, s, "kind = 'view'"); got != "3" {
		t.Errorf("views = %s, want 3", got)
	}
	if got := eventCount(t, s, "kind = 'click' AND source = 'profile_banner'"); got != "1" {
		t.Errorf("banner clicks = %s, want 1", got)
	}
	if got := eventCount(t, s, "kind = 'dismiss' AND source = 'profile_banner'"); got != "1" {
		t.Errorf("banner dismissals = %s, want 1", got)
	}
	at, _ := s.DebugQuery("SELECT created_at FROM tariff_events WHERE source = 'repeat_limit'")
	if len(at) != 1 || at[0] != day0.UTC().Format(time.RFC3339) {
		t.Errorf("created_at = %v, want %s", at, day0.UTC().Format(time.RFC3339))
	}
}

// A tab that reloads, or a component that mounts twice, must not look like two showings.
func TestTheSameEventWithinAFewSecondsIsRecordedOnce(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	first, _ := s.RecordTariffEvent(uid, TariffEventView, TariffSourceProfileBanner, day0)
	again, err := s.RecordTariffEvent(uid, TariffEventView, TariffSourceProfileBanner, day0.Add(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !first || again {
		t.Fatalf("first = %v, again = %v, want recorded then ignored", first, again)
	}
	// later it is a new showing: this is how "how many times" is told
	later, _ := s.RecordTariffEvent(uid, TariffEventView, TariffSourceProfileBanner, day0.Add(2*time.Minute))
	if !later {
		t.Error("a showing two minutes later was dropped")
	}
	if got := eventCount(t, s, "1 = 1"); got != "2" {
		t.Errorf("rows = %s, want 2", got)
	}
}

func TestEventsOfDifferentKindSourceOrLearnerDoNotHideEachOther(t *testing.T) {
	s := newStore(t)
	a, _ := s.CreateUser("Анна")
	b, _ := s.CreateUser("Борис")
	for _, c := range []struct {
		user, kind, source string
	}{
		{a, TariffEventView, TariffSourceProfileBanner},
		{a, TariffEventClick, TariffSourceProfileBanner}, // another kind
		{a, TariffEventView, TariffSourceRepeatLimit},    // another source
		{b, TariffEventView, TariffSourceProfileBanner},  // another learner
	} {
		if ok, err := s.RecordTariffEvent(c.user, c.kind, c.source, day0); err != nil || !ok {
			t.Fatalf("%+v = (%v, %v), want recorded", c, ok, err)
		}
	}
}

func TestAnUnknownKindOrSourceIsRefusedAndWritesNothing(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	for _, c := range []struct{ kind, source string }{
		{"hover", TariffSourceProfileBanner},
		{TariffEventView, "somewhere"},
		{"", ""},
	} {
		if ok, err := s.RecordTariffEvent(uid, c.kind, c.source, day0); err == nil || ok {
			t.Errorf("%q/%q = (%v, %v), want an error", c.kind, c.source, ok, err)
		}
	}
	if got := eventCount(t, s, "1 = 1"); got != "0" {
		t.Errorf("rows = %s after refused events, want 0", got)
	}
}

// A detail rides along ("phase:2"), is cleaned, and tells two events apart for the dedupe.
func TestAnEventKeepsACleanedDetailAndDetailsAreDedupedApart(t *testing.T) {
	s := newStore(t)
	uid, _ := s.CreateUser("Гриша")
	if ok, err := s.RecordTariffEventDetail(uid, TariffEventView, TariffSourceUnlockModal, "phase:2", day0); err != nil || !ok {
		t.Fatalf("phase:2 = (%v, %v), want recorded", ok, err)
	}
	// the same event about another level within seconds is a different event
	if ok, _ := s.RecordTariffEventDetail(uid, TariffEventView, TariffSourceUnlockModal, "phase:3", day0.Add(5*time.Second)); !ok {
		t.Error("an event about another level was taken for the first")
	}
	if ok, _ := s.RecordTariffEventDetail(uid, TariffEventView, TariffSourceUnlockModal, "phase:3", day0.Add(8*time.Second)); ok {
		t.Error("the same event about the same level twice in seconds was written twice")
	}
	if ok, _ := s.RecordTariffEventDetail(uid, TariffEventPress, TariffSourceCheckout, "<b>year</b> \"; DROP", day0); !ok {
		t.Fatal("a messy detail should be cleaned, not refused")
	}
	got, _ := s.DebugQuery("SELECT detail FROM tariff_events WHERE kind = 'press'")
	if len(got) != 1 || got[0] != "byearbDROP" {
		t.Errorf("cleaned detail = %v, want byearbDROP (letters and digits only survive)", got)
	}
	long := ""
	for i := 0; i < 100; i++ {
		long += "a"
	}
	s.RecordTariffEventDetail(uid, TariffEventOpen, TariffSourceCheckout, long, day0)
	if got, _ := s.DebugQuery("SELECT LENGTH(detail) FROM tariff_events WHERE kind = 'open'"); got[0] != "64" {
		t.Errorf("long detail kept %s characters, want 64", got[0])
	}
}

// The checkout takes the steps of paying, the places that sell take view/click/dismiss/earn, and not the other's.
func TestKindsBelongToTheirSources(t *testing.T) {
	for _, c := range []struct {
		kind, source string
		want         bool
	}{
		{TariffEventEarn, TariffSourceUnlockModal, true},
		{TariffEventDismiss, TariffSourceUnlockModal, true},
		{TariffEventClaimView, TariffSourceUnlockModal, true},
		{TariffEventClaim, TariffSourceUnlockModal, true},
		{TariffEventClaim, TariffSourceStoryModal, false},
		{TariffEventClaimView, TariffSourceCheckout, false},
		{TariffEventOpen, TariffSourceCheckout, true},
		{TariffEventPress, TariffSourceCheckout, true},
		{TariffEventReceiptView, TariffSourceCheckout, true},
		{TariffEventReceiptSend, TariffSourceCheckout, true},
		{TariffEventReceiptCancel, TariffSourceCheckout, true},
		{TariffEventError, TariffSourceCheckout, true},
		{TariffEventView, TariffSourceCheckout, false},
		{TariffEventPress, TariffSourceProfileBanner, false},
		{TariffEventError, TariffSourceUnlockModal, false},
	} {
		if got := ValidTariffEvent(c.kind, c.source); got != c.want {
			t.Errorf("ValidTariffEvent(%q, %q) = %v, want %v", c.kind, c.source, got, c.want)
		}
	}
}

func TestThePriceChangeCampaignSourcesTakeViewClickAndDismiss(t *testing.T) {
	for _, source := range []string{TariffSourcePriceBanner, TariffSourcePricePage, TariffSourcePriceModal} {
		for _, kind := range []string{TariffEventView, TariffEventClick, TariffEventDismiss} {
			if !ValidTariffEvent(kind, source) {
				t.Errorf("ValidTariffEvent(%q, %q) = false, want true", kind, source)
			}
		}
		if ValidTariffEvent(TariffEventOpen, source) {
			t.Errorf("%q takes a checkout step", source)
		}
	}
}
