package api

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	netmail "net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/grisha/serbian-app/server/internal/config"
	"github.com/grisha/serbian-app/server/internal/store"
)

// Paying rubles for the paid levels. Spec: docs/superpowers/specs/2026-10-14-tariffs-design.md.
//
// Everything here is closed to all but the accounts in TARIFFS_EMAILS or
// TBANK_EMAILS (or everyone with a verified email, with TARIFFS_FOR_ALL), paying
// through Т-Касса; see tariffsEnabled and payproviders.go. The screens hide themselves from the features flag in
// the account DTO, and the endpoints answer 404 as if they did not exist.

// pendingWindow is how far back a learner's unfinished payments are still
// re-checked with their payment service when they open the tariffs screen.
const pendingWindow = 48 * time.Hour

// tariffsEnabled reports whether this account may use the tariffs: its email
// must be on the list AND verified, so that typing someone else's address into
// the sign-up form unlocks nothing. An account with no email at all (Telegram
// only) has nothing to verify: it gets the tariffs when they are open to all
// and names the address for its receipt when it pays (createOrder).
func (h handlers) tariffsEnabled(s userSummary) bool {
	if s.Email == "" {
		return h.Config.PaymentProviderFor("") != ""
	}
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

// receiptContact is where the receipt for this order goes: the account's own
// verified email or, for an account that has none (Telegram only), the email or
// phone the buyer typed. The bank takes either; ok is false when what was typed
// is neither.
func receiptContact(s userSummary, typed string) (email, phone string, ok bool) {
	if s.Email != "" {
		return s.Email, "", true
	}
	typed = strings.TrimSpace(typed)
	if addr, err := netmail.ParseAddress(typed); err == nil && addr.Address == typed &&
		len(typed) <= 64 && strings.Contains(typed[strings.LastIndex(typed, "@"):], ".") {
		return strings.ToLower(typed), "", true
	}
	if p, ok := normalizePhone(typed); ok {
		return "", p, true
	}
	return "", "", false
}

// normalizePhone turns what a buyer typed into the "+79031234567" form the bank
// wants. Without a plus only a Russian number is understood (8 903…, 7 903…, or
// the ten digits 903…); with one, any international number of 8 to 15 digits.
func normalizePhone(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	plus := strings.HasPrefix(raw, "+")
	raw = strings.TrimPrefix(raw, "+")
	var d strings.Builder
	for _, r := range raw {
		switch {
		case r >= '0' && r <= '9':
			d.WriteRune(r)
		case strings.ContainsRune(" -().\u00a0", r):
		default:
			return "", false
		}
	}
	n := d.String()
	switch {
	case plus && len(n) >= 8 && len(n) <= 15 && n[0] != '0':
		return "+" + n, true
	case !plus && len(n) == 11 && (n[0] == '8' || n[0] == '7'):
		return "+7" + n[1:], true
	case !plus && len(n) == 10 && n[0] == '9':
		return "+7" + n, true
	}
	return "", false
}

// levelChoiceDTO is a level a paid "Уровень" can open: closed for this learner and
// on sale. After names the level before it in the course when that one is also still
// to be opened, so the screen can say "better after ...": the levels build on each other.
type levelChoiceDTO struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Lessons int    `json:"lessons"`
	After   string `json:"after,omitempty"`
}

// levelChoices are the levels a paid level can open, in course order: locked for
// this learner and for sale (at least one of its lessons is written; the same rule
// the course screen uses to tell a price tag from «скоро»). A running year opens
// everything, so it leaves none.
func (h handlers) levelChoices(us *store.UserStore) ([]levelChoiceDTO, error) {
	locked, _, err := h.phaseAccess(us)
	if err != nil {
		return nil, err
	}
	c := h.Course()
	var out []levelChoiceDTO
	at := map[string]int{} // phase id -> index in out
	for _, p := range c.Phases {
		if !locked[p.ID] {
			continue
		}
		written := 0
		for _, lid := range p.Lessons {
			if l := c.Lessons[lid]; l != nil && !l.Planned {
				written++
			}
		}
		if written == 0 {
			continue
		}
		at[p.ID] = len(out)
		out = append(out, levelChoiceDTO{ID: p.ID, Title: p.Title, Lessons: written})
	}
	for i := 1; i < len(c.Phases); i++ {
		if j, ok := at[c.Phases[i].ID]; ok {
			if _, prev := at[c.Phases[i-1].ID]; prev {
				out[j].After = c.Phases[i-1].Title
			}
		}
	}
	return out, nil
}

// nextLockedPhase is the first level a paid level could open (see levelChoices).
func (h handlers) nextLockedPhase(us *store.UserStore) (id, title string, ok bool, err error) {
	choices, err := h.levelChoices(us)
	if err != nil || len(choices) == 0 {
		return "", "", false, err
	}
	return choices[0].ID, choices[0].Title, true, nil
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

// tariff reads a tariff after making any announced price change whose moment has come, so every price the API quotes
// or charges is the live one at this instant (migration 041).
func (h handlers) tariff(code string) (store.Tariff, error) {
	if err := h.Store.ApplyDuePriceChanges(h.Now()); err != nil {
		log.Printf("tariffs: %v", err)
	}
	return h.Store.Tariff(code)
}

// nextLevelPriceKopecks is what the next level will cost this learner once the announced change is made: the first
// purchase price once, the regular price after, both taken from the scheduled prices.
func nextLevelPriceKopecks(t store.Tariff, hasPaid bool) int64 {
	if !hasPaid && t.NextFirstPriceKopecks > 0 {
		return t.NextFirstPriceKopecks
	}
	return t.NextPriceKopecks
}

// priceChangeDTO is a price change that has been announced and has not happened yet. Now is the server's clock, so
// the countdown does not depend on the learner's own (a phone a few minutes off would end the timer early). The prices
// are in rubles and are this learner's: LevelPrice follows the same first-purchase rule as the live one.
type priceChangeDTO struct {
	At         string `json:"at"`
	Now        string `json:"now"`
	LevelPrice int64  `json:"level_price"`
	YearPrice  int64  `json:"year_price"`
	// Preview marks the owner's own account (Config.SeesPriceChangeEarly): it is shown the announcement before its
	// date, and the screens show it whatever else is true of the account (a tariff, a closed cross, today's limit).
	Preview bool `json:"preview,omitempty"`
}

// priceChange is the announced change of the two tariffs, or nil when none is on: nothing scheduled, not announced
// yet, or already made. It follows the level tariff's moments; the year is changed together with it. early is the
// owner's own account: it gets the announcement as soon as the change is scheduled, whatever the announcement date.
func priceChange(level, year store.Tariff, hasPaid bool, now time.Time, early bool) *priceChangeDTO {
	if level.NextPriceKopecks <= 0 || level.ChangeAt.IsZero() || !now.Before(level.ChangeAt) {
		return nil
	}
	if !early && (level.AnnounceAt.IsZero() || now.Before(level.AnnounceAt)) {
		return nil
	}
	yearNext := year.NextPriceKopecks
	if yearNext <= 0 {
		yearNext = year.PriceKopecks
	}
	return &priceChangeDTO{
		At:         level.ChangeAt.UTC().Format(time.RFC3339),
		Now:        now.UTC().Format(time.RFC3339),
		LevelPrice: rubles(nextLevelPriceKopecks(level, hasPaid)),
		YearPrice:  rubles(yearNext),
		Preview:    early,
	}
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
		// Choices are the levels a paid level can open, and Credits the learner's
		// paid level orders that have not been spent on one yet (oldest first).
		Choices []levelChoiceDTO `json:"choices"`
		Credits []int64          `json:"credits"`
	} `json:"level"`
	Year struct {
		Price    int64 `json:"price"`
		OldPrice int64 `json:"old_price"`
		// Bonus is the in-app currency paying for the year credits, once.
		Bonus int64  `json:"bonus"`
		Until string `json:"until,omitempty"`
	} `json:"year"`
	// PriceChange is set from the announcement until the change is made (see priceChange).
	PriceChange *priceChangeDTO `json:"price_change,omitempty"`
	// Discount is this learner's own discount, while they have one (see personalDiscount).
	Discount *discountDTO `json:"discount,omitempty"`
}

// discountDTO is a personal discount as the tariffs screen shows it. The prices are in rubles and are what an order
// made now would be charged; the list prices in the quote stay what everybody sees. Now is the server's clock, as in
// priceChangeDTO, so the countdown does not lean on the learner's own.
type discountDTO struct {
	Percent    int    `json:"percent"`
	Until      string `json:"until"`
	Now        string `json:"now"`
	LevelPrice int64  `json:"level_price"`
	YearPrice  int64  `json:"year_price"`
}

// personalDiscount is the discount this learner can use now, or nil. A failure to read it is logged and treated as no
// discount: the learner is then charged the list price they see, never a price they were not shown.
func (h handlers) personalDiscount(userID string) *store.PersonalDiscount {
	d, ok, err := h.Store.ActiveDiscount(userID, h.Now())
	if err != nil {
		log.Printf("tariffs: personal discount: %v", err)
		return nil
	}
	if !ok {
		return nil
	}
	return &d
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

	level, err := h.tariff(store.TariffLevel)
	if err != nil {
		log.Printf("tariffs: level: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	year, err := h.tariff(store.TariffYear)
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
	choices, err := h.levelChoices(us)
	if err != nil {
		log.Printf("tariffs: level choices: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	credits, err := h.Store.UnclaimedLevelPayments(ac.UserID)
	if err != nil {
		log.Printf("tariffs: level credits: %v", err)
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
	out.Level.Choices = choices
	if out.Level.Choices == nil {
		out.Level.Choices = []levelChoiceDTO{}
	}
	out.Level.Credits = []int64{}
	for _, p := range credits {
		out.Level.Credits = append(out.Level.Credits, p.ID)
	}
	if len(choices) > 0 {
		out.Level.Next = &struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}{ID: choices[0].ID, Title: choices[0].Title}
	}
	out.Year.Price = rubles(year.PriceKopecks)
	out.Year.OldPrice = rubles(year.OldPriceKopecks)
	if running {
		out.Year.Until = until.UTC().Format(time.RFC3339)
	}
	out.PriceChange = priceChange(level, year, hasPaid, h.Now(), h.Config.SeesPriceChangeEarly(ac.Summary.Email))
	if d := h.personalDiscount(ac.UserID); d != nil {
		out.Discount = &discountDTO{
			Percent:    d.Percent,
			Until:      d.ExpiresAt.UTC().Format(time.RFC3339),
			Now:        h.Now().UTC().Format(time.RFC3339),
			LevelPrice: rubles(store.DiscountedKopecks(levelPriceKopecks(level, hasPaid), d.Percent)),
			YearPrice:  rubles(store.DiscountedKopecks(year.PriceKopecks, d.Percent)),
		}
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
	// Choose is set on a paid level order whose level has not been chosen yet: the
	// screen asks the learner which one to open (POST /api/me/orders/{id}/level).
	Choose bool `json:"choose,omitempty"`
}

// orderDTO describes one of the buyer's orders. A succeeded one carries what it
// brought; failing to look that up must not hide the order, so those fields are
// simply left out.
func (h handlers) orderDTO(p store.Payment) orderDTO {
	out := orderDTO{ID: p.ID, Tariff: p.TariffCode, Status: p.Status}
	if p.Status != store.PaymentSucceeded {
		return out
	}
	if p.TariffCode == store.TariffLevel && p.GrantedPhase == "" {
		out.Choose = true
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

// orderRefused answers a refused order and writes down which refusal it was, so the admin's «Воронка тарифов»
// can say how many buyers were turned away at the pay button and why (an order that was never made leaves no
// payment row). Best-effort, like the other events.
func (h handlers) orderRefused(w http.ResponseWriter, userID, tariff string, status int, code string) {
	detail := strings.ReplaceAll(code, " ", "_") + ":" + tariff
	if _, err := h.Store.RecordTariffEventDetail(userID, store.TariffEventError, store.TariffSourceCheckout, detail, h.Now()); err != nil {
		log.Printf("tariff events: %v", err)
	}
	fail(w, status, code)
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
		// Contact is the email or phone for the receipt, asked of an account that
		// has no email of its own; an account that has one is never sent anywhere else.
		Contact string `json:"contact"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	prov := h.providerFor(ac.Summary.Email)
	if prov == nil || !prov.Configured() {
		h.orderRefused(w, ac.UserID, req.Tariff, http.StatusServiceUnavailable, "payments_unavailable")
		return
	}
	tariff, err := h.tariff(req.Tariff)
	if err != nil || !tariff.Active {
		h.orderRefused(w, ac.UserID, req.Tariff, http.StatusNotFound, "unknown tariff")
		return
	}

	us := h.Store.User(ac.UserID)
	var amount int64
	var description string
	switch tariff.Code {
	case store.TariffLevel:
		choices, err := h.levelChoices(us)
		if err != nil {
			log.Printf("orders: level choices: %v", err)
			h.orderRefused(w, ac.UserID, req.Tariff, http.StatusInternalServerError, "internal error")
			return
		}
		if len(choices) == 0 {
			// Every level on sale is open already: charging for one would be
			// taking money for nothing.
			h.orderRefused(w, ac.UserID, req.Tariff, http.StatusConflict, "nothing_to_buy")
			return
		}
		// A level that was paid for and not chosen yet comes first: two credits at
		// once would only be a puzzle for the learner.
		if credits, err := h.Store.UnclaimedLevelPayments(ac.UserID); err != nil {
			log.Printf("orders: level credits: %v", err)
			h.orderRefused(w, ac.UserID, req.Tariff, http.StatusInternalServerError, "internal error")
			return
		} else if len(credits) > 0 {
			h.orderRefused(w, ac.UserID, req.Tariff, http.StatusConflict, "level_pending_choice")
			return
		}
		// One level a month: a level paid for recently holds the next one back.
		// The year is never held back, and neither are levels bought for зёрнышки.
		if _, held, err := h.levelCooldownEnd(ac.UserID); err != nil {
			log.Printf("orders: level cooldown: %v", err)
			h.orderRefused(w, ac.UserID, req.Tariff, http.StatusInternalServerError, "internal error")
			return
		} else if held {
			h.orderRefused(w, ac.UserID, req.Tariff, http.StatusConflict, "level_cooldown")
			return
		}
		hasPaid, err := h.Store.HasPaidLevel(ac.UserID)
		if err != nil {
			log.Printf("orders: paid level: %v", err)
			h.orderRefused(w, ac.UserID, req.Tariff, http.StatusInternalServerError, "internal error")
			return
		}
		// No level is named: the learner chooses after paying (chooseLevel).
		amount, description = levelPriceKopecks(tariff, hasPaid), "Уровень курса сербского языка (на выбор)"
	case store.TariffYear:
		amount, description = tariff.PriceKopecks, tariff.Title
	default:
		h.orderRefused(w, ac.UserID, req.Tariff, http.StatusNotFound, "unknown tariff")
		return
	}

	// The receipt goes to the account's own verified email, or to the email or
	// phone a Telegram-only buyer typed in.
	receiptMail, receiptPhone, ok := receiptContact(ac.Summary, req.Contact)
	if !ok {
		h.orderRefused(w, ac.UserID, req.Tariff, http.StatusBadRequest, "receipt_contact_invalid")
		return
	}
	receipt := receiptMail
	if receipt == "" {
		receipt = receiptPhone
	}
	// A personal discount the learner can use is taken off whatever the order costs; the same function prices the
	// number the tariffs screen showed, so the button, the order and the bank agree.
	var discountID string
	if d := h.personalDiscount(ac.UserID); d != nil {
		amount = store.DiscountedKopecks(amount, d.Percent)
		discountID = d.ID
	}
	pid, err := h.Store.CreatePayment(store.Payment{
		UserID: ac.UserID, TariffCode: tariff.Code, AmountKopecks: amount,
		Provider: prov.Name(), ReceiptContact: receipt, DiscountID: discountID,
	}, h.Now())
	if err != nil {
		log.Printf("orders: create: %v", err)
		h.orderRefused(w, ac.UserID, req.Tariff, http.StatusInternalServerError, "internal error")
		return
	}
	created, err := prov.Create(r.Context(), payCreate{
		OrderID:       pid,
		AmountKopecks: amount,
		Description:   description,
		Email:         receiptMail,
		Phone:         receiptPhone,
		Tariff:        tariff.Code,
		ReturnURL:     h.Config.AppBaseURL + "/tariffs?order=" + strconv.FormatInt(pid, 10),
		NotifyURL:     h.Config.AppBaseURL + "/api/" + prov.Name() + "/webhook",
	})
	if err != nil || created.PayURL == "" {
		log.Printf("orders: %s create (order %d): %v", prov.Name(), pid, err)
		_ = h.Store.CancelPayment(pid)
		h.orderRefused(w, ac.UserID, req.Tariff, http.StatusBadGateway, "payment_failed")
		return
	}
	if err := h.Store.SetPaymentProviderID(pid, created.ProviderPaymentID, created.Test); err != nil {
		log.Printf("orders: link provider id (order %d): %v", pid, err)
		h.orderRefused(w, ac.UserID, req.Tariff, http.StatusInternalServerError, "internal error")
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

// chooseLevel spends a paid level on the level the learner picks. It answers with
// the order as it stands afterwards (the level named, the bonus). Asking again for
// the same level is not an error; another level on a spent order is refused.
func (h handlers) chooseLevel(w http.ResponseWriter, r *http.Request) {
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
	var req struct {
		Phase string `json:"phase"`
	}
	if err := decode(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	if p.TariffCode != store.TariffLevel || p.Status != store.PaymentSucceeded {
		fail(w, http.StatusConflict, "not_claimable")
		return
	}
	if p.GrantedPhase != "" {
		if p.GrantedPhase != req.Phase {
			fail(w, http.StatusConflict, "already_chosen")
			return
		}
		writeJSON(w, http.StatusOK, h.orderDTO(p))
		return
	}
	choices, err := h.levelChoices(h.Store.User(ac.UserID))
	if err != nil {
		log.Printf("orders: choose: choices: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	allowed := false
	for _, c := range choices {
		if c.ID == req.Phase {
			allowed = true
			break
		}
	}
	if !allowed {
		fail(w, http.StatusBadRequest, "bad_phase")
		return
	}
	switch err := h.Store.ClaimLevel(ac.UserID, id, req.Phase, h.Now()); {
	case errors.Is(err, store.ErrNotClaimable):
		fail(w, http.StatusConflict, "not_claimable")
		return
	case errors.Is(err, store.ErrCreditSpent):
		fail(w, http.StatusConflict, "already_chosen")
		return
	case err != nil:
		log.Printf("orders: choose %d: %v", id, err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	done, err := h.Store.PaymentByID(id)
	if err != nil {
		log.Printf("orders: choose: reload %d: %v", id, err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, h.orderDTO(done))
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
		granted, err := h.Store.CompletePayment(p.ID, h.Now())
		if err != nil {
			return p, err
		}
		if granted {
			// Only the call that did the granting tells the operator; the notification and the return page
			// arriving together would otherwise send it twice.
			h.notifyPayment(p)
		}
	case st.Failed:
		if err := h.Store.CancelPayment(p.ID); err != nil {
			return p, err
		}
	}
	return h.Store.PaymentByID(p.ID)
}

// tbankWebhook receives Т-Касса's notifications. They are signed, so one that
// is not signed with our terminal's password is refused (400). A
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

// trackTariffEvent handles POST /api/me/tariff-events {kind, source, detail}: the places that sell the tariffs report each
// showing and each press on the way to them, and the admin's «Воронка тарифов» counts the rows. Best-effort
// analytics: a failure to write is logged and still answered 204, never shown to the learner. Only an unknown
// kind, source or body is refused (400). Closed, like the rest of the tariffs, to accounts that do not see them.
func (h handlers) trackTariffEvent(w http.ResponseWriter, r *http.Request) {
	ac, ok := authFrom(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "no session")
		return
	}
	var req struct {
		Kind   string `json:"kind"`
		Source string `json:"source"`
		// Detail is a few words on what it was about ("phase:2", "year"); cleaned before it is stored.
		Detail string `json:"detail"`
	}
	if err := decode(r, &req); err != nil || !store.ValidTariffEvent(req.Kind, req.Source) {
		fail(w, http.StatusBadRequest, "bad request body")
		return
	}
	if _, err := h.Store.RecordTariffEventDetail(ac.UserID, req.Kind, req.Source, req.Detail, h.Now()); err != nil {
		log.Printf("tariff events: %v", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

// publicPricesDTO is what the logged-out landing needs to quote: the live prices in rubles, and the announced change
// while there is one. The landing is a static page, so it asks for them instead of having them typed in, and its prices
// change by themselves at the moment of the change (migration 041).
type publicPricesDTO struct {
	Level struct {
		// First is the price of a first purchase, Regular of every later one.
		First   int64 `json:"first"`
		Regular int64 `json:"regular"`
	} `json:"level"`
	Year   int64 `json:"year"`
	Change *struct {
		At           string `json:"at"`
		LevelFirst   int64  `json:"level_first"`
		LevelRegular int64  `json:"level_regular"`
		Year         int64  `json:"year"`
	} `json:"change,omitempty"`
}

// getPublicPrices handles GET /api/prices: no session, no account, nothing personal. It is cached for half a minute at
// most, so the quote cannot outlive the change by more than that.
func (h handlers) getPublicPrices(w http.ResponseWriter, r *http.Request) {
	level, err := h.tariff(store.TariffLevel)
	if err != nil {
		log.Printf("prices: level: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	year, err := h.tariff(store.TariffYear)
	if err != nil {
		log.Printf("prices: year: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	var out publicPricesDTO
	out.Level.First = rubles(levelPriceKopecks(level, false))
	out.Level.Regular = rubles(level.PriceKopecks)
	out.Year = rubles(year.PriceKopecks)
	now := h.Now()
	if level.NextPriceKopecks > 0 && !level.AnnounceAt.IsZero() && !now.Before(level.AnnounceAt) && now.Before(level.ChangeAt) {
		yearNext := year.NextPriceKopecks
		if yearNext <= 0 {
			yearNext = year.PriceKopecks
		}
		out.Change = &struct {
			At           string `json:"at"`
			LevelFirst   int64  `json:"level_first"`
			LevelRegular int64  `json:"level_regular"`
			Year         int64  `json:"year"`
		}{
			At:           level.ChangeAt.UTC().Format(time.RFC3339),
			LevelFirst:   rubles(nextLevelPriceKopecks(level, false)),
			LevelRegular: rubles(level.NextPriceKopecks),
			Year:         rubles(yearNext),
		}
	}
	w.Header().Set("Cache-Control", "public, max-age=30")
	writeJSON(w, http.StatusOK, out)
}
