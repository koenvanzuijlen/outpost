package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestFooterShowsVersion checks that a rendered page shows the version and the source URL in the footer.
func TestFooterShowsVersion(t *testing.T) {
	server := newTestServer(t)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/library", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /library status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{testVersion, sourceURL} {
		if !strings.Contains(body, want) {
			t.Errorf("body does not contain %q", want)
		}
	}
}
