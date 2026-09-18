package api

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

// reminderInactivityAfter is how long since an account's last review or
// exercise attempt before RunReminderSweep sends the "come back" nudge.
const reminderInactivityAfter = 24 * time.Hour

// allDoneMessages are sent once per day, the first time a sweep finds an
// account's review queue empty (see dueQueueRows) — nothing due, nothing new
// left to introduce today.
var allDoneMessages = []string{
	"Все слова на сегодня сделаны. Отличная работа! 🎉",
	"Сегодняшняя очередь пуста — вы всё повторили. Так держать! 👏",
	"Готово! На сегодня слов больше нет, увидимся завтра. ✅",
	"Все карточки на сегодня пройдены. Молодец! 💪",
	"День закрыт: всё, что было на сегодня, вы сделали. 🌟",
}

// inactivityMessages are sent once per quiet spell once an account has gone
// reminderInactivityAfter without a review or exercise attempt.
var inactivityMessages = []string{
	"Вы давно не заглядывали — самое время повторить пару слов. ⏰",
	"Сербский не выучится сам :) Загляните на пару минут. 🇷🇸",
	"Слова заждались — вернитесь, когда будет пара свободных минут. 📚",
	"День без практики — не беда, но лучше вернуться сегодня. 👋",
	"Напоминаем про сербский — откройте приложение и повторите немного. 💭",
}

func pickMessage(msgs []string) string { return msgs[rand.Intn(len(msgs))] }

// RunReminderSweep is one pass of the bot's reminder job. For every account
// with a linked Telegram chat it checks two independent conditions:
//
//   - today's review queue just emptied out (dueQueueRows returns nothing
//     left to do) — sends an "all done" congrats, at most once per calendar
//     day (store.BotReminderState.AllDoneDate).
//   - the account has gone reminderInactivityAfter without a review or
//     exercise attempt — sends a "come back" nudge, at most once per quiet
//     spell: it won't repeat until the account has been active again more
//     recently than the last nudge (store.BotReminderState.InactiveSentAt).
//
// Meant to be called from a ticker (see main.go); safe to call repeatedly —
// the dedup state lives in the store, not in memory, so a crash or restart
// between calls can't cause a duplicate send.
func RunReminderSweep(deps Deps, now time.Time) error {
	h := handlers{deps}

	users, err := deps.Store.TelegramLinkedUsers()
	if err != nil {
		return fmt.Errorf("reminder sweep: telegram users: %w", err)
	}
	if len(users) == 0 {
		return nil
	}
	lastActivity, err := deps.Store.LastActivityByUser()
	if err != nil {
		return fmt.Errorf("reminder sweep: last activity: %w", err)
	}
	today := now.UTC().Format("2006-01-02")

	for _, tu := range users {
		state, err := deps.Store.BotReminderState(tu.UserID)
		if err != nil {
			log.Printf("reminder sweep: state for %s: %v", tu.UserID, err)
			continue
		}
		us := deps.Store.User(tu.UserID)

		if state.AllDoneDate != today {
			rows, err := h.dueQueueRows(us, now)
			if err != nil {
				log.Printf("reminder sweep: queue for %s: %v", tu.UserID, err)
			} else if len(rows) == 0 {
				deps.SendTelegramMessage(tu.ChatID, pickMessage(allDoneMessages))
				if err := deps.Store.SetBotReminderAllDoneDate(tu.UserID, today); err != nil {
					log.Printf("reminder sweep: save all-done state for %s: %v", tu.UserID, err)
				}
			}
		}

		// A never-active account has nothing to compare the 24h window
		// against yet — leave it alone until it has a first activity.
		last, ok := lastActivity[tu.UserID]
		if !ok {
			continue
		}
		if now.Sub(last) >= reminderInactivityAfter && state.InactiveSentAt.Before(last) {
			deps.SendTelegramMessage(tu.ChatID, pickMessage(inactivityMessages))
			if err := deps.Store.SetBotReminderInactiveSentAt(tu.UserID, now); err != nil {
				log.Printf("reminder sweep: save inactivity state for %s: %v", tu.UserID, err)
			}
		}
	}
	return nil
}
