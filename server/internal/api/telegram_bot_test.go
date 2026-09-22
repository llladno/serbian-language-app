package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
)

// sentTgMessage is one message captured by tgSink.
type sentTgMessage struct {
	ChatID   int64
	Text     string
	Button   *telegram.InlineButton
	Priority int
}

// tgSink is a capturing EnqueueTelegramMessage.
type tgSink struct {
	mu   sync.Mutex
	msgs []sentTgMessage
}

func (s *tgSink) enqueue(chatID int64, text string, button *telegram.InlineButton, priority int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, sentTgMessage{chatID, text, button, priority})
}

func (s *tgSink) all() []sentTgMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]sentTgMessage(nil), s.msgs...)
}

const tgWebhookSecret = "test-webhook-secret"
const tgBotUsername = "ucimoappbot"

// newTelegramBotAPI is newTelegramAPI plus the /start flow's wiring: a bot
// username (so telegramStartFor can build a t.me URL), a webhook secret, and
// a capturing SendTelegramMessage.
func newTelegramBotAPI(t *testing.T) (http.Handler, *store.Store, *tgSink) {
	t.Helper()
	sink := &tgSink{}
	h, st, _ := newAuthAPI(t, func(d *Deps) {
		d.Config.TelegramBotToken = tgTestToken
		d.TelegramBotUsername = tgBotUsername
		d.TelegramWebhookSecret = tgWebhookSecret
		d.EnqueueTelegramMessage = sink.enqueue
	})
	return h, st, sink
}

// startToken decodes a telegramLoginStart/telegramLinkStart response body,
// {"url":"https://t.me/<bot>?start=<token>","token":"<token>"}.
func startToken(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	got := decodeBody[struct {
		URL   string `json:"url"`
		Token string `json:"token"`
	}](t, rr)
	if got.Token == "" {
		t.Fatalf("no start token in body: %s", rr.Body.String())
	}
	return got.Token
}

// webhookUpdate builds a minimal Telegram Update JSON for a text message.
func webhookUpdate(chatID, fromID int64, username, firstName, text string) string {
	return fmt.Sprintf(
		`{"message":{"chat":{"id":%d},"text":%q,"from":{"id":%d,"username":%q,"first_name":%q}}}`,
		chatID, text, fromID, username, firstName,
	)
}

// postWebhook posts an update to the webhook with the given secret header
// (empty = header omitted entirely, matching a request that never carried
// one). The webhook is exempt from checkOrigin, so no Origin header is set.
func postWebhook(h http.Handler, secret, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/api/telegram/webhook", strings.NewReader(body))
	if secret != "" {
		r.Header.Set("X-Telegram-Bot-Api-Secret-Token", secret)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	return rr
}

func TestTelegramLoginStartReturnsBotURL(t *testing.T) {
	h, _, _ := newTelegramBotAPI(t)

	rr := anon(h, "POST", "/api/auth/telegram/start", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("start = %d %s", rr.Code, rr.Body)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "https://t.me/"+tgBotUsername+"?start=") {
		t.Errorf("body = %s, want a t.me start URL for %s", body, tgBotUsername)
	}
	startToken(t, rr) // must parse — panics the test via t.Fatalf otherwise
}

func TestTelegramLoginStartDisabledWithoutToken(t *testing.T) {
	h, _ := newTestAPI(t) // no TelegramBotToken configured
	rr := anon(h, "POST", "/api/auth/telegram/start", "")
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("start with no token = %d, want 503", rr.Code)
	}
}

func TestTelegramPollUnknownTokenIsError(t *testing.T) {
	h, _, _ := newTelegramBotAPI(t)
	rr := anon(h, "GET", "/api/auth/telegram/poll?token=never-issued", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("poll = %d %s", rr.Code, rr.Body)
	}
	if !strings.Contains(rr.Body.String(), `"status":"error"`) {
		t.Errorf("body = %s, want status:error", rr.Body.String())
	}
}

func TestTelegramWebhookWrongSecretIs401(t *testing.T) {
	h, _, _ := newTelegramBotAPI(t)
	rr := postWebhook(h, "wrong-secret", webhookUpdate(1, 42, "neo", "Neo", "/start abc"))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("webhook wrong secret = %d, want 401", rr.Code)
	}
}

func TestTelegramWebhookMissingSecretIs401(t *testing.T) {
	h, _, _ := newTelegramBotAPI(t)
	rr := postWebhook(h, "", webhookUpdate(1, 42, "neo", "Neo", "/start abc"))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("webhook missing secret = %d, want 401", rr.Code)
	}
}

func TestTelegramWebhookNonStartMessageIsNoop(t *testing.T) {
	h, st, sink := newTelegramBotAPI(t)
	before, _ := st.ListUsers()

	rr := postWebhook(h, tgWebhookSecret, webhookUpdate(1, 42, "neo", "Neo", "hello there"))
	if rr.Code != http.StatusOK {
		t.Fatalf("webhook non-start = %d, want 200", rr.Code)
	}
	after, _ := st.ListUsers()
	if len(after) != len(before) {
		t.Errorf("user count changed from %d to %d for a non-/start message", len(before), len(after))
	}
	if len(sink.all()) != 0 {
		t.Errorf("sent %d telegram messages for a non-/start message, want 0", len(sink.all()))
	}
}

func TestTelegramLoginFullRoundTrip(t *testing.T) {
	h, st, sink := newTelegramBotAPI(t)

	startRR := anon(h, "POST", "/api/auth/telegram/start", "")
	token := startToken(t, startRR)

	// Before the webhook fires, poll must report pending.
	pollBefore := anon(h, "GET", "/api/auth/telegram/poll?token="+url.QueryEscape(token), "")
	if !strings.Contains(pollBefore.Body.String(), `"status":"pending"`) {
		t.Fatalf("poll before webhook = %s, want pending", pollBefore.Body.String())
	}

	webhookRR := postWebhook(h, tgWebhookSecret, webhookUpdate(555, 424242, "neo_bot", "Neo", "/start "+token))
	if webhookRR.Code != http.StatusOK {
		t.Fatalf("webhook = %d %s", webhookRR.Code, webhookRR.Body)
	}
	if msgs := sink.all(); len(msgs) != 1 || msgs[0].ChatID != 555 {
		t.Fatalf("sent messages = %+v, want exactly one to chat 555", msgs)
	}

	pollAfter := anon(h, "GET", "/api/auth/telegram/poll?token="+url.QueryEscape(token), "")
	if pollAfter.Code != http.StatusOK {
		t.Fatalf("poll after webhook = %d %s", pollAfter.Code, pollAfter.Body)
	}
	if !strings.Contains(pollAfter.Body.String(), `"status":"ok"`) {
		t.Fatalf("poll after webhook = %s, want status ok", pollAfter.Body.String())
	}
	grabSessionCookie(t, pollAfter) // the poll response itself must carry the new session

	id, err := st.IdentityByProviderUID("telegram", "424242")
	if err != nil {
		t.Fatalf("identity after login: %v", err)
	}
	if id.TgUsername != "neo_bot" {
		t.Errorf("tg_username = %q, want neo_bot", id.TgUsername)
	}

	// The token is one-shot: a second poll must not still report done.
	pollAgain := anon(h, "GET", "/api/auth/telegram/poll?token="+url.QueryEscape(token), "")
	if strings.Contains(pollAgain.Body.String(), `"status":"ok"`) {
		t.Errorf("second poll = %s, want the token to be consumed (not replayable)", pollAgain.Body.String())
	}
}

func TestTelegramLoginClaimsPendingUsername(t *testing.T) {
	h, st, _ := newTelegramBotAPI(t)
	uid, err := st.CreateUser("Гриша")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateIdentity(store.Identity{
		ID: "idn_seed", UserID: uid, Provider: "telegram",
		ProviderUID: "pending:llladnooo", TgUsername: "llladnooo",
	}); err != nil {
		t.Fatal(err)
	}

	startRR := anon(h, "POST", "/api/auth/telegram/start", "")
	token := startToken(t, startRR)
	postWebhook(h, tgWebhookSecret, webhookUpdate(1, 999999, "llladnooo", "Grisha", "/start "+token))

	poll := anon(h, "GET", "/api/auth/telegram/poll?token="+url.QueryEscape(token), "")
	got := decodeBody[pollResponseDTO](t, poll)
	if got.User == nil || got.User.ID != uid {
		t.Errorf("resolved user = %+v, want the pre-existing account %q (claim by username)", got.User, uid)
	}
}

// pollResponseDTO mirrors telegramPoll's JSON shape for test assertions.
type pollResponseDTO struct {
	Status string          `json:"status"`
	User   *sessionUserDTO `json:"user"`
	Error  string          `json:"error"`
}

func TestTelegramLinkFullRoundTrip(t *testing.T) {
	h, st, _ := newTelegramBotAPI(t)
	uid := registerAndVerify(t, h, st, "linker@example.com", "secret1234", "Linker")
	cookie := authed(t, st, uid)

	startRR := doCookie(h, cookie, "POST", "/api/me/telegram/start", "")
	if startRR.Code != http.StatusOK {
		t.Fatalf("link start = %d %s", startRR.Code, startRR.Body)
	}
	token := startToken(t, startRR)

	postWebhook(h, tgWebhookSecret, webhookUpdate(1, 777, "linkeduser", "Link", "/start "+token))

	poll := anon(h, "GET", "/api/auth/telegram/poll?token="+url.QueryEscape(token), "")
	if !strings.Contains(poll.Body.String(), `"status":"ok"`) {
		t.Fatalf("poll after link webhook = %s, want status ok", poll.Body.String())
	}
	// A link poll must NOT mint a fresh session — the caller already has one.
	for _, c := range poll.Result().Cookies() {
		if c.Name == "session" {
			t.Errorf("link poll set a new session cookie; want none (caller already authenticated)")
		}
	}

	id, err := st.IdentityByProviderUID("telegram", "777")
	if err != nil {
		t.Fatalf("identity after link: %v", err)
	}
	if id.UserID != uid {
		t.Errorf("linked identity user_id = %q, want the caller's own %q", id.UserID, uid)
	}
}

func TestTelegramLinkTakenReportsError(t *testing.T) {
	h, st, sink := newTelegramBotAPI(t)
	ownerID := registerAndVerify(t, h, st, "owner@example.com", "secret1234", "Owner")
	if err := st.CreateIdentity(store.Identity{
		ID: "idn_owner", UserID: ownerID, Provider: "telegram", ProviderUID: "888", TgUsername: "taken",
	}); err != nil {
		t.Fatal(err)
	}

	otherID := registerAndVerify(t, h, st, "other@example.com", "secret1234", "Other")
	cookie := authed(t, st, otherID)

	startRR := doCookie(h, cookie, "POST", "/api/me/telegram/start", "")
	token := startToken(t, startRR)

	postWebhook(h, tgWebhookSecret, webhookUpdate(1, 888, "taken", "Taken", "/start "+token))
	if msgs := sink.all(); len(msgs) != 1 {
		t.Fatalf("sent messages = %+v, want exactly one (the taken notice)", msgs)
	}

	poll := anon(h, "GET", "/api/auth/telegram/poll?token="+url.QueryEscape(token), "")
	if !strings.Contains(poll.Body.String(), `"error":"telegram_taken"`) {
		t.Errorf("poll after taken webhook = %s, want error telegram_taken", poll.Body.String())
	}

	// The other account must not have gained the identity.
	if id, err := st.IdentityByProviderUID("telegram", "888"); err != nil || id.UserID != ownerID {
		t.Errorf("identity after failed link: id=%+v err=%v, want unchanged owner %q", id, err, ownerID)
	}
}

func TestTelegramWebhookUnknownTokenSendsNoMessage(t *testing.T) {
	h, _, sink := newTelegramBotAPI(t)
	rr := postWebhook(h, tgWebhookSecret, webhookUpdate(1, 42, "neo", "Neo", "/start never-issued-token"))
	if rr.Code != http.StatusOK {
		t.Fatalf("webhook unknown token = %d, want 200", rr.Code)
	}
	if len(sink.all()) != 0 {
		t.Errorf("sent %d messages for an unknown token, want 0", len(sink.all()))
	}
}
