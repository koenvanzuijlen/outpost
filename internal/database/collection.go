package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// AddToCollection adds a game to the collection, creating the game if it doesn't exist and then creating the collection
// item.
func AddToCollection(ctx context.Context, db *sql.DB, game dbgen.UpsertGameParams, item dbgen.CreateCollectionItemParams) (dbgen.CollectionItem, error) {
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

	item.GameID = storedGame.ID
	createdItem, err := q.CreateCollectionItem(ctx, item)
	if err != nil {
		return dbgen.CollectionItem{}, fmt.Errorf("create collection item: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return dbgen.CollectionItem{}, fmt.Errorf("commit tx: %w", err)
	}
	return createdItem, nil
}
