package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// maxNotesLength defines the maximum allowed length for game notes.
const maxNotesLength = 10_000

// ratingOption represents a single rating option for a game.
type ratingOption struct {
	Value int64
	Label string
}

// ratingOptions returns a list of rating options from 1 to 10, with labels formatted as stars.
func ratingOptions() []ratingOption {
	options := make([]ratingOption, 10)
	for i := 1; i <= 10; i++ {
		options[i-1] = ratingOption{
			Value: int64(i),
			Label: ratingText(int64(i)),
		}
	}
	return options
}

// gamePage represents the data needed to render the game detail page.
type gamePage struct {
	LibraryGame         database.LibraryGame
	Statuses            []database.GameStatus
	CurrentStatus       database.GameStatus
	RatingOptions       []ratingOption
	CurrentRating       int64
	PlaythroughOutcomes []database.PlaythroughOutcome
	MaxNotesLength      int64
}

// handleGame serves the detail page for a game in the user's library.
func (server *Server) handleGame(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	libraryGame, err := server.store.GetLibraryGame(r.Context(), defaultUserID, igdbID)
	if err != nil {
		if errors.Is(err, database.ErrUserGameNotFound) {
			http.Error(w, "Game not in library", http.StatusNotFound)
		} else {
			server.logger.Error("failed to get library game", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	if libraryGame.Game.DlcsFetchedAt == nil && !igdb.GameType(libraryGame.Game.GameType).DLC() {
		err := server.fetchDLCs(r.Context(), igdbID)
		if err != nil {
			server.logger.Warn("failed to fetch DLCs", "igdb_id", igdbID, "error", err)
		} else {
			libraryGame, err = server.store.GetLibraryGame(r.Context(), defaultUserID, igdbID)
			if err != nil {
				server.logger.Error("failed to get library game", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		}
	}

	server.render(w, "game.html", gamePage{
		LibraryGame:         libraryGame,
		Statuses:            database.GameStatuses,
		CurrentStatus:       database.GameStatus(derefOrZero(libraryGame.UserGame.Status)),
		RatingOptions:       ratingOptions(),
		CurrentRating:       derefOrZero(libraryGame.UserGame.Rating),
		PlaythroughOutcomes: database.PlaythroughOutcomes,
		MaxNotesLength:      maxNotesLength,
	})
}

// handleUpdateGame processes updates to a game's status and rating in the user's library.
func (server *Server) handleUpdateGame(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	status := database.GameStatus(r.PostFormValue("status"))
	if !status.Valid() {
		http.Error(w, "Invalid game status", http.StatusBadRequest)
		return
	}

	rating, ok := formOptionalPositiveInt(r, "rating")
	if !ok || (rating != nil && *rating > 10) {
		http.Error(w, "Invalid rating", http.StatusBadRequest)
		return
	}

	notes := strings.TrimSpace(r.PostFormValue("notes"))
	if utf8.RuneCountInString(notes) > maxNotesLength {
		http.Error(w, "Notes too long", http.StatusBadRequest)
		return
	}

	err := server.store.UpdateGameDetails(r.Context(), defaultUserID, igdbID, database.GameDetails{
		Status: status,
		Rating: rating,
		Notes:  ptrIfNonZero(notes),
	})
	if err != nil {
		if errors.Is(err, database.ErrUserGameNotFound) {
			http.Error(w, "Game not in library", http.StatusNotFound)
		} else if errors.Is(err, database.ErrInvalidGameStatus) {
			http.Error(w, "Invalid game status", http.StatusBadRequest)
		} else {
			server.logger.Error("failed to update library game", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/games/"+strconv.FormatInt(igdbID, 10), http.StatusSeeOther)
}

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
