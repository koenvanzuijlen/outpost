package database

import (
	"errors"
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// setupGameOnPCAndPS4 puts game 100 in user 1's library owned on PC, released on PC (6) and PS4 (48) only.
func setupGameOnPCAndPS4(t *testing.T, store *Store) {
	t.Helper()
	pc := newPlatformParams(6, "PC")
	if err := store.AddToLibrary(t.Context(), newGameParams(100, "Test Game"), &pc, 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}
	platforms := []dbgen.UpsertPlatformParams{pc, newPlatformParams(48, "PS4")}
	if err := store.SaveGame(t.Context(), newGameParams(100, "Test Game"), platforms); err != nil {
		t.Fatalf("SaveGame() failed: %v", err)
	}
}

func TestAddOwnedCopy(t *testing.T) {
	store := newTestStore(t)
	setupGameOnPCAndPS4(t, store)

	if err := store.AddOwnedCopy(t.Context(), 1, 100, 48); err != nil {
		t.Fatalf("AddOwnedCopy() failed: %v", err)
	}

	details := getGame(t, store, 100)
	assertPlatformIDs(t, details.OwnedPlatforms, 6, 48)
}

func TestAddOwnedCopyAlreadyOwned(t *testing.T) {
	store := newTestStore(t)
	setupGameOnPCAndPS4(t, store)

	err := store.AddOwnedCopy(t.Context(), 1, 100, 6)
	if !errors.Is(err, ErrAlreadyOwnedOnPlatform) {
		t.Errorf("AddOwnedCopy() error = %v, want %v", err, ErrAlreadyOwnedOnPlatform)
	}
}

func TestAddOwnedCopyWishlisted(t *testing.T) {
	store := newTestStore(t)
	if err := store.AddToWishlist(t.Context(), newGameParams(500, "Wishlisted Game"), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToWishlist() failed: %v", err)
	}

	err := store.AddOwnedCopy(t.Context(), 1, 500, 6)
	if !errors.Is(err, ErrUserGameNotFound) {
		t.Errorf("AddOwnedCopy() error = %v, want %v", err, ErrUserGameNotFound)
	}
}

func TestAddOwnedCopyPlatformNotForGame(t *testing.T) {
	store := newTestStore(t)
	setupGameOnPCAndPS4(t, store)
	if err := store.SaveGame(t.Context(), newGameParams(300, "Other"), []dbgen.UpsertPlatformParams{newPlatformParams(130, "Switch")}); err != nil {
		t.Fatalf("SaveGame() failed: %v", err)
	}

	err := store.AddOwnedCopy(t.Context(), 1, 100, 130)
	if !errors.Is(err, ErrPlatformNotForGame) {
		t.Errorf("AddOwnedCopy() error = %v, want %v", err, ErrPlatformNotForGame)
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
	playthrough, err := store.StartPlaythrough(ctx, 1, 100, &pc, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("StartPlaythrough() failed: %v", err)
	}

	if err := store.RemoveOwnedCopy(ctx, 1, 100, 6); err != nil {
		t.Fatalf("RemoveOwnedCopy(PC) failed: %v", err)
	}

	details := getGame(t, store, 100)
	assertPlatformIDs(t, details.OwnedPlatforms, 48)
	if _, ok := details.CopyDLCs[6]; ok {
		t.Errorf("CopyDLCs has key 6, want none")
	}
	got := onlyPlaythrough(t, details)
	if got.Playthrough.ID != playthrough.ID {
		t.Errorf("Playthrough ID = %d, want %d", got.Playthrough.ID, playthrough.ID)
	}
	if name := got.PlatformName; name == nil || *name != "PC" {
		t.Errorf("Playthrough PlatformName = %v, want %q", derefOrNil(name), "PC")
	}

	if err := store.RemoveOwnedCopy(ctx, 1, 100, 48); err != nil {
		t.Fatalf("RemoveOwnedCopy(PS4) failed: %v", err)
	}

	details = getGame(t, store, 100)
	if details.UserGame == nil {
		t.Fatal("UserGame = nil, want an entry")
	}
	assertPlatformIDs(t, details.OwnedPlatforms)
	if s := details.UserGame.Status; s == nil || *s != string(GameStatusPlaying) {
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
