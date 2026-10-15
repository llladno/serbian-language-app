package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// uploadClient is for sendDocument with a real file body: far slower than the
// text-only calls, so it gets its own, longer timeout.
var uploadClient = &http.Client{Timeout: 90 * time.Second}

// Document is a file to send. Set FileID to resend something Telegram already has (instant,
// no upload); otherwise Filename and Data are uploaded.
type Document struct {
	FileID   string
	Filename string
	Data     []byte
}

// SendDocument sends a file to chatID with an optional caption and one optional inline
// button (WebAppURL or URL; the channel-check preset is not supported here). It returns the
// file_id Telegram assigned, so the caller can cache it and resend by id next time.
func SendDocument(botToken string, chatID int64, doc Document, caption string, button *InlineButton) (string, error) {
	fields := map[string]string{"chat_id": strconv.FormatInt(chatID, 10)}
	if caption != "" {
		fields["caption"] = caption
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
		fields["reply_markup"] = string(markup)
	}

	var result struct {
		Document struct {
			FileID string `json:"file_id"`
		} `json:"document"`
	}
	if doc.FileID != "" {
		params := url.Values{"document": {doc.FileID}}
		for k, v := range fields {
			params.Set(k, v)
		}
		if err := call(botToken, "sendDocument", params, &result); err != nil {
			return "", err
		}
		return result.Document.FileID, nil
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			return "", fmt.Errorf("telegram sendDocument: %w", err)
		}
	}
	part, err := mw.CreateFormFile("document", doc.Filename)
	if err != nil {
		return "", fmt.Errorf("telegram sendDocument: %w", err)
	}
	if _, err := part.Write(doc.Data); err != nil {
		return "", fmt.Errorf("telegram sendDocument: %w", err)
	}
	if err := mw.Close(); err != nil {
		return "", fmt.Errorf("telegram sendDocument: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, apiBase+"/bot"+botToken+"/sendDocument", &body)
	if err != nil {
		return "", fmt.Errorf("telegram sendDocument: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := uploadClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("telegram sendDocument: %w", err)
	}
	defer res.Body.Close()
	if err := decodeEnvelope("sendDocument", res.Body, &result); err != nil {
		return "", err
	}
	return result.Document.FileID, nil
}

// decodeEnvelope is call's response handling for a request built by hand.
func decodeEnvelope(method string, r io.Reader, out any) error {
	var envelope struct {
		OK          bool            `json:"ok"`
		Result      json.RawMessage `json:"result"`
		Description string          `json:"description"`
		ErrorCode   int             `json:"error_code"`
		Parameters  *struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	if err := json.NewDecoder(r).Decode(&envelope); err != nil {
		return fmt.Errorf("telegram %s: decode response: %w", method, err)
	}
	if !envelope.OK {
		ae := apiError{Description: envelope.Description, ErrorCode: envelope.ErrorCode}
		if envelope.Parameters != nil {
			ae.RetryAfter = envelope.Parameters.RetryAfter
		}
		return ae
	}
	if out != nil && len(envelope.Result) > 0 {
		if err := json.Unmarshal(envelope.Result, out); err != nil {
			return fmt.Errorf("telegram %s: decode result: %w", method, err)
		}
	}
	return nil
}
