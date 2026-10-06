package store

import (
	"errors"
	"testing"
	"time"

	"github.com/grisha/serbian-app/server/internal/economy"
)

func seedQuest(t *testing.T, s *Store, kind string, target int, reward int64) economy.Quest {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.Exec(`INSERT INTO quests (kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
		VALUES (?, ?, '', ?, '', ?, 1, 0, ?, ?)`,
		kind, target, "Задание", reward, now, now); err != nil {
		t.Fatalf("seed quest: %v", err)
	}
	qs, err := s.ListQuests(true)
	if err != nil {
		t.Fatalf("list quests: %v", err)
	}
	return qs[len(qs)-1]
}

func seedCard(t *testing.T, s *Store, userID, cardID, kind, state string) {
	t.Helper()
	if _, err := s.db.Exec(`INSERT INTO srs_cards (user_id, card_id, kind, ref_id, state, updated_at)
		VALUES (?, ?, ?, ?, ?, '2026-09-29T00:00:00Z')`,
		userID, cardID, kind, cardID, state); err != nil {
		t.Fatalf("seed card %s: %v", cardID, err)
	}
}

func TestQuestCountersReadRealProgress(t *testing.T) {
	s := newStore(t)
	loc := belgrade(t)
	id, _ := s.CreateUser("Счёт")
	u := s.User(id)
	at := time.Date(2026, 9, 29, 10, 0, 0, 0, loc)

	answer(t, u, "01.1", true, at)
	answer(t, u, "01.2", true, at)
	if err := u.SetLessonStatus("01", "done", at); err != nil {
		t.Fatalf("lesson status: %v", err)
	}

	c, err := u.QuestCounters(at)
	if err != nil {
		t.Fatalf("counters: %v", err)
	}
	if c.AnswerBest != 2 {
		t.Fatalf("AnswerBest = %d, want 2", c.AnswerBest)
	}
	if !c.CompletedLessons["01"] {
		t.Fatal("lesson 01 not reported as completed")
	}
	if c.LessonsCompleted != 1 {
		t.Fatalf("LessonsCompleted = %d, want 1", c.LessonsCompleted)
	}
}

// "Learned a word" must mean what the rest of the app already means by it.
// CardStats — the number the learner's own dashboard shows as "знаю" — counts
// state = 'review', so this counts the same state, narrowed to vocab. Counting
// state <> 'new' instead would let the quest's word count exceed the dashboard's
// card count while measuring a strict subset of the cards, which is the kind of
// arithmetic a learner reasonably reports as a bug.
func TestQuestCountersCountOnlyGraduatedVocab(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Словарь")
	u := s.User(id)

	seedCard(t, s, id, "vocab:a", "vocab", "review")
	seedCard(t, s, id, "vocab:b", "vocab", "review")
	seedCard(t, s, id, "vocab:c", "vocab", "learning") // started, not learned
	seedCard(t, s, id, "vocab:d", "vocab", "new")      // never seen
	seedCard(t, s, id, "gram:a", "gram", "review")     // not a word
	seedCard(t, s, id, "ff:a", "ff", "review")         // not a word

	c, err := u.QuestCounters(time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("counters: %v", err)
	}
	if c.VocabLearned != 2 {
		t.Fatalf("VocabLearned = %d, want 2 (graduated vocab only)", c.VocabLearned)
	}

	// And it can never exceed what the dashboard calls known.
	_, known, err := u.CardStats()
	if err != nil {
		t.Fatalf("card stats: %v", err)
	}
	if c.VocabLearned > known {
		t.Fatalf("VocabLearned = %d exceeds dashboard known = %d", c.VocabLearned, known)
	}
}

func TestClaimQuestCreditsOnceAndOnlyOnce(t *testing.T) {
	s := newStore(t)
	id, _ := s.CreateUser("Клейм")
	u := s.User(id)
	q := seedQuest(t, s, economy.QuestLessonsCompleted, 1, 30)
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

	if err := u.ClaimQuest(q, now); err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if bal, _ := s.Balance(id); bal != 30 {
		t.Fatalf("balance = %d, want 30", bal)
	}

	if err := u.ClaimQuest(q, now); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("second claim err = %v, want ErrAlreadyClaimed", err)
	}
	if bal, _ := s.Balance(id); bal != 30 {
		t.Fatalf("balance = %d after a repeat claim, want 30", bal)
	}

	claimed, err := u.ClaimedQuestIDs()
	if err != nil {
		t.Fatalf("claimed ids: %v", err)
	}
	if !claimed[q.ID] {
		t.Fatal("quest not reported as claimed")
	}
}

// One user's claim must not block another's, and a repeat claim must leave the
// claim row alone rather than rolling it back.
func TestClaimQuestIsPerUser(t *testing.T) {
	s := newStore(t)
	a, _ := s.CreateUser("Первый")
	b, _ := s.CreateUser("Второй")
	q := seedQuest(t, s, economy.QuestLessonsCompleted, 1, 30)
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

	if err := s.User(a).ClaimQuest(q, now); err != nil {
		t.Fatalf("a claim: %v", err)
	}
	if err := s.User(b).ClaimQuest(q, now); err != nil {
		t.Fatalf("b claim: %v", err)
	}
	if err := s.User(a).ClaimQuest(q, now); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("a repeat claim err = %v, want ErrAlreadyClaimed", err)
	}

	for _, id := range []string{a, b} {
		if bal, _ := s.Balance(id); bal != 30 {
			t.Fatalf("balance of %s = %d, want 30", id, bal)
		}
		claimed, err := s.User(id).ClaimedQuestIDs()
		if err != nil {
			t.Fatalf("claimed ids: %v", err)
		}
		if !claimed[q.ID] {
			t.Fatalf("%s: claim row missing after a rejected repeat", id)
		}
	}
}

func TestListQuestsRespectsActiveFlagAndOrder(t *testing.T) {
	s := newStore(t)
	a := seedQuest(t, s, economy.QuestLessonsCompleted, 5, 5)
	b := seedQuest(t, s, economy.QuestVocabLearned, 30, 10)
	if _, err := s.db.Exec(`UPDATE quests SET active = 0 WHERE id = ?`, b.ID); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	if _, err := s.db.Exec(`UPDATE quests SET sort_order = 5 WHERE id = ?`, a.ID); err != nil {
		t.Fatalf("reorder: %v", err)
	}

	active, err := s.ListQuests(true)
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(active) != 1 || active[0].ID != a.ID {
		t.Fatalf("active quests = %+v, want only %d", active, a.ID)
	}
	all, err := s.ListQuests(false)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("all quests = %d, want 2", len(all))
	}
}
