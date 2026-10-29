package database

import (
	"context"
	"fmt"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// SaveGame caches a game in the catalog with the platforms it is released on, in one transaction. The platform list
// replaces the stored one; the cached DLC list (dlcs_fetched_at) is kept.
func (s *Store) SaveGame(
	ctx context.Context,
	game dbgen.UpsertGameParams,
	platforms []dbgen.UpsertPlatformParams,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	err = saveGameWithPlatforms(ctx, q, game, platforms)
	if err != nil {
		return fmt.Errorf("upsert game with platforms: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// saveGameWithPlatforms upserts game and replaces its platforms in game_platforms with platforms. It runs on q, so
// callers can use it inside a transaction.
func saveGameWithPlatforms(
	ctx context.Context,
	q *dbgen.Queries,
	game dbgen.UpsertGameParams,
	platforms []dbgen.UpsertPlatformParams,
) error {
	storedGame, err := q.UpsertGame(ctx, game)
	if err != nil {
		return fmt.Errorf("upsert game: %w", err)
	}
	err = q.DeleteGamePlatforms(ctx, storedGame.ID)
	if err != nil {
		return fmt.Errorf("delete game platform: %w", err)
	}
	for _, platform := range platforms {
		storedPlatform, err := q.UpsertPlatform(ctx, platform)
		if err != nil {
			return fmt.Errorf("upsert platform: %w", err)
		}
		err = q.CreateGamePlatform(ctx, dbgen.CreateGamePlatformParams{
			GameID:     storedGame.ID,
			PlatformID: storedPlatform.ID,
		})
		if err != nil {
			return fmt.Errorf("create game platform: %w", err)
		}
	}
	return nil
}
