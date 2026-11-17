package server

import (
	"strings"
	"testing"
)

// TestValidUsername checks the length and character rules for usernames.
func TestValidUsername(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"plain", "koen", true},
		{"dash, underscore and digit", "k_v-1", true},
		{"minimum length", "abc", true},
		{"too short", "ab", false},
		{"too long", strings.Repeat("a", 33), false},
		{"uppercase", "Koen", false},
		{"punctuation", "koen!", false},
		{"non-ASCII letter", "köen", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validUsername(tt.input); got != tt.want {
				t.Errorf("validUsername(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestNormalizeUsername checks that normalizeUsername trims spaces and lowercases the name.
func TestNormalizeUsername(t *testing.T) {
	if got, want := normalizeUsername("  Koen "), "koen"; got != want {
		t.Errorf("normalizeUsername() = %q, want %q", got, want)
	}
}

// TestPasswordProblem checks the password length rules. The minimum counts characters and the maximum counts bytes.
func TestPasswordProblem(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     string
	}{
		{"valid", "correct horse battery", ""},
		{"11 characters", "elevenchars", "Use at least 12 characters"},
		{"12 multi-byte characters", strings.Repeat("é", 12), ""},
		{"257 bytes", strings.Repeat("a", 257), "The password is too long"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := passwordProblem(tt.password); got != tt.want {
				t.Errorf("passwordProblem() = %q, want %q", got, tt.want)
			}
		})
	}
}
