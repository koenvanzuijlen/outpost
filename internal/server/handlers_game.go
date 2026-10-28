package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
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
	Details        database.UserGameDetails
	Statuses       []database.GameStatus
	CurrentStatus  database.GameStatus
	RatingOptions  []ratingOption
	CurrentRating  int64
	OutcomeOptions []outcomeOption
	MaxNotesLength int64
}

// handleGame renders the game page of a game in the library or on the wishlist. On the first view of a game it fetches
// and caches its DLC; if that fails, the error is logged and the page renders without DLC.
func (server *Server) handleGame(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	details, err := server.store.GetUserGame(r.Context(), defaultUserID, igdbID)
	if err != nil {
		if errors.Is(err, database.ErrUserGameNotFound) {
			http.Error(w, "Game not in library", http.StatusNotFound)
		} else {
			server.logger.Error("failed to get library game", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	if details.Game.DlcsFetchedAt == nil && !igdb.GameType(details.Game.GameType).DLC() {
		err := server.fetchDLCs(r.Context(), igdbID)
		if err != nil {
			server.logger.Warn("failed to fetch DLCs", "igdb_id", igdbID, "error", err)
		} else {
			details, err = server.store.GetUserGame(r.Context(), defaultUserID, igdbID)
			if err != nil {
				server.logger.Error("failed to get library game", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		}
	}

	server.render(w, "game.html", gamePage{
		Details:        details,
		Statuses:       database.GameStatuses,
		CurrentStatus:  database.GameStatus(derefOrZero(details.UserGame.Status)),
		RatingOptions:  ratingOptions(),
		CurrentRating:  derefOrZero(details.UserGame.Rating),
		OutcomeOptions: outcomeOptions(),
		MaxNotesLength: maxNotesLength,
	})
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
