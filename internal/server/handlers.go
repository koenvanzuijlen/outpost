package server

import (
	"net/http"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// handleHealth reports process liveness. It checks no dependencies and always returns 200 while the process is able to
// serve requests.
func (server *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

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

type searchState string

const (
	searchStateInitial   searchState = "initial"
	searchStateNoResults searchState = "no_results"
	searchStateResults   searchState = "results"
	searchStateError     searchState = "error"
)

type searchData struct {
	State   searchState
	Query   string
	Results []igdb.Game
	Error   string
}

func (server *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	var games []igdb.Game
	var errorMessage string
	state := searchStateInitial
	if query != "" {
		var err error
		games, err = server.igdbClient.SearchGames(r.Context(), query)
		if err != nil {
			server.logger.Error("game search failed", "error", err)
			errorMessage = "Game search failed"
			state = searchStateError
		} else if len(games) > 0 {
			state = searchStateResults
		} else {
			state = searchStateNoResults
		}
	}
	server.render(w, "search.html", searchData{
		State:   state,
		Query:   query,
		Results: games,
		Error:   errorMessage,
	})
}
