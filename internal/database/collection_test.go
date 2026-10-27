package database

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func TestListGamesOrder(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupTrackedDLCs(t, store)
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
	if _, err := store.AddToCollection(ctx, newGameParams(500, "Removed Game"), newPlatformParams(11, "Xbox"), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToCollection(500) failed: %v", err)
	}
	if err := store.RemoveOwnedCopy(ctx, 1, 500, 11); err != nil {
		t.Fatalf("RemoveOwnedCopy(500) failed: %v", err)
	}
	if err := store.AddToWishlist(ctx, newGameParams(600, "Wishlisted Game"), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToWishlist(600) failed: %v", err)
	}

	games, err := store.ListGames(ctx, 1)
	if err != nil {
		t.Fatalf("ListGames() failed: %v", err)
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
		t.Errorf("ListGames() = %v, want %v", got, want)
	}
}

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

	assertPlatformIDs(t, getLibraryGame(t, store, 100).Platforms, 6)
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

	assertPlatformIDs(t, getLibraryGame(t, store, 100).Platforms, 6)
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

	libraryGame := getLibraryGame(t, store, 100)
	assertPlatformIDs(t, libraryGame.Platforms, 6, 11)
	if s := libraryGame.UserGame.Status; s == nil || *s != "completed" {
		t.Errorf("Status = %v, want %q", derefOrNil(s), "completed")
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

func TestRemoveOwnedCopy(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupDLCOwnership(t, store)
	if err := store.SetDLCOwned(ctx, 1, 100, 6, 201, true); err != nil {
		t.Fatalf("SetDLCOwned() failed: %v", err)
	}
	pc := int64(6)
	pt, err := store.StartPlaythrough(ctx, 1, 100, &pc, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("StartPlaythrough() failed: %v", err)
	}

	if err := store.RemoveOwnedCopy(ctx, 1, 100, 6); err != nil {
		t.Fatalf("RemoveOwnedCopy(PC) failed: %v", err)
	}

	game := getLibraryGame(t, store, 100)
	assertPlatformIDs(t, game.Platforms, 48)
	if _, ok := game.CopyDLCs[6]; ok {
		t.Errorf("CopyDLCs has key 6, want none")
	}
	if len(game.Playthroughs) != 1 {
		t.Fatalf("len(Playthroughs) = %d, want 1", len(game.Playthroughs))
	}
	if game.Playthroughs[0].Playthrough.ID != pt.ID {
		t.Errorf("Playthrough ID = %d, want %d", game.Playthroughs[0].Playthrough.ID, pt.ID)
	}
	if name := game.Playthroughs[0].PlatformName; name == nil || *name != "PC" {
		t.Errorf("Playthrough PlatformName = %v, want %q", derefOrNil(name), "PC")
	}

	if err := store.RemoveOwnedCopy(ctx, 1, 100, 48); err != nil {
		t.Fatalf("RemoveOwnedCopy(PS4) failed: %v", err)
	}

	game, err = store.GetLibraryGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetLibraryGame() after removing last copy failed: %v", err)
	}
	assertPlatformIDs(t, game.Platforms)
	if s := game.UserGame.Status; s == nil || *s != string(GameStatusPlaying) {
		t.Errorf("Status = %v, want %q", derefOrNil(s), GameStatusPlaying)
	}
}

func TestRemoveOwnedCopyNotFound(t *testing.T) {
	tests := []struct {
		name           string
		platformIGDBID int64
	}{
		{"no copy on platform", 130},
		{"unknown platform", 999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t)
			setupDLCOwnership(t, store)

			err := store.RemoveOwnedCopy(t.Context(), 1, 100, tt.platformIGDBID)
			if !errors.Is(err, ErrOwnedCopyNotFound) {
				t.Errorf("RemoveOwnedCopy() error = %v, want %v", err, ErrOwnedCopyNotFound)
			}
		})
	}
}

func TestRemoveOwnedCopyUserGameNotFound(t *testing.T) {
	tests := []struct {
		name   string
		igdbID int64
	}{
		{"wishlisted", 500},
		{"not in library", 999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t)
			setupDLCOwnership(t, store)
			if err := store.AddToWishlist(t.Context(), newGameParams(500, "Wishlisted Game"), 1, time.Now().UTC()); err != nil {
				t.Fatalf("AddToWishlist() failed: %v", err)
			}

			err := store.RemoveOwnedCopy(t.Context(), 1, tt.igdbID, 6)
			if !errors.Is(err, ErrUserGameNotFound) {
				t.Errorf("RemoveOwnedCopy() error = %v, want %v", err, ErrUserGameNotFound)
			}
		})
	}
}
