package server

import (
	"errors"
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

// handleRemoveFromLibrary handles the removal of a game from the user's library, it deletes all ownership and
// playthroughs, including the ones from any tracked DLC. It expects the IGDB ID of the game as a path parameter. This
// function is idempotent, meaning it can be called multiple times without causing an error if the game is not in the library.
func (server *Server) handleRemoveFromLibrary(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	err := server.store.RemoveFromLibrary(r.Context(), defaultUserID, igdbID)
	if err != nil && !errors.Is(err, database.ErrUserGameNotFound) {
		server.logger.Error("failed to remove from library", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
