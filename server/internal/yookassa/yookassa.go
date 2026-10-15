// Package yookassa is the small slice of the ЮKassa payment API the app needs:
// create a payment that the buyer confirms on ЮKassa's own page, and read one
// back. Docs: https://yookassa.ru/developers/payment-acceptance/getting-started/quick-start
//
// Notifications from ЮKassa are not signed, so nothing here trusts one: callers
// take the payment id from a notification and ask Get for the truth.
package yookassa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DefaultBase is the production API. A test shop uses the same host with its
// own shop id and secret key, so there is nothing to switch but the keys.
const DefaultBase = "https://api.yookassa.ru/v3"

// Client talks to the ЮKassa API with one shop's credentials.
type Client struct {
	ShopID    string
	SecretKey string
	// Base overrides DefaultBase (tests point it at a fake server).
	Base string
	HTTP *http.Client
	// TaxSystem is the receipt's tax_system_code (1..6), 0 to omit it. ЮKassa
	// requires it only for a shop with more than one tax system.
	TaxSystem int
}

// Configured reports whether the credentials are there.
func (c *Client) Configured() bool { return c != nil && c.ShopID != "" && c.SecretKey != "" }

// Payment is the part of ЮKassa's payment object the app reads.
type Payment struct {
	ID            string
	Status        string // pending, waiting_for_capture, succeeded, canceled
	Paid          bool
	Test          bool
	AmountKopecks int64
	// ConfirmationURL is where the buyer is sent to pay (only while pending).
	ConfirmationURL string
	Metadata        map[string]string
	// CancelReason is ЮKassa's reason code for a canceled payment, if any.
	CancelReason string
}

// CreateInput is one payment to create.
type CreateInput struct {
	AmountKopecks int64
	Description   string // shown to the buyer and on the receipt, max 128
	ReturnURL     string
	Email         string // the receipt goes here
	Metadata      map[string]string
	// IdempotenceKey makes a retry of the same request return the same payment.
	IdempotenceKey string
}

// Error is an error answer from the API.
type Error struct {
	Status      int
	Code        string
	Description string
}

func (e *Error) Error() string {
	return fmt.Sprintf("yookassa: %d %s: %s", e.Status, e.Code, e.Description)
}

type amountJSON struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type paymentJSON struct {
	ID           string            `json:"id"`
	Status       string            `json:"status"`
	Paid         bool              `json:"paid"`
	Test         bool              `json:"test"`
	Amount       amountJSON        `json:"amount"`
	Metadata     map[string]string `json:"metadata"`
	Confirmation *struct {
		ConfirmationURL string `json:"confirmation_url"`
	} `json:"confirmation"`
	CancellationDetails *struct {
		Reason string `json:"reason"`
	} `json:"cancellation_details"`
}

func (c *Client) base() string {
	if c.Base != "" {
		return strings.TrimRight(c.Base, "/")
	}
	return DefaultBase
}

func (c *Client) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 15 * time.Second}
}

// Create makes a payment the buyer confirms by redirect and returns it, with
// ConfirmationURL set.
func (c *Client) Create(ctx context.Context, in CreateInput) (Payment, error) {
	if !c.Configured() {
		return Payment{}, errors.New("yookassa: not configured")
	}
	desc := in.Description
	if r := []rune(desc); len(r) > 128 {
		desc = string(r[:128])
	}
	amount := amountJSON{Value: KopecksToValue(in.AmountKopecks), Currency: "RUB"}
	receipt := map[string]any{
		"customer": map[string]string{"email": in.Email},
		"items": []map[string]any{{
			"description":     desc,
			"quantity":        "1.00",
			"amount":          amount,
			"vat_code":        1, // без НДС
			"payment_mode":    "full_payment",
			"payment_subject": "service",
		}},
	}
	if c.TaxSystem != 0 {
		receipt["tax_system_code"] = c.TaxSystem
	}
	body := map[string]any{
		"amount":       amount,
		"capture":      true,
		"description":  desc,
		"confirmation": map[string]string{"type": "redirect", "return_url": in.ReturnURL},
		"metadata":     in.Metadata,
		"receipt":      receipt,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return Payment{}, fmt.Errorf("yookassa: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base()+"/payments", bytes.NewReader(raw))
	if err != nil {
		return Payment{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotence-Key", in.IdempotenceKey)
	return c.do(req)
}

// Get reads a payment by ЮKassa's id.
func (c *Client) Get(ctx context.Context, id string) (Payment, error) {
	if !c.Configured() {
		return Payment{}, errors.New("yookassa: not configured")
	}
	if id == "" || strings.ContainsAny(id, "/?# ") {
		return Payment{}, errors.New("yookassa: bad payment id")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base()+"/payments/"+id, nil)
	if err != nil {
		return Payment{}, err
	}
	return c.do(req)
}

func (c *Client) do(req *http.Request) (Payment, error) {
	req.SetBasicAuth(c.ShopID, c.SecretKey)
	resp, err := c.client().Do(req)
	if err != nil {
		return Payment{}, fmt.Errorf("yookassa: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Payment{}, fmt.Errorf("yookassa: read body: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		var e struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		}
		_ = json.Unmarshal(data, &e)
		return Payment{}, &Error{Status: resp.StatusCode, Code: e.Code, Description: e.Description}
	}
	var pj paymentJSON
	if err := json.Unmarshal(data, &pj); err != nil {
		return Payment{}, fmt.Errorf("yookassa: decode: %w", err)
	}
	p := Payment{ID: pj.ID, Status: pj.Status, Paid: pj.Paid, Test: pj.Test, Metadata: pj.Metadata}
	p.AmountKopecks, _ = ValueToKopecks(pj.Amount.Value)
	if pj.Confirmation != nil {
		p.ConfirmationURL = pj.Confirmation.ConfirmationURL
	}
	if pj.CancellationDetails != nil {
		p.CancelReason = pj.CancellationDetails.Reason
	}
	return p, nil
}

// KopecksToValue formats kopecks as ЮKassa wants an amount: a string with
// exactly two decimals ("249.00").
func KopecksToValue(k int64) string {
	return fmt.Sprintf("%d.%02d", k/100, k%100)
}

// ValueToKopecks parses "249.00" (or "249") into kopecks without going through
// floating point.
func ValueToKopecks(v string) (int64, error) {
	whole, frac, _ := strings.Cut(strings.TrimSpace(v), ".")
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil || w < 0 {
		return 0, fmt.Errorf("yookassa: bad amount %q", v)
	}
	switch len(frac) {
	case 0:
		return w * 100, nil
	case 1:
		frac += "0"
	case 2:
	default:
		return 0, fmt.Errorf("yookassa: bad amount %q", v)
	}
	f, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("yookassa: bad amount %q", v)
	}
	return w*100 + f, nil
}
