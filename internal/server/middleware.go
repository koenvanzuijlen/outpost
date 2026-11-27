package server

import (
	"net/http"
	"runtime/debug"
	"time"
)

// contentSecurityPolicyValue is the Content-Security-Policy of every response. Scripts, styles and fonts load only
// from Outpost, and images also from the IGDB image server. Inline scripts, event handlers and style attributes are
// blocked. The only inline style that loads is the <style> element of favicon.svg, allowed by its SHA-256 hash.
// Forms submit only to Outpost, and other sites cannot show Outpost in a frame.
const contentSecurityPolicyValue = "default-src 'self'; img-src 'self' https://images.igdb.com; style-src 'self' 'sha256-ZHFz94GDLcPbjwPI4KK3hJmG8eaKEnc9V+zZi62TC6k='; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

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

// navigationRequest reports whether r loads a page in the browser: a link, a form submit or an address that the user
// typed. Browsers send the header Sec-Fetch-Mode with the value navigate for these requests, and another value, such
// as cors, for fetch. A request without the header counts as a navigation request.
func navigationRequest(r *http.Request) bool {
	mode := r.Header.Get("Sec-Fetch-Mode")
	return mode == "" || mode == "navigate"
}

// requireUser returns a handler that calls next only for a request with a valid session, and stores the user in the
// request context. A navigation request without a valid session redirects to the sign-in page, or to the setup page
// when setup is not done. Another request without a valid session, such as a fetch, gets a 401. A failed session
// lookup renders the server error page.
func (server *Server) requireUser(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok, err := server.signedInUser(w, r)
		if err != nil {
			server.logger.Error("failed to get session", "error", err)
			server.renderServerError(w, r)
			return
		}
		if !ok && !navigationRequest(r) {
			server.renderError(w, r, http.StatusUnauthorized, "You are signed out")
			return
		}
		if !ok {
			server.redirectToSignIn(w, r)
			return
		}

		next.ServeHTTP(w, r.WithContext(withUser(r.Context(), user)))
	})
}

// requireAdmin lets only signed-in admins reach next. A signed-in user who is not an admin gets a 403 error page.
// Requests without a valid session go where requireUser sends them.
func (server *Server) requireAdmin(next http.HandlerFunc) http.Handler {
	return server.requireUser(func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		if !user.IsAdmin() {
			server.renderError(w, r, http.StatusForbidden, "Only admins can open this page")
			return
		}
		next(w, r)
	})
}
