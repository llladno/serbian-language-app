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
// Level 1 is the only free level. Level 2 opens for the lessons of Level 1
// alone, with room to spare. Level 3 is earned by lessons and ordinary quests,
// Level 4 is saved up for with a streak, and Levels 5 and 6 are bought. The
// prices (40, 150, 260, 370, 480) rise by an even step after the first and are
// tied to a reference learner: one lesson a day, a streak that is never broken,
// subscribed to the channel, no invited friends. Their income at the end of each
// level is the level pays (52 / 70 / 70 / 90 / 110) plus the one-off quests that
// arrive by then plus the daily drip (217 / 446 / 544 / 738 / 884); without the
// streak quests and the drip it is much less, the floor (120 / 275 / 345 / 515 /
// 625). The cumulative price of the levels bought so far (40, 190, 450, 820,
//  1300. is kept against the two:
//     Level 2 costs 40 against the 52 that finishing Level 1 pays (42 for the
//     lessons themselves, 10 for the level): at most four fifths of it, the rest is
//     room to spare. Quests are not needed.
//     Level 3 stays under the floor: lessons and ordinary quests pay for it, no
//     streak needed (85 to spare).
//     Level 4 is above the floor and no more than the income: it takes a streak,
//     and a streak is enough (94 to spare).
//     Levels 5 and 6 are above the income, 82 and 416 short when the learner
//     finishes the level before, so they are bought.
//
// Prices never fall from one level to the next, or a later level could be bought
// before an earlier one. TestSeededEconomyMatchesTheDesign recomputes all of this
// from these tables, so a reward edited here without the price following fails
// the build. Every number is editable from the admin panel afterwards.
//
// The phase ids below ("1", "1b", "2".."5") are course.yaml's. A renamed phase
// silently breaks both halves — a phase_completed quest nobody can finish, and
// a phase that stops locking — so TestSeedMatchesTheRealCourse cross-checks them
// against the real content tree.
//
// A database seeded before Level 2 was paid is moved to these numbers by
// migration 030 instead, since this runs once.
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
		pos := 0
		for _, lesson := range ph.Lessons {
			pos++
			reward, ok := economy.LessonReward(ph.ID, pos)
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
	{"telegram_subscribed", 1, "", "Подписаться на канал", "Подпишись на наш Telegram-канал", 20, 10},
	{"friends_invited", 1, "", "Пригласить друга", "Друг пришёл по твоей ссылке", 30, 12},
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
	// One-off on purpose: a quest that paid again after every broken streak
	// would be farmable by breaking it deliberately. The renewable income is
	// the daily streak drip.
	{"streak_days", 3, "", "Стрик 3 дня", "Занимайся 3 дня подряд", 20, 58},
	{"streak_days", 7, "", "Стрик 7 дней", "Занимайся 7 дней подряд", 60, 60},
	{"streak_days", 30, "", "Стрик 30 дней", "Занимайся 30 дней подряд", 40, 61},
	{"streak_days", 100, "", "Стрик 100 дней", "Занимайся 100 дней подряд", 150, 62},
	// A level is worth 52 / 70 / 70 / 90 / 110 in total; most of it arrives
	// lesson by lesson (see seedLessonRewards), and what is left is the bonus
	// for finishing the level — the one moment worth a modal.
	{"phase_completed", 100, "1", "Уровень 1 на 100%", "Пройди все уроки первого уровня", 10, 70},
	{"phase_completed", 100, "1b", "Уровень 2 на 100%", "Пройди все уроки второго уровня", 14, 71},
	{"phase_completed", 100, "2", "Уровень 3 на 100%", "Пройди все уроки третьего уровня", 19, 72},
	{"phase_completed", 100, "3", "Уровень 4 на 100%", "Пройди все уроки четвёртого уровня", 30, 73},
	{"phase_completed", 100, "4", "Уровень 5 на 100%", "Пройди все уроки пятого уровня", 22, 74},
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
	{"phase_unlock", "1b", "Уровень 2 — Числа и магазин", "Открывает уровень 2: семья, числа, цены, время, магазин", 40, 9},
	{"phase_unlock", "2", "Уровень 3 — Быт", "Открывает уровень 3: повседневность, квартира, еда, город", 150, 10},
	{"phase_unlock", "3", "Уровень 4 — Связная речь", "Открывает уровень 4: прошлое, планы, транспорт, здоровье", 260, 11},
	{"phase_unlock", "4", "Уровень 5 — Падежи в жизни", "Открывает уровень 5: падежи, прилагательные, местоимения", 370, 13},
	{"phase_unlock", "5", "Уровень 6 — Жизнь на сербском", "Открывает уровень 6: чувства, звонки, работа, документы", 480, 14},
	{"consumable", "streak_repair", "Восстановить стрик", "Вернёт сгоревший стрик в течение 48 часов", 25, 20},
}
