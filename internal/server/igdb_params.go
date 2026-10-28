package server

import (
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// upsertGameParamsFromIGDB converts an IGDB game to UpsertGame parameters.
func upsertGameParamsFromIGDB(game igdb.Game, fetchedAt time.Time) dbgen.UpsertGameParams {
	releasedAt := time.Time{}
	if game.FirstReleaseDate != 0 {
		releasedAt = time.Unix(int64(game.FirstReleaseDate), 0).UTC()
	}
	return dbgen.UpsertGameParams{
		IgdbID:       int64(game.ID),
		FetchedAt:    fetchedAt,
		ParentIgdbID: ptrIfNonZero(int64(game.ParentGameID)),
		Title:        game.Name,
		GameType:     int64(game.GameType),
		CoverUrl:     ptrIfNonZero(game.CoverURL()),
		ReleasedAt:   ptrIfNonZero(releasedAt),
		Summary:      ptrIfNonZero(game.Summary),
	}
}

// upsertPlatformParamsFromIGDB converts an IGDB platform to UpsertPlatform parameters.
func upsertPlatformParamsFromIGDB(platform igdb.Platform, fetchedAt time.Time) dbgen.UpsertPlatformParams {
	return dbgen.UpsertPlatformParams{
		IgdbID:    int64(platform.ID),
		FetchedAt: fetchedAt,
		Name:      platform.Name,
	}
}

// dlcFromIGDB converts an IGDB DLC to a database.DLC, including its platforms.
func dlcFromIGDB(game igdb.Game, fetchedAt time.Time) database.DLC {
	platforms := make([]dbgen.UpsertPlatformParams, len(game.Platforms))
	for i, platform := range game.Platforms {
		platforms[i] = upsertPlatformParamsFromIGDB(platform, fetchedAt)
	}
	return database.DLC{
		Game:      upsertGameParamsFromIGDB(game, fetchedAt),
		Platforms: platforms,
	}
}
