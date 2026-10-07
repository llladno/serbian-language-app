package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/grisha/serbian-app/server/internal/economy"
)

// Markers for what has already been written. Each is checked and set inside
// its own transaction, so curating the catalogue in the admin panel —
// including deleting rows — is not undone by a restart. The per-lesson rewards
// have their own marker because they arrived later: a database seeded before
// them has the first marker set, and sharing it would leave that database
// without lesson rewards for ever.
const (
	seededKey        = "economy_seeded"
	lessonsSeededKey = "economy_lessons_seeded"
)

// SeedEconomyDefaults writes the starting quest list and product catalogue
// from docs/superpowers/specs/2026-09-28-currency-design.md, once. Called from
// main.go at startup; tests call it explicitly, which is why it is not a
// migration: seeding through one would put 19 quests and 3 products into every
// store test, and the behaviour would differ between the backends, since
// Postgres tests TRUNCATE after migrating while SQLite tests do not.
//
// Quest rewards total 638, which covers Level 4 (500) and deliberately does
// not cover Level 5 (1000): Level 5 is what the daily streak drip is for.
// Every number here is editable from the admin panel afterwards.
//
// The phase ids below ("1".."5") are course.yaml's. A renamed phase silently
// breaks both halves — a phase_completed quest nobody can finish, and a phase
// that stops locking — so TestSeedMatchesTheRealCourse cross-checks them
// against the real content tree.
func (s *Store) SeedEconomyDefaults(phases []economy.PhaseLessons, now time.Time) error {
	if err := s.seedOnce(seededKey, now, seedCatalogue); err != nil {
		return err
	}
	return s.seedOnce(lessonsSeededKey, now, func(tx *dbtx, ts string) error {
		return seedLessonRewards(tx, ts, phases)
	})
}

// seedOnce runs write inside a transaction, unless marker says it has already
// run. The marker is written last: a boot that dies mid-seed marks nothing
// done, and the next boot retries from a clean rollback.
func (s *Store) seedOnce(marker string, now time.Time, write func(tx *dbtx, ts string) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("seed economy: %w", err)
	}
	defer tx.Rollback()

	var found string
	err = tx.QueryRow(`SELECT value FROM economy_settings WHERE key = ?`, marker).Scan(&found)
	if err == nil {
		return nil // already seeded
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("seed economy: check marker %s: %w", marker, err)
	}

	ts := now.UTC().Format(time.RFC3339)
	if err := write(tx, ts); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO economy_settings (key, value, updated_at) VALUES (?, '1', ?)`,
		marker, ts); err != nil {
		return fmt.Errorf("seed economy: mark %s done: %w", marker, err)
	}
	return tx.Commit()
}

// seedLessonRewards writes one quest per lesson of the phases that pay for
// lessons. These are never offered on the quests screen: the server pays them
// the moment the lesson is finished (see payLessonReward), which is why they
// are quests at all — the claims table is what stops a lesson paying twice,
// and the admin panel can already edit a quest's reward one row at a time.
//
// A lesson added to the course later gets no row, since this runs once; the
// admin panel is where that row is added.
func seedLessonRewards(tx *dbtx, ts string, phases []economy.PhaseLessons) error {
	sortOrder := 100
	for _, ph := range phases {
		for i, lesson := range ph.Lessons {
			reward, ok := economy.LessonReward(ph.ID, i+1)
			if !ok {
				break // a phase that does not pay for lessons
			}
			sortOrder++
			if _, err := tx.Exec(`INSERT INTO quests
				(kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
				VALUES (?, 1, ?, ?, '', ?, 1, ?, ?, ?)`,
				economy.QuestLessonCompleted, lesson, "Урок "+lesson, reward, sortOrder, ts, ts); err != nil {
				return fmt.Errorf("seed lesson reward %s: %w", lesson, err)
			}
		}
	}
	return nil
}

func seedCatalogue(tx *dbtx, ts string) error {
	for _, q := range defaultQuests {
		if _, err := tx.Exec(`INSERT INTO quests
			(kind, target, param, title, description, reward, active, sort_order, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?, ?)`,
			q.kind, q.target, q.param, q.title, q.description, q.reward, q.sortOrder, ts, ts); err != nil {
			return fmt.Errorf("seed quest %q: %w", q.title, err)
		}
	}
	for _, p := range defaultProducts {
		if _, err := tx.Exec(`INSERT INTO products
			(kind, ref, title, description, price, discount_percent, discount_from, discount_to,
			 grant_qty, active, sort_order, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, 0, '', '', 1, 1, ?, ?, ?)
			ON CONFLICT (kind, ref) DO NOTHING`,
			p.kind, p.ref, p.title, p.description, p.price, p.sortOrder, ts, ts); err != nil {
			return fmt.Errorf("seed product %s:%s: %w", p.kind, p.ref, err)
		}
	}
	return nil
}

type defaultQuest struct {
	kind        string
	target      int
	param       string
	title       string
	description string
	reward      int64
	sortOrder   int
}

// Palettes and other cosmetics are left out on purpose: the app has no shop
// screen yet, and a cosmetic nobody can see is a row that will drift out of
// date before it is ever used.
var defaultQuests = []defaultQuest{
	{"telegram_subscribed", 1, "", "Подписаться на канал", "Подпишись на наш Telegram-канал", 15, 10},
	{"lessons_completed", 5, "", "Пройти 5 уроков", "", 5, 20},
	{"lessons_completed", 10, "", "Пройти 10 уроков", "", 10, 21},
	{"lessons_completed", 20, "", "Пройти 20 уроков", "", 20, 22},
	{"lessons_completed", 30, "", "Пройти 30 уроков", "", 30, 23},
	{"vocab_learned", 30, "", "Выучить 30 слов", "", 10, 30},
	{"vocab_learned", 100, "", "Выучить 100 слов", "", 25, 31},
	{"vocab_learned", 300, "", "Выучить 300 слов", "", 50, 32},
	{"reviews_done", 100, "", "Сделать 100 повторений", "", 10, 40},
	{"reviews_done", 500, "", "Сделать 500 повторений", "", 30, 41},
	{"correct_in_row", 5, "", "5 правильных подряд", "", 3, 50},
	{"correct_in_row", 10, "", "10 правильных подряд", "", 5, 51},
	{"correct_in_row", 20, "", "20 правильных подряд", "", 15, 52},
	{"streak_days", 7, "", "Стрик 7 дней", "Занимайся 7 дней подряд", 10, 60},
	{"streak_days", 30, "", "Стрик 30 дней", "Занимайся 30 дней подряд", 40, 61},
	{"streak_days", 100, "", "Стрик 100 дней", "Занимайся 100 дней подряд", 150, 62},
	// A level is still worth 50 / 70 / 90 in total; most of it now arrives
	// lesson by lesson (see seedLessonRewards), and what is left is the bonus
	// for finishing the level — the one moment worth a modal.
	{"phase_completed", 100, "1", "Уровень 1 на 100%", "Пройди все уроки первого уровня", 10, 70},
	{"phase_completed", 100, "2", "Уровень 2 на 100%", "Пройди все уроки второго уровня", 19, 71},
	{"phase_completed", 100, "3", "Уровень 3 на 100%", "Пройди все уроки третьего уровня", 30, 72},
}

type defaultProduct struct {
	kind        string
	ref         string
	title       string
	description string
	price       int64
	sortOrder   int
}

var defaultProducts = []defaultProduct{
	{"phase_unlock", "4", "Уровень 4 — Мнения и жизнь", "Открывает уровень A2.2", 500, 10},
	{"phase_unlock", "5", "Уровень 5 — Уверенно", "Открывает уровень B1.1", 1000, 11},
	{"consumable", "streak_repair", "Восстановить стрик", "Вернёт сгоревший стрик в течение 48 часов", 25, 20},
}
