package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

// handleAddOwnedCopy adds a copy of a library game on another platform. It reads path value igdbID and form value
// platform_id (required; a platform the game is released on), and redirects to the game page. A copy that is already
// owned counts as success.
func (server *Server) handleAddOwnedCopy(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	platformIGDBID, ok := formOptionalPositiveInt(r, "platform_id")
	if !ok || platformIGDBID == nil {
		http.Error(w, "Invalid platform", http.StatusBadRequest)
		return
	}

	err := server.store.AddOwnedCopy(r.Context(), defaultUserID, igdbID, *platformIGDBID)
	if err != nil && !errors.Is(err, database.ErrAlreadyOwnedOnPlatform) {
		if errors.Is(err, database.ErrUserGameNotFound) {
			http.Error(w, "Game not in library", http.StatusNotFound)
		} else if errors.Is(err, database.ErrPlatformNotFound) {
			http.Error(w, "Platform not found", http.StatusBadRequest)
		} else if errors.Is(err, database.ErrPlatformNotForGame) {
			http.Error(w, "Game not available on this platform", http.StatusBadRequest)
		} else {
			server.logger.Error("failed to add owned copy", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/games/"+strconv.FormatInt(igdbID, 10), http.StatusSeeOther)
}

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
