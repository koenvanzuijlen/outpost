package server

import (
	"net/http"
	"testing"
)

// TestAdminNav checks which tab of the admin submenu is marked as the current page on each admin page.
func TestAdminNav(t *testing.T) {
	tabs := []string{"/admin/users"}

	tests := []struct {
		path   string
		active string
	}{
		{"/admin/users", "/admin/users"},
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
