package server

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/auth"
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
	if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("recoverPanic() Content-Type = %q, want %q", got, "text/html; charset=utf-8")
	}
}

func TestContentSecurityPolicy(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"page", "/"},
		{"static file", "/static/css/outpost.css"},
		{"bad request", "/library?sort=bogus"},
	}

	want := "default-src 'self'; img-src 'self' https://images.igdb.com; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			server.ServeHTTP(w, req)

			got := w.Result().Header.Get("Content-Security-Policy")
			if got != want {
				t.Errorf("Content-Security-Policy = %q, want %q", got, want)
			}
		})
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
		{"cross-site POST", http.MethodPost, "/games/abc/wishlist", "cross-site", true},
		{"same-origin POST", http.MethodPost, "/games/abc/wishlist", "same-origin", false},
		{"cross-site GET", http.MethodGet, "/", "cross-site", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(""))
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

// TestRequireUser tests which requests reach the page.
func TestRequireUser(t *testing.T) {
	tests := []struct {
		name         string
		newServer    func(*testing.T) *Server
		cookie       *http.Cookie
		method       string
		path         string
		wantStatus   int
		wantLocation string
	}{
		{"no admin", newServerWithoutSetup, nil, http.MethodGet, "/library", http.StatusSeeOther, "/setup"},
		{"no cookie", newSignedOutTestServer, nil, http.MethodGet, "/library", http.StatusSeeOther, "/sign-in"},
		{
			"unknown session", newSignedOutTestServer, &http.Cookie{Name: sessionCookieName, Value: "bogus"},
			http.MethodGet, "/library", http.StatusSeeOther, "/sign-in",
		},
		{
			"post without cookie", newSignedOutTestServer, nil,
			http.MethodPost, "/games/1942/library", http.StatusSeeOther, "/sign-in",
		},
		{"signed in", newTestServer, nil, http.MethodGet, "/library", http.StatusOK, ""},
		{"admin page without cookie", newSignedOutTestServer, nil, http.MethodGet, "/admin/users", http.StatusSeeOther, "/sign-in"},
		{"health is public", newSignedOutTestServer, nil, http.MethodGet, "/health", http.StatusOK, ""},
		{
			"static is public", newSignedOutTestServer, nil,
			http.MethodGet, "/static/css/outpost.css", http.StatusOK, "",
		},
		{"unknown URL", newSignedOutTestServer, nil, http.MethodGet, "/nonexistent", http.StatusNotFound, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.newServer(t)
			w := serveWithCookie(t, server, tt.method, tt.path, "", tt.cookie)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			assertLocation(t, w, tt.wantLocation)
			if tt.name != "unknown session" {
				return
			}
			cookie := sessionCookie(w)
			if cookie == nil || cookie.MaxAge >= 0 {
				t.Errorf("session cookie = %v, want a cookie with MaxAge < 0", cookie)
			}
		})
	}
}

// TestRequireUserRenewsSession tests when the response sends the session cookie again.
func TestRequireUserRenewsSession(t *testing.T) {
	tests := []struct {
		name        string
		createdAt   time.Time
		wantRenewed bool
	}{
		{"young session", time.Now().Add(-time.Hour), false},
		{"old session", time.Now().Add(-25 * time.Hour), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newSignedOutTestServer(t)
			_, err := server.store.CreateSession(
				t.Context(), testUserID, auth.HashSessionToken("renew-token"), tt.createdAt,
			)
			if err != nil {
				t.Fatalf("CreateSession() failed: %v", err)
			}
			cookie := &http.Cookie{Name: sessionCookieName, Value: "renew-token"}
			w := serveWithCookie(t, server, http.MethodGet, "/library", "", cookie)
			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
			}
			got := sessionCookie(w)
			if !tt.wantRenewed {
				if got != nil {
					t.Errorf("session cookie = %v, want nil", got)
				}
				return
			}
			if got == nil || got.Value != "renew-token" {
				t.Errorf("session cookie = %v, want value %q", got, "renew-token")
			}
		})
	}
}

// TestRequireAdmin checks who can open an admin page: an admin can, a user who is not an admin gets a 403.
func TestRequireAdmin(t *testing.T) {
	tests := []struct {
		name       string
		asUser     bool
		wantStatus int
	}{
		{name: "admin", asUser: false, wantStatus: http.StatusOK},
		{name: "user", asUser: true, wantStatus: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			var cookie *http.Cookie
			if tt.asUser {
				_, cookie = addSignedInUser(t, server, "anna")
			}

			w := serveWithCookie(t, server, http.MethodGet, "/admin/users", "", cookie)
			if w.Code != tt.wantStatus {
				t.Errorf("GET /admin/users status = %d, want %d", w.Code, tt.wantStatus)
			}
			assertLocation(t, w, "")
		})
	}
}
