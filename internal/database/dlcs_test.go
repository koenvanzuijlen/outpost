package database

import (
	"database/sql"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

func newDLCParams(igdbID, parentIGDBID int64, title string, releasedAt time.Time) dbgen.UpsertGameParams {
	params := newGameParams(igdbID, title)
	params.ParentIgdbID = &parentIGDBID
	params.GameType = int64(igdb.GameTypeDLCAddon)
	params.ReleasedAt = &releasedAt
	return params
}

func TestSaveDLCs(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	addOwnedGame(t, store, 100)

	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	dlcs := []dbgen.UpsertGameParams{
		newDLCParams(203, 100, "DLC C", base.AddDate(0, 2, 0)),
		newDLCParams(201, 100, "DLC A", base),
		newDLCParams(202, 100, "DLC B", base.AddDate(0, 1, 0)),
	}
	if err := store.SaveDLCs(ctx, 100, dlcs, time.Now().UTC()); err != nil {
		t.Fatalf("SaveDLCs() failed: %v", err)
	}

	game := getLibraryGame(t, store, 100)
	if game.Game.DlcsFetchedAt == nil {
		t.Errorf("DlcsFetchedAt = nil, want set")
	}
	if len(game.DLCs) != 3 {
		t.Fatalf("len(DLCs) = %d, want 3", len(game.DLCs))
	}
	want := []int64{201, 202, 203}
	for i, dlc := range game.DLCs {
		if dlc.IgdbID != want[i] {
			t.Errorf("DLCs[%d].IgdbID = %d, want %d", i, dlc.IgdbID, want[i])
		}
	}
}

func TestSaveDLCsEmpty(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	addOwnedGame(t, store, 100)

	if err := store.SaveDLCs(ctx, 100, nil, time.Now().UTC()); err != nil {
		t.Fatalf("SaveDLCs() failed: %v", err)
	}

	game := getLibraryGame(t, store, 100)
	if game.Game.DlcsFetchedAt == nil {
		t.Errorf("DlcsFetchedAt = nil, want set")
	}
	if len(game.DLCs) != 0 {
		t.Errorf("len(DLCs) = %d, want 0", len(game.DLCs))
	}
}

func TestSaveDLCsMissingParent(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	releasedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	err := store.SaveDLCs(ctx, 100, []dbgen.UpsertGameParams{newDLCParams(201, 100, "DLC A", releasedAt)}, time.Now().UTC())
	if err == nil {
		t.Fatal("SaveDLCs() error = nil, want an error")
	}

	// The transaction must roll back, so the DLC is not stored either.
	if _, err := store.queries.GetGameByIGDBID(ctx, 201); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("GetGameByIGDBID(201) error = %v, want %v", err, sql.ErrNoRows)
	}
}

func TestSaveDLCsTwice(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	addOwnedGame(t, store, 100)
	releasedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	first := []dbgen.UpsertGameParams{
		newDLCParams(201, 100, "DLC A", releasedAt),
		newDLCParams(202, 100, "DLC B", releasedAt.AddDate(0, 1, 0)),
	}
	if err := store.SaveDLCs(ctx, 100, first, time.Now().UTC()); err != nil {
		t.Fatalf("SaveDLCs() first failed: %v", err)
	}
	second := []dbgen.UpsertGameParams{
		newDLCParams(201, 100, "DLC A Renamed", releasedAt),
		newDLCParams(202, 100, "DLC B", releasedAt.AddDate(0, 1, 0)),
	}
	if err := store.SaveDLCs(ctx, 100, second, time.Now().UTC()); err != nil {
		t.Fatalf("SaveDLCs() second failed: %v", err)
	}

	game := getLibraryGame(t, store, 100)
	if len(game.DLCs) != 2 {
		t.Fatalf("len(DLCs) = %d, want 2", len(game.DLCs))
	}
	i := slices.IndexFunc(game.DLCs, func(g dbgen.Game) bool { return g.IgdbID == 201 })
	if i == -1 {
		t.Fatalf("DLC 201 not found in DLCs")
	}
	if game.DLCs[i].Title != "DLC A Renamed" {
		t.Errorf("DLC 201 Title = %q, want %q", game.DLCs[i].Title, "DLC A Renamed")
	}
}

func TestSaveDLCsExcludesStandaloneExpansion(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	addOwnedGame(t, store, 100)
	releasedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	if err := store.SaveDLCs(ctx, 100, []dbgen.UpsertGameParams{newDLCParams(201, 100, "DLC A", releasedAt)}, time.Now().UTC()); err != nil {
		t.Fatalf("SaveDLCs() failed: %v", err)
	}
	standalone := newDLCParams(300, 100, "Standalone Expansion", releasedAt)
	standalone.GameType = int64(igdb.GameTypeStandaloneExpansion)
	if _, err := store.queries.UpsertGame(ctx, standalone); err != nil {
		t.Fatalf("UpsertGame() failed: %v", err)
	}

	game := getLibraryGame(t, store, 100)
	if len(game.DLCs) != 1 {
		t.Fatalf("len(DLCs) = %d, want 1", len(game.DLCs))
	}
	if slices.ContainsFunc(game.DLCs, func(g dbgen.Game) bool { return g.IgdbID == 300 }) {
		t.Errorf("DLCs contains standalone expansion 300, want it excluded")
	}
}

func TestAddToCollectionKeepsDLCsFetchedAt(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	addOwnedGame(t, store, 100)

	if err := store.SaveDLCs(ctx, 100, nil, time.Now().UTC()); err != nil {
		t.Fatalf("SaveDLCs() failed: %v", err)
	}
	if _, err := store.AddToCollection(ctx, newGameParams(100, "Test Game"), newPlatformParams(6, "PC"), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToCollection() failed: %v", err)
	}

	game := getLibraryGame(t, store, 100)
	if game.Game.DlcsFetchedAt == nil {
		t.Errorf("DlcsFetchedAt = nil, want still set")
	}
}
