package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// ErrPlaythroughNotFound is returned when a playthrough is not found in the database.
var ErrPlaythroughNotFound = errors.New("playthrough not found")

// ErrInvalidPlaythroughDates is returned when the playthrough dates are invalid.
var ErrInvalidPlaythroughDates = errors.New("invalid playthrough dates")

// StartPlaythrough starts a new playthrough for a user's game, it also marks the game status itself as playing if it
// was previously in the backlog. Returns the created playthrough, or ErrPlatformNotFound, ErrUserGameNotFound, or
// another error if the operation fails.
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

	userGame, err := getLibraryUserGame(ctx, q, userID, igdbID)
	if err != nil {
		return dbgen.Playthrough{}, fmt.Errorf("get library user game: %w", err)
	}

	platformID, err := platformIDFor(ctx, q, platformIGDBID)
	if err != nil {
		return dbgen.Playthrough{}, fmt.Errorf("resolve platform: %w", err)
	}

	playthrough, err := q.CreatePlaythrough(ctx, dbgen.CreatePlaythroughParams{
		UserGameID: userGame.ID,
		PlatformID: platformID,
		Name:       name,
		StartedAt:  &startedAt,
	})
	if err != nil {
		return dbgen.Playthrough{}, fmt.Errorf("create playthrough: %w", err)
	}

	if gameStatusFrom(userGame.Status) == GameStatusBacklog {
		err = q.UpdateUserGameStatus(ctx, dbgen.UpdateUserGameStatusParams{
			ID:     userGame.ID,
			Status: GameStatusPlaying.StringPtr(),
		})
		if err != nil {
			return dbgen.Playthrough{}, fmt.Errorf("update user game status: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return dbgen.Playthrough{}, fmt.Errorf("commit tx: %w", err)
	}
	return playthrough, nil
}

// SetPlaythroughOutcome sets the outcome of an existing playthrough for a user's game, it also updates the game status
// itself based on the outcome when it ends the last playing playthrough of a game that is playing. Setting the same
// outcome again does nothing. Returns ErrPlaythroughNotFound, ErrUserGameNotFound,
// ErrInvalidPlaythroughOutcome, or another error if the operation fails.
func (s *Store) SetPlaythroughOutcome(
	ctx context.Context,
	userID,
	igdbID,
	playthroughID int64,
	outcome PlaythroughOutcome,
	changedAt time.Time,
) error {
	if !outcome.Valid() {
		return ErrInvalidPlaythroughOutcome
	}

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

	playthrough, err := q.GetPlaythrough(ctx, dbgen.GetPlaythroughParams{
		ID:         playthroughID,
		UserGameID: userGame.ID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPlaythroughNotFound
		}
		return fmt.Errorf("get playthrough: %w", err)
	}

	currentOutcome := PlaythroughOutcome(playthrough.Outcome)
	if currentOutcome == outcome {
		return nil
	}

	err = q.UpdatePlaythroughOutcome(ctx, dbgen.UpdatePlaythroughOutcomeParams{
		ID:         playthroughID,
		Outcome:    string(outcome),
		FinishedAt: finishedAtFor(currentOutcome, outcome, playthrough.FinishedAt, changedAt),
	})
	if err != nil {
		return fmt.Errorf("update playthrough outcome: %w", err)
	}

	otherPlaying, err := q.CountPlayingPlaythroughs(ctx, userGame.ID)
	if err != nil {
		return fmt.Errorf("count playthroughs: %w", err)
	}

	nextStatus, ok := statusAfterOutcomeChange(currentOutcome, outcome, gameStatusFrom(userGame.Status), otherPlaying)
	if ok {
		err = q.UpdateUserGameStatus(ctx, dbgen.UpdateUserGameStatusParams{
			ID:     userGame.ID,
			Status: nextStatus.StringPtr(),
		})
		if err != nil {
			return fmt.Errorf("update user game status: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// PlaythroughDetails represents the details of a playthrough that can be updated. Fields set to nil will be cleared.
type PlaythroughDetails struct {
	PlatformIGDBID *int64
	Name           *string
	StartedAt      *time.Time
	FinishedAt     *time.Time
}

// UpdatePlaythroughDetails updates the details of an existing playthrough for a user's game. If fields are provided as
// nil their data will be cleared. Returns ErrPlaythroughNotFound, ErrUserGameNotFound, ErrInvalidPlaythroughDates,
// ErrPlatformNotFound, or another error if the operation fails.
func (s *Store) UpdatePlaythroughDetails(ctx context.Context, userID, igdbID, playthroughID int64, details PlaythroughDetails) error {
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

	playthrough, err := q.GetPlaythrough(ctx, dbgen.GetPlaythroughParams{
		ID:         playthroughID,
		UserGameID: userGame.ID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPlaythroughNotFound
		}
		return fmt.Errorf("get playthrough: %w", err)
	}

	if !validPlaythroughDates(PlaythroughOutcome(playthrough.Outcome), details.StartedAt, details.FinishedAt) {
		return ErrInvalidPlaythroughDates
	}

	platformID, err := platformIDFor(ctx, q, details.PlatformIGDBID)
	if err != nil {
		return fmt.Errorf("resolve platform: %w", err)
	}

	err = q.UpdatePlaythroughDetails(ctx, dbgen.UpdatePlaythroughDetailsParams{
		ID:         playthroughID,
		Name:       details.Name,
		PlatformID: platformID,
		StartedAt:  details.StartedAt,
		FinishedAt: details.FinishedAt,
	})
	if err != nil {
		return fmt.Errorf("update playthrough: %w", err)
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
	userGame, err := getLibraryUserGame(ctx, s.queries, userID, igdbID)
	if err != nil {
		return fmt.Errorf("get library user game: %w", err)
	}

	deletedRows, err := s.queries.DeletePlaythrough(ctx, dbgen.DeletePlaythroughParams{
		ID:         playthroughID,
		UserGameID: userGame.ID,
	})
	if err != nil {
		return fmt.Errorf("delete playthrough: %w", err)
	}
	if deletedRows == 0 {
		return ErrPlaythroughNotFound
	}

	return nil
}

// validPlaythroughDates checks if the provided playthrough dates are valid based on the outcome. Returns true if the
// dates are valid, false otherwise.
func validPlaythroughDates(outcome PlaythroughOutcome, startedAt, finishedAt *time.Time) bool {
	if outcome == PlaythroughOutcomePlaying && finishedAt != nil {
		return false
	}
	if finishedAt != nil && startedAt != nil && finishedAt.Before(*startedAt) {
		return false
	}
	return true
}
