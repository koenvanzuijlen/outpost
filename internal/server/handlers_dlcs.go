package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

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
		} else if errors.Is(err, database.ErrCopyDLCNotFound) {
			http.Error(w, "DLC not available for this copy", http.StatusNotFound)
		} else {
			server.logger.Error("failed to set dlc owned status", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/games/"+strconv.FormatInt(igdbID, 10), http.StatusSeeOther)
}

// handleTrackDLC handles the HTTP request to track a DLC separately for the current user. It redirects to the DLC's
// page after tracking.
func (server *Server) handleTrackDLC(w http.ResponseWriter, r *http.Request) {
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

	err := server.store.TrackDLC(r.Context(), defaultUserID, igdbID, dlcIGDBID, time.Now().UTC())
	if err != nil {
		if errors.Is(err, database.ErrUserGameNotFound) {
			http.Error(w, "Game not in library", http.StatusNotFound)
		} else if errors.Is(err, database.ErrDLCNotFound) {
			http.Error(w, "DLC not found for this game", http.StatusNotFound)
		} else {
			server.logger.Error("failed to track dlc", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/games/"+strconv.FormatInt(dlcIGDBID, 10), http.StatusSeeOther)
}
