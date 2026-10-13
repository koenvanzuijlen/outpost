package database

import (
	"errors"
	"testing"
	"time"
)

func TestGetLibraryGameCollection(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Add a game to the user's collection before attempting to retrieve it
	_, err := store.AddToCollection(ctx, newGameParams(100, "Test Game"), newPlatformParams(11, "Xbox"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() failed: %v", err)
	}

	// Attempt to retrieve the row from the library
	row, err := store.GetLibraryGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if row.Game.Title != "Test Game" {
		t.Errorf("Title = %v, want %v", row.Game.Title, "Test Game")
	}
	if row.UserGame.Status == nil || *row.UserGame.Status != "backlog" {
		t.Errorf("UserGame.Status = %v, want %v", derefOrNil(row.UserGame.Status), "backlog")
	}
	if row.UserGame.Wishlisted {
		t.Errorf("UserGame.Wishlisted = %v, want %v", row.UserGame.Wishlisted, false)
	}
	if len(row.Platforms) != 1 {
		t.Fatalf("len(Platforms) = %d, want 1", len(row.Platforms))
	}
	if row.Platforms[0].Name != "Xbox" {
		t.Errorf("Platform.Name = %v, want %v", row.Platforms[0].Name, "Xbox")
	}
}

func TestGetLibraryGameWishlist(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Add a game to the user's wishlist before attempting to retrieve it
	err := store.AddToWishlist(ctx, newGameParams(100, "Test Game"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToWishlist() failed: %v", err)
	}

	// Attempt to retrieve the row from the library
	row, err := store.GetLibraryGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if row.Game.Title != "Test Game" {
		t.Errorf("Title = %v, want %v", row.Game.Title, "Test Game")
	}
	if row.UserGame.Status != nil {
		t.Errorf("UserGame.Status = %v, want %v", derefOrNil(row.UserGame.Status), nil)
	}
	if !row.UserGame.Wishlisted {
		t.Errorf("UserGame.Wishlisted = %v, want %v", row.UserGame.Wishlisted, true)
	}
	if len(row.Platforms) != 0 {
		t.Errorf("len(Platforms) = %d, want 0", len(row.Platforms))
	}
}

func TestGetLibraryGameNotFound(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Attempt to retrieve a game that doesn't exist in the library
	_, err := store.GetLibraryGame(ctx, 1, 999)
	if !errors.Is(err, ErrUserGameNotFound) {
		t.Fatalf("GetLibraryGame() error = %v, want %v", err, ErrUserGameNotFound)
	}
}

func TestUpdateStatusAndRating(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Add a game to the user's collection before updating it
	_, err := store.AddToCollection(ctx, newGameParams(100, "Test Game"), newPlatformParams(11, "Xbox"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() failed: %v", err)
	}

	// Update the status and rating of the game in the user's library
	rating := int64(5)
	err = store.UpdateStatusAndRating(ctx, 1, 100, GameStatusPlaying, &rating)
	if err != nil {
		t.Fatalf("UpdateStatusAndRating() failed: %v", err)
	}

	// Attempt to retrieve the row from the library
	row, err := store.GetLibraryGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if row.Game.Title != "Test Game" {
		t.Errorf("Title = %v, want %v", row.Game.Title, "Test Game")
	}
	if row.UserGame.Status == nil || *row.UserGame.Status != "playing" {
		t.Errorf("UserGame.Status = %v, want %v", derefOrNil(row.UserGame.Status), "playing")
	}
	if row.UserGame.Rating == nil || *row.UserGame.Rating != 5 {
		t.Errorf("UserGame.Rating = %v, want %v", derefOrNil(row.UserGame.Rating), 5)
	}
}

func TestUpdateStatusAndRatingInvalidStatus(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Add a game to the user's collection before updating it
	_, err := store.AddToCollection(ctx, newGameParams(100, "Test Game"), newPlatformParams(11, "Xbox"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() failed: %v", err)
	}

	// Attempt to update the game with an invalid status, which should be rejected
	rating := int64(5)
	err = store.UpdateStatusAndRating(ctx, 1, 100, "invalid_status", &rating)
	if !errors.Is(err, ErrInvalidGameStatus) {
		t.Fatalf("UpdateStatusAndRating() error = %v, want %v", err, ErrInvalidGameStatus)
	}

	// Verify the status and rating were left unchanged
	row, err := store.GetLibraryGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if row.Game.Title != "Test Game" {
		t.Errorf("Title = %v, want %v", row.Game.Title, "Test Game")
	}
	if row.UserGame.Status == nil || *row.UserGame.Status != "backlog" {
		t.Errorf("UserGame.Status = %v, want %v", derefOrNil(row.UserGame.Status), "backlog")
	}
	if row.UserGame.Rating != nil {
		t.Errorf("UserGame.Rating = %v, want %v", derefOrNil(row.UserGame.Rating), nil)
	}
}

func TestUpdateStatusAndRatingClearRating(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Add a game to the user's collection
	_, err := store.AddToCollection(ctx, newGameParams(100, "Test Game"), newPlatformParams(11, "Xbox"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() failed: %v", err)
	}

	// Update the status and rating of the game in the user's library
	rating := int64(5)
	err = store.UpdateStatusAndRating(ctx, 1, 100, GameStatusPlaying, &rating)
	if err != nil {
		t.Fatalf("UpdateStatusAndRating() failed: %v", err)
	}

	// Verify the rating was set
	row, err := store.GetLibraryGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if row.UserGame.Rating == nil || *row.UserGame.Rating != 5 {
		t.Errorf("UserGame.Rating = %v, want %v", derefOrNil(row.UserGame.Rating), 5)
	}

	// Clear the rating by passing nil
	err = store.UpdateStatusAndRating(ctx, 1, 100, GameStatusPlaying, nil)
	if err != nil {
		t.Fatalf("UpdateStatusAndRating() failed: %v", err)
	}

	// Verify the rating was cleared
	row, err = store.GetLibraryGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if row.UserGame.Rating != nil {
		t.Errorf("UserGame.Rating = %v, want %v", derefOrNil(row.UserGame.Rating), nil)
	}
}

func TestUpdateStatusAndRatingWishlistedGame(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Add a game to the user's wishlist only
	err := store.AddToWishlist(ctx, newGameParams(100, "Test Game"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToWishlist() failed: %v", err)
	}

	// Attempt to update a wishlisted game, which should fail since it isn't owned
	err = store.UpdateStatusAndRating(ctx, 1, 100, GameStatusPlaying, nil)
	if !errors.Is(err, ErrUserGameNotFound) {
		t.Fatalf("UpdateStatusAndRating() error = %v, want %v", err, ErrUserGameNotFound)
	}

	// Verify the wishlisted game was left without a status
	row, err := store.GetLibraryGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if row.UserGame.Status != nil {
		t.Errorf("UserGame.Status = %v, want %v", derefOrNil(row.UserGame.Status), nil)
	}
}

func TestUpdateStatusAndRatingNotInLibrary(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Attempt to update a game that doesn't exist in the library
	err := store.UpdateStatusAndRating(ctx, 1, 100, GameStatusPlaying, nil)
	if !errors.Is(err, ErrUserGameNotFound) {
		t.Fatalf("UpdateStatusAndRating() error = %v, want %v", err, ErrUserGameNotFound)
	}
}
