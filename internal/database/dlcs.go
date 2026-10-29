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

// ErrDLCNotFound is returned when a game isn't a DLC of the given parent game.
var ErrDLCNotFound = errors.New("DLC not found")

// ErrDLCNotOnCopy is returned when a DLC isn't available on an owned copy: there is no copy on that platform, the DLC
// belongs to another game, or it isn't released on that platform.
var ErrDLCNotOnCopy = errors.New("DLC not available for this copy")

// dlcGameTypes returns the IGDB game types that count as DLC (igdb.DLCGameTypes), as int64 for sqlc.
func dlcGameTypes() []int64 {
	dlcTypes := make([]int64, 0, len(igdb.DLCGameTypes))
	for _, t := range igdb.DLCGameTypes {
		dlcTypes = append(dlcTypes, int64(t))
	}
	return dlcTypes
}

// DLC is a DLC of a parent game with the platforms it is released on, as SaveDLCs caches it.
type DLC struct {
	Game      dbgen.UpsertGameParams
	Platforms []dbgen.UpsertPlatformParams
}

// CopyDLC is a DLC that is available on one owned copy, with whether it is owned on that copy.
type CopyDLC struct {
	Game  dbgen.Game
	Owned bool
}

// SaveDLCs caches the DLC of a game: it upserts each DLC with its platforms and marks the parent's DLC list as fetched,
// in one transaction. DLC that IGDB no longer lists are not removed. Returns an error if the parent game isn't in the
// catalog.
func (s *Store) SaveDLCs(ctx context.Context, parentIGDBID int64, dlcs []DLC, fetchedAt time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	for _, dlc := range dlcs {
		err := saveGameWithPlatforms(ctx, q, dlc.Game, dlc.Platforms)
		if err != nil {
			return fmt.Errorf("upsert game with platforms: %w", err)
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

// SetDLCOwned marks a DLC as owned or not owned on the user's copy of its parent game on the given platform. Setting
// the same value again is a no-op. Returns ErrUserGameNotFound if the parent game isn't in the library, or
// ErrDLCNotOnCopy if the DLC isn't available for that copy.
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

// AddDLCToLibrary adds a DLC of a library game to the library as a separate entry with status backlog, so it gets its
// own status, rating and playthroughs. Ownership is not changed, and adding it again is a no-op. Returns
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
