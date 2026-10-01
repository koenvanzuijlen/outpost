package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBasicRouting(t *testing.T) {
	server := newTestServer(t)

	tests := []struct {
		name           string
		method         string
		route          string
		expectedStatus int
	}{
		{"health", "GET", "/health", http.StatusOK},
		{"index", "GET", "/", http.StatusOK},
		{"invalid method", "POST", "/health", http.StatusMethodNotAllowed},
		{"unknown route", "GET", "/nonexistent", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.route, nil)
			w := httptest.NewRecorder()

			server.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("ServeHTTP() status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}
