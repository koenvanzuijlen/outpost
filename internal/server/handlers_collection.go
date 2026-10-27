package server

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

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

	if !game.GameType.Standalone() {
		http.Error(w, "This game type can't be added to the library", http.StatusBadRequest)
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
	_, err = server.store.AddToCollection(r.Context(), gameParams, platformParams, defaultUserID, now)
	if err != nil && !errors.Is(err, database.ErrAlreadyOwnedOnPlatform) {
		server.logger.Error("failed to add to collection", "error", err)
		http.Error(w, "Failed to add to collection", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleRemoveOwnedCopy handles removing an owned copy of a game from the user's collection. It expects the igdbID
// and platformIGDBID as path parameters.
func (server *Server) handleRemoveOwnedCopy(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	platformIGDBID, ok := pathID(r, "platformIGDBID")
	if !ok {
		http.Error(w, "Invalid platform ID", http.StatusBadRequest)
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
