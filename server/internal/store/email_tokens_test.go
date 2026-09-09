package store

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/auth"
)

// mkIdentity creates a user + a password identity and returns the identity id.
func mkIdentity(t *testing.T, s *Store) string {
	t.Helper()
	uid := mkUser(t, s, "Гриша")
	id := auth.NewIdentityID()
	if err := s.CreateIdentity(Identity{
		ID: id, UserID: uid, Provider: "password", ProviderUID: "grisha@x.io",
		Email: "grisha@x.io", CreatedAt: day0.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	return id
}

func countTokens(t *testing.T, s *Store, identityID string) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM email_tokens WHERE identity_id = ?`, identityID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestUseEmailTokenHappy(t *testing.T) {
	s := newStore(t)
	idn := mkIdentity(t, s)

	if err := s.CreateEmailToken("tok-hash", idn, "verify", day0, day0.Add(24*time.Hour)); err != nil {
		t.Fatalf("CreateEmailToken: %v", err)
	}

	got, err := s.UseEmailToken("tok-hash", "verify", day0.Add(time.Hour))
	if err != nil {
		t.Fatalf("UseEmailToken: %v", err)
	}
	if got != idn {
		t.Errorf("identity id = %q, want %q", got, idn)
	}

	if _, err := s.UseEmailToken("tok-hash", "verify", day0.Add(2*time.Hour)); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("reuse err = %v, want sql.ErrNoRows", err)
	}
}

func TestUseEmailTokenExpired(t *testing.T) {
	s := newStore(t)
	idn := mkIdentity(t, s)

	if err := s.CreateEmailToken("tok-exp", idn, "reset", day0, day0.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UseEmailToken("tok-exp", "reset", day0); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expired token err = %v, want sql.ErrNoRows", err)
	}
}

func TestUseEmailTokenWrongKind(t *testing.T) {
	s := newStore(t)
	idn := mkIdentity(t, s)

	if err := s.CreateEmailToken("tok-kind", idn, "verify", day0, day0.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UseEmailToken("tok-kind", "reset", day0.Add(time.Hour)); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("wrong-kind err = %v, want sql.ErrNoRows", err)
	}
	// still usable with the right kind
	if _, err := s.UseEmailToken("tok-kind", "verify", day0.Add(time.Hour)); err != nil {
		t.Errorf("right-kind use failed: %v", err)
	}
}

func TestDeleteIdentityTokens(t *testing.T) {
	s := newStore(t)
	idn := mkIdentity(t, s)

	must := func(hash, kind string) {
		t.Helper()
		if err := s.CreateEmailToken(hash, idn, kind, day0, day0.Add(24*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	must("r1", "reset")
	must("r2", "reset")
	must("v1", "verify")

	if err := s.DeleteIdentityTokens(idn, "reset"); err != nil {
		t.Fatalf("DeleteIdentityTokens: %v", err)
	}
	if n := countTokens(t, s, idn); n != 1 {
		t.Errorf("tokens left = %d, want 1 (the verify)", n)
	}
	if _, err := s.UseEmailToken("v1", "verify", day0.Add(time.Hour)); err != nil {
		t.Errorf("verify token should survive: %v", err)
	}
}
