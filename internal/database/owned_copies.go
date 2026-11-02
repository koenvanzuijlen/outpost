package database

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// ErrAlreadyOwnedOnPlatform is returned when adding a copy on a platform the game is already owned on.
var ErrAlreadyOwnedOnPlatform = errors.New("game is already owned on this platform")

// ErrOwnedCopyNotFound is returned when removing a copy that doesn't exist.
var ErrOwnedCopyNotFound = errors.New("owned copy not found")

// AddOwnedCopy adds the user's copy of a library game on the given platform. The platform must be one the game is
// released on, according to the catalog. Returns ErrUserGameNotFound if the game isn't in the library,
// ErrPlatformNotFound if the platform isn't in the catalog, ErrPlatformNotForGame if the game isn't released on it, or
// ErrAlreadyOwnedOnPlatform if the copy already exists.
func (s *Store) AddOwnedCopy(ctx context.Context, userID, igdbID, platformIGDBID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	userGame, err := getLibraryUserGame(ctx, q, userID, igdbID)
	if err != nil {
		return fmt.Errorf("get library user game: %w", err)
	}

	platformID, err := platformIDFor(ctx, q, &platformIGDBID)
	if err != nil {
		return fmt.Errorf("get platform ID: %w", err)
	}

	platforms, err := q.ListPlatformsForGame(ctx, userGame.GameID)
	if err != nil {
		return fmt.Errorf("list platforms for game: %w", err)
	}

	if !slices.ContainsFunc(platforms, func(p dbgen.Platform) bool { return p.ID == *platformID }) {
		return ErrPlatformNotForGame
	}

	_, err = q.CreateOwnedCopy(ctx, dbgen.CreateOwnedCopyParams{
		UserGameID: userGame.ID,
		PlatformID: *platformID,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyOwnedOnPlatform
		}
		return fmt.Errorf("create owned copy: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

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
