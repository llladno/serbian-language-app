package store

import (
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
// and returns its identity_id. sql.ErrNoRows if the token is unknown, already
// used, expired, or the wrong kind.
func (s *Store) UseEmailToken(tokenHash, kind string, now time.Time) (string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return "", fmt.Errorf("use email token: begin: %w", err)
	}
	defer tx.Rollback()

	nowISO := now.UTC().Format(time.RFC3339)
	var identityID string
	if err := tx.QueryRow(`SELECT identity_id FROM email_tokens
		WHERE token_hash = ? AND kind = ? AND used_at IS NULL AND expires_at > ?`,
		tokenHash, kind, nowISO).Scan(&identityID); err != nil {
		return "", err // sql.ErrNoRows propagates
	}
	if _, err := tx.Exec(`UPDATE email_tokens SET used_at = ? WHERE token_hash = ?`,
		nowISO, tokenHash); err != nil {
		return "", fmt.Errorf("use email token: mark used: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("use email token: commit: %w", err)
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
