package database

import (
	"errors"
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

func newGameParams(igdbID int64, title string) dbgen.UpsertGameParams {
	return dbgen.UpsertGameParams{
		IgdbID:    igdbID,
		FetchedAt: time.Now().UTC(),
		Title:     title,
		GameType:  0,
	}
}

func newPlatformParams(igdbID int64, name string) dbgen.UpsertPlatformParams {
	return dbgen.UpsertPlatformParams{
		IgdbID:    igdbID,
		FetchedAt: time.Now().UTC(),
		Name:      name,
	}
}

func newCollectionItemParams() dbgen.CreateCollectionItemParams {
	return dbgen.CreateCollectionItemParams{
		UserID: 1,
		Status: "backlog",
	}
}

func TestAddToCollectionSameGameDifferentPlatforms(t *testing.T) {
	ctx := t.Context()
	db := newTestDatabase(t)
	game := newGameParams(100, "Test Game")
	item := newCollectionItemParams()

	first, err := AddToCollection(ctx, db, game, newPlatformParams(11, "Xbox"), item)
	if err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}
	second, err := AddToCollection(ctx, db, game, newPlatformParams(6, "PC"), item)
	if err != nil {
		t.Fatalf("AddToCollection() second failed: %v", err)
	}

	if first.ID == second.ID {
		t.Errorf("collection item IDs are equal (%d), want distinct items", first.ID)
	}
	if first.GameID != second.GameID {
		t.Errorf("GameID = %d and %d, want same game", first.GameID, second.GameID)
	}
	if first.PlatformID == second.PlatformID {
		t.Errorf("PlatformID = %d and %d, want distinct platforms", first.PlatformID, second.PlatformID)
	}
}

func TestAddToCollectionDifferentGamesSamePlatform(t *testing.T) {
	ctx := t.Context()
	db := newTestDatabase(t)
	platform := newPlatformParams(6, "PC")
	item := newCollectionItemParams()

	first, err := AddToCollection(ctx, db, newGameParams(100, "Test Game"), platform, item)
	if err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}
	second, err := AddToCollection(ctx, db, newGameParams(101, "Test Game 2"), platform, item)
	if err != nil {
		t.Fatalf("AddToCollection() second failed: %v", err)
	}

	if first.ID == second.ID {
		t.Errorf("collection item IDs are equal (%d), want distinct items", first.ID)
	}
	if first.GameID == second.GameID {
		t.Errorf("GameID = %d and %d, want distinct games", first.GameID, second.GameID)
	}
	if first.PlatformID != second.PlatformID {
		t.Errorf("PlatformID = %d and %d, want same platform", first.PlatformID, second.PlatformID)
	}
}

func TestAddToCollectionSameGameSamePlatform(t *testing.T) {
	ctx := t.Context()
	db := newTestDatabase(t)
	game := newGameParams(100, "Test Game")
	platform := newPlatformParams(6, "PC")
	item := newCollectionItemParams()

	if _, err := AddToCollection(ctx, db, game, platform, item); err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}
	if _, err := AddToCollection(ctx, db, game, platform, item); !errors.Is(err, ErrAlreadyInCollection) {
		t.Fatalf("AddToCollection() error = %v, want %v", err, ErrAlreadyInCollection)
	}

	items, err := dbgen.New(db).ListCollectionItemsWithGames(ctx, 1)
	if err != nil {
		t.Fatalf("ListCollectionItemsWithGames() failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}
}

func TestAddToCollectionUpdatesGameInfo(t *testing.T) {
	ctx := t.Context()
	db := newTestDatabase(t)
	item := newCollectionItemParams()

	first, err := AddToCollection(ctx, db, newGameParams(100, "Old Title"), newPlatformParams(6, "PC"), item)
	if err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}

	cover := "https://example.com/cover.jpg"
	updated := newGameParams(100, "New Title")
	updated.CoverUrl = &cover
	second, err := AddToCollection(ctx, db, updated, newPlatformParams(11, "Xbox"), item)
	if err != nil {
		t.Fatalf("AddToCollection() second failed: %v", err)
	}
	if first.GameID != second.GameID {
		t.Errorf("GameID = %d and %d, want same game", first.GameID, second.GameID)
	}

	game, err := dbgen.New(db).GetGameByIGDBID(ctx, 100)
	if err != nil {
		t.Fatalf("GetGameByIGDBID() failed: %v", err)
	}
	if game.Title != "New Title" {
		t.Errorf("Title = %q, want %q", game.Title, "New Title")
	}
	if game.CoverUrl == nil || *game.CoverUrl != cover {
		t.Errorf("CoverUrl = %v, want %q", game.CoverUrl, cover)
	}
}
