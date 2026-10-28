package database

import (
	"errors"
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

func TestGetLibraryGameCollection(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Add a game to the user's collection before attempting to retrieve it
	err := store.AddToLibrary(ctx, newGameParams(100, "Test Game"), new(newPlatformParams(11, "Xbox")), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
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
	if len(row.Playthroughs) != 0 {
		t.Errorf("len(Playthroughs) = %d, want 0", len(row.Playthroughs))
	}
}

func TestGetLibraryGameWithPlaythroughs(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	platformIGDBID := int64(11)

	// Add a game to the user's collection before attempting to retrieve it
	err := store.AddToLibrary(ctx, newGameParams(100, "Test Game"), new(newPlatformParams(platformIGDBID, "Xbox")), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}

	name := "Test Playthrough newer"
	_, err = store.StartPlaythrough(ctx, 1, 100, &platformIGDBID, &name, time.Now().UTC().Add(time.Hour))
	if err != nil {
		t.Fatalf("StartPlaythrough() failed: %v", err)
	}

	name = "Test Playthrough older"
	_, err = store.StartPlaythrough(ctx, 1, 100, nil, &name, time.Now().UTC())
	if err != nil {
		t.Fatalf("StartPlaythrough() failed: %v", err)
	}

	// Attempt to retrieve the row from the library
	row, err := store.GetLibraryGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if len(row.Playthroughs) != 2 {
		t.Fatalf("len(Playthroughs) = %d, want 2", len(row.Playthroughs))
	}
	// Should be ordered by the time they were started, with the newest first
	if row.Playthroughs[0].Playthrough.Name == nil || *row.Playthroughs[0].Playthrough.Name != "Test Playthrough newer" {
		t.Errorf("Playthroughs[0].Name = %v, want %v", derefOrNil(row.Playthroughs[0].Playthrough.Name), "Test Playthrough newer")
	}
	if row.Playthroughs[0].PlatformName == nil || *row.Playthroughs[0].PlatformName != "Xbox" {
		t.Errorf("Playthroughs[0].PlatformName = %v, want %v", derefOrNil(row.Playthroughs[0].PlatformName), "Xbox")
	}
	if row.Playthroughs[1].Playthrough.Name == nil || *row.Playthroughs[1].Playthrough.Name != "Test Playthrough older" {
		t.Errorf("Playthroughs[1].Name = %v, want %v", derefOrNil(row.Playthroughs[1].Playthrough.Name), "Test Playthrough older")
	}
	if row.Playthroughs[1].PlatformName != nil {
		t.Errorf("Playthroughs[1].PlatformName = %v, want %v", derefOrNil(row.Playthroughs[1].PlatformName), nil)
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

func TestUpdateGameDetails(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Add a game to the user's collection before updating it
	err := store.AddToLibrary(ctx, newGameParams(100, "Test Game"), new(newPlatformParams(11, "Xbox")), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}

	// Update the status and rating of the game in the user's library
	rating := int64(5)
	err = store.UpdateGameDetails(ctx, 1, 100, GameDetails{Status: GameStatusPlaying, Rating: &rating})
	if err != nil {
		t.Fatalf("UpdateGameDetails() failed: %v", err)
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

func TestUpdateGameDetailsInvalidStatus(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Add a game to the user's collection before updating it
	err := store.AddToLibrary(ctx, newGameParams(100, "Test Game"), new(newPlatformParams(11, "Xbox")), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}

	// Attempt to update the game with an invalid status, which should be rejected
	rating := int64(5)
	err = store.UpdateGameDetails(ctx, 1, 100, GameDetails{Status: "invalid_status", Rating: &rating})
	if !errors.Is(err, ErrInvalidGameStatus) {
		t.Fatalf("UpdateGameDetails() error = %v, want %v", err, ErrInvalidGameStatus)
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

func TestUpdateGameDetailsClearRating(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Add a game to the user's collection
	err := store.AddToLibrary(ctx, newGameParams(100, "Test Game"), new(newPlatformParams(11, "Xbox")), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}

	// Update the status and rating of the game in the user's library
	rating := int64(5)
	err = store.UpdateGameDetails(ctx, 1, 100, GameDetails{Status: GameStatusPlaying, Rating: &rating})
	if err != nil {
		t.Fatalf("UpdateGameDetails() failed: %v", err)
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
	err = store.UpdateGameDetails(ctx, 1, 100, GameDetails{Status: GameStatusPlaying})
	if err != nil {
		t.Fatalf("UpdateGameDetails() failed: %v", err)
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

func TestUpdateGameDetailsNotes(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	err := store.AddToLibrary(ctx, newGameParams(100, "Test Game"), new(newPlatformParams(11, "Xbox")), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}

	rating := int64(7)
	notes := "Buy the GOTY edition"
	err = store.UpdateGameDetails(ctx, 1, 100, GameDetails{Status: GameStatusPlaying, Rating: &rating, Notes: &notes})
	if err != nil {
		t.Fatalf("UpdateGameDetails() failed: %v", err)
	}

	row, err := store.GetLibraryGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if row.UserGame.Status == nil || *row.UserGame.Status != "playing" {
		t.Errorf("UserGame.Status = %v, want %v", derefOrNil(row.UserGame.Status), "playing")
	}
	if row.UserGame.Rating == nil || *row.UserGame.Rating != 7 {
		t.Errorf("UserGame.Rating = %v, want %v", derefOrNil(row.UserGame.Rating), 7)
	}
	if row.UserGame.Notes == nil || *row.UserGame.Notes != notes {
		t.Errorf("UserGame.Notes = %v, want %q", derefOrNil(row.UserGame.Notes), notes)
	}

	// Updating without notes clears them
	err = store.UpdateGameDetails(ctx, 1, 100, GameDetails{Status: GameStatusPlaying, Rating: &rating})
	if err != nil {
		t.Fatalf("UpdateGameDetails() failed: %v", err)
	}

	row, err = store.GetLibraryGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if row.UserGame.Notes != nil {
		t.Errorf("UserGame.Notes = %v, want %v", derefOrNil(row.UserGame.Notes), nil)
	}
}

func TestUpdateGameDetailsWishlistedGame(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Add a game to the user's wishlist only
	err := store.AddToWishlist(ctx, newGameParams(100, "Test Game"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToWishlist() failed: %v", err)
	}

	// Attempt to update a wishlisted game, which should fail since it isn't owned
	err = store.UpdateGameDetails(ctx, 1, 100, GameDetails{Status: GameStatusPlaying})
	if !errors.Is(err, ErrUserGameNotFound) {
		t.Fatalf("UpdateGameDetails() error = %v, want %v", err, ErrUserGameNotFound)
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

// setupRemoveFromLibrary extends setupTrackedDLCs with playthroughs on 100 and 201 and tracks game 300's DLC 401.
func setupRemoveFromLibrary(t *testing.T, store *Store) {
	t.Helper()
	ctx := t.Context()
	setupTrackedDLCs(t, store)
	for _, igdbID := range []int64{100, 201} {
		if _, err := store.StartPlaythrough(ctx, 1, igdbID, nil, nil, time.Now().UTC()); err != nil {
			t.Fatalf("StartPlaythrough(%d) failed: %v", igdbID, err)
		}
	}
	if err := store.TrackDLC(ctx, 1, 300, 401, time.Now().UTC()); err != nil {
		t.Fatalf("TrackDLC(401) failed: %v", err)
	}
}

func TestRemoveFromLibrary(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupRemoveFromLibrary(t, store)

	if err := store.RemoveFromLibrary(ctx, 1, 100); err != nil {
		t.Fatalf("RemoveFromLibrary() failed: %v", err)
	}

	for _, igdbID := range []int64{100, 201, 202} {
		if _, err := store.GetLibraryGame(ctx, 1, igdbID); !errors.Is(err, ErrUserGameNotFound) {
			t.Errorf("GetLibraryGame(%d) error = %v, want %v", igdbID, err, ErrUserGameNotFound)
		}
	}

	other := getLibraryGame(t, store, 300)
	assertDLCStatus(t, findDLC(t, other, 401), GameStatusBacklog)
	getLibraryGame(t, store, 401)

	if _, err := store.queries.GetGameByIGDBID(ctx, 201); err != nil {
		t.Errorf("GetGameByIGDBID(201) failed: %v", err)
	}
}

func TestRemoveFromLibraryReAdd(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupRemoveFromLibrary(t, store)

	if err := store.RemoveFromLibrary(ctx, 1, 100); err != nil {
		t.Fatalf("RemoveFromLibrary() failed: %v", err)
	}
	if err := store.AddToLibrary(ctx, newGameParams(100, "Test Game"), new(newPlatformParams(6, "PC")), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}

	game := getLibraryGame(t, store, 100)
	if game.UserGame.Status == nil || *game.UserGame.Status != string(GameStatusBacklog) {
		t.Errorf("UserGame.Status = %v, want %q", derefOrNil(game.UserGame.Status), GameStatusBacklog)
	}
	if len(game.Playthroughs) != 0 {
		t.Errorf("len(Playthroughs) = %d, want 0", len(game.Playthroughs))
	}
	if findDLC(t, game, 201).Tracked {
		t.Errorf("DLC 201 Tracked = true, want false")
	}
	assertCopyDLCs(t, game.CopyDLCs[6], map[int64]bool{201: false, 202: false})
}

func TestRemoveFromLibraryUntrackDLC(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupRemoveFromLibrary(t, store)

	if err := store.RemoveFromLibrary(ctx, 1, 201); err != nil {
		t.Fatalf("RemoveFromLibrary() failed: %v", err)
	}

	if _, err := store.GetLibraryGame(ctx, 1, 201); !errors.Is(err, ErrUserGameNotFound) {
		t.Errorf("GetLibraryGame(201) error = %v, want %v", err, ErrUserGameNotFound)
	}
	assertCopyDLCs(t, getLibraryGame(t, store, 100).CopyDLCs[6], map[int64]bool{201: true, 202: false})
}

func TestRemoveFromLibraryWishlisted(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	if err := store.AddToWishlist(ctx, newGameParams(500, "Wishlisted Game"), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToWishlist() failed: %v", err)
	}

	if err := store.RemoveFromLibrary(ctx, 1, 500); err != nil {
		t.Fatalf("RemoveFromLibrary() failed: %v", err)
	}

	wishlisted, err := store.ListWishlist(ctx, 1)
	if err != nil {
		t.Fatalf("ListWishlist() failed: %v", err)
	}
	if len(wishlisted) != 0 {
		t.Errorf("len(ListWishlist()) = %d, want 0", len(wishlisted))
	}
}

func TestRemoveFromLibraryKeepsStandaloneExpansion(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupTrackedDLCs(t, store)

	standalone := newDLC(501, 100, "Standalone Expansion", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)).Game
	standalone.GameType = int64(igdb.GameTypeStandaloneExpansion)
	if err := store.AddToLibrary(ctx, standalone, new(newPlatformParams(130, "Switch")), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}

	if err := store.RemoveFromLibrary(ctx, 1, 100); err != nil {
		t.Fatalf("RemoveFromLibrary() failed: %v", err)
	}

	assertPlatformIDs(t, getLibraryGame(t, store, 501).Platforms, 130)
}

func TestRemoveFromLibraryNotInLibrary(t *testing.T) {
	store := newTestStore(t)

	err := store.RemoveFromLibrary(t.Context(), 1, 999)
	if !errors.Is(err, ErrUserGameNotFound) {
		t.Fatalf("RemoveFromLibrary() error = %v, want %v", err, ErrUserGameNotFound)
	}
}

func TestUpdateGameDetailsNotInLibrary(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Attempt to update a game that doesn't exist in the library
	err := store.UpdateGameDetails(ctx, 1, 100, GameDetails{Status: GameStatusPlaying})
	if !errors.Is(err, ErrUserGameNotFound) {
		t.Fatalf("UpdateGameDetails() error = %v, want %v", err, ErrUserGameNotFound)
	}
}
