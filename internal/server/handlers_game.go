package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

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
	LibraryGame   database.LibraryGame
	Statuses      []database.GameStatus
	CurrentStatus database.GameStatus
	RatingOptions []ratingOption
	CurrentRating int64
}

// handleGame serves the detail page for a game in the user's library.
func (server *Server) handleGame(w http.ResponseWriter, r *http.Request) {
	igdbID, err := strconv.ParseInt(r.PathValue("igdbID"), 10, 64)
	if err != nil || igdbID <= 0 {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	libraryGame, err := server.store.GetLibraryGame(r.Context(), defaultUserID, igdbID)
	if err != nil {
		if errors.Is(err, database.ErrUserGameNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			server.logger.Error("failed to get library game", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	server.render(w, "game.html", gamePage{
		LibraryGame:   libraryGame,
		Statuses:      database.GameStatuses,
		CurrentStatus: database.GameStatus(derefOrZero(libraryGame.UserGame.Status)),
		RatingOptions: ratingOptions(),
		CurrentRating: derefOrZero(libraryGame.UserGame.Rating),
	})
}

// handleUpdateGame processes updates to a game's status and rating in the user's library.
func (server *Server) handleUpdateGame(w http.ResponseWriter, r *http.Request) {
	igdbID, err := strconv.ParseInt(r.PathValue("igdbID"), 10, 64)
	if err != nil || igdbID <= 0 {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	status := database.GameStatus(r.PostFormValue("status"))
	if !status.Valid() {
		http.Error(w, "Invalid game status", http.StatusBadRequest)
		return
	}

	ratingValue := r.PostFormValue("rating")
	var rating *int64
	if ratingValue != "" {
		ratingParsed, err := strconv.ParseInt(ratingValue, 10, 64)
		if err != nil || ratingParsed <= 0 || ratingParsed > 10 {
			http.Error(w, "Invalid rating", http.StatusBadRequest)
			return
		}
		rating = &ratingParsed
	}

	err = server.store.UpdateStatusAndRating(r.Context(), defaultUserID, igdbID, status, rating)
	if err != nil {
		if errors.Is(err, database.ErrUserGameNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
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
