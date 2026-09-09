package store

import (
	"fmt"
	"time"
)

// Identity is one way a user authenticates: a "password" row (email +
// password_hash, verified via email_verified_at) or a "telegram" row
// (provider_uid is the numeric tg id, or "pending:<username>" until the
// first login). provider_uid is unique per provider.
type Identity struct {
	ID, UserID, Provider, ProviderUID string
	Email, PasswordHash               string // password provider
	EmailVerifiedAt                   string // RFC3339 or ""
	TgUsername                        string // telegram provider
	CreatedAt                         string
}

// identityCols is the SELECT list for scanIdentity; nullable columns are
// coalesced so every field scans into a plain string.
const identityCols = `id, user_id, provider, provider_uid,
	COALESCE(email,''), COALESCE(password_hash,''), COALESCE(email_verified_at,''),
	COALESCE(tg_username,''), created_at`

func scanIdentity(row interface{ Scan(...any) error }) (Identity, error) {
	var it Identity
	err := row.Scan(&it.ID, &it.UserID, &it.Provider, &it.ProviderUID,
		&it.Email, &it.PasswordHash, &it.EmailVerifiedAt, &it.TgUsername, &it.CreatedAt)
	return it, err
}

// CreateIdentity inserts an identities row. The caller supplies a generated
// ID via auth.NewIdentityID(); CreatedAt defaults to now when left blank.
func (s *Store) CreateIdentity(in Identity) error {
	if in.CreatedAt == "" {
		in.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if _, err := s.db.Exec(`INSERT INTO identities
		(id, user_id, provider, provider_uid, email, password_hash, email_verified_at, tg_username, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.ID, in.UserID, in.Provider, in.ProviderUID,
		nullIf(in.Email), nullIf(in.PasswordHash), nullIf(in.EmailVerifiedAt), nullIf(in.TgUsername),
		in.CreatedAt); err != nil {
		return fmt.Errorf("create identity: %w", err)
	}
	return nil
}

// IdentityByProviderUID looks up one identity by (provider, provider_uid).
// sql.ErrNoRows if absent.
func (s *Store) IdentityByProviderUID(provider, uid string) (Identity, error) {
	return scanIdentity(s.db.QueryRow(
		`SELECT `+identityCols+` FROM identities WHERE provider = ? AND provider_uid = ?`,
		provider, uid))
}

// IdentitiesForUser returns every identity of a user, oldest first.
func (s *Store) IdentitiesForUser(userID string) ([]Identity, error) {
	rows, err := s.db.Query(
		`SELECT `+identityCols+` FROM identities WHERE user_id = ? ORDER BY created_at ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("identities for user: %w", err)
	}
	defer rows.Close()
	var out []Identity
	for rows.Next() {
		it, err := scanIdentity(rows)
		if err != nil {
			return nil, fmt.Errorf("identities for user: %w", err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("identities for user: %w", err)
	}
	return out, nil
}

// SetEmailVerified stamps email_verified_at = now on a password identity.
func (s *Store) SetEmailVerified(identityID string, now time.Time) error {
	if _, err := s.db.Exec(`UPDATE identities SET email_verified_at = ? WHERE id = ?`,
		now.UTC().Format(time.RFC3339), identityID); err != nil {
		return fmt.Errorf("set email verified: %w", err)
	}
	return nil
}

// SetPasswordHash updates the stored hash (used by password reset and by
// "set a password" on a Telegram-only account).
func (s *Store) SetPasswordHash(identityID, hash string) error {
	if _, err := s.db.Exec(`UPDATE identities SET password_hash = ? WHERE id = ?`,
		hash, identityID); err != nil {
		return fmt.Errorf("set password hash: %w", err)
	}
	return nil
}

// AttachPendingTelegram rewrites a "pending:<username>" telegram identity to
// a real numeric tg id on first Telegram login. Returns false if no pending
// row matched.
func (s *Store) AttachPendingTelegram(username, tgID string) (bool, error) {
	res, err := s.db.Exec(
		`UPDATE identities SET provider_uid = ?, tg_username = ?
		 WHERE provider = 'telegram' AND provider_uid = ?`,
		tgID, username, "pending:"+username)
	if err != nil {
		return false, fmt.Errorf("attach pending telegram: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
