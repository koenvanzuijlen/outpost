package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/auth"
)

// sessionCookieName is the name of the cookie that holds the session token.
const sessionCookieName = "outpost_session"

// setSessionCookie sends the session cookie with token. The browser keeps it until expiresAt. JavaScript cannot read
// it, and the browser sends it only with same-site requests and top-level navigation.
func (server *Server) setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Expires:  expiresAt,
		Path:     "/",
		HttpOnly: true,
		Secure:   server.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookie tells the browser to delete the session cookie.
func (server *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		Secure:   server.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

// startSession signs in the user: it stores a new session and sends its cookie.
// Returns an error if the session cannot be stored.
func (server *Server) startSession(ctx context.Context, w http.ResponseWriter, userID int64) error {
	token := auth.NewSessionToken()
	expiresAt, err := server.store.CreateSession(ctx, userID, auth.HashSessionToken(token), time.Now().UTC())
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	server.setSessionCookie(w, token, expiresAt)
	return nil
}

// sessionTokenHash returns the hash of the session token in the cookie of r, or "" if r has no session cookie.
// Behind requireUser, the cookie is always there.
func sessionTokenHash(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return auth.HashSessionToken(cookie.Value)
}
