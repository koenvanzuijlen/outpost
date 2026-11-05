package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStaticFiles(t *testing.T) {
	server := newTestServer(t)

	tests := []struct {
		name         string
		path         string
		wantStatus   int
		wantCacheCtl string
	}{
		{"existing file", "/static/css/outpost.css", http.StatusOK, "public, max-age=3600"},
		{"unknown file", "/static/missing.txt", http.StatusNotFound, ""},
		{"directory", "/static/", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			server.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("ServeHTTP() status = %d, want %d", w.Code, tt.wantStatus)
			}
			if got := w.Header().Get("Cache-Control"); got != tt.wantCacheCtl {
				t.Errorf("ServeHTTP() Cache-Control = %q, want %q", got, tt.wantCacheCtl)
			}
		})
	}
}
