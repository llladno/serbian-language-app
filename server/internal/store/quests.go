package store

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/grisha/serbian-app/server/internal/economy"
)

// ErrAlreadyClaimed means this user has already taken this quest's reward.
var ErrAlreadyClaimed = errors.New("quest already claimed")

// ListQuests returns quests ordered the way the profile shows them.
func (s *Store) ListQuests(activeOnly bool) ([]economy.Quest, error) {
	q := `SELECT id, kind, target, param, title, description, reward, active, sort_order
		FROM quests`
	if activeOnly {
		q += ` WHERE active = 1`
	}
	q += ` ORDER BY sort_order ASC, id ASC`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("list quests: %w", err)
	}
	defer rows.Close()
	var out []economy.Quest
	for rows.Next() {
		var qu economy.Quest
		var active int
		if err := rows.Scan(&qu.ID, &qu.Kind, &qu.Target, &qu.Param, &qu.Title,
			&qu.Description, &qu.Reward, &active, &qu.SortOrder); err != nil {
			return nil, fmt.Errorf("scan quest: %w", err)
		}
		qu.Active = active == 1
		out = append(out, qu)
	}
	return out, rows.Err()
}

// QuestCounters gathers every locally computable quest input in one go.
// TelegramSubscribed is left false — only the API layer can fill it, since it
// needs a Bot API call.
//
// Each counter deliberately reuses a definition the app already shows the
// learner, so a quest's progress bar and the dashboard can never disagree:
// a completed lesson is a lesson_progress row with status 'done', the same
// predicate the leaderboard counts, and a learned word is a vocab card in
// state 'review', the same state CardStats reports as "known".
func (u *UserStore) QuestCounters(now time.Time) (economy.Counters, error) {
	var c economy.Counters

	if err := u.db.QueryRow(`SELECT COUNT(*) FROM lesson_progress WHERE user_id = ? AND status = 'done'`,
		u.user).Scan(&c.LessonsCompleted); err != nil {
		return c, fmt.Errorf("counters: lessons: %w", err)
	}
	if err := u.db.QueryRow(`SELECT COUNT(*) FROM srs_cards WHERE user_id = ? AND kind = 'vocab' AND state = 'review'`,
		u.user).Scan(&c.VocabLearned); err != nil {
		return c, fmt.Errorf("counters: vocab: %w", err)
	}
	if err := u.db.QueryRow(`SELECT COUNT(*) FROM reviews WHERE user_id = ?`,
		u.user).Scan(&c.ReviewsDone); err != nil {
		return c, fmt.Errorf("counters: reviews: %w", err)
	}

	_, best, err := u.AnswerStreak()
	if err != nil {
		return c, err
	}
	c.AnswerBest = best

	streak, err := u.StreakDays(now)
	if err != nil {
		return c, err
	}
	c.StreakDays = streak

	rows, err := u.db.Query(`SELECT lesson FROM lesson_progress WHERE user_id = ? AND status = 'done'`, u.user)
	if err != nil {
		return c, fmt.Errorf("counters: completed lessons: %w", err)
	}
	defer rows.Close()
	c.CompletedLessons = map[string]bool{}
	for rows.Next() {
		var lesson string
		if err := rows.Scan(&lesson); err != nil {
			return c, fmt.Errorf("scan completed lesson: %w", err)
		}
		c.CompletedLessons[lesson] = true
	}
	return c, rows.Err()
}

// ClaimedQuestIDs is the set of quests this user has already taken.
func (u *UserStore) ClaimedQuestIDs() (map[int64]bool, error) {
	rows, err := u.db.Query(`SELECT quest_id FROM quest_claims WHERE user_id = ?`, u.user)
	if err != nil {
		return nil, fmt.Errorf("claimed quests: %w", err)
	}
	defer rows.Close()
	out := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan claimed quest: %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}

// ClaimQuest credits the reward and records the claim in one transaction.
//
// The caller MUST have verified two things first, because this function checks
// neither: that the quest is complete (economy.QuestDone, which needs course
// content the store layer deliberately does not know about) and that the quest
// is still active (q.Active) — an id arriving in a request is not proof that
// the quest is one the profile currently offers.
//
// The idempotency key, not the quest_claims primary key, is what makes a
// repeat claim safe: the ledger insert reports ErrDuplicateEntry and nothing is
// written, so the reward cannot be taken twice even by two concurrent
// requests. The claim row is written in the same transaction and is therefore
// never out of step with the ledger; a conflict on it would mean that
// invariant is already broken, so it is left to fail loudly rather than being
// swallowed with ON CONFLICT DO NOTHING.
func (u *UserStore) ClaimQuest(q economy.Quest, now time.Time) error {
	tx, err := u.db.Begin()
	if err != nil {
		return fmt.Errorf("claim quest: %w", err)
	}
	defer tx.Rollback()

	questID := strconv.FormatInt(q.ID, 10)
	if _, err := addLedgerEntryTx(tx, LedgerEntry{
		UserID:         u.user,
		Amount:         q.Reward,
		Kind:           "quest_reward",
		Ref:            questID,
		IdempotencyKey: "quest:" + u.user + ":" + questID,
	}, now); err != nil {
		if errors.Is(err, ErrDuplicateEntry) {
			return ErrAlreadyClaimed
		}
		return err
	}
	if _, err := tx.Exec(`INSERT INTO quest_claims (quest_id, user_id, claimed_at) VALUES (?, ?, ?)`,
		q.ID, u.user, now.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("claim quest: record claim: %w", err)
	}
	return tx.Commit()
}
