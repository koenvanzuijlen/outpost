// Package auth hashes passwords and session tokens.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// The argon2id parameters follow the minimum of the OWASP Password Storage Cheat Sheet: 19 MiB of memory,
// 2 passes and 1 thread. VerifyPassword reads the parameters from the hash, so a change here does not break
// existing hashes.
const (
	argonTime       = 2
	argonMemory     = 19 * 1024
	argonThreads    = 1
	argonKeyLength  = 32
	argonSaltLength = 16
)

// ErrMalformedHash is returned when a stored password hash is not an argon2id hash in PHC format.
// The hash "!" of an account without a password is malformed.
var ErrMalformedHash = errors.New("malformed password hash")

// HashPassword returns an argon2id hash of password with a random salt, in PHC format:
// $argon2id$v=19$m=19456,t=2,p=1$<salt>$<key>. The salt and the key use standard base64 without padding.
func HashPassword(password string) string {
	salt := make([]byte, argonSaltLength)
	_, _ = rand.Read(salt)
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLength)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory,
		argonTime,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
}

// VerifyPassword reports whether password matches the argon2id hash encoded.
// It uses the parameters and the salt in encoded, and compares in constant time.
// Returns ErrMalformedHash if encoded is not an argon2id hash in PHC format.
func VerifyPassword(encoded, password string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrMalformedHash
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil || version != argon2.Version {
		return false, ErrMalformedHash
	}

	var memory, time uint32
	var threads uint8
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil || time == 0 || threads == 0 {
		return false, ErrMalformedHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return false, ErrMalformedHash
	}

	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false, ErrMalformedHash
	}

	key := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(want, key) == 1, nil
}
