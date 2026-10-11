package database

import (
	"errors"
	"testing"
	"time"
)

func TestAddToWishlist(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")

	if err := store.AddToWishlist(ctx, game, 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToWishlist() failed: %v", err)
	}

	wishlisted, err := store.queries.ListWishlistedUserGames(ctx, 1)
	if err != nil {
		t.Fatalf("ListWishlistedUserGames() failed: %v", err)
	}

	if len(wishlisted) != 1 {
		t.Fatalf("len(wishlisted) = %d, want 1", len(wishlisted))
	}
	if wishlisted[0].Game.Title != "Test Game" {
		t.Errorf("wishlisted[0].Game.Title = %q, want %q", wishlisted[0].Game.Title, "Test Game")
	}
	if wishlisted[0].UserGame.Status != nil {
		t.Errorf("wishlisted[0].UserGame.Status = %v, want %v", wishlisted[0].UserGame.Status, nil)
	}
}

func TestAddToWishlistDuplicate(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")
	if err := store.AddToWishlist(ctx, game, 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToWishlist() first failed: %v", err)
	}
	if err := store.AddToWishlist(ctx, game, 1, time.Now().UTC()); !errors.Is(err, ErrAlreadyInLibrary) {
		t.Fatalf("AddToWishlist() second error = %v, want %v", err, ErrAlreadyInLibrary)
	}
}

func TestAddToWishlistOwnedGame(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")
	platform := newPlatformParams(6, "Test Platform")

	// Add the game to the collection first
	if _, err := store.AddToCollection(ctx, game, platform, 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToCollection() failed: %v", err)
	}

	// Attempt to add the same game to the wishlist
	if err := store.AddToWishlist(ctx, game, 1, time.Now().UTC()); !errors.Is(err, ErrAlreadyInLibrary) {
		t.Fatalf("AddToWishlist() error = %v, want %v", err, ErrAlreadyInLibrary)
	}

	// Ensure wishlist is empty
	wishlisted, err := store.queries.ListWishlistedUserGames(ctx, 1)
	if err != nil {
		t.Fatalf("ListWishlistedUserGames() failed: %v", err)
	}
	if len(wishlisted) != 0 {
		t.Fatalf("len(wishlisted) = %d, want 0", len(wishlisted))
	}

	// Ensure the game is still in the collection
	collected, err := store.queries.ListOwnedCopies(ctx, 1)
	if err != nil {
		t.Fatalf("ListOwnedCopies() failed: %v", err)
	}
	if len(collected) != 1 {
		t.Fatalf("len(collected) = %d, want 1", len(collected))
	}
	if collected[0].Game.Title != "Test Game" {
		t.Errorf("collected[0].Game.Title = %q, want %q", collected[0].Game.Title, "Test Game")
	}
}
