package server

import (
	"net/http"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

type libraryPage struct {
	Games           []database.GameListEntry
	WishlistedGames []dbgen.ListWishlistedUserGamesRow
}

// handleLibrary serves the user's library page, displaying tracked games and wishlisted games.
func (server *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	games, err := server.store.ListGames(ctx, defaultUserID)
	if err != nil {
		server.logger.Error("failed to list games", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	wishlistedGames, err := server.store.ListWishlist(ctx, defaultUserID)
	if err != nil {
		server.logger.Error("failed to list wishlisted games", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	server.render(w, "library.html", libraryPage{
		Games:           games,
		WishlistedGames: wishlistedGames,
	})
}
