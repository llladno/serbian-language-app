package yookassa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAmountsRoundTripWithoutFloats(t *testing.T) {
	for k, v := range map[int64]string{0: "0.00", 5: "0.05", 24900: "249.00", 29900: "299.00", 100000: "1000.00", 1: "0.01"} {
		if got := KopecksToValue(k); got != v {
			t.Errorf("KopecksToValue(%d) = %q, want %q", k, got, v)
		}
		if back, err := ValueToKopecks(v); err != nil || back != k {
			t.Errorf("ValueToKopecks(%q) = (%d, %v), want %d", v, back, err, k)
		}
	}
	if k, _ := ValueToKopecks("249"); k != 24900 {
		t.Errorf("ValueToKopecks(249) = %d", k)
	}
	if k, _ := ValueToKopecks("249.5"); k != 24950 {
		t.Errorf("ValueToKopecks(249.5) = %d", k)
	}
	for _, bad := range []string{"", "abc", "1.234", "-1.00", "1,00"} {
		if _, err := ValueToKopecks(bad); err == nil {
			t.Errorf("ValueToKopecks(%q) accepted", bad)
		}
	}
}

func newClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &Client{ShopID: "shop-1", SecretKey: "secret-1", Base: srv.URL}
}

func TestCreateSendsAnAuthenticatedReceiptedRedirectPayment(t *testing.T) {
	var got map[string]any
	var user, pass, idem, method, path string
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		user, pass, _ = r.BasicAuth()
		idem, method, path = r.Header.Get("Idempotence-Key"), r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"pay-1","status":"pending","paid":false,"test":true,
			"amount":{"value":"249.00","currency":"RUB"},
			"confirmation":{"type":"redirect","confirmation_url":"https://pay.example/confirm?id=pay-1"},
			"metadata":{"order_id":"7"}}`))
	})
	c.TaxSystem = 2

	p, err := c.Create(context.Background(), CreateInput{
		AmountKopecks: 24900, Description: "Уровень 4", ReturnURL: "https://ucimo.ru/tariffs?order=7",
		Email: "g@example.com", Metadata: map[string]string{"order_id": "7"}, IdempotenceKey: "order-7",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ID != "pay-1" || p.Status != "pending" || !p.Test || p.AmountKopecks != 24900 ||
		p.ConfirmationURL != "https://pay.example/confirm?id=pay-1" || p.Metadata["order_id"] != "7" {
		t.Fatalf("payment = %+v", p)
	}
	if method != "POST" || path != "/payments" || user != "shop-1" || pass != "secret-1" {
		t.Errorf("request = %s %s as %s:%s", method, path, user, pass)
	}
	if idem != "order-7" {
		t.Errorf("Idempotence-Key = %q, want order-7: a retry must not charge twice", idem)
	}
	amount := got["amount"].(map[string]any)
	if amount["value"] != "249.00" || amount["currency"] != "RUB" || got["capture"] != true {
		t.Errorf("amount/capture = %v / %v", amount, got["capture"])
	}
	conf := got["confirmation"].(map[string]any)
	if conf["type"] != "redirect" || conf["return_url"] != "https://ucimo.ru/tariffs?order=7" {
		t.Errorf("confirmation = %v", conf)
	}
	rec := got["receipt"].(map[string]any)
	if rec["customer"].(map[string]any)["email"] != "g@example.com" || rec["tax_system_code"] != float64(2) {
		t.Errorf("receipt = %v", rec)
	}
	item := rec["items"].([]any)[0].(map[string]any)
	if item["vat_code"] != float64(1) || item["payment_mode"] != "full_payment" || item["payment_subject"] != "service" ||
		item["quantity"] != "1.00" {
		t.Errorf("receipt item = %v", item)
	}
}

func TestCreateLeavesTheTaxSystemOutUnlessConfigured(t *testing.T) {
	var got map[string]any
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.Write([]byte(`{"id":"p","status":"pending"}`))
	})
	if _, err := c.Create(context.Background(), CreateInput{AmountKopecks: 100, Email: "a@b.c"}); err != nil {
		t.Fatal(err)
	}
	if _, has := got["receipt"].(map[string]any)["tax_system_code"]; has {
		t.Error("tax_system_code sent without being configured")
	}
}

func TestCreateCutsALongDescriptionToWhatTheAPIAllows(t *testing.T) {
	var got map[string]any
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.Write([]byte(`{"id":"p","status":"pending"}`))
	})
	c.Create(context.Background(), CreateInput{AmountKopecks: 100, Email: "a@b.c", Description: strings.Repeat("я", 300)})
	if n := len([]rune(got["description"].(string))); n != 128 {
		t.Errorf("description is %d runes, want 128", n)
	}
}

func TestGetReadsTheStatusAndTheCancelReason(t *testing.T) {
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/payments/pay-9" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		w.Write([]byte(`{"id":"pay-9","status":"canceled","paid":false,"amount":{"value":"1000.00","currency":"RUB"},
			"cancellation_details":{"party":"yoo_money","reason":"insufficient_funds"}}`))
	})
	p, err := c.Get(context.Background(), "pay-9")
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != "canceled" || p.Paid || p.CancelReason != "insufficient_funds" || p.AmountKopecks != 100000 {
		t.Fatalf("payment = %+v", p)
	}
}

func TestAnAPIErrorCarriesItsCode(t *testing.T) {
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type":"error","code":"invalid_credentials","description":"Basic auth required"}`))
	})
	_, err := c.Get(context.Background(), "x")
	var e *Error
	if !asError(err, &e) || e.Status != 401 || e.Code != "invalid_credentials" {
		t.Fatalf("err = %v, want a 401 invalid_credentials Error", err)
	}
}

func TestGetRefusesAnIdThatCouldEscapeThePath(t *testing.T) {
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) { t.Error("request was made") })
	for _, id := range []string{"", "../refunds", "a/b", "a?b"} {
		if _, err := c.Get(context.Background(), id); err == nil {
			t.Errorf("Get(%q) did not fail", id)
		}
	}
}

func TestWithoutCredentialsNothingIsSent(t *testing.T) {
	c := &Client{}
	if c.Configured() {
		t.Fatal("empty client says it is configured")
	}
	if _, err := c.Create(context.Background(), CreateInput{}); err == nil {
		t.Error("Create without credentials succeeded")
	}
	var nilClient *Client
	if nilClient.Configured() {
		t.Error("nil client says it is configured")
	}
}

func asError(err error, target **Error) bool {
	e, ok := err.(*Error)
	if ok {
		*target = e
	}
	return ok
}
