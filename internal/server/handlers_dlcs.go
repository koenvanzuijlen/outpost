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
		server.renderError(w, r, http.StatusBadRequest, "Invalid IGDB ID")
		return
	}

	platformIGDBID, ok := pathID(r, "platformIGDBID")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid platform IGDB ID")
		return
	}

	dlcIGDBID, ok := pathID(r, "dlcIGDBID")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid DLC IGDB ID")
		return
	}

	value := r.PostFormValue("owned")
	if value != "true" && value != "false" {
		server.renderError(w, r, http.StatusBadRequest, "Invalid owned value")
		return
	}
	owned := value == "true"

	err := server.store.SetDLCOwned(r.Context(), defaultUserID, igdbID, platformIGDBID, dlcIGDBID, owned)
	if err != nil {
		if errors.Is(err, database.ErrUserGameNotFound) {
			server.renderError(w, r, http.StatusNotFound, "Game not in library")
		} else if errors.Is(err, database.ErrDLCNotOnCopy) {
			server.renderError(w, r, http.StatusNotFound, "DLC not available for this copy")
		} else {
			server.logger.Error("failed to set dlc owned status", "error", err)
			server.renderServerError(w, r)
		}
		return
	}

	http.Redirect(w, r, "/games/"+strconv.FormatInt(igdbID, 10), http.StatusSeeOther)
}
