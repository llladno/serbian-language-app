package telegram

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// Document is a file to send, by reference: FileID resends something Telegram already has
// (instant), URL has Telegram download it itself. There is deliberately no way to upload bytes:
// the Bot API calls go through a Cloudflare relay in production, and files uploaded through it
// arrive corrupted. Exactly one of the two is set.
type Document struct {
	FileID string
	URL    string
}

// SendDocument sends a file to chatID with an optional caption and one optional inline button
// (WebAppURL or URL; the channel-check preset is not supported here). It returns the file_id
// Telegram assigned, so the caller can cache it and resend by id next time.
func SendDocument(botToken string, chatID int64, doc Document, caption string, button *InlineButton) (string, error) {
	ref := doc.FileID + doc.URL
	if ref == "" {
		return "", fmt.Errorf("telegram sendDocument: no file_id or url")
	}
	params := url.Values{
		"chat_id":  {strconv.FormatInt(chatID, 10)},
		"document": {ref},
	}
	if caption != "" {
		params.Set("caption", caption)
	}
	if button != nil {
		btn := map[string]any{"text": button.Label}
		if button.WebAppURL != "" {
			btn["web_app"] = map[string]string{"url": button.WebAppURL}
		} else {
			btn["url"] = button.URL
		}
		markup, err := json.Marshal(map[string]any{"inline_keyboard": [][]map[string]any{{btn}}})
		if err != nil {
			return "", fmt.Errorf("telegram sendDocument: marshal reply_markup: %w", err)
		}
		params.Set("reply_markup", string(markup))
	}
	var result struct {
		Document struct {
			FileID string `json:"file_id"`
		} `json:"document"`
	}
	if err := call(botToken, "sendDocument", params, &result); err != nil {
		return "", err
	}
	return result.Document.FileID, nil
}
