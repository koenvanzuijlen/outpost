package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

// handleStartPlaythrough starts a playthrough of a library game. It reads path value igdbID and form values name and
// platform (both optional), and redirects to the new playthrough on the game page.
func (server *Server) handleStartPlaythrough(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid IGDB ID")
		return
	}

	platformIGDBID, ok := formOptionalPositiveInt(r, "platform")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid platform")
		return
	}

	name := ptrIfNonZero(strings.TrimSpace(r.PostFormValue("name")))

	playthrough, err := server.store.StartPlaythrough(r.Context(), defaultUserID, igdbID, platformIGDBID, name, time.Now())
	if err != nil {
		if errors.Is(err, database.ErrUserGameNotFound) {
			server.renderError(w, r, http.StatusNotFound, "Game not in library")
		} else if errors.Is(err, database.ErrPlatformNotFound) {
			server.renderError(w, r, http.StatusBadRequest, "Unknown platform")
		} else {
			server.logger.Error("failed to start playthrough", "error", err)
			server.renderServerError(w, r)
		}
		return
	}

	http.Redirect(w, r, gameURL(igdbID)+"#"+playthroughAnchor(playthrough.ID), http.StatusSeeOther)
}

// handleSetPlaythroughOutcome sets the outcome of a playthrough. It reads path values igdbID and playthroughID and form
// value outcome, and redirects to the playthrough on the game page.
func (server *Server) handleSetPlaythroughOutcome(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid IGDB ID")
		return
	}

	playthroughID, ok := pathID(r, "playthroughID")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid playthrough ID")
		return
	}

	outcome := database.PlaythroughOutcome(r.PostFormValue("outcome"))

	err := server.store.SetPlaythroughOutcome(r.Context(), defaultUserID, igdbID, playthroughID, outcome, time.Now())
	if err != nil {
		if errors.Is(err, database.ErrUserGameNotFound) {
			server.renderError(w, r, http.StatusNotFound, "Game not in library")
		} else if errors.Is(err, database.ErrPlaythroughNotFound) {
			server.renderError(w, r, http.StatusNotFound, "Unknown playthrough")
		} else if errors.Is(err, database.ErrInvalidPlaythroughOutcome) {
			server.renderError(w, r, http.StatusBadRequest, "Invalid playthrough outcome")
		} else {
			server.logger.Error("failed to set playthrough outcome", "error", err)
			server.renderServerError(w, r)
		}
		return
	}

	http.Redirect(w, r, gameURL(igdbID)+"#"+playthroughAnchor(playthroughID), http.StatusSeeOther)
}

// handleUpdatePlaythrough updates the name, platform and dates of a playthrough. It reads path values igdbID and
// playthroughID and form values name, platform, started_at and finished_at (all optional; empty clears the field), and
// redirects to the playthrough on the game page.
func (server *Server) handleUpdatePlaythrough(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid IGDB ID")
		return
	}

	playthroughID, ok := pathID(r, "playthroughID")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid playthrough ID")
		return
	}

	platformIGDBID, ok := formOptionalPositiveInt(r, "platform")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid platform")
		return
	}

	name := ptrIfNonZero(strings.TrimSpace(r.PostFormValue("name")))

	startedAt, ok := formOptionalDate(r, "started_at")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid start date")
		return
	}
	finishedAt, ok := formOptionalDate(r, "finished_at")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid finish date")
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
			server.renderError(w, r, http.StatusNotFound, "Game not in library")
		} else if errors.Is(err, database.ErrPlaythroughNotFound) {
			server.renderError(w, r, http.StatusNotFound, "Unknown playthrough")
		} else if errors.Is(err, database.ErrPlatformNotFound) {
			server.renderError(w, r, http.StatusBadRequest, "Invalid platform")
		} else if errors.Is(err, database.ErrInvalidPlaythroughDates) {
			server.renderError(w, r, http.StatusBadRequest, "Invalid playthrough dates")
		} else {
			server.logger.Error("failed to update playthrough", "error", err)
			server.renderServerError(w, r)
		}
		return
	}

	http.Redirect(w, r, gameURL(igdbID)+"#"+playthroughAnchor(playthroughID), http.StatusSeeOther)
}

// handleDeletePlaythrough deletes a playthrough. It reads path values igdbID and playthroughID, and redirects to the
// playthroughs section of the game page. A playthrough that is already gone counts as success.
func (server *Server) handleDeletePlaythrough(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid IGDB ID")
		return
	}

	playthroughID, ok := pathID(r, "playthroughID")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid playthrough ID")
		return
	}

	err := server.store.DeletePlaythrough(r.Context(), defaultUserID, igdbID, playthroughID)
	if err != nil && !errors.Is(err, database.ErrPlaythroughNotFound) {
		if errors.Is(err, database.ErrUserGameNotFound) {
			server.renderError(w, r, http.StatusNotFound, "Game not in library")
		} else {
			server.logger.Error("failed to delete playthrough", "error", err)
			server.renderServerError(w, r)
		}
		return
	}

	http.Redirect(w, r, gameURL(igdbID)+"#playthroughs", http.StatusSeeOther)
}
