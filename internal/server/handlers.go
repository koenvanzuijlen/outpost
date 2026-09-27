package server

import "net/http"

type indexData struct {
	Title string
}

// handleIndex serves the application's root page.
func (server *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	server.render(w, "index.html", indexData{
		Title: "Outpost",
	})
}

// handleHealth reports process liveness. It checks no dependencies and always returns 200 while the process is able to
// serve requests.
func (server *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
