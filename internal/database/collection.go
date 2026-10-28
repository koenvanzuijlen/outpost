package database

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// ErrAlreadyOwnedOnPlatform is returned when trying to add an owned copy that is already present.
var ErrAlreadyOwnedOnPlatform = errors.New("game is already owned on this platform")

// ErrOwnedCopyNotFound is returned when trying to delete an owned copy that doesn't exist.
var ErrOwnedCopyNotFound = errors.New("owned copy not found")

// GameListEntry is one row of the collection list: an owned copy, or a tracked DLC on one of the copies it is
// owned on. PlatformName is nil for a tracked DLC that isn't owned on any copy.
type GameListEntry struct {
	Game         dbgen.Game
	UserGame     dbgen.UserGame
	PlatformName *string
}

// ListGames returns the user's collection of standalone games and DLC, whether owned or not. It returns one row per
// owned copy of a game, so a game owned on multiple platforms will appear multiple times. If a game is tracked but not
// owned on any platform, it will still appear with PlatformName set to nil.
func (s *Store) ListGames(ctx context.Context, userID int64) ([]GameListEntry, error) {
	libraryGames, err := s.queries.ListLibraryGames(ctx, dbgen.ListLibraryGamesParams{
		UserID:       userID,
		DlcGameTypes: dlcGameTypes(),
	})
	if err != nil {
		return nil, fmt.Errorf("list library games: %w", err)
	}

	trackedDLCs, err := s.queries.ListTrackedDLCs(ctx, dbgen.ListTrackedDLCsParams{
		UserID:    userID,
		GameTypes: dlcGameTypes(),
	})
	if err != nil {
		return nil, fmt.Errorf("list tracked dlcs: %w", err)
	}

	collection := make([]GameListEntry, 0, len(libraryGames)+len(trackedDLCs))
	for _, game := range libraryGames {
		collection = append(collection, GameListEntry{
			Game:         game.Game,
			UserGame:     game.UserGame,
			PlatformName: game.PlatformName,
		})
	}
	for _, dlc := range trackedDLCs {
		collection = append(collection, GameListEntry{
			Game:         dlc.Game,
			UserGame:     dlc.UserGame,
			PlatformName: dlc.PlatformName,
		})
	}

	platformName := func(c GameListEntry) string {
		if c.PlatformName == nil {
			return ""
		}
		return *c.PlatformName
	}

	slices.SortFunc(collection, func(a, b GameListEntry) int {
		return cmp.Or(
			cmp.Compare(a.Game.Title, b.Game.Title),
			cmp.Compare(platformName(a), platformName(b)),
		)
	})

	return collection, nil
}

// RemoveOwnedCopy removes an owned copy of a game for a user on a specific platform. It also removes DLC ownership for
// this game. The library entry and playthrough stay. It returns ErrOwnedCopyNotFound if the user does not own the game
// on the specified platform.
func (s *Store) RemoveOwnedCopy(ctx context.Context, userID, igdbID, platformIGDBID int64) error {
	userGame, err := getOwnedUserGame(ctx, s.queries, userID, igdbID)
	if err != nil {
		return fmt.Errorf("get owned user game: %w", err)
	}

	deletedRows, err := s.queries.DeleteOwnedCopy(ctx, dbgen.DeleteOwnedCopyParams{
		UserGameID: userGame.UserGame.ID,
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
