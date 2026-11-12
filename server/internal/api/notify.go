package api

import (
	"fmt"
	"html"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/grisha/serbian-app/server/internal/config"
	"github.com/grisha/serbian-app/server/internal/store"
)

// maxNoticeMessage caps a quoted support message so the whole notice stays under Telegram's 4096-character limit.
const maxNoticeMessage = 3000

// notifyAdmin sends one HTML notice to the operator, off the request's path. A nil NotifyAdmin (not configured, or a
// test that does not care) sends nothing.
func (h handlers) notifyAdmin(text string) {
	if h.NotifyAdmin == nil {
		return
	}
	h.Async(func() { h.NotifyAdmin(text) })
}

// notifyPayment tells the operator a payment has just been completed. It never fails the payment: whatever it cannot
// look up is left out of the notice.
func (h handlers) notifyPayment(p store.Payment) {
	if h.NotifyAdmin == nil {
		return
	}
	n := paymentNotice{
		AmountKopecks: p.AmountKopecks,
		Provider:      p.Provider,
		Test:          p.Test,
		PaymentID:     p.ID,
		TariffCode:    p.TariffCode,
		ReceiptTo:     p.ReceiptContact,
		Who:           h.noticeWho(p.UserID),
	}
	if t, err := h.Store.Tariff(p.TariffCode); err == nil {
		n.TariffTitle = t.Title
		n.Bonus = t.BonusCurrency
	} else {
		log.Printf("notify: tariff %s: %v", p.TariffCode, err)
	}
	h.notifyAdmin(n.String())
}

// notifySupport tells the operator about a new support message.
func (h handlers) notifySupport(userID, message string) {
	if h.NotifyAdmin == nil {
		return
	}
	h.notifyAdmin(supportNotice{Who: h.noticeWho(userID), Message: message}.String())
}

// noticeWho collects what the operator needs to recognise (and answer) a learner.
func (h handlers) noticeWho(userID string) noticeWho {
	w := noticeWho{UserID: userID}
	if u, err := h.Store.UserByID(userID); err == nil {
		w.Name = u.Name
	}
	if email, ok := h.Store.VerifiedEmail(userID); ok {
		w.Email = email
	}
	if it, err := h.Store.IdentityForUser(userID, "telegram"); err == nil {
		w.TgUsername = it.TgUsername
	}
	return w
}

// noticeWho is who a notice is about.
type noticeWho struct {
	UserID     string
	Name       string
	Email      string
	TgUsername string
}

// lines is the "who" block: the name on top, then the ways to reach them.
func (w noticeWho) lines() string {
	name := strings.TrimSpace(w.Name)
	if name == "" {
		name = "без имени"
	}
	out := "👤 <b>" + html.EscapeString(name) + "</b>"
	if w.Email != "" {
		out += "\n✉️ " + html.EscapeString(w.Email)
	}
	if tg := strings.TrimPrefix(w.TgUsername, "@"); tg != "" {
		out += "\n✈️ <a href=\"https://t.me/" + html.EscapeString(tg) + "\">@" + html.EscapeString(tg) + "</a>"
	}
	return out
}

// paymentNotice is the operator's message about a completed payment.
type paymentNotice struct {
	TariffTitle   string
	TariffCode    string
	AmountKopecks int64
	Bonus         int64
	Provider      string
	Test          bool
	PaymentID     int64
	ReceiptTo     string
	Who           noticeWho
}

func (n paymentNotice) String() string {
	var b strings.Builder
	if n.Test {
		b.WriteString("🧪 <b>Тестовая оплата</b>\n\n")
	} else {
		b.WriteString("💸 <b>Новая оплата!</b>\n\n")
	}
	title := n.TariffTitle
	if title == "" {
		title = n.TariffCode
	}
	fmt.Fprintf(&b, "🎟 %s\n💰 <b>%s</b>", html.EscapeString(title), formatRubles(n.AmountKopecks))
	if n.Bonus > 0 {
		fmt.Fprintf(&b, "\n🌱 + %d зёрнышек в подарок", n.Bonus)
	}
	b.WriteString("\n\n" + n.Who.lines())
	if n.ReceiptTo != "" && n.ReceiptTo != n.Who.Email {
		b.WriteString("\n🧾 чек: " + html.EscapeString(n.ReceiptTo))
	}
	fmt.Fprintf(&b, "\n\n<i>%s · заказ №%d</i>", html.EscapeString(providerName(n.Provider)), n.PaymentID)
	return b.String()
}

// supportNotice is the operator's message about a new support message.
type supportNotice struct {
	Who     noticeWho
	Message string
}

func (n supportNotice) String() string {
	msg := strings.TrimSpace(n.Message)
	if utf8.RuneCountInString(msg) > maxNoticeMessage {
		msg = string([]rune(msg)[:maxNoticeMessage]) + "…"
	}
	return "💬 <b>Новое обращение в поддержку</b>\n\n" + n.Who.lines() +
		"\n\n<blockquote>" + html.EscapeString(msg) + "</blockquote>"
}

// formatRubles writes kopecks as "299 ₽", or "299,50 ₽" when they are not whole roubles. The thousands are
// separated by a no-break space: "2 000 ₽".
func formatRubles(kopecks int64) string {
	rub, kop := kopecks/100, kopecks%100
	digits := fmt.Sprintf("%d", rub)
	var grouped []byte
	for i, c := range []byte(digits) {
		if i > 0 && (len(digits)-i)%3 == 0 {
			grouped = append(grouped, []byte(" ")...)
		}
		grouped = append(grouped, c)
	}
	if kop == 0 {
		return string(grouped) + " ₽"
	}
	return fmt.Sprintf("%s,%02d ₽", grouped, kop)
}

// providerName is the payment service as the operator knows it.
func providerName(p string) string {
	switch p {
	case config.ProviderTBank:
		return "Т-Касса"
	case config.ProviderDev:
		return "тестовая касса"
	case "":
		return "оплата"
	}
	return p
}
