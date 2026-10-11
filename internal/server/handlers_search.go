package server

import (
	"net/http"

	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

type searchState string

const (
	searchStateInitial   searchState = "initial"
	searchStateNoResults searchState = "no_results"
	searchStateResults   searchState = "results"
	searchStateError     searchState = "error"
)

type searchPage struct {
	State   searchState
	Query   string
	Results []igdb.Game
	Error   string
}

// handleSearch handles searching for games using the IGDB client. It can receive a query parameter "q" and renders the
// search results.
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
	server.render(w, "search.html", searchPage{
		State:   state,
		Query:   query,
		Results: games,
		Error:   errorMessage,
	})
}
