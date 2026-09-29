package server

import (
	"net/http"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

type indexData struct {
	Items []dbgen.ListCollectionItemsWithGamesRow
}

// handleIndex serves the application's root page.
func (server *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	items, err := server.queries.ListCollectionItemsWithGames(r.Context(), defaultUserID)
	if err != nil {
		server.logger.Error("failed to list collection items with games", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	server.render(w, "index.html", indexData{
		Items: items,
	})
}

// handleHealth reports process liveness. It checks no dependencies and always returns 200 while the process is able to
// serve requests.
func (server *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
