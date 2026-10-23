package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// ErrPlaythroughNotFound is returned when a playthrough is not found in the database.
var ErrPlaythroughNotFound = errors.New("playthrough not found")

// StartPlaythrough starts a new playthrough for a user's game, it also marks the game status itself as playing. Returns
// the created playthrough, or ErrPlatformNotFound, ErrUserGameNotFound, or another error if the operation fails.
func (s *Store) StartPlaythrough(
	ctx context.Context,
	userID,
	igdbID int64,
	platformIGDBID *int64,
	name *string,
	startedAt time.Time,
) (dbgen.Playthrough, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return dbgen.Playthrough{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	row, err := getOwnedUserGame(ctx, q, userID, igdbID)
	if err != nil {
		return dbgen.Playthrough{}, fmt.Errorf("get owned user game: %w", err)
	}

	var platformID *int64
	if platformIGDBID != nil {
		platform, err := q.GetPlatformByIGDBID(ctx, *platformIGDBID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return dbgen.Playthrough{}, ErrPlatformNotFound
			}
			return dbgen.Playthrough{}, fmt.Errorf("get platform by igdb id: %w", err)
		}
		platformID = &platform.ID
	}

	pt, err := q.CreatePlaythrough(ctx, dbgen.CreatePlaythroughParams{
		UserGameID: row.UserGame.ID,
		PlatformID: platformID,
		Name:       name,
		StartedAt:  &startedAt,
	})
	if err != nil {
		return dbgen.Playthrough{}, fmt.Errorf("create playthrough: %w", err)
	}

	err = q.UpdateUserGameStatus(ctx, dbgen.UpdateUserGameStatusParams{
		ID:     row.UserGame.ID,
		Status: GameStatusPlaying.StringPtr(),
	})
	if err != nil {
		return dbgen.Playthrough{}, fmt.Errorf("update user game status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return dbgen.Playthrough{}, fmt.Errorf("commit tx: %w", err)
	}
	return pt, nil
}

// FinishPlaythrough finishes an existing playthrough for a user's game, it also updates the game status itself based on
// the outcome. Returns ErrPlaythroughNotFound, ErrUserGameNotFound, ErrInvalidPlaythroughOutcome, or another error if
// the operation fails.
func (s *Store) FinishPlaythrough(
	ctx context.Context,
	userID,
	igdbID,
	playthroughID int64,
	outcome PlaythroughOutcome,
	finishedAt time.Time,
) error {
	if !slices.Contains(FinishOutcomes, outcome) {
		return ErrInvalidPlaythroughOutcome
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	row, err := getOwnedUserGame(ctx, q, userID, igdbID)
	if err != nil {
		return fmt.Errorf("get owned user game: %w", err)
	}

	rows, err := q.FinishPlaythrough(ctx, dbgen.FinishPlaythroughParams{
		ID:         playthroughID,
		UserGameID: row.UserGame.ID,
		Outcome:    string(outcome),
		FinishedAt: &finishedAt,
	})
	if err != nil {
		return fmt.Errorf("finish playthrough: %w", err)
	}
	if rows == 0 {
		return ErrPlaythroughNotFound
	}

	err = q.UpdateUserGameStatus(ctx, dbgen.UpdateUserGameStatusParams{
		ID:     row.UserGame.ID,
		Status: GameStatus(outcome).StringPtr(),
	})
	if err != nil {
		return fmt.Errorf("update user game status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// DeletePlaythrough deletes an existing playthrough for a user's game, the user game status remains unchanged. Returns
// ErrPlaythroughNotFound, ErrUserGameNotFound, or another error if the operation fails.
func (s *Store) DeletePlaythrough(
	ctx context.Context,
	userID,
	igdbID,
	playthroughID int64,
) error {
	row, err := getOwnedUserGame(ctx, s.queries, userID, igdbID)
	if err != nil {
		return fmt.Errorf("get owned user game: %w", err)
	}

	rows, err := s.queries.DeletePlaythrough(ctx, dbgen.DeletePlaythroughParams{
		ID:         playthroughID,
		UserGameID: row.UserGame.ID,
	})
	if err != nil {
		return fmt.Errorf("delete playthrough: %w", err)
	}
	if rows == 0 {
		return ErrPlaythroughNotFound
	}

	return nil
}
