package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/grisha/serbian-app/server/internal/config"
	"github.com/grisha/serbian-app/server/internal/store"
)

// The stand-in cash desk behind PAYMENTS_DEV (config.Config.PaymentsDev): a payment "service" that is
// a page with two buttons, so the flow from the order to the choice of a level can be tried on a local
// machine without a bank. Everything around it is the real code: the order is created as usual, the
// payment is settled by settlePayment (which asks this desk where it stands, checks the amount and the
// order, and only then grants), and the screens read the same answers.
//
// The desk keeps its memory in the payment's own provider id, which the store already holds:
//
//	dev-<order>-<kopecks>         waiting at the desk
//	dev-<order>-<kopecks>-paid    the buyer pressed "Оплатить"
//	dev-<order>-<kopecks>-failed  the buyer pressed "Отказаться"
//
// so nothing lives in the process, a restart loses nothing, and one test cannot leak into another.
type devProvider struct{}

func (devProvider) Name() string     { return config.ProviderDev }
func (devProvider) Configured() bool { return true }

func (devProvider) Create(_ context.Context, in payCreate) (payCreated, error) {
	return payCreated{
		ProviderPaymentID: fmt.Sprintf("dev-%d-%d", in.OrderID, in.AmountKopecks),
		// Relative on purpose: the page is the app's own, wherever the app is being served from.
		PayURL: fmt.Sprintf("/dev/pay?order=%d", in.OrderID),
		Test:   true,
	}, nil
}

func (devProvider) State(_ context.Context, id string) (payState, error) {
	parts := strings.Split(id, "-")
	if len(parts) < 3 || len(parts) > 4 || parts[0] != "dev" {
		return payState{}, fmt.Errorf("not a stand-in payment id: %q", id)
	}
	order, err1 := strconv.ParseInt(parts[1], 10, 64)
	amount, err2 := strconv.ParseInt(parts[2], 10, 64)
	if err1 != nil || err2 != nil {
		return payState{}, fmt.Errorf("not a stand-in payment id: %q", id)
	}
	st := payState{AmountKopecks: amount, OrderRef: order}
	if len(parts) == 4 {
		st.Paid = parts[3] == "paid"
		st.Failed = parts[3] == "failed"
	}
	return st, nil
}

// devOrder finds the order a stand-in page is about: the caller's own, made at this desk. Anything else
// is a 404, and so is everything when the desk is switched off (nothing here may touch a real order).
func (h handlers) devOrder(w http.ResponseWriter, r *http.Request) (store.Payment, bool) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return store.Payment{}, false
	}
	if !h.Config.PaymentsDev {
		fail(w, http.StatusNotFound, "not found")
		return store.Payment{}, false
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, http.StatusNotFound, "not found")
		return store.Payment{}, false
	}
	p, err := h.Store.PaymentByID(id)
	if err != nil || p.UserID != ac.UserID || p.Provider != config.ProviderDev {
		fail(w, http.StatusNotFound, "not found")
		return store.Payment{}, false
	}
	return p, true
}

// devOrderInfo is what the stand-in page shows: what is being bought and for how much.
func (h handlers) devOrderInfo(w http.ResponseWriter, r *http.Request) {
	p, ok := h.devOrder(w, r)
	if !ok {
		return
	}
	title := p.TariffCode
	if t, err := h.Store.Tariff(p.TariffCode); err == nil {
		title = t.Title
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": p.ID, "tariff": p.TariffCode, "title": title, "amount": rubles(p.AmountKopecks), "status": p.Status,
	})
}

// devPay records what the buyer pressed ("paid" or "failed") and settles the order the way a notification
// from a bank would: by asking the desk where the payment stands. Pressing the same button twice is
// harmless; deciding the other way once the order is settled is refused.
func (h handlers) devPay(w http.ResponseWriter, r *http.Request) {
	p, ok := h.devOrder(w, r)
	if !ok {
		return
	}
	var req struct {
		Outcome string `json:"outcome"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	if req.Outcome != "paid" && req.Outcome != "failed" {
		fail(w, http.StatusBadRequest, "bad outcome")
		return
	}
	settled := store.PaymentSucceeded
	if req.Outcome == "failed" {
		settled = store.PaymentCanceled
	}
	if p.Status != store.PaymentPending {
		if p.Status != settled {
			fail(w, http.StatusConflict, "already_settled")
			return
		}
		writeJSON(w, http.StatusOK, h.orderDTO(p))
		return
	}
	// The decision goes into the payment's id; the settlement below reads it back from there.
	id := fmt.Sprintf("dev-%d-%d-%s", p.ID, p.AmountKopecks, req.Outcome)
	if err := h.Store.SetPaymentProviderID(p.ID, id, true); err != nil {
		log.Printf("dev pay: record %d: %v", p.ID, err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	p.ProviderPaymentID = id
	done, err := h.settlePayment(r.Context(), p)
	if err != nil {
		log.Printf("dev pay: settle %d: %v", p.ID, err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, h.orderDTO(done))
}
