package server

import (
	"net/http"
	"testing"
)

// TestSettingsNav checks which tab of the settings submenu is marked as the current page on each settings page.
func TestSettingsNav(t *testing.T) {
	tabs := []string{"/settings", "/settings/export"}

	tests := []struct {
		path   string
		active string
	}{
		{"/settings", "/settings"},
		{"/settings/export", "/settings/export"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			w := getPage(t, newTestServer(t), tt.path)
			if w.Code != http.StatusOK {
				t.Fatalf("GET %s status = %d, want %d", tt.path, w.Code, http.StatusOK)
			}
			for _, tab := range tabs {
				if tab == tt.active {
					assertBodyContains(t, w, `href="`+tab+`" aria-current="page"`)
				} else {
					assertBodyNotContains(t, w, `href="`+tab+`" aria-current`)
				}
			}
		})
	}
}
