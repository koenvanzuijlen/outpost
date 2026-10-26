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

func newDLC(igdbID, parentIGDBID int64, title string, releasedAt time.Time, platforms ...dbgen.UpsertPlatformParams) DLC {
	params := newGameParams(igdbID, title)
	params.ParentIgdbID = &parentIGDBID
	params.GameType = int64(igdb.GameTypeDLCAddon)
	params.ReleasedAt = &releasedAt
	return DLC{Game: params, Platforms: platforms}
}

// countGamePlatforms returns the number of game_platforms links for the game with the given IGDB ID.
func countGamePlatforms(t *testing.T, store *Store, igdbID int64) int {
	t.Helper()
	var n int
	err := store.db.QueryRowContext(t.Context(),
		`SELECT COUNT(*) FROM game_platforms gp JOIN games g ON g.id = gp.game_id WHERE g.igdb_id = ?`, igdbID,
	).Scan(&n)
	if err != nil {
		t.Fatalf("count game_platforms failed: %v", err)
	}
	return n
}

func TestSaveDLCs(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	addOwnedGame(t, store, 100)

	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	dlcs := []DLC{
		newDLC(203, 100, "DLC C", base.AddDate(0, 2, 0)),
		newDLC(201, 100, "DLC A", base),
		newDLC(202, 100, "DLC B", base.AddDate(0, 1, 0)),
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

func TestSaveDLCsPlatforms(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	addOwnedGame(t, store, 100)
	releasedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	dlc := newDLC(201, 100, "DLC A", releasedAt, newPlatformParams(6, "PC"), newPlatformParams(48, "PS4"))
	if err := store.SaveDLCs(ctx, 100, []DLC{dlc}, time.Now().UTC()); err != nil {
		t.Fatalf("SaveDLCs() failed: %v", err)
	}

	if got := countGamePlatforms(t, store, 201); got != 2 {
		t.Errorf("game_platforms for 201 = %d, want 2", got)
	}
	for _, igdbID := range []int64{6, 48} {
		if _, err := store.queries.GetPlatformByIGDBID(ctx, igdbID); err != nil {
			t.Errorf("GetPlatformByIGDBID(%d) failed: %v", igdbID, err)
		}
	}
}

func TestSaveDLCsReplacesPlatforms(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	addOwnedGame(t, store, 100)
	releasedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	first := newDLC(201, 100, "DLC A", releasedAt, newPlatformParams(6, "PC"), newPlatformParams(48, "PS4"))
	if err := store.SaveDLCs(ctx, 100, []DLC{first}, time.Now().UTC()); err != nil {
		t.Fatalf("SaveDLCs() first failed: %v", err)
	}
	second := newDLC(201, 100, "DLC A", releasedAt, newPlatformParams(6, "PC"))
	if err := store.SaveDLCs(ctx, 100, []DLC{second}, time.Now().UTC()); err != nil {
		t.Fatalf("SaveDLCs() second failed: %v", err)
	}

	if got := countGamePlatforms(t, store, 201); got != 1 {
		t.Errorf("game_platforms for 201 = %d, want 1", got)
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

	dlc := newDLC(201, 100, "DLC A", releasedAt, newPlatformParams(6, "PC"))
	err := store.SaveDLCs(ctx, 100, []DLC{dlc}, time.Now().UTC())
	if err == nil {
		t.Fatal("SaveDLCs() error = nil, want an error")
	}

	// The transaction must roll back, so the DLC and its platform links are not stored either.
	if _, err := store.queries.GetGameByIGDBID(ctx, 201); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("GetGameByIGDBID(201) error = %v, want %v", err, sql.ErrNoRows)
	}
	var links int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM game_platforms`).Scan(&links); err != nil {
		t.Fatalf("count game_platforms failed: %v", err)
	}
	if links != 0 {
		t.Errorf("game_platforms count = %d, want 0", links)
	}
}

func TestSaveDLCsTwice(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	addOwnedGame(t, store, 100)
	releasedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	first := []DLC{
		newDLC(201, 100, "DLC A", releasedAt),
		newDLC(202, 100, "DLC B", releasedAt.AddDate(0, 1, 0)),
	}
	if err := store.SaveDLCs(ctx, 100, first, time.Now().UTC()); err != nil {
		t.Fatalf("SaveDLCs() first failed: %v", err)
	}
	second := []DLC{
		newDLC(201, 100, "DLC A Renamed", releasedAt),
		newDLC(202, 100, "DLC B", releasedAt.AddDate(0, 1, 0)),
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

	if err := store.SaveDLCs(ctx, 100, []DLC{newDLC(201, 100, "DLC A", releasedAt)}, time.Now().UTC()); err != nil {
		t.Fatalf("SaveDLCs() failed: %v", err)
	}
	standalone := newDLC(300, 100, "Standalone Expansion", releasedAt).Game
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

func TestGetLibraryGameCopyDLCs(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")
	for _, platform := range []dbgen.UpsertPlatformParams{newPlatformParams(6, "PC"), newPlatformParams(48, "PS4"), newPlatformParams(130, "Switch")} {
		if _, err := store.AddToCollection(ctx, game, platform, 1, time.Now().UTC()); err != nil {
			t.Fatalf("AddToCollection(%s) failed: %v", platform.Name, err)
		}
	}

	releasedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	dlcs := []DLC{
		newDLC(201, 100, "DLC A", releasedAt, newPlatformParams(6, "PC"), newPlatformParams(48, "PS4")),
		newDLC(202, 100, "DLC B", releasedAt.AddDate(0, 1, 0), newPlatformParams(6, "PC")),
	}
	if err := store.SaveDLCs(ctx, 100, dlcs, time.Now().UTC()); err != nil {
		t.Fatalf("SaveDLCs() failed: %v", err)
	}

	if err := store.SetDLCOwned(ctx, 1, 100, 6, 201, true); err != nil {
		t.Fatalf("SetDLCOwned() failed: %v", err)
	}

	m := getLibraryGame(t, store, 100).CopyDLCs

	assertCopyDLCs(t, m[6], map[int64]bool{201: true, 202: false})
	assertCopyDLCs(t, m[48], map[int64]bool{201: false})

	if len(m[130]) != 0 {
		t.Errorf("len(CopyDLCs[130]) = %d, want 0", len(m[130]))
	}
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
		if _, err := store.AddToCollection(ctx, newGameParams(c.igdbID, "Test Game"), c.platform, 1, time.Now().UTC()); err != nil {
			t.Fatalf("AddToCollection(%d, %s) failed: %v", c.igdbID, c.platform.Name, err)
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

func TestSetDLCOwned(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupDLCOwnership(t, store)

	if err := store.SetDLCOwned(ctx, 1, 100, 6, 201, true); err != nil {
		t.Fatalf("SetDLCOwned() failed: %v", err)
	}

	m := getLibraryGame(t, store, 100).CopyDLCs
	assertCopyDLCs(t, m[6], map[int64]bool{201: true, 202: false})
	assertCopyDLCs(t, m[48], map[int64]bool{201: false})
}

func TestSetDLCOwnedTwice(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupDLCOwnership(t, store)

	for i := range 2 {
		if err := store.SetDLCOwned(ctx, 1, 100, 6, 201, true); err != nil {
			t.Fatalf("SetDLCOwned() call %d failed: %v", i+1, err)
		}
	}

	assertCopyDLCs(t, getLibraryGame(t, store, 100).CopyDLCs[6], map[int64]bool{201: true, 202: false})
}

func TestSetDLCOwnedUnown(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupDLCOwnership(t, store)

	if err := store.SetDLCOwned(ctx, 1, 100, 6, 201, true); err != nil {
		t.Fatalf("SetDLCOwned(true) failed: %v", err)
	}
	if err := store.SetDLCOwned(ctx, 1, 100, 6, 201, false); err != nil {
		t.Fatalf("SetDLCOwned(false) failed: %v", err)
	}

	assertCopyDLCs(t, getLibraryGame(t, store, 100).CopyDLCs[6], map[int64]bool{201: false, 202: false})
}

func TestSetDLCOwnedUnownNotOwned(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupDLCOwnership(t, store)

	if err := store.SetDLCOwned(ctx, 1, 100, 6, 201, false); err != nil {
		t.Fatalf("SetDLCOwned(false) failed: %v", err)
	}

	assertCopyDLCs(t, getLibraryGame(t, store, 100).CopyDLCs[6], map[int64]bool{201: false, 202: false})
}

func TestSetDLCOwnedCopyDLCNotFound(t *testing.T) {
	tests := []struct {
		name           string
		platformIGDBID int64
		dlcIGDBID      int64
	}{
		{"dlc not on platform", 48, 202},
		{"no copy on platform", 130, 201},
		{"other game's dlc", 6, 401},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t)
			setupDLCOwnership(t, store)

			err := store.SetDLCOwned(t.Context(), 1, 100, tt.platformIGDBID, tt.dlcIGDBID, true)
			if !errors.Is(err, ErrCopyDLCNotFound) {
				t.Errorf("SetDLCOwned() error = %v, want %v", err, ErrCopyDLCNotFound)
			}
		})
	}
}

func TestSetDLCOwnedUserGameNotFound(t *testing.T) {
	tests := []struct {
		name   string
		igdbID int64
	}{
		{"wishlisted", 500},
		{"not in library", 999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t)
			setupDLCOwnership(t, store)
			if err := store.AddToWishlist(t.Context(), newGameParams(500, "Wishlisted Game"), 1, time.Now().UTC()); err != nil {
				t.Fatalf("AddToWishlist() failed: %v", err)
			}

			err := store.SetDLCOwned(t.Context(), 1, tt.igdbID, 6, 201, true)
			if !errors.Is(err, ErrUserGameNotFound) {
				t.Errorf("SetDLCOwned() error = %v, want %v", err, ErrUserGameNotFound)
			}
		})
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
