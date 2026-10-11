package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// ErrAlreadyInLibrary is returned when wishlisting a game that is already in the library.
var ErrAlreadyInLibrary = errors.New("game is already in the library")

// AddToWishlist adds a game to the user's wishlist, creating the game if it doesn't exist. If the game is already in
// the user's games, it returns ErrAlreadyInLibrary.
func AddToWishlist(
	ctx context.Context,
	db *sql.DB,
	game dbgen.UpsertGameParams,
	userID int64,
	addedAt time.Time,
) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)

	storedGame, err := q.UpsertGame(ctx, game)
	if err != nil {
		return fmt.Errorf("upsert game: %w", err)
	}

	rows, err := q.CreateWishlistedUserGame(ctx, dbgen.CreateWishlistedUserGameParams{
		UserID:  userID,
		GameID:  storedGame.ID,
		AddedAt: addedAt,
	})
	if err != nil {
		return fmt.Errorf("create wishlisted user game: %w", err)
	}
	if rows == 0 {
		return ErrAlreadyInLibrary
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
