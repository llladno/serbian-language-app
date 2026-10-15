package tbank

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestLiveDemoTerminal talks to the real API with the bank's test terminal, to
// prove that the signature, the receipt and the embedded Russian CAs are accepted
// by the real thing. It moves no money (a DEMO terminal) and is skipped unless
// the terminal's credentials are in the environment:
//
//	TBANK_LIVE_TERMINAL_KEY=...DEMO TBANK_LIVE_PASSWORD=... go test ./internal/tbank -run Live -v
func TestLiveDemoTerminal(t *testing.T) {
	key, pw := os.Getenv("TBANK_LIVE_TERMINAL_KEY"), os.Getenv("TBANK_LIVE_PASSWORD")
	if key == "" || pw == "" {
		t.Skip("set TBANK_LIVE_TERMINAL_KEY and TBANK_LIVE_PASSWORD to run against the bank")
	}
	c := &Client{TerminalKey: key, Password: pw, Taxation: "usn_income"}
	if !c.Test() {
		t.Fatal("refusing to run against a terminal that is not a DEMO one")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p, err := c.Init(ctx, InitInput{
		OrderID: fmt.Sprintf("live-%d", time.Now().Unix()), AmountKopecks: 24900,
		Description: "Уровень 4 — Падежи", Email: "buyer@example.com",
		SuccessURL: "https://ucimo.ru/tariffs?order=1", FailURL: "https://ucimo.ru/tariffs?order=1",
		NotificationURL: "https://ucimo.ru/api/tbank/webhook",
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if p.PaymentID == "" || p.URL == "" || p.Status != "NEW" || p.AmountKopecks != 24900 {
		t.Fatalf("Init = %+v", p)
	}
	s, err := c.GetState(ctx, p.PaymentID)
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if s.PaymentID != p.PaymentID || s.Status != "NEW" || s.Paid() || s.Failed() || s.AmountKopecks != 24900 {
		t.Fatalf("GetState = %+v", s)
	}
	// A request signed with a wrong password must be refused by the bank.
	bad := &Client{TerminalKey: key, Password: pw + "x", Taxation: "usn_income"}
	if _, err := bad.GetState(ctx, p.PaymentID); err == nil {
		t.Error("the bank accepted a request signed with a wrong password")
	}
}
