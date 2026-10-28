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
	if err := store.AddToWishlist(ctx, game, 1, time.Now().UTC()); !errors.Is(err, ErrAlreadyAdded) {
		t.Fatalf("AddToWishlist() second error = %v, want %v", err, ErrAlreadyAdded)
	}
}

func TestAddToWishlistOwnedGame(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")
	platform := newPlatformParams(6, "Test Platform")

	// Add the game to the collection first
	if err := store.AddToLibrary(ctx, game, &platform, 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}

	// Attempt to add the same game to the wishlist
	if err := store.AddToWishlist(ctx, game, 1, time.Now().UTC()); !errors.Is(err, ErrAlreadyAdded) {
		t.Fatalf("AddToWishlist() error = %v, want %v", err, ErrAlreadyAdded)
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
	details := getUserGame(t, store, 100)
	if details.UserGame.Wishlisted {
		t.Errorf("Wishlisted = true, want false")
	}
	assertPlatformIDs(t, details.OwnedPlatforms, 6)
}
