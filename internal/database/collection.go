package database

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// ErrAlreadyOwnedOnPlatform is returned when trying to add an owned copy that is already present.
var ErrAlreadyOwnedOnPlatform = errors.New("game is already owned on this platform")

// GameListEntry is one row of the collection list: an owned copy, or a tracked DLC on one of the copies it is
// owned on. PlatformName is nil for a tracked DLC that isn't owned on any copy.
type GameListEntry struct {
	Game         dbgen.Game
	UserGame     dbgen.UserGame
	PlatformName *string
}

// ListGames returns the user's collection of owned copies and tracked DLC games. It returns one row per owned copy or
// tracked DLC, so a game owned on multiple platforms will appear multiple times. If a DLC is tracked but not owned on
// any platform, it will still appear with PlatformName set to nil.
func (s *Store) ListGames(ctx context.Context, userID int64) ([]GameListEntry, error) {
	ownedCopies, err := s.queries.ListOwnedCopies(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list owned copies: %w", err)
	}

	trackedDLCs, err := s.queries.ListTrackedDLCs(ctx, dbgen.ListTrackedDLCsParams{
		UserID:    userID,
		GameTypes: dlcGameTypes(),
	})
	if err != nil {
		return nil, fmt.Errorf("list tracked dlcs: %w", err)
	}

	collection := make([]GameListEntry, 0, len(ownedCopies)+len(trackedDLCs))
	for _, oc := range ownedCopies {
		collection = append(collection, GameListEntry{
			Game:         oc.Game,
			UserGame:     oc.UserGame,
			PlatformName: &oc.Platform.Name,
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

// AddToCollection adds a game to the user's collection, creating the game and platform if they don't exist, and then
// creating the owned copy. New games to the collection are added with status "backlog", existing ones retain their current
// status. If the game is already in the collection on the specified platform, it returns ErrAlreadyOwnedOnPlatform.
func (s *Store) AddToCollection(
	ctx context.Context,
	game dbgen.UpsertGameParams,
	platform dbgen.UpsertPlatformParams,
	userID int64,
	addedAt time.Time,
) (dbgen.OwnedCopy, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return dbgen.OwnedCopy{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	storedGame, err := q.UpsertGame(ctx, game)
	if err != nil {
		return dbgen.OwnedCopy{}, fmt.Errorf("upsert game: %w", err)
	}

	storedPlatform, err := q.UpsertPlatform(ctx, platform)
	if err != nil {
		return dbgen.OwnedCopy{}, fmt.Errorf("upsert platform: %w", err)
	}

	storedUserGame, err := q.UpsertOwnedUserGame(ctx, dbgen.UpsertOwnedUserGameParams{
		UserID:  userID,
		GameID:  storedGame.ID,
		Status:  GameStatusBacklog.StringPtr(),
		AddedAt: addedAt,
	})
	if err != nil {
		return dbgen.OwnedCopy{}, fmt.Errorf("upsert owned user game: %w", err)
	}

	ownedCopy, err := q.CreateOwnedCopy(ctx, dbgen.CreateOwnedCopyParams{
		UserGameID: storedUserGame.ID,
		PlatformID: storedPlatform.ID,
	})
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
			return dbgen.OwnedCopy{}, ErrAlreadyOwnedOnPlatform
		}
		return dbgen.OwnedCopy{}, fmt.Errorf("create owned copy: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return dbgen.OwnedCopy{}, fmt.Errorf("commit tx: %w", err)
	}
	return ownedCopy, nil
}
