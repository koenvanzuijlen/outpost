package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBaseRoutes(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		route          string
		expectedStatus int
	}{
		{"health", "GET", "/health", http.StatusOK},
		{"invalid method", "POST", "/health", http.StatusMethodNotAllowed},
		{"unknown route", "GET", "/nonexistent", http.StatusNotFound},
	}

	server, err := New(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.route, nil)
			w := httptest.NewRecorder()

			server.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
