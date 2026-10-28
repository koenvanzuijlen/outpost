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

// ErrDLCNotOnCopy is returned when a DLC isn't available on an owned copy: there is no copy on that platform, the DLC
// belongs to another game, or it isn't released on that platform.
var ErrDLCNotOnCopy = errors.New("DLC not available for this copy")

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
// as owned; otherwise, it is removed from the owned DLCs. Both directions are idempotent. Returns ErrUserGameNotFound
// if the game isn't in the library, or ErrDLCNotOnCopy if the DLC isn't available for that copy.
func (s *Store) SetDLCOwned(ctx context.Context, userID, igdbID, platformIGDBID, dlcIGDBID int64, owned bool) error {
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

	copyDLC, err := q.GetCopyDLC(ctx, dbgen.GetCopyDLCParams{
		UserGameID:     userGame.ID,
		PlatformIgdbID: platformIGDBID,
		DlcIgdbID:      dlcIGDBID,
		ParentIgdbID:   &igdbID,
		GameTypes:      dlcGameTypes(),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrDLCNotOnCopy
		}
		return fmt.Errorf("get copy dlc: %w", err)
	}

	if owned {
		err = q.CreateOwnedDLC(ctx, dbgen.CreateOwnedDLCParams(copyDLC))
		if err != nil {
			return fmt.Errorf("create owned dlc: %w", err)
		}
	} else {
		err = q.DeleteOwnedDLC(ctx, dbgen.DeleteOwnedDLCParams(copyDLC))
		if err != nil {
			return fmt.Errorf("delete owned dlc: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// AddDLCToLibrary gives a DLC of a library game its own library entry with status backlog, so it gets its own status,
// rating and playthroughs. Ownership (owned_dlc) is not changed. Adding it twice has no additional effect. Returns
// ErrUserGameNotFound if the parent game isn't in the library, or ErrDLCNotFound if the DLC isn't a DLC of it.
func (s *Store) AddDLCToLibrary(ctx context.Context, userID, parentIGDBID, dlcIGDBID int64, addedAt time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	_, err = getLibraryUserGame(ctx, q, userID, parentIGDBID)
	if err != nil {
		return fmt.Errorf("get library user game: %w", err)
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

	err = q.CreateLibraryUserGame(ctx, dbgen.CreateLibraryUserGameParams{
		UserID:  userID,
		GameID:  dlc.ID,
		Status:  GameStatusBacklog.StringPtr(),
		AddedAt: addedAt,
	})
	if err != nil {
		return fmt.Errorf("create library user game: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
