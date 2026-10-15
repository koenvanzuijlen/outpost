package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// ErrAlreadyOwnedOnPlatform is returned when trying to add an owned copy that is already present.
var ErrAlreadyOwnedOnPlatform = errors.New("game is already owned on this platform")

// ListCollection returns the user's collection of owned copies. It returns one row per owned copy, so a game owned on
// multiple platforms will appear multiple times.
func (s *Store) ListCollection(ctx context.Context, userID int64) ([]dbgen.ListOwnedCopiesRow, error) {
	return s.queries.ListOwnedCopies(ctx, userID)
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
