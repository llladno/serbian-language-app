package telegram

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// withTestServer points apiBase at a local httptest server for the duration
// of the test and restores it afterward.
func withTestServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	prev := apiBase
	apiBase = srv.URL
	t.Cleanup(func() { apiBase = prev })
}

func TestGetMeReturnsUsername(t *testing.T) {
	var gotPath string
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":{"id":42,"username":"ucimoappbot"}}`))
	})

	username, err := GetMe("test-token")
	if err != nil {
		t.Fatalf("GetMe: %v", err)
	}
	if username != "ucimoappbot" {
		t.Errorf("username = %q, want %q", username, "ucimoappbot")
	}
	if gotPath != "/bottest-token/getMe" {
		t.Errorf("path = %q, want the bot token embedded in the path", gotPath)
	}
}

func TestGetMeAPIError(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":false,"error_code":401,"description":"Unauthorized"}`))
	})

	_, err := GetMe("bad-token")
	if err == nil {
		t.Fatal("GetMe: want error for ok:false response")
	}
}

func TestSetWebhookSendsURLAndSecret(t *testing.T) {
	var gotBody string
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotBody = r.Form.Encode()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":true}`))
	})

	if err := SetWebhook("tok", "https://ucimo.ru/api/telegram/webhook", "s3cr3t"); err != nil {
		t.Fatalf("SetWebhook: %v", err)
	}
	form, err := url.ParseQuery(gotBody)
	if err != nil {
		t.Fatal(err)
	}
	if form.Get("url") != "https://ucimo.ru/api/telegram/webhook" {
		t.Errorf("url = %q", form.Get("url"))
	}
	if form.Get("secret_token") != "s3cr3t" {
		t.Errorf("secret_token = %q", form.Get("secret_token"))
	}
}

func TestSendMessageSendsChatIDAndText(t *testing.T) {
	var gotBody string
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotBody = r.Form.Encode()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":{}}`))
	})

	if err := SendMessageWithButton("tok", 123456, "Готово!", nil); err != nil {
		t.Fatalf("SendMessageWithButton: %v", err)
	}
	form, err := url.ParseQuery(gotBody)
	if err != nil {
		t.Fatal(err)
	}
	if form.Get("chat_id") != "123456" {
		t.Errorf("chat_id = %q", form.Get("chat_id"))
	}
	if form.Get("text") != "Готово!" {
		t.Errorf("text = %q", form.Get("text"))
	}
}

func TestSetChatMenuButtonSendsWebAppButton(t *testing.T) {
	var gotBody string
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotBody = r.Form.Encode()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":true}`))
	})

	if err := SetChatMenuButton("tok", "https://ucimo.ru/profile", "Открыть ucimo"); err != nil {
		t.Fatalf("SetChatMenuButton: %v", err)
	}
	form, err := url.ParseQuery(gotBody)
	if err != nil {
		t.Fatal(err)
	}
	var button struct {
		Type   string `json:"type"`
		Text   string `json:"text"`
		WebApp struct {
			URL string `json:"url"`
		} `json:"web_app"`
	}
	if err := json.Unmarshal([]byte(form.Get("menu_button")), &button); err != nil {
		t.Fatalf("unmarshal menu_button: %v", err)
	}
	if button.Type != "web_app" {
		t.Errorf("type = %q, want web_app", button.Type)
	}
	if button.Text != "Открыть ucimo" {
		t.Errorf("text = %q", button.Text)
	}
	if button.WebApp.URL != "https://ucimo.ru/profile" {
		t.Errorf("web_app.url = %q", button.WebApp.URL)
	}
}

func TestParseStartToken(t *testing.T) {
	tests := []struct {
		text      string
		wantToken string
		wantOK    bool
	}{
		{"/start abc123", "abc123", true},
		{"/start@ucimoappbot abc123", "abc123", true},
		{"  /start   abc123  ", "abc123", true},
		{"/start", "", false},
		{"/help", "", false},
		{"hello", "", false},
		{"/start abc def", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got, ok := ParseStartToken(tt.text)
		if got != tt.wantToken || ok != tt.wantOK {
			t.Errorf("ParseStartToken(%q) = (%q, %v), want (%q, %v)", tt.text, got, ok, tt.wantToken, tt.wantOK)
		}
	}
}

func TestSendMessageWithButtonWebApp(t *testing.T) {
	var gotBody string
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotBody = r.Form.Encode()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":true}`))
	})
	err := SendMessageWithButton("tok", 555, "hi", &InlineButton{Label: "Открыть", WebAppURL: "https://ucimo.ru/profile"})
	if err != nil {
		t.Fatalf("SendMessageWithButton: %v", err)
	}
	form, err := url.ParseQuery(gotBody)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(form.Get("reply_markup"), `"web_app":{"url":"https://ucimo.ru/profile"}`) {
		t.Errorf("reply_markup = %s, want a web_app button", form.Get("reply_markup"))
	}
}

func TestSendMessageWithButtonURL(t *testing.T) {
	var gotBody string
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotBody = r.Form.Encode()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":true}`))
	})
	err := SendMessageWithButton("tok", 555, "hi", &InlineButton{Label: "Поддержка", URL: SupportURL})
	if err != nil {
		t.Fatalf("SendMessageWithButton: %v", err)
	}
	form, err := url.ParseQuery(gotBody)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(form.Get("reply_markup"), `"url":"`+SupportURL+`"`) {
		t.Errorf("reply_markup = %s, want a url button to %s", form.Get("reply_markup"), SupportURL)
	}
}

func TestSendMessageWithButtonNilOmitsMarkup(t *testing.T) {
	var gotBody string
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotBody = r.Form.Encode()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":true}`))
	})
	if err := SendMessageWithButton("tok", 555, "hi", nil); err != nil {
		t.Fatalf("SendMessageWithButton: %v", err)
	}
	form, err := url.ParseQuery(gotBody)
	if err != nil {
		t.Fatal(err)
	}
	if form.Get("reply_markup") != "" {
		t.Errorf("reply_markup = %q, want empty for a nil button", form.Get("reply_markup"))
	}
}

func TestRateLimitedParsesRetryAfter(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"ok":false,"error_code":429,"description":"Too Many Requests","parameters":{"retry_after":7}}`))
	})
	err := SendMessageWithButton("tok", 1, "hi", nil)
	if err == nil {
		t.Fatal("SendMessageWithButton: want error for a 429 response")
	}
	wait, ok := RateLimited(err)
	if !ok || wait != 7*time.Second {
		t.Errorf("RateLimited = (%v, %v), want (7s, true)", wait, ok)
	}
}

func TestRateLimitedFalseForOtherErrors(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":false,"error_code":400,"description":"Bad Request"}`))
	})
	err := SendMessageWithButton("tok", 1, "hi", nil)
	if err == nil {
		t.Fatal("SendMessageWithButton: want error for a 400 response")
	}
	if _, ok := RateLimited(err); ok {
		t.Errorf("RateLimited(400 error) = true, want false")
	}
}

func TestIsBareStart(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"/start", true},
		{"/start@ucimoappbot", true},
		{"/start abc123", false},
		{"/start@ucimoappbot abc123", false},
		{"hello", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsBareStart(c.text); got != c.want {
			t.Errorf("IsBareStart(%q) = %v, want %v", c.text, got, c.want)
		}
	}
}

func TestGetChatMember(t *testing.T) {
	cases := []struct {
		name, body string
		want       string
		wantMember bool
		wantErr    bool
	}{
		{"member", `{"ok":true,"result":{"status":"member"}}`, "member", true, false},
		{"admin", `{"ok":true,"result":{"status":"administrator"}}`, "administrator", true, false},
		{"creator", `{"ok":true,"result":{"status":"creator"}}`, "creator", true, false},
		{"left", `{"ok":true,"result":{"status":"left"}}`, "left", false, false},
		{"kicked", `{"ok":true,"result":{"status":"kicked"}}`, "kicked", false, false},
		// A restricted user who is still in the chat is subscribed; one who is
		// not is reported as "left", because "restricted" alone would otherwise
		// pay out the quest to someone who has left.
		{"restricted and present", `{"ok":true,"result":{"status":"restricted","is_member":true}}`, "restricted", true, false},
		{"restricted and gone", `{"ok":true,"result":{"status":"restricted","is_member":false}}`, "left", false, false},
		{"not found", `{"ok":false,"error_code":400,"description":"Bad Request: user not found"}`, "", false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(c.body))
			})

			status, err := GetChatMember("token", "@ucimo", 12345)
			if c.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetChatMember: %v", err)
			}
			if status != c.want {
				t.Fatalf("status = %q, want %q", status, c.want)
			}
			if IsMember(status) != c.wantMember {
				t.Fatalf("IsMember(%q) = %v, want %v", status, IsMember(status), c.wantMember)
			}
		})
	}
}

func TestGetChatMemberSendsChatAndUser(t *testing.T) {
	var gotPath, gotBody string
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotPath, gotBody = r.URL.Path, r.Form.Encode()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":{"status":"member"}}`))
	})

	if _, err := GetChatMember("test-token", "@ucimo", 12345); err != nil {
		t.Fatalf("GetChatMember: %v", err)
	}
	if gotPath != "/bottest-token/getChatMember" {
		t.Errorf("path = %q, want /bottest-token/getChatMember", gotPath)
	}
	if !strings.Contains(gotBody, "chat_id=%40ucimo") || !strings.Contains(gotBody, "user_id=12345") {
		t.Errorf("body = %q, want the chat and user id", gotBody)
	}
}
