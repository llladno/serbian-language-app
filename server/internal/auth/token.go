package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// NewToken generates an opaque 32-byte token, returns the raw base64url-encoded
// string and its SHA256 hash (hex-encoded). Panics only on cryptographic failure.
func NewToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	hash = HashToken(raw)
	return raw, hash, nil
}

// HashToken returns the hex-encoded SHA256 hash of the raw token string.
func HashToken(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}
