package database

import (
	"context"
	"fmt"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// dlcGameTypes returns a slice of int64 representing the IGDB game types that are considered DLCs.
func dlcGameTypes() []int64 {
	dlcTypes := make([]int64, 0, len(igdb.DLCGameTypes))
	for _, t := range igdb.DLCGameTypes {
		dlcTypes = append(dlcTypes, int64(t))
	}
	return dlcTypes
}

// SaveDLCs upserts the provided DLCs for the given parent IGDB ID and stores the fetched timestamp. If the parent game
// is not in the database, an error is returned.
func (s *Store) SaveDLCs(ctx context.Context, parentIGDBID int64, dlcs []dbgen.UpsertGameParams, fetchedAt time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	for _, dlc := range dlcs {
		if _, err := q.UpsertGame(ctx, dlc); err != nil {
			return fmt.Errorf("upsert game: %w", err)
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
