package api

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/grisha/serbian-app/server/internal/economy"
	"github.com/grisha/serbian-app/server/internal/store"
)

// economyAPI returns the standard fixture API plus a live session cookie for
// account "tester". The currency routes sit behind requireSession, so the
// X-User bridge the other content tests use cannot reach them.
func economyAPI(t *testing.T) (http.Handler, *store.Store, string, *http.Cookie) {
	t.Helper()
	h, st := newTestAPIWith(t, nil)
	id := testUserID(t, st)
	return h, st, id, authed(t, st, id)
}

func creditUser(t *testing.T, st *store.Store, userID string, amount int64, key string) {
	t.Helper()
	if _, err := st.AddLedgerEntry(store.LedgerEntry{
		UserID: userID, Amount: amount, Kind: "admin_adjustment",
		IdempotencyKey: key, Comment: "test", CreatedBy: "admin",
	}, fixedNow); err != nil {
		t.Fatalf("credit: %v", err)
	}
}

func seedQuestRow(t *testing.T, st *store.Store, kind string, target int, reward int64) economy.Quest {
	t.Helper()
	if err := st.UpsertQuest(economy.Quest{
		Kind: kind, Target: target, Title: "Задание", Reward: reward, Active: true,
	}, fixedNow); err != nil {
		t.Fatalf("seed quest: %v", err)
	}
	qs, err := st.ListQuests(true)
	if err != nil || len(qs) == 0 {
		t.Fatalf("list quests: %v", err)
	}
	return qs[len(qs)-1]
}

func TestWalletReportsBalanceAndCurrencyName(t *testing.T) {
	h, st, id, c := economyAPI(t)
	creditUser(t, st, id, 42, "seed:1")

	res := doCookie(h, c, "GET", "/api/me/wallet", "")
	if res.Code != 200 {
		t.Fatalf("status = %d: %s", res.Code, res.Body.String())
	}
	out := decodeBody[struct {
		Balance      int64  `json:"balance"`
		CurrencyMany string `json:"currency_many"`
	}](t, res)
	if out.Balance != 42 {
		t.Fatalf("balance = %d, want 42", out.Balance)
	}
	if out.CurrencyMany != "пёрышек" {
		t.Fatalf("currency_many = %q, want пёрышек", out.CurrencyMany)
	}
}

func TestQuestListAndClaimFlow(t *testing.T) {
	h, st, id, c := economyAPI(t)
	q := seedQuestRow(t, st, economy.QuestLessonsCompleted, 1, 30)
	claim := "/api/me/quests/" + strconv.FormatInt(q.ID, 10) + "/claim"

	res := doCookie(h, c, "GET", "/api/me/quests", "")
	if res.Code != 200 {
		t.Fatalf("list = %d: %s", res.Code, res.Body.String())
	}
	list := decodeBody[struct {
		Quests []struct {
			ID      int64 `json:"id"`
			Value   int   `json:"value"`
			Target  int   `json:"target"`
			Done    bool  `json:"done"`
			Claimed bool  `json:"claimed"`
		} `json:"quests"`
	}](t, res)
	if len(list.Quests) != 1 || list.Quests[0].Done {
		t.Fatalf("quests = %+v; want one, not done", list.Quests)
	}

	// The client's opinion that a quest is finished is never trusted.
	if res := doCookie(h, c, "POST", claim, ""); res.Code != 409 {
		t.Fatalf("premature claim = %d, want 409", res.Code)
	}
	if bal, _ := st.Balance(id); bal != 0 {
		t.Fatalf("balance = %d after a refused claim, want 0", bal)
	}

	if res := doCookie(h, c, "POST", "/api/lessons/01/complete", ""); res.Code != 200 {
		t.Fatalf("complete lesson = %d", res.Code)
	}

	if res := doCookie(h, c, "POST", claim, ""); res.Code != 200 {
		t.Fatalf("claim = %d: %s", res.Code, res.Body.String())
	}
	if bal, _ := st.Balance(id); bal != 30 {
		t.Fatalf("balance = %d, want 30", bal)
	}
	if res := doCookie(h, c, "POST", claim, ""); res.Code != 409 {
		t.Fatalf("repeat claim = %d, want 409", res.Code)
	}
	if bal, _ := st.Balance(id); bal != 30 {
		t.Fatalf("balance = %d after a repeat claim, want 30", bal)
	}

	// The reward shows up in the user's own transaction list.
	res = doCookie(h, c, "GET", "/api/me/transactions", "")
	tx := decodeBody[struct {
		Transactions []struct {
			Amount int64  `json:"amount"`
			Kind   string `json:"kind"`
		} `json:"transactions"`
	}](t, res)
	if len(tx.Transactions) != 1 || tx.Transactions[0].Kind != "quest_reward" || tx.Transactions[0].Amount != 30 {
		t.Fatalf("transactions = %+v; want one quest_reward of 30", tx.Transactions)
	}
}

func TestClaimedQuestStaysDoneWhenItsCounterFallsBack(t *testing.T) {
	// Counters are not monotonic. A streak quest paid at seven days reads zero
	// the morning after the streak breaks, and an admin can raise a target
	// after the fact — as here, which needs no clock. Either way the quest was
	// paid, and a paid quest that reads "not finished" would be shown as still
	// in progress and would hide the next rung of its ladder.
	h, st, id, c := economyAPI(t)
	q := seedQuestRow(t, st, economy.QuestLessonsCompleted, 1, 30)

	if res := doCookie(h, c, "POST", "/api/lessons/01/complete", ""); res.Code != 200 {
		t.Fatalf("complete lesson = %d", res.Code)
	}
	if res := doCookie(h, c, "POST", "/api/me/quests/"+strconv.FormatInt(q.ID, 10)+"/claim", ""); res.Code != 200 {
		t.Fatalf("claim = %d: %s", res.Code, res.Body.String())
	}

	q.Target = 10
	if err := st.UpsertQuest(q, fixedNow); err != nil {
		t.Fatalf("raise target: %v", err)
	}

	res := doCookie(h, c, "GET", "/api/me/quests", "")
	list := decodeBody[struct {
		Quests []struct {
			Value   int  `json:"value"`
			Target  int  `json:"target"`
			Done    bool `json:"done"`
			Claimed bool `json:"claimed"`
		} `json:"quests"`
	}](t, res)
	if len(list.Quests) != 1 {
		t.Fatalf("quests = %+v; want one", list.Quests)
	}
	got := list.Quests[0]
	if got.Value >= got.Target {
		t.Fatalf("value %d / target %d: the counter was supposed to fall short", got.Value, got.Target)
	}
	if !got.Done || !got.Claimed {
		t.Fatalf("quest = %+v; want done and claimed", got)
	}
	if bal, _ := st.Balance(id); bal != 30 {
		t.Fatalf("balance = %d, want 30 — nothing should have been paid twice", bal)
	}
}

func TestShopReportsEffectivePriceAndOwnership(t *testing.T) {
	h, st, id, c := economyAPI(t)
	if _, err := st.UpsertProduct(economy.Product{
		Kind: economy.ProductCosmetic, Ref: "palette:forest", Title: "Палитра",
		Price: 30, DiscountPercent: 50, GrantQty: 1, Active: true,
	}, fixedNow); err != nil {
		t.Fatalf("seed product: %v", err)
	}
	if err := st.User(id).GrantEntitlement(economy.ProductCosmetic, "palette:forest", 1, fixedNow); err != nil {
		t.Fatalf("grant: %v", err)
	}

	res := doCookie(h, c, "GET", "/api/shop", "")
	if res.Code != 200 {
		t.Fatalf("shop = %d: %s", res.Code, res.Body.String())
	}
	out := decodeBody[struct {
		Items []struct {
			Price           int64 `json:"price"`
			PriceEffective  int64 `json:"price_effective"`
			DiscountPercent int   `json:"discount_percent"`
			Owned           int64 `json:"owned"`
		} `json:"items"`
	}](t, res)
	if len(out.Items) != 1 {
		t.Fatalf("items = %+v, want 1", out.Items)
	}
	it := out.Items[0]
	if it.Price != 30 || it.PriceEffective != 15 || it.DiscountPercent != 50 || it.Owned != 1 {
		t.Fatalf("item = %+v; want price 30, effective 15, 50%%, owned 1", it)
	}
}

func TestPurchaseEndpointReportsInsufficientFunds(t *testing.T) {
	h, st, _, c := economyAPI(t)
	pid, err := st.UpsertProduct(economy.Product{
		Kind: economy.ProductConsumable, Ref: "streak_repair", Title: "Восстановление",
		Price: 25, GrantQty: 1, Active: true,
	}, fixedNow)
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}

	res := doCookie(h, c, "POST", "/api/me/purchases", `{"product_id":`+strconv.FormatInt(pid, 10)+`}`)
	if res.Code != 409 {
		t.Fatalf("status = %d, want 409: %s", res.Code, res.Body.String())
	}
	out := decodeBody[struct {
		Error string `json:"error"`
	}](t, res)
	if out.Error != "insufficient_funds" {
		t.Fatalf("error = %q, want insufficient_funds", out.Error)
	}
}

// A retried POST must not charge twice. The store layer can only honour that
// if the handler actually passes the request's key down.
func TestPurchaseEndpointHonoursTheIdempotencyKey(t *testing.T) {
	h, st, id, c := economyAPI(t)
	pid, err := st.UpsertProduct(economy.Product{
		Kind: economy.ProductConsumable, Ref: "streak_repair", Title: "Восстановление",
		Price: 25, GrantQty: 1, Active: true,
	}, fixedNow)
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}
	creditUser(t, st, id, 100, "seed:1")
	body := `{"product_id":` + strconv.FormatInt(pid, 10) + `}`

	for i := 0; i < 3; i++ {
		res := doCookieWithKey(h, c, "POST", "/api/me/purchases", body, "req-1")
		if res.Code != 200 {
			t.Fatalf("attempt %d = %d: %s", i, res.Code, res.Body.String())
		}
	}
	if bal, _ := st.Balance(id); bal != 75 {
		t.Fatalf("balance = %d after three identical requests, want 75", bal)
	}
	ent, _ := st.User(id).Entitlements()
	if ent["consumable:streak_repair"] != 1 {
		t.Fatalf("owned = %d, want 1", ent["consumable:streak_repair"])
	}

	// Without a key every request is a separate purchase, as before.
	if res := doCookie(h, c, "POST", "/api/me/purchases", body); res.Code != 200 {
		t.Fatalf("keyless purchase = %d", res.Code)
	}
	if bal, _ := st.Balance(id); bal != 50 {
		t.Fatalf("balance = %d, want 50", bal)
	}
}

func TestCurrencyEndpointsRejectTheLegacyHeaderBridge(t *testing.T) {
	h, _ := newTestAPIWith(t, nil)
	// requireAuth still honours X-User for the legacy content routes. A
	// currency endpoint registered there would let anyone spend anyone's
	// balance by setting a header.
	for _, c := range []struct{ method, path, body string }{
		{"GET", "/api/me/wallet", ""},
		{"GET", "/api/me/quests", ""},
		{"GET", "/api/me/transactions", ""},
		{"GET", "/api/shop", ""},
		{"POST", "/api/me/purchases", `{"product_id":1}`},
		{"POST", "/api/me/streak/repair", `{"day":"2026-09-28"}`},
		{"POST", "/api/me/quests/1/claim", ""},
	} {
		if res := do(h, c.method, c.path, c.body); res.Code != 401 {
			t.Errorf("%s %s with X-User = %d, want 401", c.method, c.path, res.Code)
		}
	}
}

func TestRepairStreakEndpointNeedsAConsumable(t *testing.T) {
	h, _, _, c := economyAPI(t)
	res := doCookie(h, c, "POST", "/api/me/streak/repair", `{"day":"2026-09-28"}`)
	if res.Code != 409 {
		t.Fatalf("status = %d, want 409: %s", res.Code, res.Body.String())
	}
	out := decodeBody[struct {
		Error string `json:"error"`
	}](t, res)
	if out.Error != "no_repair_available" {
		t.Fatalf("error = %q, want no_repair_available", out.Error)
	}
}
