package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// NewSessionToken returns a new random session token with at least 128 bits of randomness.
// Only the browser stores the token; the database stores its hash (HashSessionToken).
func NewSessionToken() string {
	return rand.Text()
}

// HashSessionToken returns the SHA-256 hash of token as lowercase hex.
// A fast hash is enough, because a token is random and long, unlike a password.
func HashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
