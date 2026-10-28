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

// fetchDLCs retrieves the DLCs for a given game from IGDB and saves them to the database.
func (server *Server) fetchDLCs(ctx context.Context, igdbID int64) error {
	now := time.Now().UTC()

	ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	dlcs, err := server.igdbClient.DLCs(ctxTimeout, int(igdbID))
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

// handleSetDLCOwned handles the HTTP request to set the ownership status of a DLC for a specific user's owned copy.
// It expects the IGDB ID of the parent game, the platform IGDB ID, and the DLC IGDB ID as path parameters, and the
// ownership status as a POST form value "owned" ("true" or "false").
func (server *Server) handleSetDLCOwned(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	platformIGDBID, ok := pathID(r, "platformIGDBID")
	if !ok {
		http.Error(w, "Invalid Platform IGDB ID", http.StatusBadRequest)
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

// handleAddDLCToLibrary handles the HTTP request to add a DLC to the user's library as a separate entry. It redirects
// to the DLC's page afterwards.
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
