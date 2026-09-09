package store

import (
	"database/sql"
	"fmt"
	"time"
)

// EmailToken is a single-use, expiring link token. token_hash is
// hex(sha256(rawToken)) from the caller; the raw token is only ever emailed.
// kind is "verify" (confirm an address) or "reset" (set a new password).
type EmailToken struct {
	TokenHash, IdentityID, Kind, CreatedAt, ExpiresAt, UsedAt string
}

// CreateEmailToken stores a token. tokenHash is hex(sha256(raw)) from the
// caller; kind is "verify" or "reset".
func (s *Store) CreateEmailToken(tokenHash, identityID, kind string, now, expires time.Time) error {
	if _, err := s.db.Exec(`INSERT INTO email_tokens
		(token_hash, identity_id, kind, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)`,
		tokenHash, identityID, kind,
		now.UTC().Format(time.RFC3339), expires.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("create email token: %w", err)
	}
	return nil
}

// UseEmailToken atomically marks a live, unused token of the given kind used
// and returns its identity_id. The guarded UPDATE is the atomic point: only
// one caller can flip used_at from NULL to a value, so two concurrent callers
// can never both succeed (a SELECT-then-UPDATE pair would let both through
// under READ COMMITTED). sql.ErrNoRows if the token is unknown, already used,
// expired, or the wrong kind.
func (s *Store) UseEmailToken(tokenHash, kind string, now time.Time) (string, error) {
	nowISO := now.UTC().Format(time.RFC3339)
	res, err := s.db.Exec(`UPDATE email_tokens SET used_at = ?
		WHERE token_hash = ? AND kind = ? AND used_at IS NULL AND expires_at > ?`,
		nowISO, tokenHash, kind, nowISO)
	if err != nil {
		return "", fmt.Errorf("use email token: %w", err)
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return "", sql.ErrNoRows
	}
	var identityID string
	if err := s.db.QueryRow(`SELECT identity_id FROM email_tokens WHERE token_hash = ?`,
		tokenHash).Scan(&identityID); err != nil {
		return "", err
	}
	return identityID, nil
}

// DeleteIdentityTokens drops every token of a kind for an identity (e.g. all
// "reset" tokens once a reset succeeds).
func (s *Store) DeleteIdentityTokens(identityID, kind string) error {
	if _, err := s.db.Exec(`DELETE FROM email_tokens WHERE identity_id = ? AND kind = ?`,
		identityID, kind); err != nil {
		return fmt.Errorf("delete identity tokens: %w", err)
	}
	return nil
}
