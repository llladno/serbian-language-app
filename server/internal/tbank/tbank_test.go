package tbank

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The two worked examples from the bank's documentation, so the signing rule is
// checked against the bank's own numbers and not only against itself.
func TestSignMatchesTheBanksOwnExample(t *testing.T) {
	// https://developer.tbank.ru/eacq/intro/developer/token : Receipt and DATA
	// are in the real request but not in the Token.
	req := map[string]any{
		"TerminalKey": "MerchantTerminalKey",
		"Amount":      19200,
		"OrderId":     "00000",
		"Description": "Подарочная карта на 1000 рублей",
		"DATA":        map[string]any{"Phone": "+71234567890"},
		"Receipt":     map[string]any{"Taxation": "osn"},
	}
	const want = "72dd466f8ace0a37a1f740ce5fb78101712bc0665d91a8108c7c8a0ccd426db2"
	if got := Sign(req, "11111111111111"); got != want {
		t.Errorf("Sign = %s, want %s", got, want)
	}
}

func TestSignNotificationExample(t *testing.T) {
	// https://developer.tbank.ru/eacq/intro/developer/notification : Success is a
	// bool and goes into the string as "true".
	body := `{"TerminalKey":"1234567890DEMO","OrderId":"000000","Success":true,"Status":"AUTHORIZED",
		"PaymentId":"0000000","ErrorCode":"0","Amount":1111,"CardId":"000000","Pan":"200000******0000",
		"ExpDate":"1111","RebillId":"000000",
		"Token":"1c0964277d0213349243065a0d5b838b8e90d2d25f740d0f2767836e710e80c8"}`
	c := &Client{TerminalKey: "1234567890DEMO", Password: "11111111111"}
	n, err := c.ParseNotification([]byte(body))
	if err != nil {
		t.Fatalf("ParseNotification: %v", err)
	}
	if n.PaymentID != "0000000" || n.OrderID != "000000" || n.Status != "AUTHORIZED" || !n.Success {
		t.Errorf("notification read as %+v", n)
	}
}

func TestParseNotificationRejectsWhatIsNotSignedByUs(t *testing.T) {
	c := &Client{TerminalKey: "T1", Password: "secret"}
	sign := func(m map[string]any) []byte {
		m["Token"] = Sign(m, "secret")
		b, _ := json.Marshal(m)
		return b
	}
	good := map[string]any{"TerminalKey": "T1", "PaymentId": 4242, "OrderId": "7-1", "Status": "CONFIRMED", "Success": true, "Amount": 24900}
	n, err := c.ParseNotification(sign(good))
	if err != nil || n.PaymentID != "4242" || n.Status != "CONFIRMED" {
		t.Fatalf("a correctly signed notification: %+v, %v", n, err)
	}

	tampered := sign(map[string]any{"TerminalKey": "T1", "PaymentId": 4242, "OrderId": "7-1", "Status": "CONFIRMED", "Amount": 24900})
	tampered = []byte(strings.Replace(string(tampered), `"Amount":24900`, `"Amount":100`, 1))
	foreign := sign(map[string]any{"TerminalKey": "OTHER", "PaymentId": 1, "Status": "CONFIRMED"})
	wrongPassword, _ := json.Marshal(map[string]any{"TerminalKey": "T1", "PaymentId": 1, "Status": "CONFIRMED", "Token": Sign(map[string]any{"TerminalKey": "T1", "PaymentId": 1, "Status": "CONFIRMED"}, "guess")})
	noToken, _ := json.Marshal(map[string]any{"TerminalKey": "T1", "PaymentId": 1, "Status": "CONFIRMED"})
	for name, body := range map[string][]byte{"tampered": tampered, "other terminal": foreign, "wrong password": wrongPassword, "no token": noToken} {
		if _, err := c.ParseNotification(body); !errors.Is(err, ErrBadToken) {
			t.Errorf("%s: err = %v, want ErrBadToken", name, err)
		}
	}
	if _, err := c.ParseNotification([]byte("not json")); err == nil {
		t.Error("garbage was accepted")
	}
}

// fakeBank is the API as the tests need it: it checks the Token the way the bank
// does, and answers from a canned reply.
type fakeBank struct {
	t        *testing.T
	password string
	reply    string
	status   int
	got      map[string]json.RawMessage
	path     string
}

func (f *fakeBank) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.path = r.URL.Path
	body, _ := io.ReadAll(r.Body)
	f.got = map[string]json.RawMessage{}
	_ = json.Unmarshal(body, &f.got)
	// Re-read with numbers kept, the way the signature is computed.
	var m map[string]any
	d := json.NewDecoder(strings.NewReader(string(body)))
	d.UseNumber()
	_ = d.Decode(&m)
	tok, _ := m["Token"].(string)
	if tok != Sign(m, f.password) {
		f.t.Errorf("the request's Token does not verify")
	}
	if r.Header.Get("Content-Type") != "application/json" {
		f.t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
	}
	if f.status != 0 {
		w.WriteHeader(f.status)
	}
	io.WriteString(w, f.reply)
}

func newClient(t *testing.T, f *fakeBank) *Client {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	f.t, f.password = t, "pw"
	return &Client{TerminalKey: "TERM1DEMO", Password: "pw", Base: srv.URL, Taxation: "usn_income"}
}

func TestInitSendsASignedPaymentWithAReceiptAndReturnsThePage(t *testing.T) {
	f := &fakeBank{reply: `{"Success":true,"ErrorCode":"0","TerminalKey":"TERM1DEMO","Status":"NEW","PaymentId":"9446466517","OrderId":"7-1","Amount":24900,"PaymentURL":"https://pay.example/abc"}`}
	c := newClient(t, f)
	p, err := c.Init(context.Background(), InitInput{
		OrderID: "7-1", AmountKopecks: 24900, Description: "Уровень 4 — Падежи", Email: "buyer@example.com",
		SuccessURL: "https://ucimo.ru/tariffs?order=7", FailURL: "https://ucimo.ru/tariffs?order=7",
		NotificationURL: "https://ucimo.ru/api/tbank/webhook",
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if p.PaymentID != "9446466517" || p.URL != "https://pay.example/abc" || p.Status != "NEW" || p.AmountKopecks != 24900 {
		t.Errorf("payment = %+v", p)
	}
	if f.path != "/v2/Init" {
		t.Errorf("path = %s", f.path)
	}
	var receipt struct {
		Email    string
		Taxation string
		Items    []struct {
			Name          string
			Price         int64
			Quantity      float64
			Amount        int64
			Tax           string
			PaymentMethod string
			PaymentObject string
		}
	}
	if err := json.Unmarshal(f.got["Receipt"], &receipt); err != nil {
		t.Fatalf("Receipt: %v", err)
	}
	it := receipt.Items[0]
	if receipt.Email != "buyer@example.com" || receipt.Taxation != "usn_income" || len(receipt.Items) != 1 ||
		it.Price != 24900 || it.Amount != 24900 || it.Quantity != 1 || it.Tax != "none" ||
		it.PaymentMethod != "full_payment" || it.PaymentObject != "service" {
		t.Errorf("receipt = %+v", receipt)
	}
	for k, want := range map[string]string{"PayType": `"O"`, "NotificationURL": `"https://ucimo.ru/api/tbank/webhook"`, "Amount": "24900", "TerminalKey": `"TERM1DEMO"`} {
		if string(f.got[k]) != want {
			t.Errorf("%s = %s, want %s", k, f.got[k], want)
		}
	}
}

func TestInitReportsTheBanksRefusal(t *testing.T) {
	f := &fakeBank{reply: `{"Success":false,"ErrorCode":"501","Message":"Неверные параметры","Details":"Терминал не найден."}`}
	c := newClient(t, f)
	_, err := c.Init(context.Background(), InitInput{OrderID: "1", AmountKopecks: 100, Description: "x", Email: "a@b.c"})
	var e *Error
	if !errors.As(err, &e) || e.Code != "501" || !strings.Contains(e.Details, "Терминал") {
		t.Fatalf("err = %v, want a *tbank.Error with the bank's code", err)
	}
	f.status, f.reply = http.StatusInternalServerError, `{"Success":false,"ErrorCode":"9999","Message":"Внутренняя ошибка системы."}`
	if _, err := c.Init(context.Background(), InitInput{OrderID: "1", AmountKopecks: 100, Description: "x", Email: "a@b.c"}); !errors.As(err, &e) || e.Status != 500 {
		t.Fatalf("err = %v, want the HTTP status kept", err)
	}
}

func TestGetStateReadsAPaymentWhateverTheTypeOfItsId(t *testing.T) {
	for name, reply := range map[string]string{
		"string id": `{"Success":true,"ErrorCode":"0","Message":"OK","Status":"CONFIRMED","PaymentId":"55","OrderId":"7-1","Amount":24900}`,
		"number id": `{"Success":true,"ErrorCode":"0","Message":"OK","Status":"CONFIRMED","PaymentId":55,"OrderId":"7-1","Amount":24900}`,
	} {
		f := &fakeBank{reply: reply}
		c := newClient(t, f)
		p, err := c.GetState(context.Background(), "55")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if p.PaymentID != "55" || !p.Paid() || p.OrderID != "7-1" || p.AmountKopecks != 24900 {
			t.Errorf("%s: payment = %+v", name, p)
		}
		if f.path != "/v2/GetState" || string(f.got["PaymentId"]) != `"55"` {
			t.Errorf("%s: sent %s %s", name, f.path, f.got["PaymentId"])
		}
	}
	c := &Client{TerminalKey: "T", Password: "p"}
	for _, bad := range []string{"", "a/b", "1?x", "1 2"} {
		if _, err := c.GetState(context.Background(), bad); err == nil {
			t.Errorf("GetState(%q) accepted a bad id", bad)
		}
	}
}

func TestStatuses(t *testing.T) {
	for status, want := range map[string][2]bool{ // paid, failed
		"NEW": {false, false}, "FORM_SHOWED": {false, false}, "3DS_CHECKING": {false, false},
		"AUTHORIZED": {false, false}, "CONFIRMING": {false, false},
		"CONFIRMED": {true, false},
		"REJECTED":  {false, true}, "AUTH_FAIL": {false, true}, "CANCELED": {false, true},
		"DEADLINE_EXPIRED": {false, true}, "ATTEMPTS_EXPIRED": {false, true},
		"REFUNDED": {false, false}, "REVERSED": {false, false},
	} {
		p := Payment{Status: status}
		if p.Paid() != want[0] || p.Failed() != want[1] {
			t.Errorf("%s: paid=%t failed=%t, want paid=%t failed=%t", status, p.Paid(), p.Failed(), want[0], want[1])
		}
	}
}

func TestNotConfiguredClientDoesNothing(t *testing.T) {
	c := &Client{}
	if c.Configured() {
		t.Error("an empty client claims to be configured")
	}
	if _, err := c.Init(context.Background(), InitInput{}); err == nil {
		t.Error("Init without credentials succeeded")
	}
	if _, err := c.GetState(context.Background(), "1"); err == nil {
		t.Error("GetState without credentials succeeded")
	}
	if !(&Client{TerminalKey: "1234567890DEMO"}).Test() || (&Client{TerminalKey: "1234567890"}).Test() {
		t.Error("Test() should be true for a DEMO terminal only")
	}
}

// The embedded bundle is what makes the real API reachable, so it must be a
// root and the intermediate it signed, not an empty or damaged file.
func TestEmbeddedRussianCAsAreTheRootAndItsIntermediate(t *testing.T) {
	var certs []*x509.Certificate
	rest := russianTrustedCA
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		certs = append(certs, c)
	}
	if len(certs) != 2 {
		t.Fatalf("%d certificates, want 2", len(certs))
	}
	root, sub := certs[0], certs[1]
	if root.Subject.CommonName != "Russian Trusted Root CA" || sub.Subject.CommonName != "Russian Trusted Sub CA" {
		t.Errorf("subjects: %q, %q", root.Subject.CommonName, sub.Subject.CommonName)
	}
	if err := root.CheckSignatureFrom(root); err != nil {
		t.Errorf("the root is not self-signed: %v", err)
	}
	if err := sub.CheckSignatureFrom(root); err != nil {
		t.Errorf("the intermediate is not signed by the root: %v", err)
	}
	if trustedPool() == nil {
		t.Error("no trust pool")
	}
}

// The receipt names the buyer by email or by phone, never by an empty email.
func TestInitSendsThePhoneWhenTheReceiptGoesToAPhone(t *testing.T) {
	f := &fakeBank{reply: `{"Success":true,"ErrorCode":"0","TerminalKey":"TERM1DEMO","Status":"NEW","PaymentId":"1","OrderId":"1","Amount":100,"PaymentURL":"https://pay.example/abc"}`}
	c := newClient(t, f)
	if _, err := c.Init(context.Background(), InitInput{OrderID: "1", AmountKopecks: 100, Description: "x", Phone: "+79031234567"}); err != nil {
		t.Fatal(err)
	}
	var receipt map[string]any
	if err := json.Unmarshal(f.got["Receipt"], &receipt); err != nil {
		t.Fatalf("Receipt: %v", err)
	}
	if receipt["Phone"] != "+79031234567" {
		t.Errorf("Receipt = %v, want the phone", receipt)
	}
	if _, has := receipt["Email"]; has {
		t.Errorf("Receipt = %v, want no Email key at all", receipt)
	}
}
