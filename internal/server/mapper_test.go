package server

import (
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

func TestUpsertGameParamsFromIGDB(t *testing.T) {
	released := time.Date(2001, time.November, 15, 0, 0, 0, 0, time.UTC)
	fetchedAt := time.Date(2026, time.October, 6, 12, 0, 0, 0, time.UTC)

	game := igdb.Game{
		ID:               740,
		ParentGameID:     456,
		Name:             "Halo: Combat Evolved",
		GameType:         igdb.GameTypeMain,
		Cover:            igdb.Cover{URL: "//images.igdb.com/igdb/image/upload/t_thumb/co2r2r.jpg"},
		FirstReleaseDate: int(released.Unix()),
		Summary:          "The first game in the Halo series.",
	}

	params := upsertGameParamsFromIGDB(game, fetchedAt)

	if params.IgdbID != 740 {
		t.Errorf("IgdbID = %d, want %d", params.IgdbID, 740)
	}
	if params.Title != "Halo: Combat Evolved" {
		t.Errorf("Title = %q, want %q", params.Title, "Halo: Combat Evolved")
	}
	if params.GameType != int64(igdb.GameTypeMain) {
		t.Errorf("GameType = %d, want %d", params.GameType, igdb.GameTypeMain)
	}
	if !params.FetchedAt.Equal(fetchedAt) {
		t.Errorf("FetchedAt = %v, want %v", params.FetchedAt, fetchedAt)
	}

	if params.ParentIgdbID == nil || *params.ParentIgdbID != 456 {
		t.Errorf("ParentIgdbID = %v, want 456", derefOrNil(params.ParentIgdbID))
	}
	if params.Summary == nil || *params.Summary != "The first game in the Halo series." {
		t.Errorf("Summary = %v, want %q", derefOrNil(params.Summary), "The first game in the Halo series.")
	}

	wantCover := "https://images.igdb.com/igdb/image/upload/t_cover_big/co2r2r.jpg"
	if params.CoverUrl == nil || *params.CoverUrl != wantCover {
		t.Errorf("CoverUrl = %v, want %q", derefOrNil(params.CoverUrl), wantCover)
	}
	if params.ReleaseYear == nil || *params.ReleaseYear != 2001 {
		t.Errorf("ReleaseYear = %v, want 2001", derefOrNil(params.ReleaseYear))
	}
}

func TestUpsertGameParamsFromIGDBMissingFields(t *testing.T) {
	// Only the fields IGDB always returns. Everything else is its zero value.
	game := igdb.Game{
		ID:       740,
		Name:     "Halo: Combat Evolved",
		GameType: igdb.GameTypeMain,
	}

	params := upsertGameParamsFromIGDB(game, time.Date(2026, time.October, 6, 12, 0, 0, 0, time.UTC))

	if params.ParentIgdbID != nil {
		t.Errorf("ParentIgdbID = %d, want nil", *params.ParentIgdbID)
	}
	if params.Summary != nil {
		t.Errorf("Summary = %q, want nil", *params.Summary)
	}
	if params.CoverUrl != nil {
		t.Errorf("CoverUrl = %q, want nil", *params.CoverUrl)
	}
	if params.ReleaseYear != nil {
		t.Errorf("ReleaseYear = %d, want nil", *params.ReleaseYear)
	}
}

func TestUpsertPlatformParamsFromIGDB(t *testing.T) {
	platform := igdb.Platform{
		ID:   11,
		Name: "Xbox",
	}
	fetchedAt := time.Date(2026, time.October, 6, 12, 0, 0, 0, time.UTC)

	params := upsertPlatformParamsFromIGDB(platform, fetchedAt)

	if params.IgdbID != 11 {
		t.Errorf("IgdbID = %d, want %d", params.IgdbID, 11)
	}
	if params.Name != "Xbox" {
		t.Errorf("Name = %q, want %q", params.Name, "Xbox")
	}
	if !params.FetchedAt.Equal(fetchedAt) {
		t.Errorf("FetchedAt = %v, want %v", params.FetchedAt, fetchedAt)
	}
}

// derefOrNil returns the value p points to, or nil, so failure messages show a value instead of an address.
func derefOrNil[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}
