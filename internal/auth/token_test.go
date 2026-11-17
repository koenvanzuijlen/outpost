package auth

import "testing"

// TestHashSessionToken checks that the hash is plain SHA-256 in hex.
// The expected value is the SHA-256 test vector for "abc" from FIPS 180-2.
func TestHashSessionToken(t *testing.T) {
	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got := HashSessionToken("abc"); got != want {
		t.Errorf("HashSessionToken(%q) = %q, want %q", "abc", got, want)
	}
}

// TestNewSessionToken checks that each token is new and has at least 26 characters.
func TestNewSessionToken(t *testing.T) {
	first := NewSessionToken()
	second := NewSessionToken()
	if first == second {
		t.Errorf("NewSessionToken() = %q twice, want different tokens", first)
	}
	if len(first) < 26 {
		t.Errorf("len(NewSessionToken()) = %d, want >= 26", len(first))
	}
}
