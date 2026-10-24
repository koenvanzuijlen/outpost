package database

import (
	"errors"
	"testing"
	"time"
)

func TestStartPlaythrough(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	platformIGDBID := int64(11)
	platformName := "Xbox"
	playthroughName := "Test PT"

	_, err := store.AddToCollection(ctx, newGameParams(100, "Test Game"), newPlatformParams(platformIGDBID, platformName), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() failed: %v", err)
	}

	pt, err := store.StartPlaythrough(ctx, 1, 100, &platformIGDBID, &playthroughName, time.Now().UTC())
	if err != nil {
		t.Fatalf("StartPlaythrough() failed: %v", err)
	}
	if pt.Name == nil || *pt.Name != playthroughName {
		t.Errorf("Name = %s, want %s", derefOrNil(pt.Name), playthroughName)
	}
	if pt.Outcome != "playing" {
		t.Errorf("Outcome = %s, want %s", pt.Outcome, "playing")
	}

	pts, err := store.queries.ListPlaythroughsForUserGameID(ctx, pt.UserGameID)
	if err != nil {
		t.Fatalf("ListPlaythroughsForUserGameID() failed: %v", err)
	}
	if len(pts) != 1 {
		t.Fatalf("len(pts) = %d, want 1", len(pts))
	}
	if pts[0].PlatformName == nil || *pts[0].PlatformName != platformName {
		t.Errorf("PlatformName = %s, want %s", derefOrNil(pts[0].PlatformName), platformName)
	}

	game, err := store.GetLibraryGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if game.UserGame.Status == nil || *game.UserGame.Status != "playing" {
		t.Errorf("UserGame.Status = %s, want %s", derefOrNil(game.UserGame.Status), "playing")
	}
}

func TestStartPlaythroughWithoutPlatformAndName(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	_, err := store.AddToCollection(ctx, newGameParams(100, "Test Game"), newPlatformParams(11, "Xbox"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() failed: %v", err)
	}

	pt, err := store.StartPlaythrough(ctx, 1, 100, nil, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("StartPlaythrough() failed: %v", err)
	}

	if pt.PlatformID != nil {
		t.Errorf("PlatformID = %d, want nil", derefOrNil(pt.PlatformID))
	}
	if pt.Name != nil {
		t.Errorf("Name = %s, want nil", derefOrNil(pt.Name))
	}
}

func TestStartPlaythroughNotInLibrary(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	_, err := store.StartPlaythrough(ctx, 1, 100, nil, nil, time.Now().UTC())
	if !errors.Is(err, ErrUserGameNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrUserGameNotFound)
	}
}

func TestStartPlaythroughWishlistedGame(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	err := store.AddToWishlist(ctx, newGameParams(100, "Test Game"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToWishlist() failed: %v", err)
	}

	_, err = store.StartPlaythrough(ctx, 1, 100, nil, nil, time.Now().UTC())
	if !errors.Is(err, ErrUserGameNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrUserGameNotFound)
	}
}

func TestStartPlaythroughUnknownPlatform(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	_, err := store.AddToCollection(ctx, newGameParams(100, "Test Game"), newPlatformParams(11, "Xbox"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() failed: %v", err)
	}

	unknownPlatformID := int64(999)
	_, err = store.StartPlaythrough(ctx, 1, 100, &unknownPlatformID, nil, time.Now().UTC())
	if !errors.Is(err, ErrPlatformNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrPlatformNotFound)
	}
}

func TestStartPlaythroughStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     GameStatus
		wantStatus GameStatus
	}{
		{"backlog becomes playing", GameStatusBacklog, GameStatusPlaying},
		{"shelved stays shelved", GameStatusShelved, GameStatusShelved},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			store := newTestStore(t)
			addOwnedGame(t, store, 100)

			err := store.UpdateStatusAndRating(ctx, 1, 100, tt.status, nil)
			if err != nil {
				t.Fatalf("UpdateStatusAndRating() failed: %v", err)
			}

			_, err = store.StartPlaythrough(ctx, 1, 100, nil, nil, time.Now().UTC())
			if err != nil {
				t.Fatalf("StartPlaythrough() failed: %v", err)
			}

			game := getLibraryGame(t, store, 100)
			if game.UserGame.Status == nil || *game.UserGame.Status != string(tt.wantStatus) {
				t.Errorf("UserGame.Status = %v, want %v", derefOrNil(game.UserGame.Status), tt.wantStatus)
			}
		})
	}
}

func TestSetPlaythroughOutcome(t *testing.T) {
	tests := []struct {
		outcome    PlaythroughOutcome
		wantStatus GameStatus
	}{
		{PlaythroughOutcomeCompleted, GameStatusCompleted},
		{PlaythroughOutcomeAbandoned, GameStatusAbandoned},
		{PlaythroughOutcomeRetired, GameStatusRetired},
	}

	for _, tt := range tests {
		t.Run(string(tt.outcome), func(t *testing.T) {
			ctx := t.Context()
			store := newTestStore(t)
			addOwnedGame(t, store, 100)

			pt, err := store.StartPlaythrough(ctx, 1, 100, nil, nil, time.Now().UTC())
			if err != nil {
				t.Fatalf("StartPlaythrough() failed: %v", err)
			}

			changedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
			err = store.SetPlaythroughOutcome(ctx, 1, 100, pt.ID, tt.outcome, changedAt)
			if err != nil {
				t.Fatalf("SetPlaythroughOutcome() failed: %v", err)
			}

			game := getLibraryGame(t, store, 100)
			if len(game.Playthroughs) != 1 {
				t.Fatalf("len(Playthroughs) = %d, want 1", len(game.Playthroughs))
			}
			got := game.Playthroughs[0].Playthrough
			if got.Outcome != string(tt.outcome) {
				t.Errorf("Outcome = %v, want %v", got.Outcome, tt.outcome)
			}
			if got.FinishedAt == nil || !got.FinishedAt.Equal(changedAt) {
				t.Errorf("FinishedAt = %v, want %v", derefOrNil(got.FinishedAt), changedAt)
			}
			if game.UserGame.Status == nil || *game.UserGame.Status != string(tt.wantStatus) {
				t.Errorf("UserGame.Status = %v, want %v", derefOrNil(game.UserGame.Status), tt.wantStatus)
			}
		})
	}
}

func TestSetPlaythroughOutcomeOtherPlaying(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	addOwnedGame(t, store, 100)

	pt, err := store.StartPlaythrough(ctx, 1, 100, nil, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("StartPlaythrough() failed: %v", err)
	}
	_, err = store.StartPlaythrough(ctx, 1, 100, nil, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("StartPlaythrough() failed: %v", err)
	}

	err = store.SetPlaythroughOutcome(ctx, 1, 100, pt.ID, PlaythroughOutcomeCompleted, time.Now().UTC())
	if err != nil {
		t.Fatalf("SetPlaythroughOutcome() failed: %v", err)
	}

	game := getLibraryGame(t, store, 100)
	if game.UserGame.Status == nil || *game.UserGame.Status != string(GameStatusPlaying) {
		t.Errorf("UserGame.Status = %v, want %v", derefOrNil(game.UserGame.Status), GameStatusPlaying)
	}
}

func TestSetPlaythroughOutcomeReopen(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	addOwnedGame(t, store, 100)

	pt, err := store.StartPlaythrough(ctx, 1, 100, nil, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("StartPlaythrough() failed: %v", err)
	}

	err = store.SetPlaythroughOutcome(ctx, 1, 100, pt.ID, PlaythroughOutcomeCompleted, time.Now().UTC())
	if err != nil {
		t.Fatalf("SetPlaythroughOutcome() failed: %v", err)
	}

	err = store.SetPlaythroughOutcome(ctx, 1, 100, pt.ID, PlaythroughOutcomePlaying, time.Now().UTC())
	if err != nil {
		t.Fatalf("SetPlaythroughOutcome() failed: %v", err)
	}

	libraryGame := getLibraryGame(t, store, 100)
	if len(libraryGame.Playthroughs) == 0 {
		t.Fatalf("no playthroughs found")
	}
	got := libraryGame.Playthroughs[0].Playthrough
	if got.Outcome != string(PlaythroughOutcomePlaying) {
		t.Errorf("Outcome = %v, want %v", got.Outcome, PlaythroughOutcomePlaying)
	}
	if got.FinishedAt != nil {
		t.Errorf("FinishedAt = %v, want nil", *got.FinishedAt)
	}
}

func TestSetPlaythroughOutcomeSameOutcomeTwice(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	addOwnedGame(t, store, 100)

	pt, err := store.StartPlaythrough(ctx, 1, 100, nil, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("StartPlaythrough() failed: %v", err)
	}

	firstChangedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	err = store.SetPlaythroughOutcome(ctx, 1, 100, pt.ID, PlaythroughOutcomeCompleted, firstChangedAt)
	if err != nil {
		t.Fatalf("SetPlaythroughOutcome() failed: %v", err)
	}

	libraryGame := getLibraryGame(t, store, 100)
	if len(libraryGame.Playthroughs) == 0 {
		t.Fatalf("no playthroughs found")
	}
	before := libraryGame.Playthroughs[0].Playthrough

	err = store.SetPlaythroughOutcome(ctx, 1, 100, pt.ID, PlaythroughOutcomeCompleted, firstChangedAt.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("SetPlaythroughOutcome() second call failed: %v", err)
	}

	libraryGame = getLibraryGame(t, store, 100)
	if len(libraryGame.Playthroughs) == 0 {
		t.Fatalf("no playthroughs found")
	}
	after := libraryGame.Playthroughs[0].Playthrough
	if before.FinishedAt == nil || after.FinishedAt == nil || !after.FinishedAt.Equal(*before.FinishedAt) {
		t.Errorf("FinishedAt = %v, want %v", derefOrNil(after.FinishedAt), derefOrNil(before.FinishedAt))
	}
}

func TestSetPlaythroughOutcomeErrors(t *testing.T) {
	tests := []struct {
		name    string
		igdbID  int64
		outcome PlaythroughOutcome
		wantErr error
	}{
		{"invalid outcome", 100, PlaythroughOutcome("invalid"), ErrInvalidPlaythroughOutcome},
		{"other game", 101, PlaythroughOutcomeCompleted, ErrPlaythroughNotFound},
		{"not in library", 999, PlaythroughOutcomeCompleted, ErrUserGameNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			store := newTestStore(t)
			addOwnedGame(t, store, 100)
			addOwnedGame(t, store, 101)

			pt, err := store.StartPlaythrough(ctx, 1, 100, nil, nil, time.Now().UTC())
			if err != nil {
				t.Fatalf("StartPlaythrough() failed: %v", err)
			}

			err = store.SetPlaythroughOutcome(ctx, 1, tt.igdbID, pt.ID, tt.outcome, time.Now().UTC())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeletePlaythrough(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	_, err := store.AddToCollection(ctx, newGameParams(100, "Test Game"), newPlatformParams(11, "Xbox"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() failed: %v", err)
	}

	pt, err := store.StartPlaythrough(ctx, 1, 100, nil, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("StartPlaythrough() failed: %v", err)
	}

	err = store.DeletePlaythrough(ctx, 1, 100, pt.ID)
	if err != nil {
		t.Fatalf("DeletePlaythrough() failed: %v", err)
	}

	pts, err := store.queries.ListPlaythroughsForUserGameID(ctx, pt.UserGameID)
	if err != nil {
		t.Fatalf("ListPlaythroughsForUserGameID() failed: %v", err)
	}
	if len(pts) != 0 {
		t.Fatalf("len(pts) = %d, want 0", len(pts))
	}

	game, err := store.GetLibraryGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if game.UserGame.Status == nil || *game.UserGame.Status != "playing" {
		t.Errorf("UserGame.Status = %s, want %s", derefOrNil(game.UserGame.Status), "playing")
	}
}

func TestDeletePlaythroughTwice(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	_, err := store.AddToCollection(ctx, newGameParams(100, "Test Game"), newPlatformParams(11, "Xbox"), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() failed: %v", err)
	}

	pt, err := store.StartPlaythrough(ctx, 1, 100, nil, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("StartPlaythrough() failed: %v", err)
	}

	err = store.DeletePlaythrough(ctx, 1, 100, pt.ID)
	if err != nil {
		t.Fatalf("DeletePlaythrough() failed: %v", err)
	}

	err = store.DeletePlaythrough(ctx, 1, 100, pt.ID)
	if !errors.Is(err, ErrPlaythroughNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrPlaythroughNotFound)
	}
}

func TestDeletePlaythroughOtherGame(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	platform := newPlatformParams(11, "Xbox")

	_, err := store.AddToCollection(ctx, newGameParams(100, "Test Game"), platform, 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() failed: %v", err)
	}
	_, err = store.AddToCollection(ctx, newGameParams(101, "Test Game 2"), platform, 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToCollection() failed: %v", err)
	}

	pt, err := store.StartPlaythrough(ctx, 1, 100, nil, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("StartPlaythrough() failed: %v", err)
	}

	err = store.DeletePlaythrough(ctx, 1, 101, pt.ID)
	if !errors.Is(err, ErrPlaythroughNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrPlaythroughNotFound)
	}
}

func TestDeletePlaythroughNotInLibrary(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	err := store.DeletePlaythrough(ctx, 1, 100, 1)
	if !errors.Is(err, ErrUserGameNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrUserGameNotFound)
	}
}
