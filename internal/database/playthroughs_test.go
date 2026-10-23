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

func TestFinishPlaythrough(t *testing.T) {
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

	finishedAt := time.Now().UTC()
	err = store.FinishPlaythrough(ctx, 1, 100, pt.ID, PlaythroughOutcomeAbandoned, finishedAt)
	if err != nil {
		t.Fatalf("FinishPlaythrough() failed: %v", err)
	}

	pts, err := store.queries.ListPlaythroughsForUserGameID(ctx, pt.UserGameID)
	if err != nil {
		t.Fatalf("ListPlaythroughsForUserGameID() failed: %v", err)
	}
	if len(pts) != 1 {
		t.Fatalf("len(pts) = %d, want 1", len(pts))
	}
	if pts[0].Playthrough.Outcome != string(PlaythroughOutcomeAbandoned) {
		t.Errorf("Outcome = %v, want %v", pts[0].Playthrough.Outcome, PlaythroughOutcomeAbandoned)
	}
	if pts[0].Playthrough.FinishedAt == nil || !pts[0].Playthrough.FinishedAt.Equal(finishedAt) {
		t.Errorf("FinishedAt = %v, want %v", pts[0].Playthrough.FinishedAt, finishedAt)
	}

	game, err := store.GetLibraryGame(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if game.UserGame.Status == nil || *game.UserGame.Status != "abandoned" {
		t.Errorf("UserGame.Status = %s, want %s", *game.UserGame.Status, "abandoned")
	}
}

func TestFinishPlaythroughInvalidOutcome(t *testing.T) {
	tests := []struct {
		name    string
		outcome PlaythroughOutcome
	}{
		{name: "invalid outcome", outcome: PlaythroughOutcome("invalid")},
		{name: "playing", outcome: PlaythroughOutcomePlaying},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			err = store.FinishPlaythrough(ctx, 1, 100, pt.ID, tt.outcome, time.Now().UTC())
			if !errors.Is(err, ErrInvalidPlaythroughOutcome) {
				t.Fatalf("err = %v, want %v", err, ErrInvalidPlaythroughOutcome)
			}
		})
	}
}

func TestFinishPlaythroughTwice(t *testing.T) {
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

	err = store.FinishPlaythrough(ctx, 1, 100, pt.ID, PlaythroughOutcomeAbandoned, time.Now().UTC())
	if err != nil {
		t.Fatalf("FinishPlaythrough() failed: %v", err)
	}

	err = store.FinishPlaythrough(ctx, 1, 100, pt.ID, PlaythroughOutcomeAbandoned, time.Now().UTC())
	if !errors.Is(err, ErrPlaythroughNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrPlaythroughNotFound)
	}
}

func TestFinishPlaythroughOtherGame(t *testing.T) {
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

	err = store.FinishPlaythrough(ctx, 1, 101, pt.ID, PlaythroughOutcomeAbandoned, time.Now().UTC())
	if !errors.Is(err, ErrPlaythroughNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrPlaythroughNotFound)
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
