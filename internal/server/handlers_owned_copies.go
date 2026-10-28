package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

// handleRemoveOwnedCopy removes the copy of a game on one platform; the game stays in the library. It reads path values
// igdbID and platformIGDBID, and redirects to the game page. A copy that is already gone counts as success.
func (server *Server) handleRemoveOwnedCopy(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	platformIGDBID, ok := pathID(r, "platformIGDBID")
	if !ok {
		http.Error(w, "Invalid platform IGDB ID", http.StatusBadRequest)
		return
	}

	err := server.store.RemoveOwnedCopy(r.Context(), defaultUserID, igdbID, platformIGDBID)
	if err != nil && !errors.Is(err, database.ErrOwnedCopyNotFound) {
		if errors.Is(err, database.ErrUserGameNotFound) {
			http.Error(w, "Game not in library", http.StatusNotFound)
		} else {
			server.logger.Error("failed to remove owned copy", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/games/"+strconv.FormatInt(igdbID, 10), http.StatusSeeOther)
}
