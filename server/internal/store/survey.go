package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// SurveyMinLessons is how many finished lessons a learner needs before the
// feedback survey is offered.
const SurveyMinLessons = 3

// SurveyAnswers is one filled-in survey. Every field but PainText is required.
type SurveyAnswers struct {
	Rating   int      `json:"rating"`    // 1..5
	Likes    []string `json:"likes"`     // option codes
	Pains    []string `json:"pains"`     // option codes
	PainText string   `json:"pain_text"` // optional free text
	Purpose  string   `json:"purpose"`   // option code
	Wish     string   `json:"wish"`      // option code: what the learner misses most
}

// ErrSurveyClosed means the learner has already answered the survey.
var ErrSurveyClosed = errors.New("survey already answered")

// SurveyOffer decides whether the survey should be shown to userID now and, if
// so, records that it was offered so it is never offered again. It is true at
// most once per learner: the first call after they have finished
// SurveyMinLessons lessons.
func (s *Store) SurveyOffer(userID string, now time.Time) (bool, error) {
	var done int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM lesson_progress WHERE user_id = ? AND status = 'done'`,
		userID).Scan(&done); err != nil {
		return false, fmt.Errorf("survey offer: count lessons: %w", err)
	}
	if done < SurveyMinLessons {
		return false, nil
	}
	res, err := s.db.Exec(`INSERT INTO survey_responses (user_id, status, lessons_done, shown_at)
		VALUES (?, 'shown', ?, ?) ON CONFLICT (user_id) DO NOTHING`,
		userID, done, now.UTC().Format(time.RFC3339))
	if err != nil {
		return false, fmt.Errorf("survey offer: insert: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("survey offer: rows: %w", err)
	}
	return n == 1, nil
}

// SaveSurvey stores the answers and pays the reward in one transaction. It
// returns ErrSurveyClosed if the learner already answered, so the reward can
// never be paid twice. A learner who never got the offer (no row yet) can still
// answer — the row is created here.
func (s *Store) SaveSurvey(userID string, a SurveyAnswers, reward int64, now time.Time) error {
	likes, _ := json.Marshal(nonNil(a.Likes))
	pains, _ := json.Marshal(nonNil(a.Pains))
	iso := now.UTC().Format(time.RFC3339)

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("save survey: begin: %w", err)
	}
	defer tx.Rollback()

	var status string
	err = tx.QueryRow(`SELECT status FROM survey_responses WHERE user_id = ?`, userID).Scan(&status)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := tx.Exec(`INSERT INTO survey_responses (user_id, status, shown_at) VALUES (?, 'shown', ?)`,
			userID, iso); err != nil {
			return fmt.Errorf("save survey: insert: %w", err)
		}
	case err != nil:
		return fmt.Errorf("save survey: read: %w", err)
	case status == "answered":
		return ErrSurveyClosed
	}

	if _, err := tx.Exec(`UPDATE survey_responses SET status = 'answered', rating = ?, likes = ?, pains = ?,
		pain_text = ?, purpose = ?, wish = ?, reward = ?, answered_at = ? WHERE user_id = ?`,
		a.Rating, string(likes), string(pains), a.PainText, a.Purpose, a.Wish, reward, iso, userID); err != nil {
		return fmt.Errorf("save survey: update: %w", err)
	}
	if reward > 0 {
		if _, err := addLedgerEntryTx(tx, LedgerEntry{
			UserID: userID, Amount: reward, Kind: "survey_reward", Ref: "survey",
			IdempotencyKey: "survey:" + userID, Comment: "feedback survey",
		}, now); err != nil && !errors.Is(err, ErrDuplicateEntry) {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("save survey: commit: %w", err)
	}
	return nil
}

// DismissSurvey marks an offered survey as closed without answers. It never
// downgrades an answered one.
func (s *Store) DismissSurvey(userID string, now time.Time) error {
	iso := now.UTC().Format(time.RFC3339)
	if _, err := s.db.Exec(`INSERT INTO survey_responses (user_id, status, shown_at) VALUES (?, 'dismissed', ?)
		ON CONFLICT (user_id) DO UPDATE SET status = 'dismissed' WHERE survey_responses.status = 'shown'`,
		userID, iso); err != nil {
		return fmt.Errorf("dismiss survey: %w", err)
	}
	return nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
