package server

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/auth"
	"github.com/koenvanzuijlen/outpost/internal/database"
)

const (
	validSetupBody  = "username=Koen&password=correct+horse+battery&password_confirm=correct+horse+battery"
	validSignInBody = "username=admin&password=correct+horse+battery"
)

// TestHandleSetup tests that the setup validates its input and creates the first admin with a session.
func TestHandleSetup(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantStatus   int
		wantLocation string
	}{
		{"valid", validSetupBody, http.StatusSeeOther, "/"},
		{
			"short username",
			"username=ko&password=correct+horse+battery&password_confirm=correct+horse+battery",
			http.StatusBadRequest, "",
		},
		{
			"invalid character",
			"username=koen%21&password=correct+horse+battery&password_confirm=correct+horse+battery",
			http.StatusBadRequest, "",
		},
		{
			"short password",
			"username=Koen&password=elevenchars&password_confirm=elevenchars",
			http.StatusBadRequest, "",
		},
		{
			"different repeat",
			"username=Koen&password=correct+horse+battery&password_confirm=correct+horse+batterz",
			http.StatusBadRequest, "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newServerWithoutSetup(t)
			w := serveWithCookie(t, server, http.MethodPost, "/setup", tt.body, nil)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			assertLocation(t, w, tt.wantLocation)
			if tt.name != "valid" {
				return
			}
			cookie := sessionCookie(w)
			if cookie == nil || cookie.Value == "" || !cookie.HttpOnly {
				t.Fatalf("session cookie = %+v, want a value and HttpOnly", cookie)
			}
			user, _, err := server.store.UserByUsername(t.Context(), "koen")
			if err != nil {
				t.Fatalf("UserByUsername() failed: %v", err)
			}
			if user.Role != database.UserRoleAdmin || user.ID != 1 {
				t.Errorf("user = %+v, want role %q and ID 1", user, database.UserRoleAdmin)
			}
		})
	}
}

// TestHandleSetupDone tests that the setup pages answer 404 after the setup is done.
func TestHandleSetupDone(t *testing.T) {
	server := newSignedOutTestServer(t)
	tests := []struct {
		name, method, body string
	}{
		{"get", http.MethodGet, ""},
		{"post", http.MethodPost, validSetupBody},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := serveWithCookie(t, server, tt.method, "/setup", tt.body, nil)
			if w.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
			}
		})
	}
}

// TestHandleSignInPage tests that the sign-in page redirects to the setup until the setup is done.
func TestHandleSignInPage(t *testing.T) {
	tests := []struct {
		name         string
		newServer    func(*testing.T) *Server
		wantStatus   int
		wantLocation string
	}{
		{"before setup", newServerWithoutSetup, http.StatusSeeOther, "/setup"},
		{"after setup", newSignedOutTestServer, http.StatusOK, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := serveWithCookie(t, tt.newServer(t), http.MethodGet, "/sign-in", "", nil)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			assertLocation(t, w, tt.wantLocation)
		})
	}
}

// TestHandleSignIn tests that sign-in starts a session only for a correct username and password.
func TestHandleSignIn(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantStatus   int
		wantLocation string
		wantCookie   bool
	}{
		{"right", validSignInBody, http.StatusSeeOther, "/", true},
		{"uppercase username", "username=ADMIN&password=correct+horse+battery", http.StatusSeeOther, "/", true},
		{"wrong password", "username=admin&password=wrong+password+123", http.StatusUnauthorized, "", false},
		{"unknown user", "username=nobody&password=correct+horse+battery", http.StatusUnauthorized, "", false},
		{"empty", "", http.StatusBadRequest, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newSignedOutTestServer(t)
			w := serveWithCookie(t, server, http.MethodPost, "/sign-in", tt.body, nil)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			assertLocation(t, w, tt.wantLocation)
			if got := sessionCookie(w) != nil; got != tt.wantCookie {
				t.Errorf("has session cookie = %v, want %v", got, tt.wantCookie)
			}
		})
	}
}

// TestHandleSignInPageNext tests that the sign-in form keeps only a local next page in its hidden field.
func TestHandleSignInPageNext(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"next", "/sign-in?next=%2Fgames%2F1942", `name="next" value="/games/1942"`},
		{"no next", "/sign-in", `name="next" value="/"`},
		{"other site", "/sign-in?next=https%3A%2F%2Fevil.example%2F", `name="next" value="/"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := serveWithCookie(t, newSignedOutTestServer(t), http.MethodGet, tt.path, "", nil)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
			}
			assertBodyContains(t, w, tt.want)
		})
	}
}

// TestHandleSignInNext tests that sign-in redirects to a local next page, and that a failed attempt keeps next.
func TestHandleSignInNext(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantStatus   int
		wantLocation string
		wantBody     string
	}{
		{"next", validSignInBody + "&next=%2Fgames%2F1942", http.StatusSeeOther, "/games/1942", ""},
		{
			"next with query", validSignInBody + "&next=%2Flibrary%3Fstatus%3Dbacklog",
			http.StatusSeeOther, "/library?status=backlog", "",
		},
		{"other site", validSignInBody + "&next=%2F%2Fevil.example%2F", http.StatusSeeOther, "/", ""},
		{"no next", validSignInBody, http.StatusSeeOther, "/", ""},
		{
			"wrong password keeps next", "username=admin&password=wrong+password+123&next=%2Fgames%2F1942",
			http.StatusUnauthorized, "", `name="next" value="/games/1942"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := serveWithCookie(t, newSignedOutTestServer(t), http.MethodPost, "/sign-in", tt.body, nil)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			assertLocation(t, w, tt.wantLocation)
			if tt.wantBody != "" {
				assertBodyContains(t, w, tt.wantBody)
			}
		})
	}
}

// TestHandleSignInBeforeSetup tests that the seeded user cannot sign in before the setup, because its password hash
// matches no password.
func TestHandleSignInBeforeSetup(t *testing.T) {
	server := newServerWithoutSetup(t)
	w := serveWithCookie(t, server, http.MethodPost, "/sign-in", "username=default&password=correct+horse+battery", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// TestHandleSignOut tests that sign-out deletes the session, clears the cookie and also works without a cookie.
func TestHandleSignOut(t *testing.T) {
	server := newSignedOutTestServer(t)
	signedIn := serveWithCookie(t, server, http.MethodPost, "/sign-in", validSignInBody, nil)
	cookie := sessionCookie(signedIn)
	if cookie == nil {
		t.Fatal("sign-in set no session cookie")
	}

	w := serveWithCookie(t, server, http.MethodPost, "/sign-out", "", cookie)
	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/sign-in")
	cleared := sessionCookie(w)
	if cleared == nil || cleared.MaxAge >= 0 {
		t.Errorf("cookie = %+v, want MaxAge < 0", cleared)
	}
	_, err := server.store.SessionUser(t.Context(), auth.HashSessionToken(cookie.Value), time.Now().UTC())
	if !errors.Is(err, database.ErrSessionNotFound) {
		t.Errorf("SessionUser() error = %v, want %v", err, database.ErrSessionNotFound)
	}

	w = serveWithCookie(t, server, http.MethodPost, "/sign-out", "", nil)
	if w.Code != http.StatusSeeOther {
		t.Errorf("status without cookie = %d, want %d", w.Code, http.StatusSeeOther)
	}
}

// TestSessionCookieSecure tests that the session cookie has the Secure attribute when secureCookies is set.
func TestSessionCookieSecure(t *testing.T) {
	for _, secure := range []bool{false, true} {
		t.Run(map[bool]string{false: "insecure", true: "secure"}[secure], func(t *testing.T) {
			server := newSignedOutTestServer(t)
			server.secureCookies = secure
			w := serveWithCookie(t, server, http.MethodPost, "/sign-in", validSignInBody, nil)
			cookie := sessionCookie(w)
			if cookie == nil {
				t.Fatal("sign-in set no session cookie")
			}
			if cookie.Secure != secure {
				t.Errorf("Secure = %v, want %v", cookie.Secure, secure)
			}
		})
	}
}
