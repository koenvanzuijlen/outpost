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

func TestPlaythroughChecks(t *testing.T) {
	day1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		playName   *string
		outcome    string
		startedAt  *time.Time
		finishedAt *time.Time
		wantErr    bool
	}{
		{"valid", new("PT"), "completed", &day1, &day2, false},
		{"nil name and dates", nil, "completed", nil, nil, false},
		{"empty name", new(""), "playing", nil, nil, true},
		{"playing with finished", nil, "playing", &day1, &day2, true},
		{"finished before started", nil, "completed", &day2, &day1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			store := newTestStore(t)
			addOwnedGame(t, store, 100)
			userGameID := getUserGame(t, store, 100).UserGame.ID

			_, err := store.db.ExecContext(ctx,
				"INSERT INTO playthroughs(user_game_id, name, outcome, started_at, finished_at) VALUES (?, ?, ?, ?, ?)",
				userGameID, tt.playName, tt.outcome, tt.startedAt, tt.finishedAt)
			if (err != nil) != tt.wantErr {
				t.Errorf("insert err = %v, wantErr %v", err, tt.wantErr)
			}
		})
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
