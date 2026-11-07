package store

import (
	"errors"
	"fmt"
	"time"
)

// Onboarding states, as stored in onboarding.status.
const (
	OnboardingPostponed = "postponed"
	OnboardingDone      = "done"
)

// OnboardingDue reports whether the first-lesson tour should start by itself for userID: no answer on record and at
// least one finished lesson. It records nothing, so a closed tab brings the tour back from the first line.
func (s *Store) OnboardingDue(userID string) (bool, error) {
	var answered int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM onboarding WHERE user_id = ?`, userID).Scan(&answered); err != nil {
		return false, fmt.Errorf("onboarding due: state: %w", err)
	}
	if answered > 0 {
		return false, nil
	}
	var finished int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM lesson_progress WHERE user_id = ? AND status = 'done'`,
		userID).Scan(&finished); err != nil {
		return false, fmt.Errorf("onboarding due: lessons: %w", err)
	}
	return finished > 0, nil
}

// SetOnboarding records the learner's answer to the tour. 'postponed' never downgrades 'done'.
func (s *Store) SetOnboarding(userID, status string, now time.Time) error {
	if status != OnboardingPostponed && status != OnboardingDone {
		return errors.New("set onboarding: unknown status " + status)
	}
	iso := now.UTC().Format(time.RFC3339)
	if _, err := s.db.Exec(`INSERT INTO onboarding (user_id, status, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET status = ?, updated_at = ? WHERE onboarding.status <> 'done'`,
		userID, status, iso, status, iso); err != nil {
		return fmt.Errorf("set onboarding: %w", err)
	}
	return nil
}
