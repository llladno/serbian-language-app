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

	"github.com/grisha/serbian-app/server/internal/config"
	"github.com/grisha/serbian-app/server/internal/store"
)

// Paying rubles for the paid levels. Spec: docs/superpowers/specs/2026-10-14-tariffs-design.md.
//
// Everything here is closed to all but the accounts in TARIFFS_EMAILS (paying
// through ЮKassa) and TBANK_EMAILS (paying through Т-Касса); see tariffsEnabled
// and payproviders.go. The screens hide themselves from the features flag in
// the account DTO, and the endpoints answer 404 as if they did not exist.

// pendingWindow is how far back a learner's unfinished payments are still
// re-checked with their payment service when they open the tariffs screen.
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

// levelCooldownEnd is when this learner may buy another level for rubles: a
// month (TARIFF_LEVEL_COOLDOWN_DAYS) after the last one they paid for. held is
// false when nothing holds them back. It limits "Уровень" only; the year and
// levels bought for зёрнышки are not affected.
func (h handlers) levelCooldownEnd(userID string) (end time.Time, held bool, err error) {
	days := h.Config.TariffLevelCooldownDays
	if days <= 0 {
		return time.Time{}, false, nil
	}
	last, ok, err := h.Store.LastLevelPaymentAt(userID)
	if err != nil || !ok {
		return time.Time{}, false, err
	}
	end = last.AddDate(0, 0, days)
	if !end.After(h.Now()) {
		return time.Time{}, false, nil
	}
	return end, true, nil
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
	HasPaidLevel  bool `json:"has_paid_level"`
	Level         struct {
		Price        int64 `json:"price"`
		RegularPrice int64 `json:"regular_price"`
		// AvailableFrom is set while the monthly limit holds the learner back: the
		// moment they may buy the next level for rubles.
		AvailableFrom string `json:"available_from,omitempty"`
		// Bonus is the in-app currency (зёрнышки) paying for a level credits, once.
		Bonus int64 `json:"bonus"`
		Next  *struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"next"`
	} `json:"level"`
	Year struct {
		Price    int64 `json:"price"`
		OldPrice int64 `json:"old_price"`
		// Bonus is the in-app currency paying for the year credits, once.
		Bonus int64  `json:"bonus"`
		Until string `json:"until,omitempty"`
	} `json:"year"`
}

func (h handlers) getTariffs(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	// A learner who just paid and came back before the notification did should
	// see their level open, so ask the payment service about anything still waiting.
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

	cooldownEnd, held, err := h.levelCooldownEnd(ac.UserID)
	if err != nil {
		log.Printf("tariffs: level cooldown: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}

	var out tariffsDTO
	prov := h.providerFor(ac.Summary.Email)
	out.PaymentsReady = prov != nil && prov.Configured()
	out.HasPaidLevel = hasPaid
	out.Level.Price = rubles(levelPriceKopecks(level, hasPaid))
	out.Level.RegularPrice = rubles(level.PriceKopecks)
	out.Level.Bonus = level.BonusCurrency
	out.Year.Bonus = year.BonusCurrency
	if held {
		out.Level.AvailableFrom = cooldownEnd.UTC().Format(time.RFC3339)
	}
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
	// Once the order is paid, what it brought: the level it opened (a "level"
	// order) and the in-app currency credited with it. The thank-you screen
	// reads these; a year's end date comes from GET /api/me/tariffs.
	PhaseTitle string `json:"phase_title,omitempty"`
	Bonus      int64  `json:"bonus,omitempty"`
}

// orderDTO describes one of the buyer's orders. A succeeded one carries what it
// brought; failing to look that up must not hide the order, so those fields are
// simply left out.
func (h handlers) orderDTO(p store.Payment) orderDTO {
	out := orderDTO{ID: p.ID, Tariff: p.TariffCode, Status: p.Status}
	if p.Status != store.PaymentSucceeded {
		return out
	}
	if p.TariffCode == store.TariffLevel && p.GrantedPhase != "" {
		for _, ph := range h.Course().Phases {
			if ph.ID == p.GrantedPhase {
				out.PhaseTitle = ph.Title
				break
			}
		}
	}
	if bonus, err := h.Store.PaymentBonus(p.ID); err != nil {
		log.Printf("orders: bonus of %d: %v", p.ID, err)
	} else {
		out.Bonus = bonus
	}
	return out
}

// createOrder starts a payment: it records it, asks the account's payment
// service for a payment the buyer completes on the service's own page, and
// returns that page's address. Nothing is granted here; that happens in
// settlePayment, once the service says it was paid.
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
	prov := h.providerFor(ac.Summary.Email)
	if prov == nil || !prov.Configured() {
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
		// One level a month: a level paid for recently holds the next one back.
		// The year is never held back, and neither are levels bought for зёрнышки.
		if _, held, err := h.levelCooldownEnd(ac.UserID); err != nil {
			log.Printf("orders: level cooldown: %v", err)
			fail(w, http.StatusInternalServerError, "internal error")
			return
		} else if held {
			fail(w, http.StatusConflict, "level_cooldown")
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
		Provider: prov.Name(), ReceiptContact: ac.Summary.Email,
	}, h.Now())
	if err != nil {
		log.Printf("orders: create: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	created, err := prov.Create(r.Context(), payCreate{
		OrderID:       pid,
		AmountKopecks: amount,
		Description:   description,
		Email:         ac.Summary.Email,
		Tariff:        tariff.Code,
		ReturnURL:     h.Config.AppBaseURL + "/tariffs?order=" + strconv.FormatInt(pid, 10),
		NotifyURL:     h.Config.AppBaseURL + "/api/" + prov.Name() + "/webhook",
	})
	if err != nil || created.PayURL == "" {
		log.Printf("orders: %s create (order %d): %v", prov.Name(), pid, err)
		_ = h.Store.CancelPayment(pid)
		fail(w, http.StatusBadGateway, "payment_failed")
		return
	}
	if err := h.Store.SetPaymentProviderID(pid, created.ProviderPaymentID, created.Test); err != nil {
		log.Printf("orders: link provider id (order %d): %v", pid, err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, orderDTO{ID: pid, Tariff: tariff.Code, Status: store.PaymentPending, ConfirmationURL: created.PayURL})
}

// getOrder is what the page the buyer returns to calls: it asks the payment
// service where the payment stands, settles it, and says. Only the buyer's own
// orders.
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
	writeJSON(w, http.StatusOK, h.orderDTO(p))
}

// settlePending settles every unfinished payment of one learner.
func (h handlers) settlePending(ctx context.Context, userID string) {
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

// settlePayment asks the service that made a payment where it stands and acts
// on the answer: paid opens what was bought, failed closes the order. It is the
// only place that grants anything, and it never believes the caller about the
// outcome. It is safe to run twice at once (a notification and the return page):
// CompletePayment grants once.
func (h handlers) settlePayment(ctx context.Context, p store.Payment) (store.Payment, error) {
	if p.Status != store.PaymentPending || p.ProviderPaymentID == "" {
		return p, nil
	}
	prov := h.providerNamed(p.Provider)
	if prov == nil || !prov.Configured() {
		return p, nil
	}
	st, err := prov.State(ctx, p.ProviderPaymentID)
	if err != nil {
		return p, err
	}
	// The payment the service describes must be the one we asked for. A mismatch
	// is not "not paid yet", it is something wrong, so it is left pending and
	// logged.
	if st.AmountKopecks != p.AmountKopecks {
		return p, errors.New("amount differs from the order")
	}
	if st.OrderRef != 0 && st.OrderRef != p.ID {
		return p, errOrderMismatch
	}
	switch {
	case st.Paid:
		if _, err := h.Store.CompletePayment(p.ID, h.Now()); err != nil {
			return p, err
		}
	case st.Failed:
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
	p, err := h.Store.PaymentByProviderID(config.ProviderYooKassa, n.Object.ID)
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

// tbankWebhook receives Т-Касса's notifications. Unlike ЮKassa's they are signed,
// so one that is not signed with our terminal's password is refused (400). A
// signed one is still only a nudge: the payment id is taken from it and the truth
// is fetched with GetState by settlePayment. The bank wants exactly "OK" with a
// 200 within ten seconds, and otherwise retries for a day and then daily for a
// month, so a failure to ask the bank or to write the grant is answered 500;
// anything we have nothing to do with is answered OK.
func (h handlers) tbankWebhook(w http.ResponseWriter, r *http.Request) {
	ok := func() {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "OK")
	}
	prov, _ := h.providerNamed(config.ProviderTBank).(tbankProvider)
	if !prov.Configured() {
		ok()
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		fail(w, http.StatusBadRequest, "bad body")
		return
	}
	n, err := prov.c.ParseNotification(body)
	if err != nil {
		fail(w, http.StatusBadRequest, "bad notification")
		return
	}
	// Receipt and card-binding notifications arrive here too; they carry no
	// payment status and there is nothing to settle for them.
	if n.PaymentID == "" || n.Status == "" {
		ok()
		return
	}
	p, err := h.Store.PaymentByProviderID(config.ProviderTBank, n.PaymentID)
	if err != nil {
		ok()
		return
	}
	if _, err := h.settlePayment(r.Context(), p); err != nil {
		log.Printf("tbank webhook: settle %d: %v", p.ID, err)
		fail(w, http.StatusInternalServerError, "try again")
		return
	}
	ok()
}
