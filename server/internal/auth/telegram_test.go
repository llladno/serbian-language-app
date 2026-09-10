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

// TestVerifyInitDataGoldenVector pins the check-string construction against a
// known-answer vector computed out-of-band (not via signInitData or the impl).
//
// Generation (see task-3 report "Fix round 1" for the full script):
//
//	secret      = HMAC_SHA256(key=b"WebAppData", msg=botToken)
//	checkString = "auth_date=1700000000\n" +
//	              `user={"id":42,"username":"grisha","first_name":"Grisha"}`
//	hash        = hex(HMAC_SHA256(key=secret, msg=checkString))
//
// with botToken "7654321:AAHgolden-vector-fixed-bot-token", cross-checked with
// an independent openssl computation.
func TestVerifyInitDataGoldenVector(t *testing.T) {
	const (
		goldenBotToken = "7654321:AAHgolden-vector-fixed-bot-token"
		goldenHash     = "009a76a3e9ecc1b4b69a52540625f06ffd0a88b89251a5e9741a2e2f183ca25c"
		goldenInitData = "auth_date=1700000000&user=%7B%22id%22%3A42%2C%22username%22%3A%22grisha%22%2C%22first_name%22%3A%22Grisha%22%7D&hash=" + goldenHash
	)
	now := time.Unix(1_700_000_000+60, 0)

	u, err := VerifyInitData(goldenInitData, goldenBotToken, now, 24*time.Hour)
	if err != nil {
		t.Fatalf("golden vector rejected: %v", err)
	}
	if u.ID != 42 || u.Username != "grisha" || u.FirstName != "Grisha" {
		t.Fatalf("bad TelegramUser: %+v", u)
	}
	if !u.AuthDate.Equal(time.Unix(1_700_000_000, 0)) {
		t.Fatalf("bad AuthDate: got %v want %v", u.AuthDate, time.Unix(1_700_000_000, 0))
	}
}

// TestVerifyInitDataUserID rejects a correctly-signed payload that carries no
// usable identity (empty user object, missing user field, explicit id=0).
func TestVerifyInitDataUserID(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	authDate := strconv.FormatInt(now.Add(-time.Minute).Unix(), 10)

	cases := map[string]map[string]string{
		"empty user object": {"auth_date": authDate, "user": `{}`},
		"user omitted":      {"auth_date": authDate},
		"explicit id 0":     {"auth_date": authDate, "user": `{"id":0,"username":"grisha"}`},
	}
	for name, fields := range cases {
		t.Run(name, func(t *testing.T) {
			initData := signInitData(fields, testBotToken)
			if _, err := VerifyInitData(initData, testBotToken, now, 24*time.Hour); err != ErrMalformed {
				t.Fatalf("want ErrMalformed, got %v", err)
			}
		})
	}
}

// TestVerifyInitDataFutureDate treats an auth_date in the future as a freshness
// failure, not a valid sign-in.
func TestVerifyInitDataFutureDate(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	fields := map[string]string{
		"auth_date": strconv.FormatInt(now.Add(2*time.Hour).Unix(), 10),
		"user":      `{"id":42,"username":"grisha","first_name":"Grisha"}`,
	}
	initData := signInitData(fields, testBotToken)

	if _, err := VerifyInitData(initData, testBotToken, now, 24*time.Hour); err != ErrStale {
		t.Fatalf("want ErrStale, got %v", err)
	}
}

// TestVerifyInitDataPrecedence pins ErrBadHash ahead of ErrStale: a bad
// signature is reported even when the payload is also stale.
func TestVerifyInitDataPrecedence(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	user := `{"id":42,"username":"grisha","first_name":"Grisha"}`

	staleFields := map[string]string{
		"auth_date": strconv.FormatInt(now.Add(-25*time.Hour).Unix(), 10),
		"user":      user,
	}
	freshFields := map[string]string{
		"auth_date": strconv.FormatInt(now.Add(-time.Minute).Unix(), 10),
		"user":      user,
	}

	// stale payload + GOOD hash -> ErrStale
	if _, err := VerifyInitData(signInitData(staleFields, testBotToken), testBotToken, now, 24*time.Hour); err != ErrStale {
		t.Fatalf("stale + good hash: want ErrStale, got %v", err)
	}

	// fresh payload + BAD hash -> ErrBadHash
	badFresh := mutateHash(t, signInitData(freshFields, testBotToken))
	if _, err := VerifyInitData(badFresh, testBotToken, now, 24*time.Hour); err != ErrBadHash {
		t.Fatalf("fresh + bad hash: want ErrBadHash, got %v", err)
	}

	// stale payload + BAD hash -> ErrBadHash (hash check wins)
	badStale := mutateHash(t, signInitData(staleFields, testBotToken))
	if _, err := VerifyInitData(badStale, testBotToken, now, 24*time.Hour); err != ErrBadHash {
		t.Fatalf("stale + bad hash: want ErrBadHash, got %v", err)
	}
}

// mutateHash flips one hex nibble of the hash in an initData query string,
// keeping it well-formed hex.
func mutateHash(t *testing.T, initData string) string {
	t.Helper()
	values, err := url.ParseQuery(initData)
	if err != nil {
		t.Fatal(err)
	}
	h := []byte(values.Get("hash"))
	h[0] = flipHex(h[0])
	values.Set("hash", string(h))
	return values.Encode()
}

// TestVerifyWidgetMalformed covers the negative paths of the widget surface.
func TestVerifyWidgetMalformed(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	authDate := strconv.FormatInt(now.Add(-time.Minute).Unix(), 10)

	t.Run("non-numeric id", func(t *testing.T) {
		params := signWidget(map[string]string{"id": "abc", "auth_date": authDate}, testBotToken)
		if _, err := VerifyWidget(params, testBotToken, now, 24*time.Hour); err != ErrMalformed {
			t.Fatalf("want ErrMalformed, got %v", err)
		}
	})
	t.Run("id 0", func(t *testing.T) {
		params := signWidget(map[string]string{"id": "0", "auth_date": authDate}, testBotToken)
		if _, err := VerifyWidget(params, testBotToken, now, 24*time.Hour); err != ErrMalformed {
			t.Fatalf("want ErrMalformed, got %v", err)
		}
	})
	t.Run("id missing", func(t *testing.T) {
		params := signWidget(map[string]string{"auth_date": authDate}, testBotToken)
		if _, err := VerifyWidget(params, testBotToken, now, 24*time.Hour); err != ErrMalformed {
			t.Fatalf("want ErrMalformed, got %v", err)
		}
	})
	t.Run("missing auth_date", func(t *testing.T) {
		params := signWidget(map[string]string{"id": "42", "username": "grisha"}, testBotToken)
		if _, err := VerifyWidget(params, testBotToken, now, 24*time.Hour); err != ErrMalformed {
			t.Fatalf("want ErrMalformed, got %v", err)
		}
	})
	t.Run("non-hex hash", func(t *testing.T) {
		params := signWidget(map[string]string{"id": "42", "auth_date": authDate}, testBotToken)
		params["hash"] = "zzzz" + params["hash"][4:]
		if _, err := VerifyWidget(params, testBotToken, now, 24*time.Hour); err != ErrMalformed {
			t.Fatalf("want ErrMalformed, got %v", err)
		}
	})
}
