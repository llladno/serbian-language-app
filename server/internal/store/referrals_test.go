package store

import (
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
)

func addVerifiedEmail(t *testing.T, s *Store, userID, email string) {
	t.Helper()
	if err := s.CreateIdentity(Identity{
		ID: auth.NewIdentityID(), UserID: userID, Provider: "password", ProviderUID: email,
		Email: email, PasswordHash: "x", EmailVerifiedAt: "2026-10-09T10:00:00Z",
	}); err != nil {
		t.Fatalf("identity: %v", err)
	}
}

func TestReferralCodeIsStableAndWellFormed(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Пригласитель")
	a, err := s.ReferralCode(id)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := s.ReferralCode(id)
	if a != b {
		t.Fatalf("code changed between calls: %q then %q", a, b)
	}
	if norm, ok := NormalizeReferralCode(a); !ok || norm != a {
		t.Fatalf("code %q is not in the canonical form", a)
	}
	other, _ := s.CreateUser("Другой")
	c, _ := s.ReferralCode(other)
	if c == a {
		t.Fatal("two accounts got the same code")
	}
}

func TestNormalizeReferralCodeRejectsJunk(t *testing.T) {
	for _, bad := range []string{"", "short", "waytoolongcode", "abcdefg!", "ABCDEFG0", "abcd efg"} {
		if _, ok := NormalizeReferralCode(bad); ok {
			t.Errorf("%q was accepted", bad)
		}
	}
	if got, ok := NormalizeReferralCode("  ABCDEFGH "); !ok || got != "abcdefgh" {
		t.Errorf("a padded upper-case code gave %q, %v", got, ok)
	}
}

func TestApplyReferralRules(t *testing.T) {
	s := newStore(t)
	now := time.Now().UTC()
	inviter, _ := s.CreateUser("Пригласитель")
	code, _ := s.ReferralCode(inviter)

	t.Run("a new account is attached once", func(t *testing.T) {
		friend, _ := s.CreateUser("Друг")
		ok, err := s.ApplyReferral(friend, code, now)
		if err != nil || !ok {
			t.Fatalf("apply = %v, %v", ok, err)
		}
		// first touch wins: a second inviter does not replace the first
		second, _ := s.CreateUser("Второй")
		code2, _ := s.ReferralCode(second)
		if ok, _ := s.ApplyReferral(friend, code2, now); ok {
			t.Fatal("a second inviter replaced the first")
		}
	})
	t.Run("own code and unknown code do nothing", func(t *testing.T) {
		if ok, _ := s.ApplyReferral(inviter, code, now); ok {
			t.Fatal("applied to its own owner")
		}
		friend, _ := s.CreateUser("Ещё друг")
		if ok, err := s.ApplyReferral(friend, "zzzzzzzz", now); ok || err != nil {
			t.Fatalf("unknown code = %v, %v", ok, err)
		}
		if ok, err := s.ApplyReferral(friend, "not a code!", now); ok || err != nil {
			t.Fatalf("junk code = %v, %v", ok, err)
		}
	})
	t.Run("two accounts cannot credit each other", func(t *testing.T) {
		a, _ := s.CreateUser("A")
		b, _ := s.CreateUser("B")
		codeA, _ := s.ReferralCode(a)
		codeB, _ := s.ReferralCode(b)
		if ok, _ := s.ApplyReferral(b, codeA, now); !ok {
			t.Fatal("b should join through a")
		}
		if ok, _ := s.ApplyReferral(a, codeB, now); ok {
			t.Fatal("a joined through the account it invited")
		}
	})
	t.Run("an old account is not attached", func(t *testing.T) {
		old, _ := s.CreateUser("Старожил")
		if ok, _ := s.ApplyReferral(old, code, now.Add(ReferralWindow+time.Hour)); ok {
			t.Fatal("an account past the window was attached")
		}
	})
}

func TestFriendsInvitedCountsOnlyConfirmedAccounts(t *testing.T) {
	s := newStore(t)
	now := time.Now().UTC()
	inviter, _ := s.CreateUser("Пригласитель")
	code, _ := s.ReferralCode(inviter)

	// An email nobody confirmed: does not count.
	unverified, _ := s.CreateUser("Неподтверждённый")
	if err := s.CreateIdentity(Identity{ID: auth.NewIdentityID(), UserID: unverified, Provider: "password",
		ProviderUID: "u@example.test", Email: "u@example.test", PasswordHash: "x"}); err != nil {
		t.Fatal(err)
	}
	s.ApplyReferral(unverified, code, now)
	if n, _ := s.FriendsInvited(inviter); n != 0 {
		t.Fatalf("an unconfirmed friend counted: %d", n)
	}

	verified, _ := s.CreateUser("Подтверждённый")
	addVerifiedEmail(t, s, verified, "v@example.test")
	s.ApplyReferral(verified, code, now)
	if n, _ := s.FriendsInvited(inviter); n != 1 {
		t.Fatalf("friends = %d, want 1", n)
	}

	viaTelegram, _ := s.CreateUser("Телеграм")
	if err := s.CreateIdentity(Identity{ID: auth.NewIdentityID(), UserID: viaTelegram, Provider: "telegram", ProviderUID: "42"}); err != nil {
		t.Fatal(err)
	}
	s.ApplyReferral(viaTelegram, code, now)
	if n, _ := s.FriendsInvited(inviter); n != 2 {
		t.Fatalf("friends = %d, want 2 (email + telegram)", n)
	}

	c, err := s.User(inviter).QuestCounters(now)
	if err != nil || c.FriendsInvited != 2 {
		t.Fatalf("counters: %d, %v", c.FriendsInvited, err)
	}
}
