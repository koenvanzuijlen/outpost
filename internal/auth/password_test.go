package auth

import (
	"errors"
	"strings"
	"testing"
)

// TestVerifyPassword checks that a hash matches only its own password.
func TestVerifyPassword(t *testing.T) {
	hash := HashPassword("correct horse battery")

	tests := []struct {
		password string
		want     bool
	}{
		{"correct horse battery", true},
		{"Correct horse battery", false},
		{"", false},
	}
	for _, tt := range tests {
		ok, err := VerifyPassword(hash, tt.password)
		if err != nil || ok != tt.want {
			t.Errorf("VerifyPassword(%q) = %v, %v, want %v, nil", tt.password, ok, err, tt.want)
		}
	}
}

// TestHashPasswordFormat checks that the hash has the PHC format and a new salt each time.
func TestHashPasswordFormat(t *testing.T) {
	const prefix = "$argon2id$v=19$m=19456,t=2,p=1$"

	hash := HashPassword("same")
	if !strings.HasPrefix(hash, prefix) {
		t.Errorf("HashPassword() = %q, want prefix %q", hash, prefix)
	}

	other := HashPassword("same")
	if other == hash {
		t.Errorf("HashPassword() = %q twice, want different hashes", hash)
	}
}

// TestVerifyPasswordMalformed checks which hashes are malformed.
// Each row fails before the key derivation, so the test stays fast.
func TestVerifyPasswordMalformed(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
	}{
		{"locked account", "!"},
		{"empty", ""},
		{"missing key", "$argon2id$v=19$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA"},
		{"other algorithm", "$argon2i$v=19$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$a2V5"},
		{"other version", "$argon2id$v=16$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$a2V5"},
		{"zero passes", "$argon2id$v=19$m=19456,t=0,p=1$c2FsdHNhbHRzYWx0c2FsdA$a2V5"},
		{"zero threads", "$argon2id$v=19$m=19456,t=2,p=0$c2FsdHNhbHRzYWx0c2FsdA$a2V5"},
		{"bad base64", "$argon2id$v=19$m=19456,t=2,p=1$!!!$a2V5"},
		{"empty key", "$argon2id$v=19$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := VerifyPassword(tt.encoded, "password")
			if ok || !errors.Is(err, ErrMalformedHash) {
				t.Errorf("VerifyPassword(%q) = %v, %v, want false, %v", tt.encoded, ok, err, ErrMalformedHash)
			}
		})
	}
}
