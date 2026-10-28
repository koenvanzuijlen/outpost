package database

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

func TestGetUserGameOwned(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Add a game to the user's collection before attempting to retrieve it
	err := store.AddToLibrary(ctx, newGameParams(100, "Test Game"), new(newPlatformParams(11, "Xbox")), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}

	// Attempt to retrieve the row from the library
	row, err := store.GetUserGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetUserGame() failed: %v", err)
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
	if len(row.OwnedPlatforms) != 1 {
		t.Fatalf("len(Platforms) = %d, want 1", len(row.OwnedPlatforms))
	}
	if row.OwnedPlatforms[0].Name != "Xbox" {
		t.Errorf("Platform.Name = %v, want %v", row.OwnedPlatforms[0].Name, "Xbox")
	}
	if len(row.Playthroughs) != 0 {
		t.Errorf("len(Playthroughs) = %d, want 0", len(row.Playthroughs))
	}
}

func TestGetUserGameWithPlaythroughs(t *testing.T) {
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
	row, err := store.GetUserGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetUserGame() failed: %v", err)
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

func TestGetUserGameWishlist(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Add a game to the user's wishlist before attempting to retrieve it
	err := store.AddToWishlist(ctx, newGameParams(100, "Test Game"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToWishlist() failed: %v", err)
	}

	// Attempt to retrieve the row from the library
	row, err := store.GetUserGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetUserGame() failed: %v", err)
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
	if len(row.OwnedPlatforms) != 0 {
		t.Errorf("len(Platforms) = %d, want 0", len(row.OwnedPlatforms))
	}
}

func TestGetUserGameNotFound(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	// Attempt to retrieve a game that doesn't exist in the library
	_, err := store.GetUserGame(ctx, 1, 999)
	if !errors.Is(err, ErrUserGameNotFound) {
		t.Fatalf("GetUserGame() error = %v, want %v", err, ErrUserGameNotFound)
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
	row, err := store.GetUserGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetUserGame() failed: %v", err)
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
	row, err := store.GetUserGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetUserGame() failed: %v", err)
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
	row, err := store.GetUserGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetUserGame() failed: %v", err)
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
	row, err = store.GetUserGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetUserGame() failed: %v", err)
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

	row, err := store.GetUserGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetUserGame() failed: %v", err)
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

	row, err = store.GetUserGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetUserGame() failed: %v", err)
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
	row, err := store.GetUserGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetUserGame() failed: %v", err)
	}
	if row.UserGame.Status != nil {
		t.Errorf("UserGame.Status = %v, want %v", derefOrNil(row.UserGame.Status), nil)
	}
}

// setupRemoveUserGame extends setupLibraryDLCs with playthroughs on 100 and 201 and adds game 300's DLC 401 to the
// library.
func setupRemoveUserGame(t *testing.T, store *Store) {
	t.Helper()
	ctx := t.Context()
	setupLibraryDLCs(t, store)
	for _, igdbID := range []int64{100, 201} {
		if _, err := store.StartPlaythrough(ctx, 1, igdbID, nil, nil, time.Now().UTC()); err != nil {
			t.Fatalf("StartPlaythrough(%d) failed: %v", igdbID, err)
		}
	}
	if err := store.AddDLCToLibrary(ctx, 1, 300, 401, time.Now().UTC()); err != nil {
		t.Fatalf("AddDLCToLibrary(401) failed: %v", err)
	}
}

func TestRemoveUserGame(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupRemoveUserGame(t, store)

	if err := store.RemoveUserGame(ctx, 1, 100); err != nil {
		t.Fatalf("RemoveUserGame() failed: %v", err)
	}

	for _, igdbID := range []int64{100, 201, 202} {
		if _, err := store.GetUserGame(ctx, 1, igdbID); !errors.Is(err, ErrUserGameNotFound) {
			t.Errorf("GetUserGame(%d) error = %v, want %v", igdbID, err, ErrUserGameNotFound)
		}
	}

	details := getUserGame(t, store, 300)
	assertDLCStatus(t, findDLC(t, details, 401), GameStatusBacklog)
	getUserGame(t, store, 401)

	if _, err := store.queries.GetGameByIGDBID(ctx, 201); err != nil {
		t.Errorf("GetGameByIGDBID(201) failed: %v", err)
	}
}

func TestRemoveUserGameReAdd(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupRemoveUserGame(t, store)

	if err := store.RemoveUserGame(ctx, 1, 100); err != nil {
		t.Fatalf("RemoveUserGame() failed: %v", err)
	}
	if err := store.AddToLibrary(ctx, newGameParams(100, "Test Game"), new(newPlatformParams(6, "PC")), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}

	details := getUserGame(t, store, 100)
	if details.UserGame.Status == nil || *details.UserGame.Status != string(GameStatusBacklog) {
		t.Errorf("UserGame.Status = %v, want %q", derefOrNil(details.UserGame.Status), GameStatusBacklog)
	}
	if len(details.Playthroughs) != 0 {
		t.Errorf("len(Playthroughs) = %d, want 0", len(details.Playthroughs))
	}
	if findDLC(t, details, 201).InLibrary {
		t.Errorf("DLC 201 InLibrary = true, want false")
	}
	assertCopyDLCs(t, details.CopyDLCs[6], map[int64]bool{201: false, 202: false})
}

func TestRemoveUserGameLibraryDLC(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupRemoveUserGame(t, store)

	if err := store.RemoveUserGame(ctx, 1, 201); err != nil {
		t.Fatalf("RemoveUserGame() failed: %v", err)
	}

	if _, err := store.GetUserGame(ctx, 1, 201); !errors.Is(err, ErrUserGameNotFound) {
		t.Errorf("GetUserGame(201) error = %v, want %v", err, ErrUserGameNotFound)
	}
	assertCopyDLCs(t, getUserGame(t, store, 100).CopyDLCs[6], map[int64]bool{201: true, 202: false})
}

func TestRemoveUserGameWishlisted(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	if err := store.AddToWishlist(ctx, newGameParams(500, "Wishlisted Game"), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToWishlist() failed: %v", err)
	}

	if err := store.RemoveUserGame(ctx, 1, 500); err != nil {
		t.Fatalf("RemoveUserGame() failed: %v", err)
	}

	wishlisted, err := store.ListWishlist(ctx, 1)
	if err != nil {
		t.Fatalf("ListWishlist() failed: %v", err)
	}
	if len(wishlisted) != 0 {
		t.Errorf("len(ListWishlist()) = %d, want 0", len(wishlisted))
	}
}

func TestRemoveUserGameKeepsStandaloneExpansion(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupLibraryDLCs(t, store)

	standalone := newDLC(501, 100, "Standalone Expansion", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)).Game
	standalone.GameType = int64(igdb.GameTypeStandaloneExpansion)
	if err := store.AddToLibrary(ctx, standalone, new(newPlatformParams(130, "Switch")), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}

	if err := store.RemoveUserGame(ctx, 1, 100); err != nil {
		t.Fatalf("RemoveUserGame() failed: %v", err)
	}

	assertPlatformIDs(t, getUserGame(t, store, 501).OwnedPlatforms, 130)
}

func TestRemoveUserGameNotFound(t *testing.T) {
	store := newTestStore(t)

	err := store.RemoveUserGame(t.Context(), 1, 999)
	if !errors.Is(err, ErrUserGameNotFound) {
		t.Fatalf("RemoveUserGame() error = %v, want %v", err, ErrUserGameNotFound)
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

func TestListLibraryOrder(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupLibraryDLCs(t, store)
	if err := store.SetDLCOwned(ctx, 1, 100, 48, 201, true); err != nil {
		t.Fatalf("SetDLCOwned(201, PS4) failed: %v", err)
	}
	if err := store.SetDLCOwned(ctx, 1, 300, 6, 401, true); err != nil {
		t.Fatalf("SetDLCOwned(401, PC) failed: %v", err)
	}
	// Both main games are titled "Test Game"; rename 300 so their PC copies don't tie on (title, platform).
	if _, err := store.queries.UpsertGame(ctx, newGameParams(300, "Other Game")); err != nil {
		t.Fatalf("UpsertGame(300) failed: %v", err)
	}
	if err := store.AddToLibrary(ctx, newGameParams(500, "Removed Game"), new(newPlatformParams(11, "Xbox")), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary(500) failed: %v", err)
	}
	if err := store.RemoveOwnedCopy(ctx, 1, 500, 11); err != nil {
		t.Fatalf("RemoveOwnedCopy(500) failed: %v", err)
	}
	if err := store.AddToWishlist(ctx, newGameParams(600, "Wishlisted Game"), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToWishlist(600) failed: %v", err)
	}

	games, err := store.ListLibrary(ctx, 1)
	if err != nil {
		t.Fatalf("ListLibrary() failed: %v", err)
	}

	type entry struct {
		igdbID   int64
		platform string
	}
	got := make([]entry, 0, len(games))
	for _, g := range games {
		e := entry{igdbID: g.Game.IgdbID, platform: "<nil>"}
		if g.PlatformName != nil {
			e.platform = *g.PlatformName
		}
		got = append(got, e)
	}
	want := []entry{
		{201, "PC"},
		{201, "PS4"},
		{202, "<nil>"},
		{300, "PC"},
		{500, "<nil>"},
		{100, "PC"},
		{100, "PS4"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("ListLibrary() = %v, want %v", got, want)
	}
}

func TestAddToLibraryClearsWishlist(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")

	if err := store.AddToWishlist(ctx, game, 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToWishlist() failed: %v", err)
	}
	if err := store.AddToLibrary(ctx, game, new(newPlatformParams(6, "PC")), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}

	wishlisted, err := store.queries.ListWishlistedUserGames(ctx, 1)
	if err != nil {
		t.Fatalf("ListWishlistedUserGames() failed: %v", err)
	}
	if len(wishlisted) != 0 {
		t.Errorf("len(wishlisted) = %d, want 0", len(wishlisted))
	}

	assertPlatformIDs(t, getUserGame(t, store, 100).OwnedPlatforms, 6)
}

func TestAddToLibraryNotOwned(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")

	if err := store.AddToLibrary(ctx, game, nil, 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}
	// Adding it again without a platform is a no-op.
	if err := store.AddToLibrary(ctx, game, nil, 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() second failed: %v", err)
	}

	details := getUserGame(t, store, 100)
	assertPlatformIDs(t, details.OwnedPlatforms)
	if s := details.UserGame.Status; s == nil || *s != string(GameStatusBacklog) {
		t.Errorf("Status = %v, want %q", derefOrNil(s), GameStatusBacklog)
	}

	games, err := store.ListLibrary(ctx, 1)
	if err != nil {
		t.Fatalf("ListLibrary() failed: %v", err)
	}
	if len(games) != 1 || games[0].Game.IgdbID != 100 || games[0].PlatformName != nil {
		t.Errorf("ListLibrary() = %v, want one entry for 100 without platform", games)
	}
}

func TestAddToLibraryNotOwnedThenOwned(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")

	if err := store.AddToLibrary(ctx, game, nil, 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary(nil) failed: %v", err)
	}
	if err := store.UpdateGameDetails(ctx, 1, 100, GameDetails{Status: GameStatusCompleted}); err != nil {
		t.Fatalf("UpdateGameDetails() failed: %v", err)
	}
	if err := store.AddToLibrary(ctx, game, new(newPlatformParams(6, "PC")), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary(PC) failed: %v", err)
	}

	details := getUserGame(t, store, 100)
	assertPlatformIDs(t, details.OwnedPlatforms, 6)
	if s := details.UserGame.Status; s == nil || *s != string(GameStatusCompleted) {
		t.Errorf("Status = %v, want %q", derefOrNil(s), GameStatusCompleted)
	}
}

func TestAddToLibraryNotOwnedClearsWishlist(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")

	if err := store.AddToWishlist(ctx, game, 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToWishlist() failed: %v", err)
	}
	if err := store.AddToLibrary(ctx, game, nil, 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}

	details := getUserGame(t, store, 100)
	if details.UserGame.Wishlisted {
		t.Errorf("Wishlisted = true, want false")
	}
	if s := details.UserGame.Status; s == nil || *s != string(GameStatusBacklog) {
		t.Errorf("Status = %v, want %q", derefOrNil(s), GameStatusBacklog)
	}
}

func TestAddToLibrarySameGameDifferentPlatforms(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")
	if err := store.AddToLibrary(ctx, game, new(newPlatformParams(11, "Xbox")), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() first failed: %v", err)
	}
	if err := store.AddToLibrary(ctx, game, new(newPlatformParams(6, "PC")), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() second failed: %v", err)
	}

	// One library entry with two owned copies.
	assertPlatformIDs(t, getUserGame(t, store, 100).OwnedPlatforms, 6, 11)
}

func TestAddToLibraryDifferentGamesSamePlatform(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	platform := newPlatformParams(6, "PC")

	if err := store.AddToLibrary(ctx, newGameParams(100, "Test Game"), &platform, 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() first failed: %v", err)
	}
	if err := store.AddToLibrary(ctx, newGameParams(101, "Test Game 2"), &platform, 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() second failed: %v", err)
	}

	// Two library entries, each with its own copy on the same platform.
	assertPlatformIDs(t, getUserGame(t, store, 100).OwnedPlatforms, 6)
	assertPlatformIDs(t, getUserGame(t, store, 101).OwnedPlatforms, 6)
}

func TestAddToLibrarySameGameSamePlatform(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")
	platform := newPlatformParams(6, "PC")

	if err := store.AddToLibrary(ctx, game, &platform, 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() first failed: %v", err)
	}
	if err := store.AddToLibrary(ctx, game, &platform, 1, time.Now().UTC()); !errors.Is(err, ErrAlreadyOwnedOnPlatform) {
		t.Fatalf("AddToLibrary() error = %v, want %v", err, ErrAlreadyOwnedOnPlatform)
	}

	assertPlatformIDs(t, getUserGame(t, store, 100).OwnedPlatforms, 6)
}

func TestAddToLibraryPreservesStatus(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")

	if err := store.AddToLibrary(ctx, game, new(newPlatformParams(6, "PC")), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() first failed: %v", err)
	}

	if _, err := store.db.ExecContext(ctx, "UPDATE user_games SET status = ?", "completed"); err != nil {
		t.Fatalf("ExecContext() failed: %v", err)
	}

	if err := store.AddToLibrary(ctx, game, new(newPlatformParams(11, "Xbox")), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() second failed: %v", err)
	}

	details := getUserGame(t, store, 100)
	assertPlatformIDs(t, details.OwnedPlatforms, 6, 11)
	if s := details.UserGame.Status; s == nil || *s != "completed" {
		t.Errorf("Status = %v, want %q", derefOrNil(s), "completed")
	}
}

func TestAddToLibraryUpdatesGameInfo(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	if err := store.AddToLibrary(ctx, newGameParams(100, "Old Title"), new(newPlatformParams(6, "PC")), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() first failed: %v", err)
	}

	cover := "https://example.com/cover.jpg"
	updated := newGameParams(100, "New Title")
	updated.CoverUrl = &cover
	if err := store.AddToLibrary(ctx, updated, new(newPlatformParams(11, "Xbox")), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() second failed: %v", err)
	}
	assertPlatformIDs(t, getUserGame(t, store, 100).OwnedPlatforms, 6, 11)

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
