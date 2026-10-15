package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/yookassa"
)

// Paying rubles for the paid levels. Spec: docs/superpowers/specs/2026-10-14-tariffs-design.md.
//
// Everything here is closed to all but the accounts in TARIFFS_EMAILS (see
// tariffsEnabled): the screens hide themselves from the features flag in the
// account DTO, and the endpoints answer 404 as if they did not exist.

// providerYooKassa is the payments.provider value for ЮKassa.
const providerYooKassa = "yookassa"

// pendingWindow is how far back a learner's unfinished payments are still
// re-checked with ЮKassa when they open the tariffs screen.
const pendingWindow = 48 * time.Hour

// tariffsEnabled reports whether this account may use the tariffs: its email
// must be on the list AND verified, so that typing someone else's address into
// the sign-up form unlocks nothing.
func (h handlers) tariffsEnabled(s userSummary) bool {
	return s.EmailVerified && h.Config.TariffsAllowed(s.Email)
}

func (h handlers) requireTariffs(next http.HandlerFunc) http.HandlerFunc {
	return h.requireSession(func(w http.ResponseWriter, r *http.Request) {
		ac, ok := authFrom(r)
		if !ok || !h.tariffsEnabled(ac.Summary) {
			fail(w, http.StatusNotFound, "not found")
			return
		}
		next(w, r)
	})
}

func (h handlers) yookassa() *yookassa.Client {
	return &yookassa.Client{
		ShopID:    h.Config.YooKassaShopID,
		SecretKey: h.Config.YooKassaSecretKey,
		Base:      h.Config.YooKassaAPIBase,
		TaxSystem: h.Config.YooKassaTaxSystem,
	}
}

// nextLockedPhase is the level "Уровень" would open: the first level in course
// order that is locked for this learner and for sale (at least one of its lessons
// is written; the same rule the course screen uses to tell a price tag from
// «скоро»).
func (h handlers) nextLockedPhase(us *store.UserStore) (id, title string, ok bool, err error) {
	locked, _, err := h.phaseAccess(us)
	if err != nil {
		return "", "", false, err
	}
	c := h.Course()
	for _, p := range c.Phases {
		if !locked[p.ID] {
			continue
		}
		for _, lid := range p.Lessons {
			if l := c.Lessons[lid]; l != nil && !l.Planned {
				return p.ID, p.Title, true, nil
			}
		}
	}
	return "", "", false, nil
}

// levelPriceKopecks is what the next level costs this learner: the first
// purchase price once, the regular price after.
func levelPriceKopecks(t store.Tariff, hasPaid bool) int64 {
	if !hasPaid && t.FirstPriceKopecks > 0 {
		return t.FirstPriceKopecks
	}
	return t.PriceKopecks
}

// rubles turns kopecks into the whole rubles the screens show. Prices are set in
// whole rubles; a stray kopeck is dropped from the display, never from the charge.
func rubles(kopecks int64) int64 { return kopecks / 100 }

type tariffsDTO struct {
	// PaymentsReady is false until the shop's credentials are set: the screens
	// still show, the buttons say payment is not connected yet.
	PaymentsReady bool `json:"payments_ready"`
	// TestMode: the shop is ЮKassa's test shop, no money moves.
	TestMode     bool `json:"test_mode"`
	HasPaidLevel bool `json:"has_paid_level"`
	Level        struct {
		Price        int64 `json:"price"`
		RegularPrice int64 `json:"regular_price"`
		Next         *struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"next"`
	} `json:"level"`
	Year struct {
		Price    int64  `json:"price"`
		OldPrice int64  `json:"old_price"`
		Until    string `json:"until,omitempty"`
	} `json:"year"`
}

func (h handlers) getTariffs(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	// A learner who just paid and came back before the notification did should
	// see their level open, so ask ЮKassa about anything still waiting.
	h.settlePending(r.Context(), ac.UserID)

	level, err := h.Store.Tariff(store.TariffLevel)
	if err != nil {
		log.Printf("tariffs: level: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	year, err := h.Store.Tariff(store.TariffYear)
	if err != nil {
		log.Printf("tariffs: year: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	hasPaid, err := h.Store.HasPaidLevel(ac.UserID)
	if err != nil {
		log.Printf("tariffs: paid level: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	us := h.Store.User(ac.UserID)
	id, title, found, err := h.nextLockedPhase(us)
	if err != nil {
		log.Printf("tariffs: next level: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	until, running, err := us.AllAccessUntil(h.Now())
	if err != nil {
		log.Printf("tariffs: year end: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}

	var out tariffsDTO
	out.PaymentsReady = h.Config.YooKassaConfigured()
	out.TestMode = h.Config.YooKassaTestMode
	out.HasPaidLevel = hasPaid
	out.Level.Price = rubles(levelPriceKopecks(level, hasPaid))
	out.Level.RegularPrice = rubles(level.PriceKopecks)
	if found {
		out.Level.Next = &struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}{ID: id, Title: title}
	}
	out.Year.Price = rubles(year.PriceKopecks)
	out.Year.OldPrice = rubles(year.OldPriceKopecks)
	if running {
		out.Year.Until = until.UTC().Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, out)
}

type orderDTO struct {
	ID              int64  `json:"id"`
	Tariff          string `json:"tariff"`
	Status          string `json:"status"`
	ConfirmationURL string `json:"confirmation_url,omitempty"`
}

func orderToDTO(p store.Payment) orderDTO {
	return orderDTO{ID: p.ID, Tariff: p.TariffCode, Status: p.Status}
}

// createOrder starts a payment: it records it, asks ЮKassa for a payment the
// buyer confirms on ЮKassa's own page, and returns that page's address. Nothing
// is granted here; that happens in settlePayment, once ЮKassa says it was paid.
func (h handlers) createOrder(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	var req struct {
		Tariff string `json:"tariff"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	if !h.Config.YooKassaConfigured() {
		fail(w, http.StatusServiceUnavailable, "payments_unavailable")
		return
	}
	tariff, err := h.Store.Tariff(req.Tariff)
	if err != nil || !tariff.Active {
		fail(w, http.StatusNotFound, "unknown tariff")
		return
	}

	us := h.Store.User(ac.UserID)
	var amount int64
	var phase, description string
	switch tariff.Code {
	case store.TariffLevel:
		id, title, found, err := h.nextLockedPhase(us)
		if err != nil {
			log.Printf("orders: next level: %v", err)
			fail(w, http.StatusInternalServerError, "internal error")
			return
		}
		if !found {
			// Every level on sale is open already: charging for one would be
			// taking money for nothing.
			fail(w, http.StatusConflict, "nothing_to_buy")
			return
		}
		hasPaid, err := h.Store.HasPaidLevel(ac.UserID)
		if err != nil {
			log.Printf("orders: paid level: %v", err)
			fail(w, http.StatusInternalServerError, "internal error")
			return
		}
		amount, phase, description = levelPriceKopecks(tariff, hasPaid), id, title
	case store.TariffYear:
		amount, description = tariff.PriceKopecks, tariff.Title
	default:
		fail(w, http.StatusNotFound, "unknown tariff")
		return
	}

	// The receipt is mailed to the account's own address; the tariffs are only
	// open to accounts that have a verified one.
	pid, err := h.Store.CreatePayment(store.Payment{
		UserID: ac.UserID, TariffCode: tariff.Code, GrantedPhase: phase, AmountKopecks: amount,
		Provider: providerYooKassa, ReceiptContact: ac.Summary.Email,
	}, h.Now())
	if err != nil {
		log.Printf("orders: create: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	yp, err := h.yookassa().Create(r.Context(), yookassa.CreateInput{
		AmountKopecks:  amount,
		Description:    description,
		ReturnURL:      h.Config.AppBaseURL + "/tariffs?order=" + strconv.FormatInt(pid, 10),
		Email:          ac.Summary.Email,
		Metadata:       map[string]string{"order_id": strconv.FormatInt(pid, 10), "tariff": tariff.Code},
		IdempotenceKey: "order-" + strconv.FormatInt(pid, 10),
	})
	if err != nil || yp.ConfirmationURL == "" {
		log.Printf("orders: yookassa create (order %d): %v", pid, err)
		_ = h.Store.CancelPayment(pid)
		fail(w, http.StatusBadGateway, "payment_failed")
		return
	}
	if err := h.Store.SetPaymentProviderID(pid, yp.ID, yp.Test); err != nil {
		log.Printf("orders: link provider id (order %d): %v", pid, err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, orderDTO{ID: pid, Tariff: tariff.Code, Status: store.PaymentPending, ConfirmationURL: yp.ConfirmationURL})
}

// getOrder is what the page the buyer returns to calls: it asks ЮKassa where the
// payment stands, settles it, and says. Only the buyer's own orders.
func (h handlers) getOrder(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, http.StatusNotFound, "not found")
		return
	}
	p, err := h.Store.PaymentByID(id)
	if err != nil || p.UserID != ac.UserID {
		fail(w, http.StatusNotFound, "not found")
		return
	}
	if settled, err := h.settlePayment(r.Context(), p); err != nil {
		log.Printf("orders: settle %d: %v", id, err)
	} else {
		p = settled
	}
	writeJSON(w, http.StatusOK, orderToDTO(p))
}

// settlePending settles every unfinished payment of one learner.
func (h handlers) settlePending(ctx context.Context, userID string) {
	if !h.Config.YooKassaConfigured() {
		return
	}
	pending, err := h.Store.PendingPayments(userID, h.Now().Add(-pendingWindow))
	if err != nil {
		log.Printf("tariffs: pending payments: %v", err)
		return
	}
	for _, p := range pending {
		if _, err := h.settlePayment(ctx, p); err != nil {
			log.Printf("tariffs: settle %d: %v", p.ID, err)
		}
	}
}

// settlePayment asks ЮKassa where a payment stands and acts on the answer: paid
// opens what was bought, canceled closes the order. It is the only place that
// grants anything, and it never believes the caller about the outcome. It is safe
// to run twice at once (a notification and the return page): CompletePayment
// grants once.
func (h handlers) settlePayment(ctx context.Context, p store.Payment) (store.Payment, error) {
	if p.Status != store.PaymentPending || p.ProviderPaymentID == "" || !h.Config.YooKassaConfigured() {
		return p, nil
	}
	yp, err := h.yookassa().Get(ctx, p.ProviderPaymentID)
	if err != nil {
		return p, err
	}
	// The payment ЮKassa describes must be the one we asked for. A mismatch is
	// not "not paid yet", it is something wrong, so it is left pending and logged.
	if yp.AmountKopecks != p.AmountKopecks {
		return p, errors.New("amount differs from the order")
	}
	if oid := yp.Metadata["order_id"]; oid != "" && oid != strconv.FormatInt(p.ID, 10) {
		return p, errors.New("payment belongs to another order")
	}
	switch {
	case yp.Status == "succeeded" && yp.Paid:
		if _, err := h.Store.CompletePayment(p.ID, h.Now()); err != nil {
			return p, err
		}
	case yp.Status == "canceled":
		if err := h.Store.CancelPayment(p.ID); err != nil {
			return p, err
		}
	}
	return h.Store.PaymentByID(p.ID)
}

// yookassaWebhook receives ЮKassa's payment notifications. They are not signed,
// so the body is used for one thing only: the payment id. The truth about that
// payment is fetched from ЮKassa by settlePayment. An answer other than 200 makes
// ЮKassa retry for a day, so a failure to ask ЮKassa or to write the grant is
// answered 500; anything we simply have nothing to do with is answered 200.
func (h handlers) yookassaWebhook(w http.ResponseWriter, r *http.Request) {
	if !h.Config.YooKassaConfigured() {
		w.WriteHeader(http.StatusOK)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		fail(w, http.StatusBadRequest, "bad body")
		return
	}
	var n struct {
		Event  string `json:"event"`
		Object struct {
			ID string `json:"id"`
		} `json:"object"`
	}
	if err := json.Unmarshal(body, &n); err != nil || n.Object.ID == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if n.Event != "payment.succeeded" && n.Event != "payment.canceled" {
		w.WriteHeader(http.StatusOK)
		return
	}
	p, err := h.Store.PaymentByProviderID(providerYooKassa, n.Object.ID)
	if err != nil {
		// Not one of ours (another integration on the same shop, or a payment
		// from before this one existed).
		w.WriteHeader(http.StatusOK)
		return
	}
	if _, err := h.settlePayment(r.Context(), p); err != nil {
		log.Printf("yookassa webhook: settle %d: %v", p.ID, err)
		fail(w, http.StatusInternalServerError, "try again")
		return
	}
	w.WriteHeader(http.StatusOK)
}
