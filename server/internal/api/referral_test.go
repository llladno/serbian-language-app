package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/grisha/serbian-app/server/internal/economy"
	"github.com/grisha/serbian-app/server/internal/store"
)

// referralAPI is economyAPI with registration run inline: register hands its
// work to Deps.Async, which the fixture leaves to the default goroutine.
func referralAPI(t *testing.T) (http.Handler, *store.Store, string, *http.Cookie) {
	t.Helper()
	h, st := newTestAPIWith(t, func(d *Deps) {
		d.SendMail = (&mailSink{}).send
		d.Async = func(f func()) { f() }
	})
	id := testUserID(t, st)
	return h, st, id, authed(t, st, id)
}

type referralBody struct {
	Code    string `json:"code"`
	URL     string `json:"url"`
	Friends int    `json:"friends"`
}

func TestReferralLinkIsStableAndPointsAtTheRegisterPage(t *testing.T) {
	h, _, _, c := economyAPI(t)
	first := decodeBody[referralBody](t, doCookie(h, c, "GET", "/api/me/referral", ""))
	second := decodeBody[referralBody](t, doCookie(h, c, "GET", "/api/me/referral", ""))
	if first.Code == "" || first.Code != second.Code {
		t.Fatalf("code %q then %q; want one stable code", first.Code, second.Code)
	}
	if !strings.HasSuffix(first.URL, "/register?ref="+first.Code) {
		t.Fatalf("url = %q, want it to end in /register?ref=%s", first.URL, first.Code)
	}
}

func TestReferralRequiresASession(t *testing.T) {
	h, _, _, _ := economyAPI(t)
	if rr := anon(h, "GET", "/api/me/referral", ""); rr.Code != 401 {
		t.Fatalf("anonymous referral = %d, want 401", rr.Code)
	}
	if rr := anon(h, "POST", "/api/me/referral/claim", `{"code":"abcdefgh"}`); rr.Code != 401 {
		t.Fatalf("anonymous claim = %d, want 401", rr.Code)
	}
}

// The whole promise: a friend registers through the link, confirms the email,
// and the inviter's quest becomes claimable and pays.
func TestInviteQuestPaysWhenAFriendRegistersThroughTheLink(t *testing.T) {
	h, st, inviterID, c := referralAPI(t)
	q := seedQuestRow(t, st, economy.QuestFriendsInvited, 1, 30)
	link := decodeBody[referralBody](t, doCookie(h, c, "GET", "/api/me/referral", ""))

	questDone := func() bool {
		res := doCookie(h, c, "GET", "/api/me/quests", "")
		list := decodeBody[struct {
			Quests []struct {
				Done  bool `json:"done"`
				Value int  `json:"value"`
			} `json:"quests"`
		}](t, res)
		return len(list.Quests) == 1 && list.Quests[0].Done
	}
	if questDone() {
		t.Fatal("the quest was done before any friend joined")
	}

	body := fmt.Sprintf(`{"email":"friend@example.com","password":"password123","name":"Друг","ref":%q}`, link.Code)
	if rr := anon(h, "POST", "/api/auth/register", body); rr.Code != 200 {
		t.Fatalf("register = %d %s", rr.Code, rr.Body)
	}
	if questDone() {
		t.Fatal("an unconfirmed friend counted")
	}
	ident, err := st.IdentityByProviderUID("password", "friend@example.com")
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	if err := st.SetEmailVerified(ident.ID, fixedNow); err != nil {
		t.Fatal(err)
	}
	if !questDone() {
		t.Fatal("a confirmed friend did not finish the quest")
	}

	claim := fmt.Sprintf("/api/me/quests/%d/claim", q.ID)
	if res := doCookie(h, c, "POST", claim, ""); res.Code != 200 {
		t.Fatalf("claim = %d: %s", res.Code, res.Body.String())
	}
	if bal, _ := st.Balance(inviterID); bal != 30 {
		t.Fatalf("balance = %d, want 30", bal)
	}
}

// A signup that could not carry the code (email confirmed on another device)
// is attached afterwards, once, from the first session.
func TestClaimReferralAttachesAfterTheFact(t *testing.T) {
	h, st, inviterID, c := referralAPI(t)
	link := decodeBody[referralBody](t, doCookie(h, c, "GET", "/api/me/referral", ""))

	friendID := registerAndVerify(t, h, st, "late@example.com", "password123", "Поздний")
	login := anon(h, "POST", "/api/auth/login", loginBody("late@example.com", "password123"))
	fc := grabSessionCookie(t, login)

	res := doCookie(h, fc, "POST", "/api/me/referral/claim", fmt.Sprintf(`{"code":%q}`, link.Code))
	if res.Code != 200 || !decodeBody[struct {
		Applied bool `json:"applied"`
	}](t, res).Applied {
		t.Fatalf("claim = %d %s, want applied", res.Code, res.Body.String())
	}
	if n, _ := st.FriendsInvited(inviterID); n != 1 {
		t.Fatalf("friends = %d, want 1 (friend %s)", n, friendID)
	}
	// a stale or junk code is not an error for the new learner
	if res := doCookie(h, fc, "POST", "/api/me/referral/claim", `{"code":"nonsense"}`); res.Code != 200 {
		t.Fatalf("junk claim = %d, want 200", res.Code)
	}
}
