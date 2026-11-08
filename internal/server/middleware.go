package server

import (
	"net/http"
	"runtime/debug"
	"time"
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

// recoverPanic middleware recovers from panics in HTTP handlers and returns a 500.
func (server *Server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				server.logger.Error("panic recovered", "error", rec, "path", r.URL.Path, "stack", string(debug.Stack()))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
