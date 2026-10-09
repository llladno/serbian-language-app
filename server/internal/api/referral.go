// Invite-a-friend endpoints. The link is a plain page of the site
// (/register?ref=CODE), not a bot deep link: the site is the one thing every
// signup path goes through, and nothing about it depends on the Telegram Bot
// API being reachable from the server.
package api

import (
	"log"
	"net/http"
	"strings"
)

type referralDTO struct {
	Code    string `json:"code"`
	URL     string `json:"url"`
	Friends int    `json:"friends"`
}

// getReferral returns the caller's invite code and the link built from it.
func (h handlers) getReferral(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	code, err := h.Store.ReferralCode(ac.UserID)
	if err != nil {
		log.Printf("referral: code: %v", err)
		fail(w, 500, "internal error")
		return
	}
	friends, err := h.Store.FriendsInvited(ac.UserID)
	if err != nil {
		log.Printf("referral: friends: %v", err)
		fail(w, 500, "internal error")
		return
	}
	writeJSON(w, 200, referralDTO{
		Code:    code,
		URL:     strings.TrimRight(h.Config.AppBaseURL, "/") + "/register?ref=" + code,
		Friends: friends,
	})
}

// claimReferral attaches the caller to an inviter after the fact. It exists for
// the signups that cannot carry the code with them — the email was confirmed on
// another device, or the account came through the bot — so the browser that
// saw the link hands it over on the first session. It answers 200 whatever the
// code was: a stale link is not an error the new learner can do anything about.
func (h handlers) claimReferral(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	applied, err := h.Store.ApplyReferral(ac.UserID, req.Code, h.Now())
	if err != nil {
		log.Printf("referral: claim: %v", err)
		fail(w, 500, "internal error")
		return
	}
	writeJSON(w, 200, map[string]bool{"applied": applied})
}

// applyReferral is the best-effort version used while an account is being
// created: a failure is logged and never fails the signup.
func (h handlers) applyReferral(userID, code string) {
	if strings.TrimSpace(code) == "" {
		return
	}
	if _, err := h.Store.ApplyReferral(userID, code, h.Now()); err != nil {
		log.Printf("signup: apply referral: %v", err)
	}
}
