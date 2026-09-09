// Package auth holds authentication primitives: id generation, password
// hashing, opaque token handling, and Telegram signature validation.
package auth

import (
	"crypto/rand"
	"math/big"
)

const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// NewID returns "<prefix>_" followed by 22 base62 characters drawn from a
// cryptographic source (~131 bits of entropy).
func NewID(prefix string) string {
	b := make([]byte, 22)
	max := big.NewInt(int64(len(base62)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err) // crypto/rand failure is unrecoverable
		}
		b[i] = base62[n.Int64()]
	}
	return prefix + "_" + string(b)
}

// NewUserID mints an id for a users row.
func NewUserID() string { return NewID("usr") }

// NewIdentityID mints an id for an identities row.
func NewIdentityID() string { return NewID("idn") }
