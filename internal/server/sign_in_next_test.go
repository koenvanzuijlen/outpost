package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSignInURL tests which requests carry the requested page as next.
func TestSignInURL(t *testing.T) {
	tests := []struct {
		name   string
		method string
		target string
		want   string
	}{
		{"page", http.MethodGet, "/library", "/sign-in?next=%2Flibrary"},
		{
			"page with query", http.MethodGet, "/library?status=backlog&status=shelved",
			"/sign-in?next=%2Flibrary%3Fstatus%3Dbacklog%26status%3Dshelved",
		},
		{"home", http.MethodGet, "/", "/sign-in"},
		{"post", http.MethodPost, "/games/1942/library", "/sign-in"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.target, nil)
			if got := signInURL(r); got != tt.want {
				t.Errorf("signInURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestNextPath tests which next values are accepted as local paths.
func TestNextPath(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"empty", "", "/"},
		{"home", "/", "/"},
		{"page", "/games/1942", "/games/1942"},
		{"page with query", "/library?status=backlog&sort=title", "/library?status=backlog&sort=title"},
		{"relative path", "library", "/"},
		{"other host", "//evil.example/", "/"},
		{"backslash host", "/\\evil.example/", "/"},
		{"absolute URL", "https://evil.example/", "/"},
		{"javascript URL", "javascript:alert(1)", "/"},
		{"control character", "/\t/evil.example/", "/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextPath(tt.raw); got != tt.want {
				t.Errorf("nextPath(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
