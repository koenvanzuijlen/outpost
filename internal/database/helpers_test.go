package database

import (
	"log/slog"
	"path/filepath"
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

// newTestStore creates a new test database store with a temporary database file. It fails the test if the store cannot
// be opened.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(t.Context(), slog.New(slog.DiscardHandler), path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

// addOwnedGame adds a game to user 1's collection on a default platform. It fails the test if the game cannot be added.
func addOwnedGame(t *testing.T, store *Store, igdbID int64) {
	t.Helper()
	err := store.AddToLibrary(t.Context(), newGameParams(igdbID, "Test Game"), new(newPlatformParams(11, "Xbox")), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}
}

// getLibraryGame retrieves a game from user 1's library. It fails the test if the game cannot be retrieved.
func getLibraryGame(t *testing.T, store *Store, igdbID int64) LibraryGame {
	t.Helper()
	game, err := store.GetLibraryGame(t.Context(), 1, igdbID)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	return game
}

// derefOrNil returns the value p points to, or nil, so failure messages show a value instead of an address.
func derefOrNil[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}
