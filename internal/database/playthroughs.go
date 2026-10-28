package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// ErrPlaythroughNotFound is returned when a playthrough doesn't exist or doesn't belong to the given game.
var ErrPlaythroughNotFound = errors.New("playthrough not found")

// ErrInvalidPlaythroughDates is returned when playthrough dates don't fit its outcome: a finished date while playing,
// or finished before started.
var ErrInvalidPlaythroughDates = errors.New("invalid playthrough dates")

// StartPlaythrough starts a playthrough of a library game, optionally named and on a platform. If the game is on the
// backlog, its status becomes playing. Returns ErrUserGameNotFound if the game isn't in the library, or
// ErrPlatformNotFound if the platform isn't in the catalog.
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

	startedAtDate := dateOf(startedAt)
	playthrough, err := q.CreatePlaythrough(ctx, dbgen.CreatePlaythroughParams{
		UserGameID: userGame.ID,
		PlatformID: platformID,
		Name:       name,
		StartedAt:  &startedAtDate,
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

// SetPlaythroughOutcome sets the outcome of a playthrough and updates finished_at to match. When this ends the last
// playing playthrough of a game that is playing, the game status becomes the outcome. Setting the same outcome again is
// a no-op. Returns ErrInvalidPlaythroughOutcome if the outcome is unknown, ErrUserGameNotFound if the game isn't in the
// library, or ErrPlaythroughNotFound if the playthrough doesn't belong to it.
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
		FinishedAt: finishedAtFor(currentOutcome, outcome, playthrough.FinishedAt, dateOf(changedAt)),
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

// PlaythroughDetails are the fields of a playthrough that UpdatePlaythroughDetails overwrites. A nil field is cleared.
type PlaythroughDetails struct {
	PlatformIGDBID *int64
	Name           *string
	StartedAt      *time.Time
	FinishedAt     *time.Time
}

// UpdatePlaythroughDetails updates the name, platform and dates of a playthrough. Every field is overwritten; nil
// clears it. Returns ErrUserGameNotFound if the game isn't in the library, ErrPlaythroughNotFound if the playthrough
// doesn't belong to it, ErrInvalidPlaythroughDates if the dates don't fit its outcome, or ErrPlatformNotFound if the
// platform isn't in the catalog.
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

	startedAtDate := optionalDateOf(details.StartedAt)
	finishedAtDate := optionalDateOf(details.FinishedAt)
	if !validPlaythroughDates(PlaythroughOutcome(playthrough.Outcome), startedAtDate, finishedAtDate) {
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
		StartedAt:  startedAtDate,
		FinishedAt: finishedAtDate,
	})
	if err != nil {
		return fmt.Errorf("update playthrough: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// DeletePlaythrough deletes a playthrough of a library game. The game status is not changed. Returns
// ErrUserGameNotFound if the game isn't in the library, or ErrPlaythroughNotFound if the playthrough doesn't belong to
// it.
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

// dateOf returns the calendar date of t in its own location, as midnight UTC
func dateOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// optionalDateOf returns the calendar date of t in its own location, as midnight UTC. Returns nil if the input is nil.
func optionalDateOf(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	return new(dateOf(*t))
}

// validPlaythroughDates reports whether the dates fit a playthrough with the given outcome: no finished date while
// playing, and not finished before started. Both dates are optional.
func validPlaythroughDates(outcome PlaythroughOutcome, startedAt, finishedAt *time.Time) bool {
	if outcome == PlaythroughOutcomePlaying && finishedAt != nil {
		return false
	}
	if finishedAt != nil && startedAt != nil && finishedAt.Before(*startedAt) {
		return false
	}
	return true
}
