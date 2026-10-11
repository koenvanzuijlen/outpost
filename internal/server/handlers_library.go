package server

import (
	"net/http"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

type libraryPage struct {
	OwnedCopies     []dbgen.ListOwnedCopiesRow
	WishlistedGames []dbgen.ListWishlistedUserGamesRow
}

// handleLibrary serves the user's library page, displaying owned copies and wishlisted games.
func (server *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ownedCopies, err := server.store.ListCollection(ctx, defaultUserID)
	if err != nil {
		server.logger.Error("failed to list owned copies", "error", err)
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
		OwnedCopies:     ownedCopies,
		WishlistedGames: wishlistedGames,
	})
}
