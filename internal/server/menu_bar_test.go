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
		assertBodyContains(t, w, `href="/admin/users"`)
	})

	t.Run("signed in as user", func(t *testing.T) {
		server := newTestServer(t)
		_, cookie := addSignedInUser(t, server, "anna")
		w := serveWithCookie(t, server, http.MethodGet, "/library", "", cookie)
		if w.Code != http.StatusOK {
			t.Fatalf("GET /library status = %d, want %d", w.Code, http.StatusOK)
		}
		assertBodyContains(t, w, `popovertarget="account-menu"`)
		assertBodyNotContains(t, w, `href="/admin/users"`)
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
