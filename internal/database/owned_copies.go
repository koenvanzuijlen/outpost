package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// ErrAlreadyOwnedOnPlatform is returned when trying to add an owned copy that is already present.
var ErrAlreadyOwnedOnPlatform = errors.New("game is already owned on this platform")

// ErrOwnedCopyNotFound is returned when trying to delete an owned copy that doesn't exist.
var ErrOwnedCopyNotFound = errors.New("owned copy not found")

// RemoveOwnedCopy removes an owned copy of a game for a user on a specific platform. It also removes DLC ownership for
// this game. The library entry and playthrough stay. It returns ErrOwnedCopyNotFound if the user does not own the game
// on the specified platform.
func (s *Store) RemoveOwnedCopy(ctx context.Context, userID, igdbID, platformIGDBID int64) error {
	userGame, err := getLibraryUserGame(ctx, s.queries, userID, igdbID)
	if err != nil {
		return fmt.Errorf("get library user game: %w", err)
	}

	deletedRows, err := s.queries.DeleteOwnedCopy(ctx, dbgen.DeleteOwnedCopyParams{
		UserGameID: userGame.ID,
		IgdbID:     platformIGDBID,
	})
	if err != nil {
		return fmt.Errorf("delete owned copy: %w", err)
	}
	if deletedRows == 0 {
		return ErrOwnedCopyNotFound
	}
	return nil
}
