package api

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/grisha/serbian-app/server/internal/config"
	"github.com/grisha/serbian-app/server/internal/tbank"
)

// A payment service, as the order and settlement code sees it. Т-Касса is the
// only real one (the stand-in cash desk of PAYMENTS_DEV is the other, for a
// developer's machine); the interface keeps the order and settlement code free of
// how a service is called and signed. A service does three things for us: make a
// payment the buyer completes on the service's own page, say where a payment
// stands, and (through a notification) tell us to ask. The store, the grant and
// the screens know nothing about which one was used; the payment row records it
// (payments.provider) so that a notification or the return page is settled with
// the service that made the payment.
type payProvider interface {
	// Name is the payments.provider value.
	Name() string
	Configured() bool
	Create(ctx context.Context, in payCreate) (payCreated, error)
	// State asks the service for the truth about one of its payments.
	State(ctx context.Context, providerPaymentID string) (payState, error)
}

type payCreate struct {
	OrderID       int64
	AmountKopecks int64
	Description   string
	// The receipt goes to Email or, when the buyer gave a phone instead, to Phone.
	Email, Phone string
	Tariff       string
	// ReturnURL is where the buyer lands afterwards, NotifyURL where the service
	// posts about the payment (a service that is told this per payment).
	ReturnURL string
	NotifyURL string
}

type payCreated struct {
	ProviderPaymentID string
	PayURL            string
	Test              bool
}

type payState struct {
	Paid, Failed  bool
	AmountKopecks int64
	// OrderRef is our order id as the service echoed it back: 0 when it did not
	// echo one, -1 when what it echoed is not an order id of ours.
	OrderRef int64
}

// providerNamed is the service a stored payment was made with, nil if unknown.
func (h handlers) providerNamed(name string) payProvider {
	switch name {
	case config.ProviderDev:
		// Only with PAYMENTS_DEV: a stored stand-in payment is inert everywhere else.
		if h.Config.PaymentsDev {
			return devProvider{}
		}
		return nil
	case config.ProviderTBank:
		return tbankProvider{c: &tbank.Client{
			TerminalKey: h.Config.TBankTerminalKey,
			Password:    h.Config.TBankPassword,
			Base:        h.Config.TBankAPIBase,
			Taxation:    h.Config.TBankTaxation,
		}}
	}
	return nil
}

// providerFor is the service the account with this email pays through, nil for
// an account that has none.
func (h handlers) providerFor(email string) payProvider {
	return h.providerNamed(h.Config.PaymentProviderFor(email))
}

// ---- Т-Касса ----

type tbankProvider struct{ c *tbank.Client }

func (tbankProvider) Name() string       { return config.ProviderTBank }
func (p tbankProvider) Configured() bool { return p.c.Configured() }

// tbankOrderID is the OrderId sent to the bank: our order id, then the time. The
// bank wants a fresh one per payment, and the same terminal can serve more than
// one environment (a local copy against the test terminal, say), each counting
// its own order ids from 1.
func tbankOrderID(orderID int64, now time.Time) string {
	return fmt.Sprintf("%d-%d", orderID, now.Unix())
}

func (p tbankProvider) Create(ctx context.Context, in payCreate) (payCreated, error) {
	tp, err := p.c.Init(ctx, tbank.InitInput{
		OrderID:         tbankOrderID(in.OrderID, time.Now()),
		AmountKopecks:   in.AmountKopecks,
		Description:     in.Description,
		Email:           in.Email,
		Phone:           in.Phone,
		SuccessURL:      in.ReturnURL,
		FailURL:         in.ReturnURL,
		NotificationURL: in.NotifyURL,
	})
	if err != nil {
		return payCreated{}, err
	}
	return payCreated{ProviderPaymentID: tp.PaymentID, PayURL: tp.URL, Test: p.c.Test()}, nil
}

func (p tbankProvider) State(ctx context.Context, id string) (payState, error) {
	tp, err := p.c.GetState(ctx, id)
	if err != nil {
		return payState{}, err
	}
	return payState{
		Paid:          tp.Paid(),
		Failed:        tp.Failed(),
		AmountKopecks: tp.AmountKopecks,
		OrderRef:      orderRef(strings.SplitN(tp.OrderID, "-", 2)[0]),
	}, nil
}

// orderRef reads an order id echoed by a service: "" is 0 (nothing echoed),
// anything that is not a positive number is -1.
func orderRef(s string) int64 {
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return -1
	}
	return n
}

var errOrderMismatch = errors.New("payment belongs to another order")
