package api

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/grisha/serbian-app/server/internal/leadmagnet"
	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
)

// The lead-magnet flow: a post links to t.me/<bot>?start=lm_<slug>[_<source>]. The bot checks
// the channel subscription straight away; a subscriber gets the file at once, anyone else is
// asked to subscribe and press "check", and the file follows. Every step is counted in
// lead_magnet_claims, so the funnel (followed the link / got the file / subscribed for it)
// can be read with the bot's /stats command.
//
// Unlike the /start login replies these go out directly, not through bot_outbox: the outbox
// only carries text with one button, a file needs its own call, and each of these is a single
// reply to something the person just pressed.

const (
	leadPayloadPrefix  = "lm_" // start payload: lm_<slug>[_<source>]
	leadCallbackPrefix = "lm:" // callback_data of the "check" button: lm:<slug>:<source>
	leadDefaultSource  = "direct"
	leadMaxSource      = 24
)

// The bot's wording for this flow. {name} is the person's first name.
const (
	leadPromptText = "Привет, {name}! 👋 Топ-100 сербских слов: PDF с переводом, транскрипцией и примерами.\n\n" +
		"Чтобы забрать файл, подпишись на наш канал: нажми «Подписаться на канал», а потом «Проверить подписку». Файл придёт сюда."
	leadNotYetText = "Пока не вижу подписки, {name} 🤔\n\n" +
		"Подпишись на канал и нажми «Проверить подписку» ещё раз."
	leadCaptionText = "Держи, {name}! 📘 Топ-100 сербских слов с переводом, транскрипцией и примерами.\n\n" +
		"Сохрани файл, чтобы не потерять. А учить эти слова удобнее в приложении: уроки, озвучка и повторение. Три первых уровня бесплатно."
	leadCantCheckText = "Не получилось проверить подписку 😔\n\nПопробуй ещё раз через минуту: нажми «Проверить подписку»."
	leadFailedText    = "Упс, не получилось отправить файл 😔\n\nНапиши нам, пришлём вручную: @ucimosupport"
)

// leadSleep is how a 429 from Telegram is waited out; a var so tests do not.
var leadSleep = time.Sleep

// parseLeadPayload splits a start payload "lm_top100_tgpost" into slug and source.
func parseLeadPayload(token string) (slug, source string, ok bool) {
	rest, found := strings.CutPrefix(token, leadPayloadPrefix)
	if !found {
		return "", "", false
	}
	slug, source, _ = strings.Cut(rest, "_")
	if slug == "" {
		return "", "", false
	}
	return slug, cleanSource(source), true
}

func cleanSource(s string) string {
	if s == "" {
		return leadDefaultSource
	}
	if len(s) > leadMaxSource {
		s = s[:leadMaxSource]
	}
	return s
}

func leadCheckData(slug, source string) string { return leadCallbackPrefix + slug + ":" + source }

func parseLeadCallback(data string) (slug, source string, ok bool) {
	rest, found := strings.CutPrefix(data, leadCallbackPrefix)
	if !found {
		return "", "", false
	}
	slug, source, _ = strings.Cut(rest, ":")
	if slug == "" {
		return "", "", false
	}
	return slug, cleanSource(source), true
}

// leadKeyboard is the subscribe-and-check keyboard whose check button answers for this magnet,
// or nil when no channel is configured.
func (h handlers) leadKeyboard(slug, source string) *telegram.InlineButton {
	kb := h.channelKeyboard()
	if kb == nil {
		return nil
	}
	kb.CheckData = leadCheckData(slug, source)
	return kb
}

// channelConfigured reports whether there is a channel to gate on.
func (h handlers) channelConfigured() bool {
	set, err := h.Store.EconomySettings()
	return err == nil && set.TelegramChannel != ""
}

// leadSay sends a text reply right away, waiting out a rate limit once or twice.
func (h handlers) leadSay(chatID int64, text, name string, button *telegram.InlineButton) {
	text = telegram.Substitute(text, name)
	_, err := withRetry(func() (string, error) {
		return "", telegram.SendMessageWithButton(h.Config.TelegramBotToken, chatID, text, button)
	})
	if err != nil {
		log.Printf("lead magnet: send message to %d: %v", chatID, err)
	}
}

// withRetry runs send, and on a Telegram 429 waits the requested time (at most a few
// seconds) and tries again, up to three attempts in all.
func withRetry(send func() (string, error)) (string, error) {
	var (
		id  string
		err error
	)
	for attempt := 0; attempt < 3; attempt++ {
		id, err = send()
		wait, limited := telegram.RateLimited(err)
		if err == nil || !limited {
			return id, err
		}
		if wait > 5*time.Second {
			wait = 5 * time.Second
		}
		leadSleep(wait)
	}
	return id, err
}

// leadMagnetStart handles /start lm_<slug>[_<source>].
func (h handlers) leadMagnetStart(chatID, tgID int64, name, slug, source string) {
	magnet, ok := leadmagnet.Get(slug)
	if !ok {
		h.sendBotMessage(chatID, telegram.MsgStartGreeting, name, h.miniAppButton())
		return
	}
	if !h.channelConfigured() {
		// Nothing to gate on: record the visit and hand the file over.
		_ = h.Store.LeadMagnetStart(slug, tgID, source, false, h.Now())
		h.deliverLeadMagnet(chatID, tgID, name, magnet)
		return
	}
	member, err := h.memberOfChannel(tgID)
	if err != nil {
		log.Printf("lead magnet: check channel for %d: %v", tgID, err)
		_ = h.Store.LeadMagnetStart(slug, tgID, source, false, h.Now())
		h.leadSay(chatID, leadCantCheckText, name, h.leadKeyboard(slug, source))
		return
	}
	if err := h.Store.LeadMagnetStart(slug, tgID, source, member, h.Now()); err != nil {
		log.Printf("lead magnet: record start: %v", err)
	}
	if member {
		h.deliverLeadMagnet(chatID, tgID, name, magnet)
		return
	}
	h.leadSay(chatID, leadPromptText, name, h.leadKeyboard(slug, source))
}

// leadMagnetCheckPressed answers the "check subscription" button of the lead-magnet flow.
func (h handlers) leadMagnetCheckPressed(queryID, data string, tgID int64, name string) {
	answer := func(text string) {
		if err := telegram.AnswerCallbackQuery(h.Config.TelegramBotToken, queryID, text); err != nil {
			log.Printf("lead magnet: answer callback: %v", err)
		}
	}
	slug, source, ok := parseLeadCallback(data)
	magnet, known := leadmagnet.Get(slug)
	if !ok || !known {
		answer("")
		return
	}
	chatID := tgID // a private chat's id is the user's id
	if !h.channelConfigured() {
		answer("")
		h.deliverLeadMagnet(chatID, tgID, name, magnet)
		return
	}
	member, err := h.memberOfChannel(tgID)
	if err != nil {
		log.Printf("lead magnet: check channel for %d: %v", tgID, err)
		answer("Не получилось проверить, попробуй ещё раз")
		return
	}
	// Normally the visit was recorded at /start (and this is then a no-op); it only writes for a
	// button pressed on a message that predates the tracking.
	_ = h.Store.LeadMagnetStart(slug, tgID, source, false, h.Now())
	if !member {
		answer("Подписки пока не видно")
		h.leadSay(chatID, leadNotYetText, name, h.leadKeyboard(slug, source))
		return
	}
	answer("Подписка найдена ✅")
	h.deliverLeadMagnet(chatID, tgID, name, magnet)
}

// deliverLeadMagnet sends the file, by cached file_id when there is one (and by upload when not,
// remembering the id Telegram returns), and counts the delivery.
func (h handlers) deliverLeadMagnet(chatID, tgID int64, name string, m leadmagnet.Magnet) {
	token := h.Config.TelegramBotToken
	caption := telegram.Substitute(leadCaptionText, name)

	cached, _, err := h.Store.LeadMagnetFileID(m.Slug, m.SHA)
	if err != nil {
		log.Printf("lead magnet: file id lookup: %v", err)
	}
	send := func(doc telegram.Document) (string, error) {
		return withRetry(func() (string, error) {
			return telegram.SendDocument(token, chatID, doc, caption, h.miniAppButton())
		})
	}
	var id string
	if cached != "" {
		id, err = send(telegram.Document{FileID: cached})
		if err != nil {
			log.Printf("lead magnet: resend by file_id failed, uploading again: %v", err)
			cached = ""
		}
	}
	if cached == "" {
		id, err = send(telegram.Document{Filename: m.Filename, Data: m.Data})
		if err == nil && id != "" {
			if saveErr := h.Store.SaveLeadMagnetFileID(m.Slug, m.SHA, id, h.Now()); saveErr != nil {
				log.Printf("lead magnet: save file id: %v", saveErr)
			}
		}
	}
	if err != nil {
		log.Printf("lead magnet: send %s to %d: %v", m.Slug, tgID, err)
		h.leadSay(chatID, leadFailedText, name, h.supportButton())
		return
	}
	if err := h.Store.LeadMagnetDelivered(m.Slug, tgID, h.Now()); err != nil {
		log.Printf("lead magnet: record delivery: %v", err)
	}
}

// botCommand handles the bot's own commands that are not part of the login flow: /id (anyone:
// shows their Telegram id, which is what goes into TELEGRAM_ADMIN_IDS) and /stats (admins only:
// the lead-magnet funnel). Returns true when text was one of them. A /stats from a non-admin
// is swallowed without an answer, so the command does not announce itself.
func (h handlers) botCommand(chatID, fromID int64, name, text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return false
	}
	cmd, _, _ := strings.Cut(fields[0], "@")
	switch cmd {
	case "/id":
		h.leadSay(chatID, fmt.Sprintf("Твой Telegram ID: %d", fromID), name, nil)
		return true
	case "/stats":
		if h.Config.IsTelegramAdmin(fromID) {
			h.leadSay(chatID, h.leadStatsText("top100"), name, nil)
		}
		return true
	}
	return false
}

// leadStatsText formats the funnel of one magnet for /stats.
func (h handlers) leadStatsText(slug string) string {
	st, err := h.Store.LeadMagnetStats(slug)
	if err != nil {
		log.Printf("lead magnet: stats: %v", err)
		return "Не получилось посчитать статистику"
	}
	return formatLeadStats(slug, st)
}

func formatLeadStats(slug string, st store.LeadStats) string {
	var b strings.Builder
	fmt.Fprintf(&b, "📊 Топ-100 слов (PDF)\n\n")
	fmt.Fprintf(&b, "Перешли по ссылке: %d\n", st.Started)
	fmt.Fprintf(&b, "• уже были подписаны: %d\n", st.AlreadyMember)
	fmt.Fprintf(&b, "• подписались ради файла: %d\n", st.SubscribedForIt)
	fmt.Fprintf(&b, "• ждут (не подписались или не нажали «проверить»): %d\n", st.Waiting)
	fmt.Fprintf(&b, "Получили файл: %d", st.Received)
	if st.Started > 0 {
		fmt.Fprintf(&b, " (%d%%)", st.Received*100/st.Started)
	}
	if len(st.BySource) > 0 {
		b.WriteString("\n\nПо источникам (ссылка lm_" + slug + "_<источник>):")
		for _, s := range st.BySource {
			fmt.Fprintf(&b, "\n• %s: перешли %d, получили %d, подписались %d", s.Source, s.Started, s.Received, s.SubscribedForIt)
		}
	}
	return b.String()
}
