package telegram

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
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

	if err := SendMessage("tok", 123456, "Готово!"); err != nil {
		t.Fatalf("SendMessage: %v", err)
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
