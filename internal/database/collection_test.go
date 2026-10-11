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

func newUserGameParams() dbgen.UpsertOwnedUserGameParams {
	return dbgen.UpsertOwnedUserGameParams{
		UserID:  1,
		AddedAt: time.Now().UTC(),
	}
}

func TestAddToCollectionSameGameDifferentPlatforms(t *testing.T) {
	ctx := t.Context()
	db := newTestDatabase(t)
	game := newGameParams(100, "Test Game")
	userGame := newUserGameParams()

	first, err := AddToCollection(ctx, db, game, newPlatformParams(11, "Xbox"), userGame)
	if err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}
	second, err := AddToCollection(ctx, db, game, newPlatformParams(6, "PC"), userGame)
	if err != nil {
		t.Fatalf("AddToCollection() second failed: %v", err)
	}

	if first.ID == second.ID {
		t.Errorf("collection item IDs are equal (%d), want distinct items", first.ID)
	}
	if first.UserGameID != second.UserGameID {
		t.Errorf("UserGameID = %d and %d, want same game", first.UserGameID, second.UserGameID)
	}
	if first.PlatformID == second.PlatformID {
		t.Errorf("PlatformID = %d and %d, want distinct platforms", first.PlatformID, second.PlatformID)
	}
}

func TestAddToCollectionDifferentGamesSamePlatform(t *testing.T) {
	ctx := t.Context()
	db := newTestDatabase(t)
	platform := newPlatformParams(6, "PC")
	userGame := newUserGameParams()

	first, err := AddToCollection(ctx, db, newGameParams(100, "Test Game"), platform, userGame)
	if err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}
	second, err := AddToCollection(ctx, db, newGameParams(101, "Test Game 2"), platform, userGame)
	if err != nil {
		t.Fatalf("AddToCollection() second failed: %v", err)
	}

	if first.ID == second.ID {
		t.Errorf("collection item IDs are equal (%d), want distinct items", first.ID)
	}
	if first.UserGameID == second.UserGameID {
		t.Errorf("UserGameID = %d and %d, want distinct games", first.UserGameID, second.UserGameID)
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
	userGame := newUserGameParams()

	if _, err := AddToCollection(ctx, db, game, platform, userGame); err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}
	if _, err := AddToCollection(ctx, db, game, platform, userGame); !errors.Is(err, ErrAlreadyInCollection) {
		t.Fatalf("AddToCollection() error = %v, want %v", err, ErrAlreadyInCollection)
	}

	items, err := dbgen.New(db).ListOwnedCopies(ctx, 1)
	if err != nil {
		t.Fatalf("ListOwnedCopies() failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}
}

func TestAddToCollectionPreservesStatus(t *testing.T) {
	ctx := t.Context()
	db := newTestDatabase(t)
	game := newGameParams(100, "Test Game")
	userGame := newUserGameParams()

	first, err := AddToCollection(ctx, db, game, newPlatformParams(6, "PC"), userGame)
	if err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}

	_, err = db.ExecContext(ctx, "UPDATE user_games SET status = ?", "completed")
	if err != nil {
		t.Fatalf("Failed to update user game status: %v", err)
	}

	second, err := AddToCollection(ctx, db, game, newPlatformParams(11, "Xbox"), userGame)
	if err != nil && !errors.Is(err, ErrAlreadyInCollection) {
		t.Fatalf("AddToCollection() second failed: %v", err)
	}

	if first.UserGameID != second.UserGameID {
		t.Errorf("UserGameID = %d and %d, want same game", first.UserGameID, second.UserGameID)
	}

	ownedCopies, err := dbgen.New(db).ListOwnedCopies(ctx, 1)
	if err != nil {
		t.Fatalf("ListOwnedCopies() failed: %v", err)
	}
	if len(ownedCopies) != 2 {
		t.Fatalf("ListOwnedCopies() returned %d items, want 2", len(ownedCopies))
	}
	status := ownedCopies[0].UserGame.Status
	if status == nil || *status != "completed" {
		t.Errorf("Status = %v, want %q", *status, "completed")
	}
}

func TestAddToCollectionUpdatesGameInfo(t *testing.T) {
	ctx := t.Context()
	db := newTestDatabase(t)
	userGame := newUserGameParams()

	first, err := AddToCollection(ctx, db, newGameParams(100, "Old Title"), newPlatformParams(6, "PC"), userGame)
	if err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}

	cover := "https://example.com/cover.jpg"
	updated := newGameParams(100, "New Title")
	updated.CoverUrl = &cover
	second, err := AddToCollection(ctx, db, updated, newPlatformParams(11, "Xbox"), userGame)
	if err != nil {
		t.Fatalf("AddToCollection() second failed: %v", err)
	}
	if first.UserGameID != second.UserGameID {
		t.Errorf("UserGameID = %d and %d, want same game", first.UserGameID, second.UserGameID)
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
