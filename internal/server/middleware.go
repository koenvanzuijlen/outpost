package server

import (
	"net/http"
	"runtime/debug"
	"time"
)

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
