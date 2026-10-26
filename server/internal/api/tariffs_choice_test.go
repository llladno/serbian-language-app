package api

import (
	"fmt"
	"github.com/grisha/serbian-app/server/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A level payment buys the right to open one level; which one, the learner says
// after paying (docs/superpowers/specs/2026-10-26-level-choice-design.md).

func chooseLevel(h http.Handler, c *http.Cookie, orderID int64, phase string) *httptest.ResponseRecorder {
	return doCookie(h, c, "POST", fmt.Sprintf("/api/me/orders/%d/level", orderID), fmt.Sprintf(`{"phase":%q}`, phase))
}

// payForALevel orders a level, pays it at the fake bank and returns the settled order.
func payForALevel(t *testing.T, tb *fakeTBank, h http.Handler, c *http.Cookie) orderDTO {
	t.Helper()
	o := order(t, h, c, "level")
	tb.set(tb.latestID(), "CONFIRMED")
	rr := doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	if rr.Code != 200 {
		t.Fatalf("GET order = %d %s", rr.Code, rr.Body)
	}
	return decodeBody[orderDTO](t, rr)
}

func TestTariffsListTheLevelsAPaidLevelCanOpenInCourseOrder(t *testing.T) {
	h, st := tariffsAPI(t, newFakeTBank(t))
	_, c := listedUser(t, h, st)
	got := getTariffsOf(t, h, c)

	if len(got.Level.Choices) != 2 || got.Level.Choices[0].ID != "B" || got.Level.Choices[1].ID != "C" {
		t.Fatalf("choices = %+v, want B then C", got.Level.Choices)
	}
	b, cc := got.Level.Choices[0], got.Level.Choices[1]
	if b.Title != "Уровень 4 — Падежи" || b.Lessons != 1 || cc.Lessons != 2 {
		t.Errorf("B = %+v, C = %+v: want titles and the number of written lessons", b, cc)
	}
	// B follows an open level, so nothing needs to come first; C follows B, which is still closed
	if b.After != "" || cc.After != "Уровень 4 — Падежи" {
		t.Errorf("after: B %q, C %q, want none for B and B's title for C", b.After, cc.After)
	}
	if got.Level.Next == nil || got.Level.Next.ID != "B" {
		t.Errorf("next = %+v, want the first choice", got.Level.Next)
	}
	if len(got.Level.Credits) != 0 {
		t.Errorf("credits = %v before any payment", got.Level.Credits)
	}
}

func TestAnOrderForALevelNamesNoLevelUntilTheLearnerChooses(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPI(t, tb)
	_, c := listedUser(t, h, st)

	o := order(t, h, c, "level")
	pay, err := st.PaymentByID(o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if pay.GrantedPhase != "" || pay.AmountKopecks != 19900 {
		t.Errorf("payment = %+v, want no level and 199 ₽", pay)
	}
	desc, _ := tb.get("9001").Request["Description"].(string)
	if !strings.Contains(desc, "на выбор") {
		t.Errorf("description = %q, the bank and the receipt must say the level is chosen", desc)
	}
}

func TestPayingForALevelGivesACreditAndOpensNoLevel(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPI(t, tb)
	_, c := listedUser(t, h, st)

	o := payForALevel(t, tb, h, c)
	if o.Status != "succeeded" || !o.Choose || o.PhaseTitle != "" || o.Bonus != 1000 {
		t.Fatalf("order = %+v, want paid, waiting for a choice, no level yet, 1000 зёрнышек", o)
	}
	for _, id := range []string{"B", "C"} {
		if !phaseLocked(t, h, c, id) {
			t.Errorf("level %s opened by a payment that named none", id)
		}
	}
	got := getTariffsOf(t, h, c)
	if !got.HasPaidLevel || got.Level.Price != 299 || len(got.Level.Credits) != 1 || got.Level.Credits[0] != o.ID {
		t.Errorf("after paying: %+v, want paid, 299 next time and the order as a credit", got.Level)
	}
	if len(got.Level.Choices) != 2 {
		t.Errorf("choices = %+v, both levels are still there to choose", got.Level.Choices)
	}
}

func TestChoosingALevelOpensItAndTheCreditIsSpent(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPI(t, tb)
	_, c := listedUser(t, h, st)
	o := payForALevel(t, tb, h, c)

	rr := chooseLevel(h, c, o.ID, "C") // not the next one: the learner decides
	if rr.Code != 200 {
		t.Fatalf("choose = %d %s", rr.Code, rr.Body)
	}
	done := decodeBody[orderDTO](t, rr)
	if done.PhaseTitle != "Уровень 5 — Жизнь" || done.Choose || done.Bonus != 1000 {
		t.Errorf("answer = %+v, want the chosen level named, nothing left to choose", done)
	}
	if phaseLocked(t, h, c, "C") {
		t.Error("level C is still locked after choosing it")
	}
	if !phaseLocked(t, h, c, "B") {
		t.Error("level B opened too: a credit opens one level")
	}
	got := getTariffsOf(t, h, c)
	if len(got.Level.Credits) != 0 || len(got.Level.Choices) != 1 || got.Level.Choices[0].ID != "B" {
		t.Errorf("after choosing: %+v, want no credit and only B left to choose", got.Level)
	}
	// a second tap on the same level is fine; another level is refused
	if rr := chooseLevel(h, c, o.ID, "C"); rr.Code != 200 {
		t.Errorf("same level again = %d, want 200", rr.Code)
	}
	if rr := chooseLevel(h, c, o.ID, "B"); rr.Code != http.StatusConflict || !strings.Contains(rr.Body.String(), "already_chosen") {
		t.Errorf("other level on a spent credit = %d %s, want 409 already_chosen", rr.Code, rr.Body)
	}
	if !phaseLocked(t, h, c, "B") {
		t.Error("a refused choice opened level B")
	}
}

func TestAChoiceThatIsNotAnOpenableLevelIsRefused(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPI(t, tb)
	uid, c := listedUser(t, h, st)
	o := payForALevel(t, tb, h, c)

	for _, ph := range []string{"A", "Z", ""} { // A is free, Z does not exist
		if rr := chooseLevel(h, c, o.ID, ph); rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "bad_phase") {
			t.Errorf("choose %q = %d %s, want 400 bad_phase", ph, rr.Code, rr.Body)
		}
	}
	// a level the learner already has (here: bought with зёрнышки) cannot be chosen again
	if err := st.User(uid).GrantEntitlement("phase_unlock", "B", 1, fixedNow); err != nil {
		t.Fatal(err)
	}
	if rr := chooseLevel(h, c, o.ID, "B"); rr.Code != http.StatusBadRequest {
		t.Errorf("choose an open level = %d, want 400", rr.Code)
	}
	if rr := chooseLevel(h, c, o.ID, "C"); rr.Code != 200 {
		t.Errorf("choose the other one = %d %s", rr.Code, rr.Body)
	}
}

func TestOnlyAPaidLevelOrderOfTheBuyerCanBeChosenOn(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPI(t, tb)
	_, c := listedUser(t, h, st)

	unpaid := order(t, h, c, "level")
	if rr := chooseLevel(h, c, unpaid.ID, "B"); rr.Code != http.StatusConflict || !strings.Contains(rr.Body.String(), "not_claimable") {
		t.Errorf("an unpaid order = %d %s, want 409 not_claimable", rr.Code, rr.Body)
	}
	other, _ := st.CreateUser("Other")
	pid, err := st.CreatePayment(store.Payment{UserID: other, TariffCode: "level", AmountKopecks: 19900, Provider: "tbank"}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	st.CompletePayment(pid, fixedNow)
	if rr := chooseLevel(h, c, pid, "B"); rr.Code != http.StatusNotFound {
		t.Errorf("somebody else's credit = %d, want 404", rr.Code)
	}
	if !phaseLocked(t, h, c, "B") {
		t.Error("level B opened on a refused choice")
	}
	if rr := doCookie(h, c, "POST", "/api/me/orders/abc/level", `{"phase":"B"}`); rr.Code != http.StatusNotFound {
		t.Errorf("order abc = %d, want 404", rr.Code)
	}
	if rr := doCookie(h, c, "POST", fmt.Sprintf("/api/me/orders/%d/level", unpaid.ID), `not json`); rr.Code != http.StatusBadRequest {
		t.Errorf("a broken body = %d, want 400", rr.Code)
	}
}

func TestANewLevelCannotBeBoughtWhileACreditIsUnspent(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPI(t, tb)
	_, c := listedUser(t, h, st)
	o := payForALevel(t, tb, h, c)

	rr := doCookie(h, c, "POST", "/api/me/orders", `{"tariff":"level"}`)
	if rr.Code != http.StatusConflict || !strings.Contains(rr.Body.String(), "level_pending_choice") {
		t.Fatalf("second level order = %d %s, want 409 level_pending_choice", rr.Code, rr.Body)
	}
	if tb.count() != 1 {
		t.Error("the bank was asked to charge for a second level while the first was not chosen")
	}
	if rr := chooseLevel(h, c, o.ID, "B"); rr.Code != 200 {
		t.Fatal(rr.Body)
	}
	if rr := doCookie(h, c, "POST", "/api/me/orders", `{"tariff":"level"}`); rr.Code != 200 {
		t.Errorf("after choosing, a level order = %d %s, want 200", rr.Code, rr.Body)
	}
}

// The notification arrives with nobody signed in: the credit waits in the account.
func TestACreditPaidWhileAwayIsWaitingAtTheNextVisit(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPI(t, tb)
	_, c := listedUser(t, h, st)
	o := order(t, h, c, "level")
	tb.set("9001", "CONFIRMED")
	body := tb.notification("9001", "CONFIRMED")
	if rr := anon(h, "POST", "/api/tbank/webhook", body); rr.Code != 200 {
		t.Fatalf("webhook = %d %s", rr.Code, rr.Body)
	}
	got := getTariffsOf(t, h, c)
	if len(got.Level.Credits) != 1 || got.Level.Credits[0] != o.ID {
		t.Fatalf("credits = %v, want the paid order waiting", got.Level.Credits)
	}
}

// Everything was opened some other way while the credit waited: it keeps waiting
// for the next level to appear, and nothing can be chosen meanwhile.
func TestACreditWaitsWhenThereIsNothingLeftToChoose(t *testing.T) {
	tb := newFakeTBank(t)
	h, st := tariffsAPI(t, tb)
	uid, c := listedUser(t, h, st)
	o := payForALevel(t, tb, h, c)
	for _, ref := range []string{"B", "C"} {
		if err := st.User(uid).GrantEntitlement("phase_unlock", ref, 1, fixedNow); err != nil {
			t.Fatal(err)
		}
	}
	got := getTariffsOf(t, h, c)
	if len(got.Level.Choices) != 0 || len(got.Level.Credits) != 1 || got.Level.Credits[0] != o.ID {
		t.Fatalf("level = %+v, want no choices and the credit kept", got.Level)
	}
	if rr := chooseLevel(h, c, o.ID, "B"); rr.Code != http.StatusBadRequest {
		t.Errorf("choose with nothing left = %d, want 400", rr.Code)
	}
}
