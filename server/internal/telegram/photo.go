package telegram

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// Photo is an image to send, by reference: FileID resends one Telegram already has (instant), URL has Telegram download it
// itself. Like Document there is no way to upload bytes, because files uploaded through the production relay arrive corrupted.
// Exactly one of the two is set.
type Photo struct {
	FileID string
	URL    string
}

// SendPhoto sends an image with caption as its text (Telegram allows 1024 characters after the tags are parsed; the caption is HTML,
// so the caller escapes it) and the keyboard for button, if any. It returns
// the file_id Telegram assigned to the image, so the caller can cache it and resend by id.
func SendPhoto(botToken string, chatID int64, photo Photo, caption string, button *InlineButton) (string, error) {
	ref := photo.FileID + photo.URL
	if ref == "" {
		return "", fmt.Errorf("telegram sendPhoto: no file_id or url")
	}
	params := url.Values{
		"chat_id": {strconv.FormatInt(chatID, 10)},
		"photo":   {ref},
	}
	if caption != "" {
		params.Set("caption", caption)
		params.Set("parse_mode", "HTML")
	}
	if button != nil {
		markup, err := json.Marshal(map[string]any{"inline_keyboard": inlineRows(button)})
		if err != nil {
			return "", fmt.Errorf("telegram sendPhoto: marshal reply_markup: %w", err)
		}
		params.Set("reply_markup", string(markup))
	}
	var result struct {
		Photo []struct {
			FileID string `json:"file_id"`
		} `json:"photo"`
	}
	if err := call(botToken, "sendPhoto", params, &result); err != nil {
		return "", err
	}
	// Telegram returns the photo in several sizes, smallest first; the largest is the one to resend.
	if n := len(result.Photo); n > 0 {
		return result.Photo[n-1].FileID, nil
	}
	return "", nil
}
