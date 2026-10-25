package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// ErrPlatformNotFound is returned when a platform is not found in the database.
var ErrPlatformNotFound = errors.New("platform not found")

// platformIDFor retrieves the internal platform ID for a given IGDB platform ID, if nil is provided it returns nil.
// Returns ErrPlatformNotFound if the platform does not exist.
func platformIDFor(ctx context.Context, q *dbgen.Queries, platformIGDBID *int64) (*int64, error) {
	if platformIGDBID == nil {
		return nil, nil
	}

	platform, err := q.GetPlatformByIGDBID(ctx, *platformIGDBID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlatformNotFound
		}
		return nil, fmt.Errorf("get platform by igdb id: %w", err)
	}
	return &platform.ID, nil
}
