package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCheckHealth tests that checkHealth reports the state of the server.
func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{"ok", http.StatusOK, false},
		{"unavailable", http.StatusServiceUnavailable, true},
		{"not found", http.StatusNotFound, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			err := checkHealth(t.Context(), srv.URL)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkHealth() error = %v, want error %v", err, tt.wantErr)
			}
		})
	}
}

// TestCheckHealthUnreachable tests that checkHealth returns an error when the server is closed.
func TestCheckHealthUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	if err := checkHealth(t.Context(), url); err == nil {
		t.Errorf("checkHealth() error = %v, want error %v", err, true)
	}
}
