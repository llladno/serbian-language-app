// Package tbank is the small slice of the Т-Касса (Т-Банк internet acquiring)
// API the app needs: start a payment that the buyer completes on the bank's own
// page, read one back, and check a notification. Docs:
// https://developer.tbank.ru/eacq/scenarios/payments/nonPCI/
//
// Every request and every notification carries a Token: a SHA-256 over the
// request's top-level values and the terminal's password (see Sign). A
// notification's Token is checked, but callers still take only the payment id
// from it and ask GetState for the truth, as they do for ЮKassa.
package tbank

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultBase is the API. A terminal whose key ends in DEMO is the bank's test
// terminal: it is reached at this same address and moves no money. (The separate
// rest-api-test host is only for real terminals from a whitelisted IP.)
const DefaultBase = "https://securepay.tinkoff.ru"

// russianTrustedCA is the Минцифры root and intermediate certificates. The bank's
// API hosts present chains that end in them, and neither is in the system trust
// stores of the images we run on, so without these every request fails with an
// x509 error. They are trusted for this client only, never system-wide.
//
//go:embed russian_trusted_ca.pem
var russianTrustedCA []byte

var (
	poolOnce sync.Once
	pool     *x509.CertPool
)

// trustedPool is the system roots plus the Russian ones above.
func trustedPool() *x509.CertPool {
	poolOnce.Do(func() {
		p, err := x509.SystemCertPool()
		if err != nil || p == nil {
			p = x509.NewCertPool()
		}
		p.AppendCertsFromPEM(russianTrustedCA)
		pool = p
	})
	return pool
}

// Client talks to the Т-Касса API with one terminal's credentials.
type Client struct {
	TerminalKey string
	Password    string
	// Base overrides DefaultBase (tests point it at a fake server).
	Base string
	HTTP *http.Client
	// Taxation is the receipt's tax system: osn, usn_income, usn_income_outcome,
	// esn or patent. Required by the bank whenever a cash register is connected.
	Taxation string
}

// Configured reports whether the credentials are there.
func (c *Client) Configured() bool { return c != nil && c.TerminalKey != "" && c.Password != "" }

// Test reports whether this is the bank's test terminal.
func (c *Client) Test() bool { return strings.HasSuffix(c.TerminalKey, "DEMO") }

// Payment is the part of the bank's payment object the app reads.
type Payment struct {
	PaymentID string
	OrderID   string
	Status    string // NEW, FORM_SHOWED, AUTHORIZED, CONFIRMED, REJECTED, ...
	// AmountKopecks is the amount the bank holds for the payment.
	AmountKopecks int64
	// URL is where the buyer is sent to pay (set by Init).
	URL string
}

// Paid reports whether the money has been taken. For the one-stage payments we
// make, AUTHORIZED comes first and CONFIRMED right after; only CONFIRMED is a
// completed payment.
func (p Payment) Paid() bool { return p.Status == "CONFIRMED" }

// Failed reports whether the payment ended without money moving: the card was
// refused, the buyer never finished, or it was canceled. Anything else (a
// payment still on its way, or one already refunded) is neither.
func (p Payment) Failed() bool {
	switch p.Status {
	case "REJECTED", "AUTH_FAIL", "CANCELED", "DEADLINE_EXPIRED", "ATTEMPTS_EXPIRED":
		return true
	}
	return false
}

// InitInput is one payment to create.
type InitInput struct {
	// OrderID is our identifier for it, at most 50 characters, unique per payment.
	OrderID       string
	AmountKopecks int64
	Description   string // shown on the payment page and on the receipt
	Email         string // the receipt goes here
	// SuccessURL and FailURL are where the buyer lands afterwards.
	SuccessURL, FailURL string
	// NotificationURL gets the bank's POST about the payment.
	NotificationURL string
}

// Error is an error answer from the API.
type Error struct {
	Status  int // HTTP status
	Code    string
	Message string
	Details string
}

func (e *Error) Error() string {
	return fmt.Sprintf("tbank: http %d code %s: %s %s", e.Status, e.Code, e.Message, e.Details)
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
	return &http.Client{
		Timeout:   15 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: trustedPool(), MinVersion: tls.VersionTLS12}},
	}
}

// Init creates a payment and returns it with URL set, the page to send the buyer to.
func (c *Client) Init(ctx context.Context, in InitInput) (Payment, error) {
	if !c.Configured() {
		return Payment{}, errors.New("tbank: not configured")
	}
	desc := truncate(in.Description, 140)
	body := map[string]any{
		"Amount":      in.AmountKopecks,
		"OrderId":     in.OrderID,
		"Description": desc,
		"Language":    "ru",
		"PayType":     "O", // one stage: the money is taken at once
	}
	if in.SuccessURL != "" {
		body["SuccessURL"] = in.SuccessURL
	}
	if in.FailURL != "" {
		body["FailURL"] = in.FailURL
	}
	if in.NotificationURL != "" {
		body["NotificationURL"] = in.NotificationURL
	}
	// Objects below are not part of the Token, so they are added to the body
	// either way; Sign skips anything that is not a plain value.
	body["DATA"] = map[string]string{"OperationInitiatorType": "0"}
	body["Receipt"] = map[string]any{
		"Email":    in.Email,
		"Taxation": c.Taxation,
		"Items": []map[string]any{{
			"Name":          truncate(in.Description, 128),
			"Price":         in.AmountKopecks,
			"Quantity":      1,
			"Amount":        in.AmountKopecks,
			"Tax":           "none", // the seller is not a VAT payer
			"PaymentMethod": "full_payment",
			"PaymentObject": "service",
		}},
	}
	return c.call(ctx, "Init", body)
}

// GetState asks the bank where a payment stands.
func (c *Client) GetState(ctx context.Context, paymentID string) (Payment, error) {
	if !c.Configured() {
		return Payment{}, errors.New("tbank: not configured")
	}
	if paymentID == "" || strings.ContainsAny(paymentID, "/?# ") {
		return Payment{}, errors.New("tbank: bad payment id")
	}
	return c.call(ctx, "GetState", map[string]any{"PaymentId": paymentID})
}

// flex reads a JSON string or number into a string: the bank sends PaymentId
// both ways.
type flex string

func (f *flex) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" {
		return nil
	}
	if strings.HasPrefix(s, `"`) {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*f = flex(v)
		return nil
	}
	*f = flex(s)
	return nil
}

type answerJSON struct {
	Success    bool   `json:"Success"`
	ErrorCode  string `json:"ErrorCode"`
	Message    string `json:"Message"`
	Details    string `json:"Details"`
	Status     string `json:"Status"`
	PaymentID  flex   `json:"PaymentId"`
	OrderID    string `json:"OrderId"`
	Amount     int64  `json:"Amount"`
	PaymentURL string `json:"PaymentURL"`
}

func (c *Client) call(ctx context.Context, method string, params map[string]any) (Payment, error) {
	params["TerminalKey"] = c.TerminalKey
	params["Token"] = Sign(params, c.Password)
	raw, err := json.Marshal(params)
	if err != nil {
		return Payment{}, fmt.Errorf("tbank: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base()+"/v2/"+method, bytes.NewReader(raw))
	if err != nil {
		return Payment{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client().Do(req)
	if err != nil {
		return Payment{}, fmt.Errorf("tbank: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Payment{}, fmt.Errorf("tbank: read body: %w", err)
	}
	var a answerJSON
	jsonErr := json.Unmarshal(data, &a)
	if resp.StatusCode/100 != 2 {
		return Payment{}, &Error{Status: resp.StatusCode, Code: a.ErrorCode, Message: a.Message, Details: a.Details}
	}
	if jsonErr != nil {
		return Payment{}, fmt.Errorf("tbank: decode: %w", jsonErr)
	}
	// The bank answers 200 with Success=false for a refused request.
	if !a.Success || (a.ErrorCode != "" && a.ErrorCode != "0") {
		return Payment{}, &Error{Status: resp.StatusCode, Code: a.ErrorCode, Message: a.Message, Details: a.Details}
	}
	return Payment{
		PaymentID: string(a.PaymentID), OrderID: a.OrderID, Status: a.Status,
		AmountKopecks: a.Amount, URL: a.PaymentURL,
	}, nil
}

// Sign computes the Token: take the request's top-level parameters that are
// plain values (nested objects and arrays, like Receipt and DATA, are left out),
// add the terminal password as "Password", sort by parameter name, join the
// values and hash the result with SHA-256. A bool is written "true"/"false".
func Sign(params map[string]any, password string) string {
	vals := map[string]string{"Password": password}
	for k, v := range params {
		if k == "Token" {
			continue
		}
		if s, ok := scalar(v); ok {
			vals[k] = s
		}
	}
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(vals[k])
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// scalar writes a plain JSON value the way the bank concatenates it, and says
// whether v is one (objects, arrays and null are not).
func scalar(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, true
	case bool:
		return strconv.FormatBool(x), true
	case json.Number:
		return x.String(), true
	case int:
		return strconv.Itoa(x), true
	case int64:
		return strconv.FormatInt(x, 10), true
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), true
	}
	return "", false
}

// Notification is the bank's POST about a payment, after its Token was checked.
type Notification struct {
	PaymentID string
	OrderID   string
	Status    string // empty for the bank's other notifications (receipts, card binding)
	Success   bool
}

// ErrBadToken is returned for a notification that is not signed with our
// terminal's password.
var ErrBadToken = errors.New("tbank: bad notification token")

// ParseNotification checks a notification's Token against the terminal's
// password and reads it. It does not decide anything about the payment.
func (c *Client) ParseNotification(body []byte) (Notification, error) {
	if !c.Configured() {
		return Notification{}, errors.New("tbank: not configured")
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return Notification{}, fmt.Errorf("tbank: decode notification: %w", err)
	}
	got, _ := m["Token"].(string)
	if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(Sign(m, c.Password))) != 1 {
		return Notification{}, ErrBadToken
	}
	if key, _ := m["TerminalKey"].(string); key != c.TerminalKey {
		return Notification{}, ErrBadToken
	}
	n := Notification{}
	n.PaymentID, _ = scalar(m["PaymentId"])
	n.OrderID, _ = m["OrderId"].(string)
	n.Status, _ = m["Status"].(string)
	n.Success, _ = m["Success"].(bool)
	return n, nil
}

func truncate(s string, max int) string {
	if r := []rune(s); len(r) > max {
		return string(r[:max])
	}
	return s
}
