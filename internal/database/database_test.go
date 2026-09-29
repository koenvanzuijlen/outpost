package database

import (
	"database/sql"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

func newTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(t.Context(), slog.New(slog.DiscardHandler), path)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func TestPragmasApplied(t *testing.T) {
	ctx := t.Context()
	db := newTestDatabase(t)

	var fk int
	err := db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk)
	if err != nil {
		t.Fatalf("failed to check foreign keys: %v", err)
	}
	if fk != 1 {
		t.Fatalf("foreign keys not enabled")
	}
}

func TestTypeRoundTrip(t *testing.T) {
	ctx := t.Context()
	db := newTestDatabase(t)
	queries := dbgen.New(db)

	fetchedAt := time.Now()
	_, err := queries.CreateGame(ctx, dbgen.CreateGameParams{
		IgdbID:       1,
		FetchedAt:    fetchedAt,
		ParentIgdbID: nil,
		Title:        "Test Game",
		GameType:     "Test Type",
		CoverUrl:     nil,
		ReleaseYear:  nil,
		Summary:      nil,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}
	game, err := queries.GetGameByIGDBID(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get game by IGDB ID: %v", err)
	}
	t.Run("Datetime_field", func(t *testing.T) {
		if !game.FetchedAt.Equal(fetchedAt) {
			t.Errorf("expected fetched_at to be %v, got %v", fetchedAt, game.FetchedAt)
		}
	})
	t.Run("Nullable_field", func(t *testing.T) {
		if game.CoverUrl != nil {
			t.Errorf("expected cover_url to be nil, got %v", game.CoverUrl)
		}
	})
}
