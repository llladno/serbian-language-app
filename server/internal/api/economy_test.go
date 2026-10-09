package api

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/grisha/serbian-app/server/internal/auth"
	"github.com/grisha/serbian-app/server/internal/economy"
	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
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
	if out.CurrencyMany != "зёрнышек" {
		t.Fatalf("currency_many = %q, want зёрнышек", out.CurrencyMany)
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

func TestFinishingALessonPaysItsRewardOnceAndIsNotAQuestToClaim(t *testing.T) {
	h, st, id, c := economyAPI(t)
	if err := st.UpsertQuest(economy.Quest{
		Kind: economy.QuestLessonCompleted, Target: 1, Param: "01",
		Title: "Урок 01", Reward: 7, Active: true,
	}, fixedNow); err != nil {
		t.Fatalf("seed lesson quest: %v", err)
	}
	quests, _ := st.ListQuests(true)
	lessonQuest := quests[len(quests)-1]

	// Per-lesson rewards are not on offer anywhere: they are paid by finishing
	// the lesson, and a list of them would bury the real quests.
	res := doCookie(h, c, "GET", "/api/me/quests", "")
	list := decodeBody[struct {
		Quests []struct {
			ID int64 `json:"id"`
		} `json:"quests"`
	}](t, res)
	for _, q := range list.Quests {
		if q.ID == lessonQuest.ID {
			t.Fatalf("the lesson reward is offered on the quests screen")
		}
	}
	claim := "/api/me/quests/" + strconv.FormatInt(lessonQuest.ID, 10) + "/claim"
	if res := doCookie(h, c, "POST", claim, ""); res.Code != 404 {
		t.Fatalf("claiming a lesson reward by hand = %d, want 404", res.Code)
	}

	res = doCookie(h, c, "POST", "/api/lessons/01/complete", "")
	if res.Code != 200 {
		t.Fatalf("complete = %d: %s", res.Code, res.Body.String())
	}
	paid := decodeBody[struct {
		Reward int64 `json:"reward"`
	}](t, res)
	if paid.Reward != 7 {
		t.Fatalf("reward = %d, want 7", paid.Reward)
	}
	if bal, _ := st.Balance(id); bal != 7 {
		t.Fatalf("balance = %d, want 7", bal)
	}

	// Finishing it again is a normal thing to do and is worth nothing.
	res = doCookie(h, c, "POST", "/api/lessons/01/complete", "")
	if res.Code != 200 {
		t.Fatalf("second complete = %d", res.Code)
	}
	again := decodeBody[struct {
		Reward int64 `json:"reward"`
	}](t, res)
	if again.Reward != 0 {
		t.Fatalf("second completion paid %d, want 0", again.Reward)
	}
	if bal, _ := st.Balance(id); bal != 7 {
		t.Fatalf("balance = %d after finishing twice, want 7", bal)
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

// The screen that closes a lesson reports how much of it was right the first
// time. Redoing an exercise later does not buy the score back.
func TestFinishingALessonReportsFirstTryStats(t *testing.T) {
	h, _, _, c := economyAPI(t)
	check := func(ex, answer string) {
		t.Helper()
		body := `{"answer":"` + answer + `"}`
		if res := doCookie(h, c, "POST", "/api/lessons/01/exercises/"+ex+"/check", body); res.Code != 200 {
			t.Fatalf("check %s = %d", ex, res.Code)
		}
	}
	check("01-A-1", "nope")             // wrong first,
	check("01-A-1", "Zdravo! Kako si?") // right on the second go: still a miss
	check("01-A-2", "Zdravo")           // right first time
	check("01-A-4", "Zdravo, kako si?") // right first time
	check("01-A-4", "Zdravo, kako si?") // asking again changes nothing

	res := doCookie(h, c, "POST", "/api/lessons/01/complete", "")
	got := decodeBody[struct {
		Stats lessonStatsDTO `json:"stats"`
	}](t, res).Stats
	if got.Answered != 3 || got.Mistakes != 1 || got.Percent != 67 {
		t.Fatalf("stats = %+v, want 3 answered, 1 mistake, 67%%", got)
	}
}

func TestFinishingALessonWithNothingAnsweredHasNoPercent(t *testing.T) {
	h, _, _, c := economyAPI(t)
	res := doCookie(h, c, "POST", "/api/lessons/01/complete", "")
	got := decodeBody[struct {
		Stats lessonStatsDTO `json:"stats"`
	}](t, res).Stats
	if got.Answered != 0 || got.Percent != 0 || got.Mistakes != 0 {
		t.Fatalf("stats = %+v, want all zero", got)
	}
}

// "New words" are the lesson's own vocabulary, however the lesson names it.
func TestLessonStatsCountTheLessonsOwnWords(t *testing.T) {
	h, _, _, c := economyAPI(t)
	res := doCookie(h, c, "POST", "/api/lessons/01/complete", "")
	got := decodeBody[struct {
		Stats lessonStatsDTO `json:"stats"`
	}](t, res).Stats
	if got.NewWords != 4 {
		t.Fatalf("new words = %d, want the 4 vocabulary entries filed under lesson 01", got.NewWords)
	}
}

func TestTelegramChannelURL(t *testing.T) {
	for in, want := range map[string]string{
		"@ucimosrb":             "https://t.me/ucimosrb",
		"  @ucimosrb ":          "https://t.me/ucimosrb",
		"https://t.me/ucimosrb": "https://t.me/ucimosrb",
		"-1001234567890":        "", // a private channel's id has no address to open
		"@":                     "",
		"":                      "",
	} {
		if got := telegramChannelURL(in); got != want {
			t.Errorf("telegramChannelURL(%q) = %q, want %q", in, got, want)
		}
	}
}

// The subscription quest tells the app where to go, and whether subscribing
// alone could ever finish it for this account.
func TestSubscriptionQuestCarriesTheChannelLink(t *testing.T) {
	h, st, id, c := economyAPI(t)
	seedQuestRow(t, st, economy.QuestTelegramSubscribed, 1, 15)

	var status string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{"status":"` + status + `"}}`))
	}))
	defer srv.Close()
	defer telegram.SetAPIBase(srv.URL)()

	quest := func() questDTO {
		t.Helper()
		res := doCookie(h, c, "GET", "/api/me/quests", "")
		list := decodeBody[struct {
			Quests []questDTO `json:"quests"`
		}](t, res)
		for _, q := range list.Quests {
			if q.Kind == economy.QuestTelegramSubscribed {
				return q
			}
		}
		t.Fatalf("the subscription quest is not listed")
		return questDTO{}
	}

	// A fresh database already has the channel (migration 019). No Telegram is
	// linked, so subscribing would not be seen.
	q := quest()
	if q.URL != "https://t.me/ucimosrb" || !q.NeedsTelegram || q.Done {
		t.Fatalf("unlinked account: %+v, want the channel link, needs_telegram, not done", q)
	}

	// Linked, but not in the channel yet: the link stays, the warning goes.
	if err := st.CreateIdentity(store.Identity{
		ID: auth.NewIdentityID(), UserID: id, Provider: "telegram", ProviderUID: "5550123",
	}); err != nil {
		t.Fatalf("link telegram: %v", err)
	}
	status = "left"
	q = quest()
	if q.URL == "" || q.NeedsTelegram || q.Done {
		t.Fatalf("linked, not subscribed: %+v, want the link, no warning, not done", q)
	}

	// Subscribed: nothing left to send the learner to.
	status = "member"
	q = quest()
	if !q.Done || q.URL != "" || q.NeedsTelegram {
		t.Fatalf("subscribed: %+v, want done with no link and no warning", q)
	}
}
