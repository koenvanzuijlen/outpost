package server

import (
	"strings"
	"unicode/utf8"
)

// The limits for usernames and passwords. The maximum password length only blocks large inputs; argon2 has no limit.
const (
	minUsernameLength = 3
	maxUsernameLength = 32
	minPasswordLength = 12
	maxPasswordBytes  = 256
)

// usernameRule is the message for an invalid username. The setup form also shows it as a hint.
const usernameRule = "Use 3 to 32 characters: lowercase letters, digits, - and _"

// normalizeUsername removes the spaces around s and converts it to lowercase, so "Koen" and "koen" are the same
// username.
func normalizeUsername(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// validUsername reports whether s is a valid normalized username: 3 to 32 characters from a–z, 0–9, - and _.
func validUsername(s string) bool {
	if len(s) < minUsernameLength || len(s) > maxUsernameLength {
		return false
	}
	for _, c := range s {
		isLetter := c >= 'a' && c <= 'z'
		isDigit := c >= '0' && c <= '9'
		if !isLetter && !isDigit && c != '-' && c != '_' {
			return false
		}
	}
	return true
}

// passwordProblem returns the message for a password that breaks the rules, or "" if the password is valid.
func passwordProblem(password string) string {
	if utf8.RuneCountInString(password) < minPasswordLength {
		return "Use at least 12 characters"
	}
	if len(password) > maxPasswordBytes {
		return "The password is too long"
	}
	return ""
}
