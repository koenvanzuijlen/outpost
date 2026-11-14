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
		server.renderError(w, r, http.StatusBadRequest, "Invalid IGDB ID")
		return
	}

	platformIGDBID, ok := formOptionalPositiveInt(r, "platform_id")
	if !ok || platformIGDBID == nil {
		server.renderError(w, r, http.StatusBadRequest, "Invalid platform")
		return
	}

	err := server.store.AddOwnedCopy(r.Context(), defaultUserID, igdbID, *platformIGDBID)
	if err != nil && !errors.Is(err, database.ErrAlreadyOwnedOnPlatform) {
		if errors.Is(err, database.ErrUserGameNotFound) {
			server.renderError(w, r, http.StatusNotFound, "Game not in library")
		} else if errors.Is(err, database.ErrPlatformNotFound) {
			server.renderError(w, r, http.StatusBadRequest, "Platform not found")
		} else if errors.Is(err, database.ErrPlatformNotForGame) {
			server.renderError(w, r, http.StatusBadRequest, "Game not available on this platform")
		} else {
			server.logger.Error("failed to add owned copy", "error", err)
			server.renderServerError(w, r)
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
		server.renderError(w, r, http.StatusBadRequest, "Invalid IGDB ID")
		return
	}

	platformIGDBID, ok := pathID(r, "platformIGDBID")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid platform IGDB ID")
		return
	}

	err := server.store.RemoveOwnedCopy(r.Context(), defaultUserID, igdbID, platformIGDBID)
	if err != nil && !errors.Is(err, database.ErrOwnedCopyNotFound) {
		if errors.Is(err, database.ErrUserGameNotFound) {
			server.renderError(w, r, http.StatusNotFound, "Game not in library")
		} else {
			server.logger.Error("failed to remove owned copy", "error", err)
			server.renderServerError(w, r)
		}
		return
	}

	http.Redirect(w, r, "/games/"+strconv.FormatInt(igdbID, 10), http.StatusSeeOther)
}
