package database

import (
	"log/slog"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// newGameParams returns UpsertGame parameters for a main game with the given IGDB ID and title.
func newGameParams(igdbID int64, title string) dbgen.UpsertGameParams {
	return dbgen.UpsertGameParams{
		IgdbID:    igdbID,
		FetchedAt: time.Now().UTC(),
		Title:     title,
		GameType:  0,
	}
}

// newPlatformParams returns UpsertPlatform parameters for a platform with the given IGDB ID and name.
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

// getUserGame returns the game with the given IGDB ID from user 1's library or wishlist. It fails the test if it isn't
// there.
func getUserGame(t *testing.T, store *Store, igdbID int64) UserGameDetails {
	t.Helper()
	details, err := store.GetUserGame(t.Context(), 1, igdbID)
	if err != nil {
		t.Fatalf("GetUserGame() failed: %v", err)
	}
	return details
}

// onlyPlaythrough returns the single playthrough of details. It fails the test if there isn't exactly one.
func onlyPlaythrough(t *testing.T, details UserGameDetails) dbgen.ListPlaythroughsForUserGameIDRow {
	t.Helper()
	if len(details.Playthroughs) != 1 {
		t.Fatalf("len(Playthroughs) = %d, want 1", len(details.Playthroughs))
	}
	return details.Playthroughs[0]
}

// derefOrNil returns the value p points to, or nil, so failure messages show a value instead of an address.
func derefOrNil[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}

// newDLC returns a DLC of parentIGDBID, released at releasedAt and available on platforms, as SaveDLCs takes it.
func newDLC(igdbID, parentIGDBID int64, title string, releasedAt time.Time, platforms ...dbgen.UpsertPlatformParams) DLC {
	params := newGameParams(igdbID, title)
	params.ParentIgdbID = &parentIGDBID
	params.GameType = int64(igdb.GameTypeDLCAddon)
	params.ReleasedAt = &releasedAt
	return DLC{Game: params, Platforms: platforms}
}

// setupDLCOwnership owns game 100 on PC (6) and PS4 (48) with DLC 201 (PC, PS4) and DLC 202 (PC only), and game 300
// on PC with DLC 401 (PC).
func setupDLCOwnership(t *testing.T, store *Store) {
	t.Helper()
	ctx := t.Context()
	// Game 300 goes first so game 100's user game ID (2) differs from the user ID (1).
	copies := []struct {
		igdbID   int64
		platform dbgen.UpsertPlatformParams
	}{
		{300, newPlatformParams(6, "PC")},
		{100, newPlatformParams(6, "PC")},
		{100, newPlatformParams(48, "PS4")},
	}
	for _, c := range copies {
		if err := store.AddToLibrary(ctx, newGameParams(c.igdbID, "Test Game"), &c.platform, 1, time.Now().UTC()); err != nil {
			t.Fatalf("AddToLibrary(%d, %s) failed: %v", c.igdbID, c.platform.Name, err)
		}
	}

	releasedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	dlcs := []DLC{
		newDLC(201, 100, "DLC A", releasedAt, newPlatformParams(6, "PC"), newPlatformParams(48, "PS4")),
		newDLC(202, 100, "DLC B", releasedAt, newPlatformParams(6, "PC")),
	}
	if err := store.SaveDLCs(ctx, 100, dlcs, time.Now().UTC()); err != nil {
		t.Fatalf("SaveDLCs(100) failed: %v", err)
	}
	if err := store.SaveDLCs(ctx, 300, []DLC{newDLC(401, 300, "Other DLC", releasedAt, newPlatformParams(6, "PC"))}, time.Now().UTC()); err != nil {
		t.Fatalf("SaveDLCs(300) failed: %v", err)
	}
}

// setupLibraryDLCs extends setupDLCOwnership by owning DLC 201 on PC and tracking DLCs 201 and 202.
func setupLibraryDLCs(t *testing.T, store *Store) {
	t.Helper()
	ctx := t.Context()
	setupDLCOwnership(t, store)
	if err := store.SetDLCOwned(ctx, 1, 100, 6, 201, true); err != nil {
		t.Fatalf("SetDLCOwned() failed: %v", err)
	}
	for _, dlcIGDBID := range []int64{201, 202} {
		if err := store.AddDLCToLibrary(ctx, 1, 100, dlcIGDBID, time.Now().UTC()); err != nil {
			t.Fatalf("AddDLCToLibrary(%d) failed: %v", dlcIGDBID, err)
		}
	}
}

// findDLC returns the DLC row with the given IGDB ID from the library game, failing the test if it's missing.
func findDLC(t *testing.T, game UserGameDetails, igdbID int64) dbgen.ListDLCGamesRow {
	t.Helper()
	i := slices.IndexFunc(game.DLCs, func(g dbgen.ListDLCGamesRow) bool { return g.Game.IgdbID == igdbID })
	if i == -1 {
		t.Fatalf("DLC %d not found in DLCs", igdbID)
	}
	return game.DLCs[i]
}

// assertCopyDLCs checks that got contains exactly the DLCs in want (keyed by IGDB ID) with the given owned state.
func assertCopyDLCs(t *testing.T, got []CopyDLC, want map[int64]bool) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(CopyDLCs) = %d, want %d", len(got), len(want))
	}
	for igdbID, wantOwned := range want {
		i := slices.IndexFunc(got, func(d CopyDLC) bool { return d.Game.IgdbID == igdbID })
		if i == -1 {
			t.Errorf("DLC %d not found", igdbID)
			continue
		}
		if got[i].Owned != wantOwned {
			t.Errorf("DLC %d Owned = %v, want %v", igdbID, got[i].Owned, wantOwned)
		}
	}
}

// assertDLCStatus checks that the DLC row is tracked with the given status.
func assertDLCStatus(t *testing.T, dlc dbgen.ListDLCGamesRow, want GameStatus) {
	t.Helper()
	if !dlc.InLibrary {
		t.Errorf("DLC %d InLibrary = false, want true", dlc.Game.IgdbID)
	}
	if dlc.Status == nil || *dlc.Status != string(want) {
		t.Errorf("DLC %d Status = %v, want %q", dlc.Game.IgdbID, derefOrNil(dlc.Status), want)
	}
}

// assertPlatformIDs checks that got contains exactly the platforms with the given IGDB IDs, in any order.
func assertPlatformIDs(t *testing.T, got []dbgen.Platform, want ...int64) {
	t.Helper()
	ids := make([]int64, 0, len(got))
	for _, p := range got {
		ids = append(ids, p.IgdbID)
	}
	slices.Sort(ids)
	slices.Sort(want)
	if !slices.Equal(ids, want) {
		t.Errorf("Platforms IGDB IDs = %v, want %v", ids, want)
	}
}
