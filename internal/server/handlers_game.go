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
	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// maxNotesLength defines the maximum allowed length for game notes.
const maxNotesLength = 10_000

// ratingOption represents a single rating option for a game.
type ratingOption struct {
	Value int64
	Label string
}

// ratingOptions returns the ratings 1 to 10 with their star labels, for the rating select.
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

// outcomeOption is a playthrough outcome as offered on the game page.
type outcomeOption struct {
	Value string
	Label string
}

// outcomeOptions returns all playthrough outcomes in display order with their labels, for the outcome buttons.
func outcomeOptions() []outcomeOption {
	options := make([]outcomeOption, 0, len(database.PlaythroughOutcomes))
	for _, outcome := range database.PlaythroughOutcomes {
		options = append(options, outcomeOption{
			Value: string(outcome),
			Label: outcomeLabel(string(outcome)),
		})
	}
	return options
}

// gamePage represents the data needed to render the game detail page.
type gamePage struct {
	Details        database.GameOverview
	Statuses       []database.GameStatus
	CurrentStatus  database.GameStatus
	RatingOptions  []ratingOption
	CurrentRating  int64
	OutcomeOptions []outcomeOption
	MaxNotesLength int64
	CanAddGame     bool // the game is a standalone game outside the library and wishlist
	CanAddDLC      bool // the game is a DLC outside the library, and its base game is in the library
}

// fetchGame fetches a game from IGDB and caches it in the catalog with its platforms. Only the IGDB call is limited to 3
// seconds, because it blocks the page. Errors wrap the IGDB error, so callers can check for igdb.ErrGameNotFound.
func (server *Server) fetchGame(ctx context.Context, igdbID int64) error {
	now := time.Now().UTC()

	igdbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	game, err := server.igdbClient.GameByID(igdbCtx, int(igdbID))
	if err != nil {
		return fmt.Errorf("fetch from igdb: %w", err)
	}

	platforms := make([]dbgen.UpsertPlatformParams, 0, len(game.Platforms))
	for _, platform := range game.Platforms {
		platforms = append(platforms, upsertPlatformParamsFromIGDB(platform, now))
	}

	err = server.store.SaveGame(ctx, upsertGameParamsFromIGDB(game, now), platforms)
	if err != nil {
		return fmt.Errorf("save game: %w", err)
	}
	return nil
}

// handleGame renders the game page of any game: from the library, the wishlist, or only the catalog. Every view fetches
// the game from IGDB and caches it with its platforms; if that fails, the cached data is shown, or 404 (unknown to
// IGDB) or 500 when there is none. On the first view of a base game it fetches and caches its DLC; if that fails, the
// error is logged and the page renders without DLC.
func (server *Server) handleGame(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	details, err := server.store.GetGame(ctx, defaultUserID, igdbID)
	if err != nil && !errors.Is(err, database.ErrGameNotInCatalog) {
		server.logger.Error("failed to get game", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	cached := err == nil

	fetchErr := server.fetchGame(ctx, igdbID)
	switch {
	case fetchErr == nil:
		details, err = server.store.GetGame(ctx, defaultUserID, igdbID)
		if err != nil {
			server.logger.Error("failed to get game", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	case cached:
		server.logger.Warn("failed to refresh game, showing cached data", "igdb_id", igdbID, "error", fetchErr)
	case errors.Is(fetchErr, igdb.ErrGameNotFound):
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	default:
		server.logger.Error("failed to fetch game", "igdb_id", igdbID, "error", fetchErr)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if details.Game.DlcsFetchedAt == nil && !igdb.GameType(details.Game.GameType).DLC() {
		err := server.fetchDLCs(ctx, igdbID)
		if err != nil {
			server.logger.Warn("failed to fetch DLCs", "igdb_id", igdbID, "error", err)
		} else {
			details, err = server.store.GetGame(ctx, defaultUserID, igdbID)
			if err != nil {
				server.logger.Error("failed to get game", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		}
	}

	pageData := gamePage{
		Details:        details,
		Statuses:       database.GameStatuses,
		RatingOptions:  ratingOptions(),
		OutcomeOptions: outcomeOptions(),
		MaxNotesLength: maxNotesLength,
		CanAddGame: details.UserGame == nil &&
			igdb.GameType(details.Game.GameType).Standalone(),
		CanAddDLC: details.UserGame == nil &&
			igdb.GameType(details.Game.GameType).DLC() &&
			details.Parent != nil &&
			details.ParentInLibrary,
	}
	if details.UserGame != nil {
		pageData.CurrentStatus = database.GameStatus(derefOrZero(details.UserGame.Status))
		pageData.CurrentRating = derefOrZero(details.UserGame.Rating)
	}

	server.render(w, "game.html", pageData)
}

// handleUpdateGameDetails updates the status, rating and notes of a library game. It reads path value igdbID and form
// values status, rating (optional) and notes (optional), and redirects to the game page.
func (server *Server) handleUpdateGameDetails(w http.ResponseWriter, r *http.Request) {
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
