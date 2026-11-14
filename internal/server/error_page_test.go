package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorPage(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		body    string
		closeDB bool
		status  int
		message string
	}{
		{"bad request", http.MethodGet, "/games/abc", "", false, http.StatusBadRequest, "Invalid IGDB ID"},
		{"not found", http.MethodPost, "/games/1942/details", "status=playing", false, http.StatusNotFound, "Game not in library"},
		{"server error", http.MethodGet, "/", "", true, http.StatusInternalServerError, "Something went wrong"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			if tt.closeDB {
				_ = server.store.Close()
			}
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			server.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("ServeHTTP() status = %d, want %d", w.Code, tt.status)
			}
			if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
				t.Errorf("ServeHTTP() Content-Type = %q, want %q", got, "text/html; charset=utf-8")
			}
			assertBodyContains(t, w, tt.message)
		})
	}
}
