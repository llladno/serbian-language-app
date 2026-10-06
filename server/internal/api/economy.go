// Economy endpoints: wallet, quests, shop, purchases and streak repair.
//
// Every handler here resolves the account from the session, never from the
// X-User bridge, and every one of them is registered with requireSession for
// that reason. The bridge exists for legacy content routes; a currency
// endpoint behind it would let anyone spend anyone else's balance by setting a
// header.
package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/grisha/serbian-app/server/internal/economy"
	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
)

const (
	transactionPageDefault = 50
	transactionPageMax     = 200
)

type walletDTO struct {
	Balance      int64  `json:"balance"`
	CurrencyOne  string `json:"currency_one"`
	CurrencyFew  string `json:"currency_few"`
	CurrencyMany string `json:"currency_many"`
	StreakDays   int    `json:"streak_days"`
}

type questDTO struct {
	ID          int64  `json:"id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Target      int    `json:"target"`
	Value       int    `json:"value"`
	Reward      int64  `json:"reward"`
	Done        bool   `json:"done"`
	Claimed     bool   `json:"claimed"`
}

type shopItemDTO struct {
	ID              int64  `json:"id"`
	Kind            string `json:"kind"`
	Ref             string `json:"ref"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	Price           int64  `json:"price"`
	PriceEffective  int64  `json:"price_effective"`
	DiscountPercent int    `json:"discount_percent,omitempty"`
	Owned           int64  `json:"owned"`
}

type transactionDTO struct {
	ID        int64  `json:"id"`
	Amount    int64  `json:"amount"`
	Kind      string `json:"kind"`
	Ref       string `json:"ref"`
	Comment   string `json:"comment"`
	CreatedAt string `json:"created_at"`
}

func (h handlers) getWallet(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	set, err := h.Store.EconomySettings()
	if err != nil {
		log.Printf("wallet: settings: %v", err)
		fail(w, 500, "internal error")
		return
	}
	bal, err := h.Store.Balance(ac.UserID)
	if err != nil {
		log.Printf("wallet: balance: %v", err)
		fail(w, 500, "internal error")
		return
	}
	streak, err := h.Store.User(ac.UserID).StreakDays(h.Now())
	if err != nil {
		log.Printf("wallet: streak: %v", err)
		fail(w, 500, "internal error")
		return
	}
	writeJSON(w, 200, walletDTO{
		Balance:      bal,
		CurrencyOne:  set.CurrencyNameOne,
		CurrencyFew:  set.CurrencyNameFew,
		CurrencyMany: set.CurrencyNameMany,
		StreakDays:   streak,
	})
}

// phaseLessons maps phase id to its lesson ids, for phase_completed quests.
func (h handlers) phaseLessons() map[string][]string {
	out := map[string][]string{}
	for _, p := range h.Course().Phases {
		out[p.ID] = p.Lessons
	}
	return out
}

// questCounters gathers the local counters and, only if some quest in the list
// needs it, the Telegram subscription flag — which costs a Bot API round trip,
// so it is not paid for on a profile with no such quest.
func (h handlers) questCounters(userID string, quests []economy.Quest) (economy.Counters, error) {
	c, err := h.Store.User(userID).QuestCounters(h.Now())
	if err != nil {
		return c, err
	}
	for _, q := range quests {
		if q.Kind == economy.QuestTelegramSubscribed {
			c.TelegramSubscribed = h.telegramSubscribed(userID)
			break
		}
	}
	return c, nil
}

// telegramSubscribed answers the channel-membership question, degrading to
// false on every failure: an unreachable Bot API must leave the profile screen
// working, just with that one quest unfinished.
func (h handlers) telegramSubscribed(userID string) bool {
	set, err := h.Store.EconomySettings()
	if err != nil || set.TelegramChannel == "" {
		return false
	}
	chatID, ok := h.Store.TelegramChatID(userID)
	if !ok {
		return false
	}
	status, err := telegram.GetChatMember(h.Config.TelegramBotToken, set.TelegramChannel, chatID)
	if err != nil {
		log.Printf("quests: getChatMember: %v", err)
		return false
	}
	return telegram.IsMember(status)
}

// visibleQuests drops quests nobody could finish: a telegram_subscribed quest
// with no channel configured would sit on the profile for ever.
func (h handlers) visibleQuests() ([]economy.Quest, error) {
	quests, err := h.Store.ListQuests(true)
	if err != nil {
		return nil, err
	}
	set, err := h.Store.EconomySettings()
	if err != nil {
		return nil, err
	}
	if set.TelegramChannel != "" {
		return quests, nil
	}
	out := make([]economy.Quest, 0, len(quests))
	for _, q := range quests {
		if q.Kind == economy.QuestTelegramSubscribed {
			continue
		}
		out = append(out, q)
	}
	return out, nil
}

func (h handlers) listQuests(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	quests, err := h.visibleQuests()
	if err != nil {
		log.Printf("quests: list: %v", err)
		fail(w, 500, "internal error")
		return
	}
	claimed, err := h.Store.User(ac.UserID).ClaimedQuestIDs()
	if err != nil {
		log.Printf("quests: claimed: %v", err)
		fail(w, 500, "internal error")
		return
	}
	// Counters only answer "how far along is this quest", which is settled for
	// the claimed ones. Passing just the open quests also keeps the Bot API out
	// of the request once the subscription quest is paid: the screen asks for
	// this list on every navigation.
	counters, err := h.questCounters(ac.UserID, openQuests(quests, claimed))
	if err != nil {
		log.Printf("quests: counters: %v", err)
		fail(w, 500, "internal error")
		return
	}
	phases := h.phaseLessons()
	out := struct {
		Quests []questDTO `json:"quests"`
	}{Quests: []questDTO{}}
	for _, q := range quests {
		// A claimed quest is finished by definition — the server checked it
		// when it paid — and some counters fall back afterwards: break a streak
		// and streak_days drops to 0. Recomputing "done" from the counter alone
		// would make a paid quest look unfinished for ever.
		done := claimed[q.ID] || economy.QuestDone(q, counters, phases)
		out.Quests = append(out.Quests, questDTO{
			ID: q.ID, Kind: q.Kind, Title: q.Title, Description: q.Description,
			Target: q.Target, Value: economy.QuestValue(q, counters, phases),
			Reward: q.Reward, Done: done,
			Claimed: claimed[q.ID],
		})
	}
	writeJSON(w, 200, out)
}

// openQuests drops the ones this user has already been paid for.
func openQuests(quests []economy.Quest, claimed map[int64]bool) []economy.Quest {
	out := make([]economy.Quest, 0, len(quests))
	for _, q := range quests {
		if !claimed[q.ID] {
			out = append(out, q)
		}
	}
	return out
}

func (h handlers) claimQuest(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, 400, "bad quest id")
		return
	}
	// Resolved from the list of quests the profile actually offers, which is
	// also what makes ClaimQuest's "the caller has checked Active" contract
	// hold: an id in a request is not proof the quest is on offer.
	quests, err := h.visibleQuests()
	if err != nil {
		log.Printf("claim: list: %v", err)
		fail(w, 500, "internal error")
		return
	}
	var quest economy.Quest
	for _, q := range quests {
		if q.ID == id {
			quest = q
			break
		}
	}
	if quest.ID == 0 {
		fail(w, 404, "unknown quest")
		return
	}
	counters, err := h.questCounters(ac.UserID, []economy.Quest{quest})
	if err != nil {
		log.Printf("claim: counters: %v", err)
		fail(w, 500, "internal error")
		return
	}
	// Completion is re-checked here, on the server, every time. The client's
	// opinion that a quest is done is never trusted.
	if !economy.QuestDone(quest, counters, h.phaseLessons()) {
		fail(w, 409, "not_completed")
		return
	}
	if err := h.Store.User(ac.UserID).ClaimQuest(quest, h.Now()); err != nil {
		if errors.Is(err, store.ErrAlreadyClaimed) {
			fail(w, 409, "already_claimed")
			return
		}
		log.Printf("claim: %v", err)
		fail(w, 500, "internal error")
		return
	}
	bal, _ := h.Store.Balance(ac.UserID)
	writeJSON(w, 200, map[string]any{"reward": quest.Reward, "balance": bal})
}

func (h handlers) listTransactions(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	limit := transactionPageDefault
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = min(v, transactionPageMax)
	}
	offset := 0
	if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && v > 0 {
		offset = v
	}
	rows, err := h.Store.ListLedger(ac.UserID, limit, offset)
	if err != nil {
		log.Printf("transactions: %v", err)
		fail(w, 500, "internal error")
		return
	}
	out := struct {
		Transactions []transactionDTO `json:"transactions"`
	}{Transactions: []transactionDTO{}}
	for _, e := range rows {
		out.Transactions = append(out.Transactions, transactionDTO{
			ID: e.ID, Amount: e.Amount, Kind: e.Kind, Ref: e.Ref,
			Comment: e.Comment, CreatedAt: e.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, 200, out)
}

func (h handlers) getShop(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	products, err := h.Store.ListProducts(true)
	if err != nil {
		log.Printf("shop: %v", err)
		fail(w, 500, "internal error")
		return
	}
	owned, err := h.Store.User(ac.UserID).Entitlements()
	if err != nil {
		log.Printf("shop: entitlements: %v", err)
		fail(w, 500, "internal error")
		return
	}
	now := h.Now()
	out := struct {
		Items []shopItemDTO `json:"items"`
	}{Items: []shopItemDTO{}}
	for _, p := range products {
		sale := p.SalePercent(now)
		effective, _ := economy.EffectivePrice(p.Price, sale, 0)
		out.Items = append(out.Items, shopItemDTO{
			ID: p.ID, Kind: p.Kind, Ref: p.Ref, Title: p.Title, Description: p.Description,
			Price: p.Price, PriceEffective: effective, DiscountPercent: sale,
			Owned: owned[p.Kind+":"+p.Ref],
		})
	}
	writeJSON(w, 200, out)
}

func (h handlers) purchase(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	var req struct {
		ProductID int64  `json:"product_id"`
		PromoCode string `json:"promo_code"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, 400, "bad request body")
		return
	}
	// A client that lost the response and retries sends the same
	// Idempotency-Key, and gets the original purchase back instead of a second
	// charge. Without the header every request is a distinct purchase, which is
	// also correct — buying three repairs is three purchases — so the header is
	// how the client says "this is the same one", not something we can infer.
	res, err := h.Store.User(ac.UserID).Purchase(req.ProductID, req.PromoCode,
		r.Header.Get("Idempotency-Key"), h.Now())
	switch {
	case err == nil:
	case errors.Is(err, store.ErrInsufficientFunds):
		fail(w, 409, "insufficient_funds")
		return
	case errors.Is(err, store.ErrAlreadyOwned):
		fail(w, 409, "already_owned")
		return
	case errors.Is(err, store.ErrProductUnavailable):
		fail(w, 404, "product_unavailable")
		return
	case errors.Is(err, store.ErrPromoInvalid):
		fail(w, 400, "promo_invalid")
		return
	default:
		log.Printf("purchase: %v", err)
		fail(w, 500, "internal error")
		return
	}
	bal, _ := h.Store.Balance(ac.UserID)
	writeJSON(w, 200, map[string]any{
		"paid": res.Paid, "applied": res.Applied, "balance": bal, "replayed": res.Replayed,
	})
}

func (h handlers) repairStreak(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	var req struct {
		Day string `json:"day"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, 400, "bad request body")
		return
	}
	if err := h.Store.User(ac.UserID).SpendStreakRepair(req.Day, h.Now()); err != nil {
		if errors.Is(err, store.ErrNoEntitlement) {
			fail(w, 409, "no_repair_available")
			return
		}
		// The remaining failures are all "that day cannot be repaired": not in
		// the past, already active, or outside the window. They carry their own
		// explanation and none of them is a fault on our side.
		fail(w, 400, err.Error())
		return
	}
	streak, _ := h.Store.User(ac.UserID).StreakDays(h.Now())
	writeJSON(w, 200, map[string]any{"streak_days": streak})
}
