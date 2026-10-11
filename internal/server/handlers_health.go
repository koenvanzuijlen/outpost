package server

import "net/http"

// handleHealth reports process liveness. It checks no dependencies and always returns 200 while the process is able to
// serve requests.
func (server *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
