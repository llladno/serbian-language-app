package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TelegramUser is the identity carried by a verified Telegram sign-in.
type TelegramUser struct {
	ID        int64 // tg user id
	Username  string
	FirstName string
	AuthDate  time.Time
}

// Sentinel errors returned by VerifyInitData and VerifyWidget.
var (
	ErrBadHash   = errors.New("auth: telegram signature mismatch")
	ErrStale     = errors.New("auth: telegram auth_date too old")
	ErrMalformed = errors.New("auth: telegram payload malformed")
)

// VerifyInitData parses and verifies a Telegram Mini App initData query string.
//
// The shared secret is HMAC_SHA256(key=[]byte("WebAppData"), msg=botToken); the
// check string is the "\n"-joined, key-sorted list of "k=v" for every field
// except hash, using raw (URL-decoded) values. The data is valid iff the
// recomputed HMAC equals the supplied hash, the user carries a non-zero id,
// and 0 <= now.Sub(auth_date) < maxAge.
func VerifyInitData(initData, botToken string, now time.Time, maxAge time.Duration) (TelegramUser, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return TelegramUser{}, ErrMalformed
	}
	fields := make(map[string]string, len(values))
	for k := range values {
		fields[k] = values.Get(k)
	}

	if err := verifyHash(fields, hmacSHA256([]byte("WebAppData"), []byte(botToken))); err != nil {
		return TelegramUser{}, err
	}

	rawUser, ok := fields["user"]
	if !ok {
		return TelegramUser{}, ErrMalformed
	}
	var u struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		FirstName string `json:"first_name"`
	}
	if err := json.Unmarshal([]byte(rawUser), &u); err != nil {
		return TelegramUser{}, ErrMalformed
	}
	if u.ID == 0 {
		return TelegramUser{}, ErrMalformed
	}

	authDate, err := parseAuthDate(fields, now, maxAge)
	if err != nil {
		return TelegramUser{}, err
	}
	return TelegramUser{
		ID:        u.ID,
		Username:  u.Username,
		FirstName: u.FirstName,
		AuthDate:  authDate,
	}, nil
}

// checkHash recomputes the Telegram data-check hash over fields (every key
// except "hash") with secret and reports whether it matches. The returned hash
// is the raw value from fields, empty when absent.
func checkHash(fields map[string]string, secret []byte) (hash string, ok bool) {
	hash = fields["hash"]
	if hash == "" {
		return "", false
	}
	want, err := hex.DecodeString(hash)
	if err != nil {
		return hash, false
	}

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

	got := hmacSHA256(secret, []byte(strings.Join(pairs, "\n")))
	return hash, hmac.Equal(got, want)
}

// verifyHash maps checkHash's result onto the sentinel errors: a missing or
// non-hex hash is ErrMalformed, a well-formed hash that does not match is
// ErrBadHash.
func verifyHash(fields map[string]string, secret []byte) error {
	hash, ok := checkHash(fields, secret)
	if ok {
		return nil
	}
	if hash == "" {
		return ErrMalformed
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return ErrMalformed
	}
	return ErrBadHash
}

// parseAuthDate reads auth_date (unix seconds) and enforces freshness. A
// missing or unparseable value is ErrMalformed; a value older than maxAge or
// dated in the future is ErrStale.
func parseAuthDate(fields map[string]string, now time.Time, maxAge time.Duration) (time.Time, error) {
	raw, ok := fields["auth_date"]
	if !ok {
		return time.Time{}, ErrMalformed
	}
	secs, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, ErrMalformed
	}
	authDate := time.Unix(secs, 0)
	if d := now.Sub(authDate); d < 0 || d >= maxAge {
		return authDate, ErrStale
	}
	return authDate, nil
}

// hmacSHA256 returns HMAC-SHA256(key, msg).
func hmacSHA256(key, msg []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(msg)
	return h.Sum(nil)
}
