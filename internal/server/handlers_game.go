package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

type gamePage struct {
	LibraryGame database.LibraryGame
}

// handleGame serves the detail page for a game in the user's library.
func (server *Server) handleGame(w http.ResponseWriter, r *http.Request) {
	igdbID, err := strconv.ParseInt(r.PathValue("igdbID"), 10, 64)
	if err != nil || igdbID <= 0 {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	libraryGame, err := server.store.GetLibraryGame(r.Context(), defaultUserID, igdbID)
	if err != nil {
		if errors.Is(err, database.ErrUserGameNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			server.logger.Error("failed to get library game", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	server.render(w, "game.html", gamePage{
		LibraryGame: libraryGame,
	})
}
