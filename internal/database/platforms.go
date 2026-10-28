package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// ErrPlatformNotFound is returned when a platform isn't in the catalog.
var ErrPlatformNotFound = errors.New("platform not found")

// platformIDFor returns the internal ID of the platform with the given IGDB ID, or nil if platformIGDBID is nil.
// Returns ErrPlatformNotFound if the platform isn't in the catalog.
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
