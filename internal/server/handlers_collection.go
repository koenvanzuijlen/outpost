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
