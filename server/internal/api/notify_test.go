package api

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// noticeSink collects what would have gone to the operator's bot.
type noticeSink struct {
	mu   sync.Mutex
	msgs []string
}

func (s *noticeSink) send(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, text)
}

func (s *noticeSink) all() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.msgs...)
}

func TestFormatRubles(t *testing.T) {
	for kop, want := range map[int64]string{
		19900:   "199\u00a0₽",
		100000:  "1\u00a0000\u00a0₽",
		29950:   "299,50\u00a0₽",
		1234500: "12\u00a0345\u00a0₽",
		0:       "0\u00a0₽",
	} {
		if got := formatRubles(kop); got != want {
			t.Errorf("formatRubles(%d) = %q, want %q", kop, got, want)
		}
	}
}

func TestPaymentNoticeIsEscapedAndComplete(t *testing.T) {
	got := paymentNotice{
		TariffTitle: "Год", AmountKopecks: 100000, Bonus: 15000, Provider: "tbank", PaymentID: 42,
		ReceiptTo: "+79031234567",
		Who:       noticeWho{Name: "Ann <b>&", Email: "ann@example.com", TgUsername: "ann_tg"},
	}.String()
	for _, want := range []string{
		"💸 <b>Новая оплата!</b>", "Год", "<b>1\u00a0000\u00a0₽</b>", "15000 зёрнышек",
		"Ann &lt;b&gt;&amp;", "ann@example.com", `<a href="https://t.me/ann_tg">@ann_tg</a>`,
		"чек: +79031234567", "Т-Касса", "заказ №42",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("notice lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Ann <b>") {
		t.Errorf("user's name was not escaped:\n%s", got)
	}
	if !strings.Contains(paymentNotice{Test: true}.String(), "🧪 <b>Тестовая оплата</b>") {
		t.Error("a test payment is not marked as one")
	}
}

func TestSupportNoticeQuotesAndTruncates(t *testing.T) {
	got := supportNotice{Who: noticeWho{Name: "Bob"}, Message: "a < b & c"}.String()
	if !strings.Contains(got, "<blockquote>a &lt; b &amp; c</blockquote>") {
		t.Errorf("message not quoted and escaped:\n%s", got)
	}
	long := supportNotice{Who: noticeWho{Name: "Bob"}, Message: strings.Repeat("я", 9000)}.String()
	if len([]rune(long)) > 4096 {
		t.Errorf("notice is %d characters, over Telegram's limit", len([]rune(long)))
	}
}

func TestASupportMessageNotifiesTheOperator(t *testing.T) {
	sink := &noticeSink{}
	h, st, _ := newAuthAPI(t, func(d *Deps) { d.NotifyAdmin = sink.send })
	uid := registerAndVerify(t, h, st, "bob@example.com", "password123", "Bob")
	c := authed(t, st, uid)

	if rr := doCookie(h, c, "POST", "/api/me/support", `{"message":"Не открывается урок 3"}`); rr.Code != 200 {
		t.Fatalf("support = %d", rr.Code)
	}
	// a blank message is refused and says nothing to the operator
	doCookie(h, c, "POST", "/api/me/support", `{"message":"  "}`)

	msgs := sink.all()
	if len(msgs) != 1 {
		t.Fatalf("operator got %d notices, want 1: %q", len(msgs), msgs)
	}
	for _, want := range []string{"Новое обращение в поддержку", "Bob", "bob@example.com", "Не открывается урок 3"} {
		if !strings.Contains(msgs[0], want) {
			t.Errorf("notice lacks %q:\n%s", want, msgs[0])
		}
	}
}

func TestAPaymentNotifiesTheOperatorOnce(t *testing.T) {
	sink := &noticeSink{}
	tb := newFakeTBank(t)
	h, st := tariffsAPIWith(t, tb, func(d *Deps) { d.NotifyAdmin = sink.send })
	_, c := listedUser(t, h, st)

	o := order(t, h, c, "level")
	if n := len(sink.all()); n != 0 {
		t.Fatalf("operator told about an unpaid order: %q", sink.all())
	}
	tb.set("9001", "CONFIRMED")
	// the return page and the bank's notification both arrive
	doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	anon(h, "POST", "/api/tbank/webhook", tb.notification("9001", "CONFIRMED"))
	doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")

	msgs := sink.all()
	if len(msgs) != 1 {
		t.Fatalf("operator got %d notices, want exactly 1: %q", len(msgs), msgs)
	}
	for _, want := range []string{"оплата", "Т-Касса", fmt.Sprintf("заказ №%d", o.ID), "₽"} {
		if !strings.Contains(msgs[0], want) {
			t.Errorf("notice lacks %q:\n%s", want, msgs[0])
		}
	}
}

func TestACanceledPaymentDoesNotNotifyTheOperator(t *testing.T) {
	sink := &noticeSink{}
	tb := newFakeTBank(t)
	h, st := tariffsAPIWith(t, tb, func(d *Deps) { d.NotifyAdmin = sink.send })
	_, c := listedUser(t, h, st)
	o := order(t, h, c, "year")
	tb.set("9001", "REJECTED")
	doCookie(h, c, "GET", fmt.Sprintf("/api/me/orders/%d", o.ID), "")
	if n := len(sink.all()); n != 0 {
		t.Fatalf("operator got %d notices for a canceled payment", n)
	}
}
