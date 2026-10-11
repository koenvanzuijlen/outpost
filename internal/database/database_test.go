package database

import (
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

func TestPragmasApplied(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	var fk int
	err := store.db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk)
	if err != nil {
		t.Fatalf("PRAGMA foreign_keys query failed: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}
}

func TestTypeRoundTrip(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	fetchedAt := time.Now()
	_, err := store.queries.UpsertGame(ctx, dbgen.UpsertGameParams{
		IgdbID:       1,
		FetchedAt:    fetchedAt,
		ParentIgdbID: nil,
		Title:        "Test Game",
		GameType:     1,
		CoverUrl:     nil,
		ReleasedAt:   nil,
		Summary:      nil,
	})
	if err != nil {
		t.Fatalf("UpsertGame() failed: %v", err)
	}
	game, err := store.queries.GetGameByIGDBID(ctx, 1)
	if err != nil {
		t.Fatalf("GetGameByIGDBID() failed: %v", err)
	}
	t.Run("Datetime_field", func(t *testing.T) {
		if !game.FetchedAt.Equal(fetchedAt) {
			t.Errorf("FetchedAt = %v, want %v", game.FetchedAt, fetchedAt)
		}
	})
	t.Run("Nullable_field", func(t *testing.T) {
		if game.CoverUrl != nil {
			t.Errorf("CoverUrl = %v, want nil", game.CoverUrl)
		}
	})
}
