package store

import (
	"fmt"
	"time"
)

// Session is one logged-in browser. token_hash is hex(sha256(rawToken))
// computed by the caller; the raw token never reaches the store.
type Session struct {
	TokenHash, UserID, CreatedAt, LastSeenAt, ExpiresAt, UserAgent string
}

const (
	// sessionSlide is how far TouchSession pushes expires_at from "now".
	sessionSlide = 30 * 24 * time.Hour
	// sessionMaxAge caps a session's lifetime measured from created_at.
	sessionMaxAge = 90 * 24 * time.Hour
)

const sessionCols = `token_hash, user_id, created_at, last_seen_at, expires_at, COALESCE(user_agent,'')`

func scanSession(row interface{ Scan(...any) error }) (Session, error) {
	var ss Session
	err := row.Scan(&ss.TokenHash, &ss.UserID, &ss.CreatedAt, &ss.LastSeenAt, &ss.ExpiresAt, &ss.UserAgent)
	return ss, err
}

// CreateSession stores a session. tokenHash is hex(sha256(rawToken)) from the
// caller; last_seen_at starts equal to now.
func (s *Store) CreateSession(tokenHash, userID, userAgent string, now, expires time.Time) error {
	iso := now.UTC().Format(time.RFC3339)
	if _, err := s.db.Exec(`INSERT INTO sessions
		(token_hash, user_id, created_at, last_seen_at, expires_at, user_agent)
		VALUES (?, ?, ?, ?, ?, ?)`,
		tokenHash, userID, iso, iso, expires.UTC().Format(time.RFC3339), nullIf(userAgent)); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// SessionByHash returns a live session. sql.ErrNoRows if the hash is unknown
// or the session's expires_at is at or before now.
func (s *Store) SessionByHash(tokenHash string, now time.Time) (Session, error) {
	return scanSession(s.db.QueryRow(`SELECT `+sessionCols+`
		FROM sessions WHERE token_hash = ? AND expires_at > ?`,
		tokenHash, now.UTC().Format(time.RFC3339)))
}

// TouchSession bumps last_seen_at to now and slides expires_at to now +
// sessionSlide, clamped so it never passes created_at + sessionMaxAge.
func (s *Store) TouchSession(tokenHash string, now time.Time) error {
	var createdStr string
	if err := s.db.QueryRow(`SELECT created_at FROM sessions WHERE token_hash = ?`, tokenHash).
		Scan(&createdStr); err != nil {
		return err // sql.ErrNoRows propagates
	}
	created, _ := time.Parse(time.RFC3339, createdStr)
	slide := now.Add(sessionSlide)
	if hardCap := created.Add(sessionMaxAge); slide.After(hardCap) {
		slide = hardCap
	}
	if _, err := s.db.Exec(`UPDATE sessions SET last_seen_at = ?, expires_at = ? WHERE token_hash = ?`,
		now.UTC().Format(time.RFC3339), slide.UTC().Format(time.RFC3339), tokenHash); err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return nil
}

// DeleteSession removes one session (logout).
func (s *Store) DeleteSession(tokenHash string) error {
	if _, err := s.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteUserSessions removes every session of a user (log out everywhere,
// account deletion).
func (s *Store) DeleteUserSessions(userID string) error {
	if _, err := s.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("delete user sessions: %w", err)
	}
	return nil
}

// DeleteUserSessionsExcept removes every session of a user except the one
// with keepTokenHash (used after a password change to keep the current tab).
func (s *Store) DeleteUserSessionsExcept(userID, keepTokenHash string) error {
	if _, err := s.db.Exec(`DELETE FROM sessions WHERE user_id = ? AND token_hash <> ?`,
		userID, keepTokenHash); err != nil {
		return fmt.Errorf("delete user sessions except: %w", err)
	}
	return nil
}

// ListUserSessions returns a user's sessions newest-first for the profile
// "devices" list.
func (s *Store) ListUserSessions(userID string) ([]Session, error) {
	rows, err := s.db.Query(
		`SELECT `+sessionCols+` FROM sessions WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list user sessions: %w", err)
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		ss, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("list user sessions: %w", err)
		}
		out = append(out, ss)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list user sessions: %w", err)
	}
	return out, nil
}

// DeleteExpiredSessions drops every session whose expires_at is at or before
// now and reports how many rows went. Called opportunistically as housekeeping.
func (s *Store) DeleteExpiredSessions(now time.Time) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, now.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
