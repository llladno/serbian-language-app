package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword hashes a plaintext password using SHA256 pre-hash followed by bcrypt.
// Returns the bcrypt PHC-formatted string.
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(pre(plain), bcryptCost)
	return string(hash), err
}

// VerifyPassword verifies a plaintext password against a bcrypt hash.
func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), pre(plain)) == nil
}

// pre applies SHA256 pre-hashing followed by base64 encoding to handle passwords longer than 72 bytes.
func pre(p string) []byte {
	hash := sha256.Sum256([]byte(p))
	return []byte(base64.StdEncoding.EncodeToString(hash[:]))
}
