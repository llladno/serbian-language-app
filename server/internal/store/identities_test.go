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

func TestIdentityForUser(t *testing.T) {
	s := newStore(t)
	uid := mkUser(t, s, "Гриша")

	pw := Identity{
		ID: auth.NewIdentityID(), UserID: uid, Provider: "password",
		ProviderUID: "grisha@x.io", Email: "grisha@x.io", PasswordHash: "argon2id$abc",
		CreatedAt: day0.Format(time.RFC3339),
	}
	tg := Identity{
		ID: auth.NewIdentityID(), UserID: uid, Provider: "telegram",
		ProviderUID: "123456", TgUsername: "llladnooo",
		CreatedAt: day0.Add(time.Second).Format(time.RFC3339),
	}
	for _, in := range []Identity{pw, tg} {
		if err := s.CreateIdentity(in); err != nil {
			t.Fatalf("CreateIdentity: %v", err)
		}
	}

	got, err := s.IdentityForUser(uid, "telegram")
	if err != nil {
		t.Fatalf("IdentityForUser telegram: %v", err)
	}
	if got != tg {
		t.Fatalf("telegram identity:\n got %+v\nwant %+v", got, tg)
	}

	got, err = s.IdentityForUser(uid, "password")
	if err != nil {
		t.Fatalf("IdentityForUser password: %v", err)
	}
	if got != pw {
		t.Fatalf("password identity:\n got %+v\nwant %+v", got, pw)
	}

	if _, err := s.IdentityForUser(uid, "github"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unknown provider err = %v, want sql.ErrNoRows", err)
	}
}

func TestIdentityByID(t *testing.T) {
	s := newStore(t)
	uid := mkUser(t, s, "Гриша")

	in := Identity{
		ID: auth.NewIdentityID(), UserID: uid, Provider: "password",
		ProviderUID: "grisha@x.io", Email: "grisha@x.io", PasswordHash: "argon2id$abc",
		CreatedAt: day0.Format(time.RFC3339),
	}
	if err := s.CreateIdentity(in); err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}

	got, err := s.IdentityByID(in.ID)
	if err != nil {
		t.Fatalf("IdentityByID: %v", err)
	}
	if got != in {
		t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v", got, in)
	}

	if _, err := s.IdentityByID("idn_missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unknown id err = %v, want sql.ErrNoRows", err)
	}
}

func TestDeleteUserIdentity(t *testing.T) {
	s := newStore(t)
	uid := mkUser(t, s, "Гриша")
	other := mkUser(t, s, "Оля")

	pw := Identity{
		ID: auth.NewIdentityID(), UserID: uid, Provider: "password",
		ProviderUID: "grisha@x.io", Email: "grisha@x.io",
		CreatedAt: day0.Format(time.RFC3339),
	}
	tg := Identity{
		ID: auth.NewIdentityID(), UserID: uid, Provider: "telegram",
		ProviderUID: "123456", TgUsername: "grisha",
		CreatedAt: day0.Format(time.RFC3339),
	}
	otherTg := Identity{
		ID: auth.NewIdentityID(), UserID: other, Provider: "telegram",
		ProviderUID: "999", TgUsername: "olya",
		CreatedAt: day0.Format(time.RFC3339),
	}
	for _, in := range []Identity{pw, tg, otherTg} {
		if err := s.CreateIdentity(in); err != nil {
			t.Fatalf("CreateIdentity: %v", err)
		}
	}
	// a token hanging off the telegram identity we're about to delete
	if err := s.CreateEmailToken("tok-del", tg.ID, "verify", day0, day0.Add(24*time.Hour)); err != nil {
		t.Fatalf("CreateEmailToken: %v", err)
	}

	if err := s.DeleteUserIdentity(uid, "telegram"); err != nil {
		t.Fatalf("DeleteUserIdentity: %v", err)
	}

	// the matching (user_id, provider) row is gone
	if _, err := s.IdentityForUser(uid, "telegram"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("telegram identity still present: %v", err)
	}
	// the same user's other-provider row is untouched
	if _, err := s.IdentityForUser(uid, "password"); err != nil {
		t.Errorf("same user's password identity affected: %v", err)
	}
	// a different user's telegram row is untouched
	if _, err := s.IdentityForUser(other, "telegram"); err != nil {
		t.Errorf("other user's telegram identity affected: %v", err)
	}
	// email_tokens on the deleted identity cascaded away
	if n := countTokens(t, s, tg.ID); n != 0 {
		t.Errorf("email_tokens after identity delete = %d, want 0 (FK cascade)", n)
	}

	// deleting a provider the user no longer has is a no-op, not an error
	if err := s.DeleteUserIdentity(uid, "telegram"); err != nil {
		t.Errorf("second DeleteUserIdentity: %v", err)
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
