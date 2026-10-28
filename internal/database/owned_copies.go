package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// ErrAlreadyOwnedOnPlatform is returned when adding a copy on a platform the game is already owned on.
var ErrAlreadyOwnedOnPlatform = errors.New("game is already owned on this platform")

// ErrOwnedCopyNotFound is returned when removing a copy that doesn't exist.
var ErrOwnedCopyNotFound = errors.New("owned copy not found")

// RemoveOwnedCopy removes the user's copy of a game on the given platform, together with the DLC ownership on that
// copy. The library entry and its playthroughs stay, also when it was the last copy. Returns ErrUserGameNotFound if the
// game isn't in the library, or ErrOwnedCopyNotFound if there is no copy on that platform.
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
