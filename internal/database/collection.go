package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// ErrAlreadyInCollection is returned when trying to add a game to the collection that is already present.
var ErrAlreadyInCollection = errors.New("game is already in the collection on this platform")

// AddToCollection adds a game to the collection, creating the game and platform if they don't exist, and then creating
// the collection item.
func AddToCollection(
	ctx context.Context,
	db *sql.DB,
	game dbgen.UpsertGameParams,
	platform dbgen.UpsertPlatformParams,
	item dbgen.CreateCollectionItemParams,
) (dbgen.CollectionItem, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return dbgen.CollectionItem{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	storedGame, err := q.UpsertGame(ctx, game)
	if err != nil {
		return dbgen.CollectionItem{}, fmt.Errorf("upsert game: %w", err)
	}

	storedPlatform, err := q.UpsertPlatform(ctx, platform)
	if err != nil {
		return dbgen.CollectionItem{}, fmt.Errorf("upsert platform: %w", err)
	}

	item.GameID = storedGame.ID
	item.PlatformID = storedPlatform.ID
	createdItem, err := q.CreateCollectionItem(ctx, item)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
			return dbgen.CollectionItem{}, ErrAlreadyInCollection
		}
		return dbgen.CollectionItem{}, fmt.Errorf("create collection item: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return dbgen.CollectionItem{}, fmt.Errorf("commit tx: %w", err)
	}
	return createdItem, nil
}
