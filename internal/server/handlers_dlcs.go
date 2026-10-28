package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

// fetchDLCs fetches the DLC of a game from IGDB and caches them in the database. Only the IGDB call is limited to 3
// seconds, because it blocks the page.
func (server *Server) fetchDLCs(ctx context.Context, igdbID int64) error {
	now := time.Now().UTC()

	igdbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	dlcs, err := server.igdbClient.DLCs(igdbCtx, int(igdbID))
	if err != nil {
		return fmt.Errorf("fetch from igdb: %w", err)
	}

	dlcsToSave := make([]database.DLC, len(dlcs))
	for i, dlc := range dlcs {
		dlcsToSave[i] = dlcFromIGDB(dlc, now)
	}

	err = server.store.SaveDLCs(ctx, igdbID, dlcsToSave, now)
	if err != nil {
		return fmt.Errorf("save dlcs: %w", err)
	}
	return nil
}

// handleSetDLCOwned marks a DLC as owned or not owned on one copy of its parent game. It reads path values igdbID,
// platformIGDBID and dlcIGDBID and form value owned ("true" or "false"), and redirects to the game page.
func (server *Server) handleSetDLCOwned(w http.ResponseWriter, r *http.Request) {
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

	dlcIGDBID, ok := pathID(r, "dlcIGDBID")
	if !ok {
		http.Error(w, "Invalid DLC IGDB ID", http.StatusBadRequest)
		return
	}

	value := r.PostFormValue("owned")
	if value != "true" && value != "false" {
		http.Error(w, "Invalid owned value", http.StatusBadRequest)
		return
	}
	owned := value == "true"

	err := server.store.SetDLCOwned(r.Context(), defaultUserID, igdbID, platformIGDBID, dlcIGDBID, owned)
	if err != nil {
		if errors.Is(err, database.ErrUserGameNotFound) {
			http.Error(w, "Game not in library", http.StatusNotFound)
		} else if errors.Is(err, database.ErrDLCNotOnCopy) {
			http.Error(w, "DLC not available for this copy", http.StatusNotFound)
		} else {
			server.logger.Error("failed to set dlc owned status", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/games/"+strconv.FormatInt(igdbID, 10), http.StatusSeeOther)
}

// handleAddDLCToLibrary adds a DLC of a library game to the library separately. It reads path values igdbID and
// dlcIGDBID, and redirects to the DLC's page.
func (server *Server) handleAddDLCToLibrary(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	dlcIGDBID, ok := pathID(r, "dlcIGDBID")
	if !ok {
		http.Error(w, "Invalid DLC IGDB ID", http.StatusBadRequest)
		return
	}

	err := server.store.AddDLCToLibrary(r.Context(), defaultUserID, igdbID, dlcIGDBID, time.Now().UTC())
	if err != nil {
		if errors.Is(err, database.ErrUserGameNotFound) {
			http.Error(w, "Game not in library", http.StatusNotFound)
		} else if errors.Is(err, database.ErrDLCNotFound) {
			http.Error(w, "DLC not found for this game", http.StatusNotFound)
		} else {
			server.logger.Error("failed to add dlc to library", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/games/"+strconv.FormatInt(dlcIGDBID, 10), http.StatusSeeOther)
}
