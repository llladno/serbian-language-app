package store

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
)

func TestCreateAndFindIdentity(t *testing.T) {
	s := newStore(t)
	uid, err := s.CreateUser("Гриша")
	if err != nil {
		t.Fatal(err)
	}
	in := Identity{
		ID:              auth.NewIdentityID(),
		UserID:          uid,
		Provider:        "password",
		ProviderUID:     "grisha@x.io",
		Email:           "grisha@x.io",
		PasswordHash:    "argon2id$abc",
		EmailVerifiedAt: "",
		CreatedAt:       day0.Format(time.RFC3339),
	}
	if err := s.CreateIdentity(in); err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}

	got, err := s.IdentityByProviderUID("password", "grisha@x.io")
	if err != nil {
		t.Fatalf("IdentityByProviderUID: %v", err)
	}
	if got != in {
		t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v", got, in)
	}

	if _, err := s.IdentityByProviderUID("password", "nope"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing identity err = %v, want sql.ErrNoRows", err)
	}
}

func TestIdentitiesForUser(t *testing.T) {
	s := newStore(t)
	uid, err := s.CreateUser("Гриша")
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.CreateUser("Оля")
	if err != nil {
		t.Fatal(err)
	}

	must := func(in Identity) {
		t.Helper()
		if err := s.CreateIdentity(in); err != nil {
			t.Fatalf("CreateIdentity: %v", err)
		}
	}
	must(Identity{ID: auth.NewIdentityID(), UserID: uid, Provider: "password", ProviderUID: "grisha@x.io", Email: "grisha@x.io", CreatedAt: day0.Format(time.RFC3339)})
	must(Identity{ID: auth.NewIdentityID(), UserID: uid, Provider: "telegram", ProviderUID: "123456", TgUsername: "llladnooo", CreatedAt: day0.Add(time.Second).Format(time.RFC3339)})
	must(Identity{ID: auth.NewIdentityID(), UserID: other, Provider: "password", ProviderUID: "olya@x.io", Email: "olya@x.io", CreatedAt: day0.Format(time.RFC3339)})

	list, err := s.IdentitiesForUser(uid)
	if err != nil {
		t.Fatalf("IdentitiesForUser: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("identities for user = %d, want 2 (%+v)", len(list), list)
	}
}

func TestSetEmailVerifiedAndPasswordHash(t *testing.T) {
	s := newStore(t)
	uid, err := s.CreateUser("Гриша")
	if err != nil {
		t.Fatal(err)
	}
	id := auth.NewIdentityID()
	if err := s.CreateIdentity(Identity{
		ID: id, UserID: uid, Provider: "password", ProviderUID: "grisha@x.io",
		Email: "grisha@x.io", PasswordHash: "old-hash", CreatedAt: day0.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	if err := s.SetEmailVerified(id, day0.Add(time.Hour)); err != nil {
		t.Fatalf("SetEmailVerified: %v", err)
	}
	if err := s.SetPasswordHash(id, "new-hash"); err != nil {
		t.Fatalf("SetPasswordHash: %v", err)
	}

	got, err := s.IdentityByProviderUID("password", "grisha@x.io")
	if err != nil {
		t.Fatal(err)
	}
	if got.EmailVerifiedAt == "" {
		t.Errorf("EmailVerifiedAt still empty after SetEmailVerified")
	}
	if want := day0.Add(time.Hour).UTC().Format(time.RFC3339); got.EmailVerifiedAt != want {
		t.Errorf("EmailVerifiedAt = %q, want %q", got.EmailVerifiedAt, want)
	}
	if got.PasswordHash != "new-hash" {
		t.Errorf("PasswordHash = %q, want %q", got.PasswordHash, "new-hash")
	}
}

func TestAttachPendingTelegram(t *testing.T) {
	s := newStore(t)
	uid, err := s.CreateUser("Гриша")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateIdentity(Identity{
		ID: auth.NewIdentityID(), UserID: uid, Provider: "telegram",
		ProviderUID: "pending:llladnooo", TgUsername: "llladnooo",
		CreatedAt: day0.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	ok, err := s.AttachPendingTelegram("llladnooo", "123456")
	if err != nil {
		t.Fatalf("AttachPendingTelegram: %v", err)
	}
	if !ok {
		t.Fatal("AttachPendingTelegram = false, want true for a pending row")
	}

	got, err := s.IdentityByProviderUID("telegram", "123456")
	if err != nil {
		t.Fatalf("lookup after attach: %v", err)
	}
	if got.ProviderUID != "123456" || got.TgUsername != "llladnooo" {
		t.Errorf("after attach: provider_uid=%q tg_username=%q", got.ProviderUID, got.TgUsername)
	}

	ok, err = s.AttachPendingTelegram("llladnooo", "123456")
	if err != nil {
		t.Fatalf("second AttachPendingTelegram: %v", err)
	}
	if ok {
		t.Error("second AttachPendingTelegram = true, want false (nothing pending)")
	}
}
