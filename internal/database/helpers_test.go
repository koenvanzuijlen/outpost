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
