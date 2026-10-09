package store

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Referral codes are public by design — they sit in a link — so they only need
// to be unguessable enough that nobody can enumerate accounts by trying codes:
// 8 characters of a 31-letter alphabet is about 40 bits. The alphabet drops
// the look-alikes (0/o, 1/l/i) because people retype these from screenshots.
const (
	referralAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"
	referralCodeLen  = 8
)

// ReferralWindow is how long after sign-up a link can still be credited. A code
// is only meant for people who have just arrived; without a limit an old
// account could be attached to anyone's link by opening it.
const ReferralWindow = 7 * 24 * time.Hour

func newReferralCode() (string, error) {
	b := make([]byte, referralCodeLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, referralCodeLen)
	for i, x := range b {
		out[i] = referralAlphabet[int(x)%len(referralAlphabet)]
	}
	return string(out), nil
}

// NormalizeReferralCode lower-cases and trims a code and reports whether it has
// the shape of one. Anything else is rejected before it reaches the database.
func NormalizeReferralCode(code string) (string, bool) {
	code = strings.ToLower(strings.TrimSpace(code))
	if len(code) != referralCodeLen {
		return "", false
	}
	for _, r := range code {
		if !strings.ContainsRune(referralAlphabet, r) {
			return "", false
		}
	}
	return code, true
}

// ReferralCode returns the account's invite code, creating it on first use. Two
// concurrent first calls end up with the same code: the UPDATE only fills an
// empty column, and the loser re-reads what the winner wrote.
func (s *Store) ReferralCode(userID string) (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		var code sql.NullString
		if err := s.db.QueryRow(`SELECT referral_code FROM users WHERE id = ?`, userID).Scan(&code); err != nil {
			return "", fmt.Errorf("referral code: %w", err)
		}
		if code.Valid && code.String != "" {
			return code.String, nil
		}
		fresh, err := newReferralCode()
		if err != nil {
			return "", fmt.Errorf("referral code: %w", err)
		}
		// A collision with another account's code trips the unique index and
		// is retried with a new random code; so is losing the race above.
		_, _ = s.db.Exec(`UPDATE users SET referral_code = ? WHERE id = ? AND referral_code IS NULL`, fresh, userID)
	}
	return "", errors.New("referral code: could not allocate one")
}

// UserIDByReferralCode resolves a code to its owner.
func (s *Store) UserIDByReferralCode(code string) (string, bool, error) {
	code, ok := NormalizeReferralCode(code)
	if !ok {
		return "", false, nil
	}
	var id string
	err := s.db.QueryRow(`SELECT id FROM users WHERE referral_code = ?`, code).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("referral owner: %w", err)
	}
	return id, true, nil
}

// ApplyReferral records that userID joined through the owner of code. It
// reports whether anything was recorded, and never errors on a bad code: a
// mistyped or stale link must not break signing up. It refuses
//   - an unknown code,
//   - your own code, or a code of someone you invited (no two accounts crediting
//     each other),
//   - an account that already has an inviter (first touch wins),
//   - an account created more than ReferralWindow ago.
func (s *Store) ApplyReferral(userID, code string, now time.Time) (bool, error) {
	inviter, ok, err := s.UserIDByReferralCode(code)
	if err != nil || !ok || inviter == userID {
		return false, err
	}
	var inviterRef sql.NullString
	if err := s.db.QueryRow(`SELECT referred_by FROM users WHERE id = ?`, inviter).Scan(&inviterRef); err != nil {
		return false, fmt.Errorf("referral inviter: %w", err)
	}
	if inviterRef.Valid && inviterRef.String == userID {
		return false, nil
	}
	cutoff := now.Add(-ReferralWindow).UTC().Format(time.RFC3339)
	res, err := s.db.Exec(`UPDATE users SET referred_by = ?
		WHERE id = ? AND referred_by IS NULL AND created_at >= ?`, inviter, userID, cutoff)
	if err != nil {
		return false, fmt.Errorf("apply referral: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("apply referral: %w", err)
	}
	return n > 0, nil
}

// FriendsInvited counts the accounts that joined through userID's link and are
// real: a verified email, or a Telegram account (Telegram has already checked
// that person). An account nobody confirmed does not count.
func (s *Store) FriendsInvited(userID string) (int, error) { return friendsInvited(s.db, userID) }

func friendsInvited(db *database, userID string) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM users f
		WHERE f.referred_by = ?
		  AND EXISTS (SELECT 1 FROM identities i WHERE i.user_id = f.id
		              AND (i.provider = 'telegram' OR COALESCE(i.email_verified_at, '') <> ''))`,
		userID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("friends invited: %w", err)
	}
	return n, nil
}
