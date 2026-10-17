package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

// handleStartPlaythrough handles the HTTP request to start a new playthrough for a game. It expects the IGDB ID of the
// game as a path parameter and optionally accepts the platform and name as form values.
func (server *Server) handleStartPlaythrough(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	platformIGDBID, ok := formOptionalPositiveInt(r, "platform")
	if !ok {
		http.Error(w, "Invalid platform", http.StatusBadRequest)
		return
	}

	name := ptrIfNonZero(strings.TrimSpace(r.PostFormValue("name")))

	_, err := server.store.StartPlaythrough(r.Context(), defaultUserID, igdbID, platformIGDBID, name, time.Now().UTC())
	if err != nil {
		if errors.Is(err, database.ErrUserGameNotFound) {
			http.Error(w, "Game not in library", http.StatusNotFound)
		} else if errors.Is(err, database.ErrPlatformNotFound) {
			http.Error(w, "Unknown platform", http.StatusBadRequest)
		} else {
			server.logger.Error("failed to start playthrough", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/games/"+strconv.FormatInt(igdbID, 10), http.StatusSeeOther)
}

// handleFinishPlaythrough handles the HTTP request to finish an existing playthrough for a game. It expects the IGDB ID
// of the game and the playthrough ID as path parameters and the outcome as a form value.
func (server *Server) handleFinishPlaythrough(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	playthroughID, ok := pathID(r, "playthroughID")
	if !ok {
		http.Error(w, "Invalid playthrough ID", http.StatusBadRequest)
		return
	}

	outcome := database.PlaythroughOutcome(r.PostFormValue("outcome"))

	err := server.store.FinishPlaythrough(r.Context(), defaultUserID, igdbID, playthroughID, outcome, time.Now().UTC())
	if err != nil {
		if errors.Is(err, database.ErrUserGameNotFound) {
			http.Error(w, "Game not in library", http.StatusNotFound)
		} else if errors.Is(err, database.ErrPlaythroughNotFound) {
			http.Error(w, "Unknown playthrough", http.StatusNotFound)
		} else if errors.Is(err, database.ErrInvalidPlaythroughOutcome) {
			http.Error(w, "Invalid playthrough outcome", http.StatusBadRequest)
		} else {
			server.logger.Error("failed to finish playthrough", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/games/"+strconv.FormatInt(igdbID, 10), http.StatusSeeOther)
}
