package server

import (
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// ptrIfNonZero returns a pointer to the given value if it is non-zero, otherwise it returns nil.
func ptrIfNonZero[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

// upsertGameParamsFromIGDB maps an IGDB game to the parameters required for the UpsertGame database operation.
func upsertGameParamsFromIGDB(game igdb.Game, fetchedAt time.Time) dbgen.UpsertGameParams {
	releaseYear := 0
	if game.FirstReleaseDate != 0 {
		releaseYear = time.Unix(int64(game.FirstReleaseDate), 0).UTC().Year()
	}
	return dbgen.UpsertGameParams{
		IgdbID:       int64(game.ID),
		FetchedAt:    fetchedAt,
		ParentIgdbID: ptrIfNonZero(int64(game.ParentGameID)),
		Title:        game.Name,
		GameType:     int64(game.GameType),
		CoverUrl:     ptrIfNonZero(game.CoverURL()),
		ReleaseYear:  ptrIfNonZero(int64(releaseYear)),
		Summary:      ptrIfNonZero(game.Summary),
	}
}

// upsertPlatformParamsFromIGDB maps an IGDB platform to the parameters required for the UpsertPlatform database operation.
func upsertPlatformParamsFromIGDB(platform igdb.Platform, fetchedAt time.Time) dbgen.UpsertPlatformParams {
	return dbgen.UpsertPlatformParams{
		IgdbID:    int64(platform.ID),
		FetchedAt: fetchedAt,
		Name:      platform.Name,
	}
}
