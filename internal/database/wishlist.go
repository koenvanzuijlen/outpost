package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// ErrAlreadyAdded is returned when wishlisting a game that is already in the library or on the wishlist.
var ErrAlreadyAdded = errors.New("game is already in the library or wishlist")

// ListWishlist returns all wishlisted games for the specified user.
func (s *Store) ListWishlist(ctx context.Context, userID int64) ([]dbgen.ListWishlistedUserGamesRow, error) {
	return s.queries.ListWishlistedUserGames(ctx, userID)
}

// AddToWishlist adds a game to the user's wishlist, storing the game in the catalog if it isn't there yet. If the game
// is already in the library or on the wishlist, it returns ErrAlreadyAdded.
func (s *Store) AddToWishlist(ctx context.Context, game dbgen.UpsertGameParams, userID int64, addedAt time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)

	storedGame, err := q.UpsertGame(ctx, game)
	if err != nil {
		return fmt.Errorf("upsert game: %w", err)
	}

	insertedRows, err := q.CreateWishlistedUserGame(ctx, dbgen.CreateWishlistedUserGameParams{
		UserID:  userID,
		GameID:  storedGame.ID,
		AddedAt: addedAt,
	})
	if err != nil {
		return fmt.Errorf("create wishlisted user game: %w", err)
	}
	if insertedRows == 0 {
		return ErrAlreadyAdded
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
