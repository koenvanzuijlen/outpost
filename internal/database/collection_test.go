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

func newCollectionItemParams(platform string) dbgen.CreateCollectionItemParams {
	return dbgen.CreateCollectionItemParams{
		UserID:   1,
		Platform: platform,
		Status:   "backlog",
	}
}

func TestAddToCollectionSameGameDifferentPlatforms(t *testing.T) {
	ctx := t.Context()
	db := newTestDatabase(t)
	game := newGameParams(100, "Test Game")

	first, err := AddToCollection(ctx, db, game, newCollectionItemParams("pc"))
	if err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}
	second, err := AddToCollection(ctx, db, game, newCollectionItemParams("switch"))
	if err != nil {
		t.Fatalf("AddToCollection() second failed: %v", err)
	}

	if first.ID == second.ID {
		t.Errorf("collection item IDs are equal (%d), want distinct items", first.ID)
	}
	if first.GameID != second.GameID {
		t.Errorf("GameID = %d and %d, want same game", first.GameID, second.GameID)
	}
}

func TestAddToCollectionSameGameSamePlatform(t *testing.T) {
	ctx := t.Context()
	db := newTestDatabase(t)
	game := newGameParams(100, "Test Game")

	if _, err := AddToCollection(ctx, db, game, newCollectionItemParams("pc")); err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}
	if _, err := AddToCollection(ctx, db, game, newCollectionItemParams("pc")); !errors.Is(err, ErrAlreadyInCollection) {
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

	first, err := AddToCollection(ctx, db, newGameParams(100, "Old Title"), newCollectionItemParams("pc"))
	if err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}

	cover := "https://example.com/cover.jpg"
	updated := newGameParams(100, "New Title")
	updated.CoverUrl = &cover
	second, err := AddToCollection(ctx, db, updated, newCollectionItemParams("switch"))
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
