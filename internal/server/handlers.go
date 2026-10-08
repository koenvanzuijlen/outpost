package server

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
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

// handleAddToCollection handles adding a game to the user's collection. It expects the igdb_id and platform as form
// values.
func (server *Server) handleAddToCollection(w http.ResponseWriter, r *http.Request) {
	igdbID, err := strconv.Atoi(r.PostFormValue("igdb_id"))
	if err != nil || igdbID <= 0 {
		http.Error(w, "Invalid igdb_id", http.StatusBadRequest)
		return
	}

	platformID, err := strconv.Atoi(r.PostFormValue("platform_id"))
	if err != nil || platformID <= 0 {
		http.Error(w, "Invalid platform", http.StatusBadRequest)
		return
	}

	game, err := server.igdbClient.GameByID(r.Context(), igdbID)
	if err != nil {
		if errors.Is(err, igdb.ErrGameNotFound) {
			http.Error(w, "Game not found", http.StatusNotFound)
		} else {
			server.logger.Error("failed to fetch game", "error", err)
			http.Error(w, "Failed to fetch game", http.StatusInternalServerError)
		}
		return
	}

	platformIndex := slices.IndexFunc(game.Platforms, func(p igdb.Platform) bool { return p.ID == platformID })
	if platformIndex == -1 {
		http.Error(w, "Game not available for this platform", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	gameParams := upsertGameParamsFromIGDB(game, now)
	platformParams := upsertPlatformParamsFromIGDB(game.Platforms[platformIndex], now)
	itemParams := dbgen.CreateCollectionItemParams{
		UserID: defaultUserID,
		Status: "backlog",
	}
	_, err = database.AddToCollection(r.Context(), server.db, gameParams, platformParams, itemParams)
	if err != nil && !errors.Is(err, database.ErrAlreadyInCollection) {
		server.logger.Error("failed to add to collection", "error", err)
		http.Error(w, "Failed to add to collection", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
