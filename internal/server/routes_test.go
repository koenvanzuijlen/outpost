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

	server := New(slog.New(slog.DiscardHandler))
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
