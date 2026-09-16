// Package telegram is a thin client for the Telegram Bot API — just the
// calls the bot-login flow needs (GetMe, SetWebhook, SendMessage,
// SetChatMenuButton). It does not touch signature verification
// (initData/widget HMAC checks live in internal/auth) or webhook update
// parsing (that's api-layer concern, since only a couple of fields off the
// update actually matter).
package telegram

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// apiBase is the Bot API origin. A var (not const) so tests can point it at
// an httptest server instead of the real Telegram servers.
var apiBase = "https://api.telegram.org"

// httpClient is used for every call; a package-level var with a sane timeout
// so a hung Telegram request cannot block a caller forever.
var httpClient = &http.Client{Timeout: 10 * time.Second}

// apiError is the shape Telegram's Bot API returns on ok:false.
type apiError struct {
	Description string `json:"description"`
	ErrorCode   int    `json:"error_code"`
}

func (e apiError) Error() string {
	return fmt.Sprintf("telegram: %d %s", e.ErrorCode, e.Description)
}

// call POSTs form-encoded params to method and decodes the result field of a
// successful response into out (nil to discard it). A non-2xx-shaped
// {"ok":false,...} response is returned as an apiError.
func call(botToken, method string, params url.Values, out any) error {
	endpoint := apiBase + "/bot" + botToken + "/" + method
	res, err := httpClient.PostForm(endpoint, params)
	if err != nil {
		return fmt.Errorf("telegram %s: %w", method, err)
	}
	defer res.Body.Close()

	var envelope struct {
		OK          bool            `json:"ok"`
		Result      json.RawMessage `json:"result"`
		Description string          `json:"description"`
		ErrorCode   int             `json:"error_code"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("telegram %s: decode response: %w", method, err)
	}
	if !envelope.OK {
		return apiError{Description: envelope.Description, ErrorCode: envelope.ErrorCode}
	}
	if out != nil && len(envelope.Result) > 0 {
		if err := json.Unmarshal(envelope.Result, out); err != nil {
			return fmt.Errorf("telegram %s: decode result: %w", method, err)
		}
	}
	return nil
}

// GetMe returns the bot's own @username (without the "@"), used to build the
// t.me/<username>?start=<token> deep link.
func GetMe(botToken string) (username string, err error) {
	var result struct {
		Username string `json:"username"`
	}
	if err := call(botToken, "getMe", nil, &result); err != nil {
		return "", err
	}
	if result.Username == "" {
		return "", fmt.Errorf("telegram getMe: empty username in response")
	}
	return result.Username, nil
}

// SetWebhook registers webhookURL as the bot's update endpoint. secretToken is
// echoed back by Telegram on every webhook call as the
// X-Telegram-Bot-Api-Secret-Token header, so the receiver can reject anything
// that didn't come from Telegram's own servers.
func SetWebhook(botToken, webhookURL, secretToken string) error {
	params := url.Values{
		"url":          {webhookURL},
		"secret_token": {secretToken},
	}
	return call(botToken, "setWebhook", params, nil)
}

// SendMessage sends a plain-text message to chatID (a Telegram chat id, e.g.
// the id of whoever just messaged the bot).
func SendMessage(botToken string, chatID int64, text string) error {
	params := url.Values{
		"chat_id": {fmt.Sprintf("%d", chatID)},
		"text":    {text},
	}
	return call(botToken, "sendMessage", params, nil)
}

// SetChatMenuButton sets the bot's default menu button — shown to every user
// in their private chat with the bot — to open webAppURL as a Telegram Mini
// App. text is the button label (Telegram caps it at 64 characters).
func SetChatMenuButton(botToken, webAppURL, text string) error {
	button, err := json.Marshal(map[string]any{
		"type":    "web_app",
		"text":    text,
		"web_app": map[string]string{"url": webAppURL},
	})
	if err != nil {
		return fmt.Errorf("telegram setChatMenuButton: marshal menu_button: %w", err)
	}
	params := url.Values{"menu_button": {string(button)}}
	return call(botToken, "setChatMenuButton", params, nil)
}

// ParseStartToken extracts the token from a "/start <token>" command message,
// tolerating a bot-mention suffix ("/start@mybot <token>") and surrounding
// whitespace. Returns "", false for anything else (no command, wrong command,
// or no token).
func ParseStartToken(text string) (string, bool) {
	fields := strings.Fields(text)
	if len(fields) != 2 {
		return "", false
	}
	cmd := fields[0]
	if cmd != "/start" && !strings.HasPrefix(cmd, "/start@") {
		return "", false
	}
	if fields[1] == "" {
		return "", false
	}
	return fields[1], true
}
