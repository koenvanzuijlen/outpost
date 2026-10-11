package database

import (
	"errors"
	"testing"
	"time"
)

func TestAddToCollectionClearsWishlist(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")

	if err := store.AddToWishlist(ctx, game, 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToWishlist() failed: %v", err)
	}
	if _, err := store.AddToCollection(ctx, game, newPlatformParams(6, "PC"), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToCollection() failed: %v", err)
	}

	wishlisted, err := store.queries.ListWishlistedUserGames(ctx, 1)
	if err != nil {
		t.Fatalf("ListWishlistedUserGames() failed: %v", err)
	}
	if len(wishlisted) != 0 {
		t.Errorf("len(wishlisted) = %d, want 0", len(wishlisted))
	}

	owned, err := store.queries.ListOwnedCopies(ctx, 1)
	if err != nil {
		t.Fatalf("ListOwnedCopies() failed: %v", err)
	}
	if len(owned) != 1 {
		t.Errorf("len(owned) = %d, want 1", len(owned))
	}
}

func TestAddToCollectionSameGameDifferentPlatforms(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")
	first, err := store.AddToCollection(ctx, game, newPlatformParams(11, "Xbox"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}
	second, err := store.AddToCollection(ctx, game, newPlatformParams(6, "PC"), 1, time.Now().UTC())
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
	store := newTestStore(t)
	platform := newPlatformParams(6, "PC")

	first, err := store.AddToCollection(ctx, newGameParams(100, "Test Game"), platform, 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}
	second, err := store.AddToCollection(ctx, newGameParams(101, "Test Game 2"), platform, 1, time.Now().UTC())
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
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")
	platform := newPlatformParams(6, "PC")

	if _, err := store.AddToCollection(ctx, game, platform, 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}
	if _, err := store.AddToCollection(ctx, game, platform, 1, time.Now().UTC()); !errors.Is(err, ErrAlreadyOwnedOnPlatform) {
		t.Fatalf("AddToCollection() error = %v, want %v", err, ErrAlreadyOwnedOnPlatform)
	}

	items, err := store.queries.ListOwnedCopies(ctx, 1)
	if err != nil {
		t.Fatalf("ListOwnedCopies() failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}
}

func TestAddToCollectionPreservesStatus(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")

	first, err := store.AddToCollection(ctx, game, newPlatformParams(6, "PC"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}

	_, err = store.db.ExecContext(ctx, "UPDATE user_games SET status = ?", "completed")
	if err != nil {
		t.Fatalf("ExecContext() failed: %v", err)
	}

	second, err := store.AddToCollection(ctx, game, newPlatformParams(11, "Xbox"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() second failed: %v", err)
	}

	if first.UserGameID != second.UserGameID {
		t.Errorf("UserGameID = %d and %d, want same game", first.UserGameID, second.UserGameID)
	}

	ownedCopies, err := store.queries.ListOwnedCopies(ctx, 1)
	if err != nil {
		t.Fatalf("ListOwnedCopies() failed: %v", err)
	}
	if len(ownedCopies) != 2 {
		t.Fatalf("ListOwnedCopies() returned %d items, want 2", len(ownedCopies))
	}
	status := ownedCopies[0].UserGame.Status
	if status == nil || *status != "completed" {
		t.Errorf("Status = %v, want %q", status, "completed")
	}
}

func TestAddToCollectionUpdatesGameInfo(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	first, err := store.AddToCollection(ctx, newGameParams(100, "Old Title"), newPlatformParams(6, "PC"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() first failed: %v", err)
	}

	cover := "https://example.com/cover.jpg"
	updated := newGameParams(100, "New Title")
	updated.CoverUrl = &cover
	second, err := store.AddToCollection(ctx, updated, newPlatformParams(11, "Xbox"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() second failed: %v", err)
	}
	if first.UserGameID != second.UserGameID {
		t.Errorf("UserGameID = %d and %d, want same game", first.UserGameID, second.UserGameID)
	}

	game, err := store.queries.GetGameByIGDBID(ctx, 100)
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
