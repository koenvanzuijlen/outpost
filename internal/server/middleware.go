package server

import (
	"errors"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/auth"
	"github.com/koenvanzuijlen/outpost/internal/database"
)

// contentSecurityPolicyValue is the Content-Security-Policy of every response. Scripts, styles and fonts load only
// from Outpost, and images also from the IGDB image server. Inline scripts, event handlers and style attributes are
// blocked. Forms submit only to Outpost, and other sites cannot show Outpost in a frame.
const contentSecurityPolicyValue = "default-src 'self'; img-src 'self' https://images.igdb.com; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

// statusRecorder wraps a response writer to save the status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader saves the status code and calls the underlying WriteHeader method.
func (rec *statusRecorder) WriteHeader(status int) {
	rec.status = status
	rec.ResponseWriter.WriteHeader(status)
}

// chainMiddleware applies the given middlewares, with the first middleware being the innermost one, and the last
// middleware being the outermost one.
func chainMiddleware(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}
	return handler
}

// contentSecurityPolicy middleware sets the Content-Security-Policy header to contentSecurityPolicyValue on each
// response.
func contentSecurityPolicy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", contentSecurityPolicyValue)
		next.ServeHTTP(w, r)
	})
}

// requestLogging middleware logs the details of each HTTP request.
func (server *Server) requestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		server.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start),
		)
	})
}

// recoverPanic middleware recovers from panics in HTTP handlers, logs them, and renders the error page with a 500.
func (server *Server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				server.logger.Error("panic recovered", "error", rec, "path", r.URL.Path, "stack", string(debug.Stack()))
				server.renderServerError(w, r)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// requireUser returns a handler that calls next only for a request with a valid session. It reads the session cookie
// and stores the user in the request context. It sends the session cookie again when the session was renewed. A
// request without a valid session redirects to the sign-in page, or to the setup page when setup is not done. An
// unknown session also clears the cookie. A failed session lookup renders the server error page.
func (server *Server) requireUser(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			server.redirectToSignIn(w, r)
			return
		}

		session, err := server.store.SessionUser(
			r.Context(),
			auth.HashSessionToken(cookie.Value),
			time.Now().UTC(),
		)
		if err != nil {
			if errors.Is(err, database.ErrSessionNotFound) {
				server.clearSessionCookie(w)
				server.redirectToSignIn(w, r)
				return
			}
			server.logger.Error("failed to get session", "error", err)
			server.renderServerError(w, r)
			return
		}

		if session.Renewed {
			server.setSessionCookie(w, cookie.Value, session.ExpiresAt)
		}

		next.ServeHTTP(w, r.WithContext(withUser(r.Context(), session.User)))
	})
}
