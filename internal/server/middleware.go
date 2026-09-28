package server

import (
	"net/http"
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
