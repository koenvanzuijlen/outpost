package server

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestChainMiddleware(t *testing.T) {
	var calls []string
	appending := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		calls = append(calls, "handler")
	})

	chained := chainMiddleware(handler, appending("A"), appending("B"))
	chained.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	// B was passed last, so it is outermost and runs first.
	want := []string{"B", "A", "handler"}
	if !slices.Equal(calls, want) {
		t.Errorf("chainMiddleware() call order = %v, want %v", calls, want)
	}
}

func TestRecoverPanic(t *testing.T) {
	server := newTestServer(t)
	handler := server.recoverPanic(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic")
	}))
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	status := w.Result().StatusCode
	if status != http.StatusInternalServerError {
		t.Errorf("recoverPanic() status = %d, want %d", status, http.StatusInternalServerError)
	}
}

func TestCSRFProtection(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		path          string
		secFetchSite  string
		wantForbidden bool
	}{
		{"cross-site POST", http.MethodPost, "/wishlist", "cross-site", true},
		{"same-origin POST", http.MethodPost, "/wishlist", "same-origin", false},
		{"cross-site GET", http.MethodGet, "/", "cross-site", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader("igdb_id=abc"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Sec-Fetch-Site", tt.secFetchSite)
			w := httptest.NewRecorder()

			server.ServeHTTP(w, req)

			if (w.Code == http.StatusForbidden) != tt.wantForbidden {
				t.Errorf("status = %d, want forbidden: %t", w.Code, tt.wantForbidden)
			}
		})
	}
}
