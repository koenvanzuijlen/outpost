package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// ErrUserGameNotFound is returned when a game is not found in the user's library.
var ErrUserGameNotFound = errors.New("game not found in library")

// LibraryGame represents a game in the user's library, including the game details, the user's game entry, and the
// platforms the user owns it on.
type LibraryGame struct {
	Game      dbgen.Game
	UserGame  dbgen.UserGame
	Platforms []dbgen.Platform
}

// GetLibraryGame retrieves a game from the user's library by its IGDB ID. Returns ErrUserGameNotFound if the game is
// not in the library.
func (s *Store) GetLibraryGame(ctx context.Context, userID, igdbID int64) (LibraryGame, error) {
	row, err := s.queries.GetUserGameByIGDBID(ctx, dbgen.GetUserGameByIGDBIDParams{
		UserID: userID,
		IgdbID: igdbID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return LibraryGame{}, ErrUserGameNotFound
		}
		return LibraryGame{}, fmt.Errorf("get user game: %w", err)
	}

	platforms, err := s.queries.ListPlatformsForUserGameID(ctx, row.UserGame.ID)
	if err != nil {
		return LibraryGame{}, fmt.Errorf("list owned platforms: %w", err)
	}

	return LibraryGame{
		Game:      row.Game,
		UserGame:  row.UserGame,
		Platforms: platforms,
	}, nil
}

// UpdateStatusAndRating updates the status and rating of a game in the user's library. Returns ErrUserGameNotFound if
// the game is not in the library.
func (s *Store) UpdateStatusAndRating(ctx context.Context, userID, igdbID int64, status GameStatus, rating *int64) error {
	if !status.Valid() {
		return ErrInvalidGameStatus
	}
	statusString := string(status)
	updatedRows, err := s.queries.UpdateUserGameStatusAndRating(ctx, dbgen.UpdateUserGameStatusAndRatingParams{
		Status: &statusString,
		Rating: rating,
		UserID: userID,
		IgdbID: igdbID,
	})
	if err != nil {
		return fmt.Errorf("update status and rating: %w", err)
	}
	if updatedRows == 0 {
		return ErrUserGameNotFound
	}
	return nil
}
