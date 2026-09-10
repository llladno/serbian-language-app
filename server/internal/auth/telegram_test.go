package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const testBotToken = "12345:test-token"

// checkString reproduces the Telegram data-check string: the "\n"-joined,
// key-sorted list of "k=v" for every field except hash.
func checkString(fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		if k == "hash" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, len(keys))
	for i, k := range keys {
		pairs[i] = k + "=" + fields[k]
	}
	return strings.Join(pairs, "\n")
}

// signInitData builds a valid Mini App initData query string, computing hash
// with the same algorithm VerifyInitData uses: secret = HMAC_SHA256(key=
// "WebAppData", msg=botToken).
func signInitData(fields map[string]string, botToken string) string {
	sk := hmac.New(sha256.New, []byte("WebAppData"))
	sk.Write([]byte(botToken))
	secret := sk.Sum(nil)

	h := hmac.New(sha256.New, secret)
	h.Write([]byte(checkString(fields)))
	sig := hex.EncodeToString(h.Sum(nil))

	q := url.Values{}
	for k, v := range fields {
		q.Set(k, v)
	}
	q.Set("hash", sig)
	return q.Encode()
}

// signWidget builds a valid Login Widget param map, computing hash with
// secret = SHA256(botToken).
func signWidget(fields map[string]string, botToken string) map[string]string {
	sum := sha256.Sum256([]byte(botToken))

	h := hmac.New(sha256.New, sum[:])
	h.Write([]byte(checkString(fields)))
	sig := hex.EncodeToString(h.Sum(nil))

	out := make(map[string]string, len(fields)+1)
	for k, v := range fields {
		out[k] = v
	}
	out["hash"] = sig
	return out
}

// flipHex returns a different valid hex digit, so a mutated hash stays
// well-formed hex and the failure is a mismatch, not a parse error.
func flipHex(c byte) byte {
	if c == '0' {
		return '1'
	}
	return '0'
}

func TestVerifyInitDataValid(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	authDate := now.Add(-time.Minute)
	fields := map[string]string{
		"auth_date": strconv.FormatInt(authDate.Unix(), 10),
		"query_id":  "AAABBBCCC",
		"user":      `{"id":42,"username":"grisha","first_name":"Grisha"}`,
	}
	initData := signInitData(fields, testBotToken)

	u, err := VerifyInitData(initData, testBotToken, now, 24*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != 42 || u.Username != "grisha" || u.FirstName != "Grisha" {
		t.Fatalf("bad TelegramUser: %+v", u)
	}
	if !u.AuthDate.Equal(time.Unix(authDate.Unix(), 0)) {
		t.Fatalf("bad AuthDate: got %v want %v", u.AuthDate, authDate)
	}
}

func TestVerifyInitDataBadHash(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	fields := map[string]string{
		"auth_date": strconv.FormatInt(now.Unix(), 10),
		"user":      `{"id":42,"username":"grisha","first_name":"Grisha"}`,
	}
	initData := signInitData(fields, testBotToken)

	values, err := url.ParseQuery(initData)
	if err != nil {
		t.Fatal(err)
	}
	h := []byte(values.Get("hash"))
	h[0] = flipHex(h[0])
	values.Set("hash", string(h))

	if _, err := VerifyInitData(values.Encode(), testBotToken, now, 24*time.Hour); err != ErrBadHash {
		t.Fatalf("want ErrBadHash, got %v", err)
	}
}

func TestVerifyInitDataStale(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	fields := map[string]string{
		"auth_date": strconv.FormatInt(now.Add(-25*time.Hour).Unix(), 10),
		"user":      `{"id":42,"username":"grisha","first_name":"Grisha"}`,
	}
	initData := signInitData(fields, testBotToken)

	if _, err := VerifyInitData(initData, testBotToken, now, 24*time.Hour); err != ErrStale {
		t.Fatalf("want ErrStale, got %v", err)
	}
}

func TestVerifyInitDataMalformed(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)

	// missing hash field
	noHash := url.Values{}
	noHash.Set("auth_date", strconv.FormatInt(now.Unix(), 10))
	noHash.Set("user", `{"id":42}`)
	if _, err := VerifyInitData(noHash.Encode(), testBotToken, now, 24*time.Hour); err != ErrMalformed {
		t.Fatalf("missing hash: want ErrMalformed, got %v", err)
	}

	// correctly signed, but user JSON is broken
	bad := map[string]string{
		"auth_date": strconv.FormatInt(now.Unix(), 10),
		"user":      `{"id":42,`,
	}
	initData := signInitData(bad, testBotToken)
	if _, err := VerifyInitData(initData, testBotToken, now, 24*time.Hour); err != ErrMalformed {
		t.Fatalf("broken user JSON: want ErrMalformed, got %v", err)
	}
}

func TestVerifyWidgetValid(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	authDate := now.Add(-time.Minute)
	fields := map[string]string{
		"id":         "42",
		"first_name": "Grisha",
		"username":   "grisha",
		"auth_date":  strconv.FormatInt(authDate.Unix(), 10),
	}
	params := signWidget(fields, testBotToken)

	u, err := VerifyWidget(params, testBotToken, now, 24*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != 42 || u.Username != "grisha" || u.FirstName != "Grisha" {
		t.Fatalf("bad TelegramUser: %+v", u)
	}
	if !u.AuthDate.Equal(time.Unix(authDate.Unix(), 0)) {
		t.Fatalf("bad AuthDate: got %v want %v", u.AuthDate, authDate)
	}
}

func TestVerifyWidgetBadHash(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	fields := map[string]string{
		"id":        "42",
		"username":  "grisha",
		"auth_date": strconv.FormatInt(now.Unix(), 10),
	}
	params := signWidget(fields, testBotToken)
	h := []byte(params["hash"])
	h[0] = flipHex(h[0])
	params["hash"] = string(h)

	if _, err := VerifyWidget(params, testBotToken, now, 24*time.Hour); err != ErrBadHash {
		t.Fatalf("want ErrBadHash, got %v", err)
	}
}
