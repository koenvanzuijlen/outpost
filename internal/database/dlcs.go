package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// ErrDLCNotFound is returned when a DLC is not a DLC for the given parent game.
var ErrDLCNotFound = errors.New("DLC not found")

// ErrCopyDLCNotFound is returned when a specific copy of a DLC is not found in the database.
var ErrCopyDLCNotFound = errors.New("DLC copy not found")

// dlcGameTypes returns a slice of int64 representing the IGDB game types that are considered DLCs.
func dlcGameTypes() []int64 {
	dlcTypes := make([]int64, 0, len(igdb.DLCGameTypes))
	for _, t := range igdb.DLCGameTypes {
		dlcTypes = append(dlcTypes, int64(t))
	}
	return dlcTypes
}

// DLC represents a downloadable content (DLC) for a parent game, including its associated platforms.
type DLC struct {
	Game      dbgen.UpsertGameParams
	Platforms []dbgen.UpsertPlatformParams
}

// CopyDLC represents a DLC for a parent game along with its ownership status for a specific owned copy.
type CopyDLC struct {
	Game  dbgen.Game
	Owned bool
}

// SaveDLCs upserts the provided DLCs for the given parent IGDB ID and stores the fetched timestamp. If the parent game
// is not in the database, an error is returned.
func (s *Store) SaveDLCs(ctx context.Context, parentIGDBID int64, dlcs []DLC, fetchedAt time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	for _, dlc := range dlcs {
		storedGame, err := q.UpsertGame(ctx, dlc.Game)
		if err != nil {
			return fmt.Errorf("upsert game: %w", err)
		}
		err = q.DeleteGamePlatforms(ctx, storedGame.ID)
		if err != nil {
			return fmt.Errorf("delete game platform: %w", err)
		}
		for _, platform := range dlc.Platforms {
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
	}

	updatedRows, err := q.MarkDLCsFetched(ctx, dbgen.MarkDLCsFetchedParams{
		DlcsFetchedAt: &fetchedAt,
		IgdbID:        parentIGDBID,
	})
	if err != nil {
		return fmt.Errorf("mark dlcs fetched: %w", err)
	}
	if updatedRows == 0 {
		return fmt.Errorf("parent game %d not in database", parentIGDBID)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// SetDLCOwned sets the ownership status of a DLC for a specific user's owned copy. If owned is true, the DLC is marked
// as owned; otherwise, it is removed from the owned DLCs. Both directions are idempotent. "Returns ErrUserGameNotFound
// if the game isn't owned, or ErrCopyDLCNotFound if the DLC isn't available for that copy
func (s *Store) SetDLCOwned(ctx context.Context, userID, igdbID, platformIGDBID, dlcIGDBID int64, owned bool) error {
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

	ownedCopy, err := q.GetCopyDLC(ctx, dbgen.GetCopyDLCParams{
		UserGameID:     row.UserGame.ID,
		PlatformIgdbID: platformIGDBID,
		DlcIgdbID:      dlcIGDBID,
		ParentIgdbID:   &igdbID,
		GameTypes:      dlcGameTypes(),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCopyDLCNotFound
		}
		return fmt.Errorf("get copy dlc: %w", err)
	}

	if owned {
		err = q.CreateOwnedDLC(ctx, dbgen.CreateOwnedDLCParams(ownedCopy))
		if err != nil {
			return fmt.Errorf("create owned dlc: %w", err)
		}
	} else {
		err = q.DeleteOwnedDLC(ctx, dbgen.DeleteOwnedDLCParams(ownedCopy))
		if err != nil {
			return fmt.Errorf("delete owned dlc: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// TrackDLC gives a DLC of an owned game its own library entry with status backlog, so it gets its own status, rating
// and playthroughs. Tracking twice has no additional effect. Returns ErrUserGameNotFound if the user doesn't own the
// parent game, or ErrDLCNotFound if the DLC isn't available for the parent game.
func (s *Store) TrackDLC(ctx context.Context, userID, parentIGDBID, dlcIGDBID int64, addedAt time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	_, err = getOwnedUserGame(ctx, q, userID, parentIGDBID)
	if err != nil {
		return fmt.Errorf("get owned user game: %w", err)
	}

	dlc, err := q.GetDLCGame(ctx, dbgen.GetDLCGameParams{
		DlcIgdbID:    dlcIGDBID,
		ParentIgdbID: &parentIGDBID,
		GameTypes:    dlcGameTypes(),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrDLCNotFound
		}
		return fmt.Errorf("get dlc game: %w", err)
	}

	err = q.CreateTrackedUserGame(ctx, dbgen.CreateTrackedUserGameParams{
		UserID:  userID,
		GameID:  dlc.ID,
		Status:  GameStatusBacklog.StringPtr(),
		AddedAt: addedAt,
	})
	if err != nil {
		return fmt.Errorf("create tracked user game: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
