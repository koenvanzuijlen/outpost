package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

// handleStartPlaythrough starts a playthrough of a library game. It reads path value igdbID and form values name and
// platform (both optional), and redirects to the game page.
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

// handleSetPlaythroughOutcome sets the outcome of a playthrough. It reads path values igdbID and playthroughID and form
// value outcome, and redirects to the game page.
func (server *Server) handleSetPlaythroughOutcome(w http.ResponseWriter, r *http.Request) {
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

	err := server.store.SetPlaythroughOutcome(r.Context(), defaultUserID, igdbID, playthroughID, outcome, time.Now().UTC())
	if err != nil {
		if errors.Is(err, database.ErrUserGameNotFound) {
			http.Error(w, "Game not in library", http.StatusNotFound)
		} else if errors.Is(err, database.ErrPlaythroughNotFound) {
			http.Error(w, "Unknown playthrough", http.StatusNotFound)
		} else if errors.Is(err, database.ErrInvalidPlaythroughOutcome) {
			http.Error(w, "Invalid playthrough outcome", http.StatusBadRequest)
		} else {
			server.logger.Error("failed to set playthrough outcome", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/games/"+strconv.FormatInt(igdbID, 10), http.StatusSeeOther)
}

// handleUpdatePlaythrough updates the name, platform and dates of a playthrough. It reads path values igdbID and
// playthroughID and form values name, platform, started_at and finished_at (all optional; empty clears the field), and
// redirects to the game page.
func (server *Server) handleUpdatePlaythrough(w http.ResponseWriter, r *http.Request) {
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

	platformIGDBID, ok := formOptionalPositiveInt(r, "platform")
	if !ok {
		http.Error(w, "Invalid platform", http.StatusBadRequest)
		return
	}

	name := ptrIfNonZero(strings.TrimSpace(r.PostFormValue("name")))

	startedAt, ok := formOptionalDate(r, "started_at")
	if !ok {
		http.Error(w, "Invalid start date", http.StatusBadRequest)
		return
	}
	finishedAt, ok := formOptionalDate(r, "finished_at")
	if !ok {
		http.Error(w, "Invalid finish date", http.StatusBadRequest)
		return
	}

	err := server.store.UpdatePlaythroughDetails(r.Context(), defaultUserID, igdbID, playthroughID, database.PlaythroughDetails{
		PlatformIGDBID: platformIGDBID,
		Name:           name,
		StartedAt:      startedAt,
		FinishedAt:     finishedAt,
	})
	if err != nil {
		if errors.Is(err, database.ErrUserGameNotFound) {
			http.Error(w, "Game not in library", http.StatusNotFound)
		} else if errors.Is(err, database.ErrPlaythroughNotFound) {
			http.Error(w, "Unknown playthrough", http.StatusNotFound)
		} else if errors.Is(err, database.ErrPlatformNotFound) {
			http.Error(w, "Invalid platform", http.StatusBadRequest)
		} else if errors.Is(err, database.ErrInvalidPlaythroughDates) {
			http.Error(w, "Invalid playthrough dates", http.StatusBadRequest)
		} else {
			server.logger.Error("failed to update playthrough", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/games/"+strconv.FormatInt(igdbID, 10), http.StatusSeeOther)
}

// handleDeletePlaythrough deletes a playthrough. It reads path values igdbID and playthroughID, and redirects to the
// game page. A playthrough that is already gone counts as success.
func (server *Server) handleDeletePlaythrough(w http.ResponseWriter, r *http.Request) {
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

	err := server.store.DeletePlaythrough(r.Context(), defaultUserID, igdbID, playthroughID)
	if err != nil && !errors.Is(err, database.ErrPlaythroughNotFound) {
		if errors.Is(err, database.ErrUserGameNotFound) {
			http.Error(w, "Game not in library", http.StatusNotFound)
		} else {
			server.logger.Error("failed to delete playthrough", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/games/"+strconv.FormatInt(igdbID, 10), http.StatusSeeOther)
}
