package server

import (
	"net/http"
	"testing"
)

// TestMenuBar tests what the menu bar shows to a signed-in user and to a signed-out visitor.
func TestMenuBar(t *testing.T) {
	t.Run("signed in", func(t *testing.T) {
		w := getPage(t, newTestServer(t), "/library")
		assertBodyContains(t, w, `popovertarget="account-menu"`)
		assertBodyContains(t, w, "admin")
		assertBodyContains(t, w, `action="/sign-out"`)
		assertBodyNotContains(t, w, `href="/export.json"`)
	})

	t.Run("signed out", func(t *testing.T) {
		w := serveWithCookie(t, newSignedOutTestServer(t), http.MethodGet, "/sign-in", "", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("GET /sign-in status = %d, want %d", w.Code, http.StatusOK)
		}
		assertBodyContains(t, w, `class="menu-bar__brand"`)
		assertBodyNotContains(t, w, `href="/library"`)
		assertBodyNotContains(t, w, `class="search-form"`)
	})
}
