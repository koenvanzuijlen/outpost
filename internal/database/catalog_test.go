package database

import (
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

func TestSaveGame(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	platforms := []dbgen.UpsertPlatformParams{newPlatformParams(6, "PC"), newPlatformParams(48, "PS4")}
	if err := store.SaveGame(ctx, newGameParams(100, "Test Game"), platforms); err != nil {
		t.Fatalf("SaveGame() failed: %v", err)
	}

	details := getGame(t, store, 100)
	if details.Game.Title != "Test Game" {
		t.Errorf("Title = %q, want %q", details.Game.Title, "Test Game")
	}
	if details.UserGame != nil {
		t.Errorf("UserGame = %v, want nil", *details.UserGame)
	}
	assertPlatformIDs(t, details.Platforms, 6, 48)
}

func TestSaveGameReplacesPlatforms(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	first := []dbgen.UpsertPlatformParams{newPlatformParams(6, "PC"), newPlatformParams(48, "PS4")}
	if err := store.SaveGame(ctx, newGameParams(100, "Test Game"), first); err != nil {
		t.Fatalf("SaveGame() first failed: %v", err)
	}
	second := []dbgen.UpsertPlatformParams{newPlatformParams(130, "Switch")}
	if err := store.SaveGame(ctx, newGameParams(100, "Test Game"), second); err != nil {
		t.Fatalf("SaveGame() second failed: %v", err)
	}

	details := getGame(t, store, 100)
	assertPlatformIDs(t, details.Platforms, 130)
}

func TestSaveGameKeepsDLCsFetchedAt(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	if err := store.SaveGame(ctx, newGameParams(100, "Test Game"), nil); err != nil {
		t.Fatalf("SaveGame() first failed: %v", err)
	}
	fetchedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := store.SaveDLCs(ctx, 100, nil, fetchedAt); err != nil {
		t.Fatalf("SaveDLCs() failed: %v", err)
	}
	if err := store.SaveGame(ctx, newGameParams(100, "Test Game"), nil); err != nil {
		t.Fatalf("SaveGame() second failed: %v", err)
	}

	details := getGame(t, store, 100)
	if details.Game.DlcsFetchedAt == nil {
		t.Fatal("DlcsFetchedAt = nil, want still set")
	}
	if !details.Game.DlcsFetchedAt.Equal(fetchedAt) {
		t.Errorf("DlcsFetchedAt = %v, want %v", *details.Game.DlcsFetchedAt, fetchedAt)
	}
}
