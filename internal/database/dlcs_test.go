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

	details := getGame(t, store, 100)
	if details.Game.DlcsFetchedAt == nil {
		t.Errorf("DlcsFetchedAt = nil, want set")
	}
	if len(details.DLCs) != 3 {
		t.Fatalf("len(DLCs) = %d, want 3", len(details.DLCs))
	}
	want := []int64{201, 202, 203}
	for i, dlc := range details.DLCs {
		if dlc.Game.IgdbID != want[i] {
			t.Errorf("DLCs[%d].IgdbID = %d, want %d", i, dlc.Game.IgdbID, want[i])
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

	details := getGame(t, store, 100)
	if details.Game.DlcsFetchedAt == nil {
		t.Errorf("DlcsFetchedAt = nil, want set")
	}
	if len(details.DLCs) != 0 {
		t.Errorf("len(DLCs) = %d, want 0", len(details.DLCs))
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

	details := getGame(t, store, 100)
	if len(details.DLCs) != 2 {
		t.Fatalf("len(DLCs) = %d, want 2", len(details.DLCs))
	}
	i := slices.IndexFunc(details.DLCs, func(g dbgen.ListDLCGamesRow) bool { return g.Game.IgdbID == 201 })
	if i == -1 {
		t.Fatalf("DLC 201 not found in DLCs")
	}
	if details.DLCs[i].Game.Title != "DLC A Renamed" {
		t.Errorf("DLC 201 Title = %q, want %q", details.DLCs[i].Game.Title, "DLC A Renamed")
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

	details := getGame(t, store, 100)
	if len(details.DLCs) != 1 {
		t.Fatalf("len(DLCs) = %d, want 1", len(details.DLCs))
	}
	if slices.ContainsFunc(details.DLCs, func(g dbgen.ListDLCGamesRow) bool { return g.Game.IgdbID == 300 }) {
		t.Errorf("DLCs contains standalone expansion 300, want it excluded")
	}
}

func TestGetGameCopyDLCs(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	game := newGameParams(100, "Test Game")
	for _, platform := range []dbgen.UpsertPlatformParams{newPlatformParams(6, "PC"), newPlatformParams(48, "PS4"), newPlatformParams(130, "Switch")} {
		if err := store.AddToLibrary(ctx, game, &platform, 1, time.Now().UTC()); err != nil {
			t.Fatalf("AddToLibrary(%s) failed: %v", platform.Name, err)
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

	copyDLCs := getGame(t, store, 100).CopyDLCs

	assertCopyDLCs(t, copyDLCs[6], map[int64]bool{201: true, 202: false})
	assertCopyDLCs(t, copyDLCs[48], map[int64]bool{201: false})

	if len(copyDLCs[130]) != 0 {
		t.Errorf("len(CopyDLCs[130]) = %d, want 0", len(copyDLCs[130]))
	}
}

func TestSetDLCOwned(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupDLCOwnership(t, store)

	if err := store.SetDLCOwned(ctx, 1, 100, 6, 201, true); err != nil {
		t.Fatalf("SetDLCOwned() failed: %v", err)
	}

	copyDLCs := getGame(t, store, 100).CopyDLCs
	assertCopyDLCs(t, copyDLCs[6], map[int64]bool{201: true, 202: false})
	assertCopyDLCs(t, copyDLCs[48], map[int64]bool{201: false})
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

	assertCopyDLCs(t, getGame(t, store, 100).CopyDLCs[6], map[int64]bool{201: true, 202: false})
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

	assertCopyDLCs(t, getGame(t, store, 100).CopyDLCs[6], map[int64]bool{201: false, 202: false})
}

func TestSetDLCOwnedUnownNotOwned(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupDLCOwnership(t, store)

	if err := store.SetDLCOwned(ctx, 1, 100, 6, 201, false); err != nil {
		t.Fatalf("SetDLCOwned(false) failed: %v", err)
	}

	assertCopyDLCs(t, getGame(t, store, 100).CopyDLCs[6], map[int64]bool{201: false, 202: false})
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
			if !errors.Is(err, ErrDLCNotOnCopy) {
				t.Errorf("SetDLCOwned() error = %v, want %v", err, ErrDLCNotOnCopy)
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

func TestAddDLCToLibrary(t *testing.T) {
	store := newTestStore(t)
	setupDLCOwnership(t, store)

	if err := store.AddDLCToLibrary(t.Context(), 1, 100, 201, time.Now().UTC()); err != nil {
		t.Fatalf("AddDLCToLibrary() failed: %v", err)
	}

	details := getGame(t, store, 100)
	assertDLCStatus(t, findDLC(t, details, 201), GameStatusBacklog)
	if findDLC(t, details, 202).InLibrary {
		t.Errorf("DLC 202 InLibrary = true, want false")
	}
}

func TestAddDLCToLibraryTwice(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupDLCOwnership(t, store)

	if err := store.AddDLCToLibrary(ctx, 1, 100, 201, time.Now().UTC()); err != nil {
		t.Fatalf("AddDLCToLibrary() first failed: %v", err)
	}
	if err := store.UpdateGameDetails(ctx, 1, 201, GameDetails{Status: GameStatusCompleted}); err != nil {
		t.Fatalf("UpdateGameDetails() failed: %v", err)
	}
	if err := store.AddDLCToLibrary(ctx, 1, 100, 201, time.Now().UTC()); err != nil {
		t.Fatalf("AddDLCToLibrary() second failed: %v", err)
	}

	assertDLCStatus(t, findDLC(t, getGame(t, store, 100), 201), GameStatusCompleted)
}

func TestAddDLCToLibraryKeepsOwnership(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupDLCOwnership(t, store)

	if err := store.SetDLCOwned(ctx, 1, 100, 6, 201, true); err != nil {
		t.Fatalf("SetDLCOwned() failed: %v", err)
	}
	if err := store.AddDLCToLibrary(ctx, 1, 100, 201, time.Now().UTC()); err != nil {
		t.Fatalf("AddDLCToLibrary() failed: %v", err)
	}

	assertCopyDLCs(t, getGame(t, store, 100).CopyDLCs[6], map[int64]bool{201: true, 202: false})
}

func TestAddDLCToLibraryNotFound(t *testing.T) {
	tests := []struct {
		name      string
		dlcIGDBID int64
	}{
		{"unknown dlc", 999},
		{"other game's dlc", 401},
		{"standalone expansion", 501},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			store := newTestStore(t)
			setupDLCOwnership(t, store)
			standalone := newDLC(501, 100, "Standalone Expansion", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)).Game
			standalone.GameType = int64(igdb.GameTypeStandaloneExpansion)
			if _, err := store.queries.UpsertGame(ctx, standalone); err != nil {
				t.Fatalf("UpsertGame() failed: %v", err)
			}

			err := store.AddDLCToLibrary(ctx, 1, 100, tt.dlcIGDBID, time.Now().UTC())
			if !errors.Is(err, ErrDLCNotFound) {
				t.Errorf("AddDLCToLibrary() error = %v, want %v", err, ErrDLCNotFound)
			}
		})
	}
}

func TestAddDLCToLibraryUserGameNotFound(t *testing.T) {
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

			err := store.AddDLCToLibrary(t.Context(), 1, tt.igdbID, 201, time.Now().UTC())
			if !errors.Is(err, ErrUserGameNotFound) {
				t.Errorf("AddDLCToLibrary() error = %v, want %v", err, ErrUserGameNotFound)
			}
		})
	}
}

func TestGetGameOwnedDLC(t *testing.T) {
	store := newTestStore(t)
	setupLibraryDLCs(t, store)

	details := getGame(t, store, 201)
	if details.Parent == nil || details.Parent.IgdbID != 100 {
		t.Errorf("Parent = %v, want game 100", derefOrNil(details.Parent))
	}
	assertPlatformIDs(t, details.OwnedPlatforms, 6)
}

func TestGetGameUnownedLibraryDLC(t *testing.T) {
	store := newTestStore(t)
	setupLibraryDLCs(t, store)

	details := getGame(t, store, 202)
	if details.Parent == nil || details.Parent.IgdbID != 100 {
		t.Errorf("Parent = %v, want game 100", derefOrNil(details.Parent))
	}
	assertPlatformIDs(t, details.OwnedPlatforms)
}

func TestGetGameParentOfLibraryDLC(t *testing.T) {
	store := newTestStore(t)
	setupLibraryDLCs(t, store)

	details := getGame(t, store, 100)
	if details.Parent != nil {
		t.Errorf("Parent = %v, want nil", *details.Parent)
	}
	assertPlatformIDs(t, details.OwnedPlatforms, 6, 48)
}

func TestGetGameStandaloneExpansion(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupLibraryDLCs(t, store)

	standalone := newDLC(501, 100, "Standalone Expansion", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)).Game
	standalone.GameType = int64(igdb.GameTypeStandaloneExpansion)
	if err := store.AddToLibrary(ctx, standalone, new(newPlatformParams(130, "Switch")), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}

	details := getGame(t, store, 501)
	if details.Parent != nil {
		t.Errorf("Parent = %v, want nil", *details.Parent)
	}
	assertPlatformIDs(t, details.OwnedPlatforms, 130)
}

func TestAddToLibraryKeepsDLCsFetchedAt(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	addOwnedGame(t, store, 100)

	if err := store.SaveDLCs(ctx, 100, nil, time.Now().UTC()); err != nil {
		t.Fatalf("SaveDLCs() failed: %v", err)
	}
	if err := store.AddToLibrary(ctx, newGameParams(100, "Test Game"), new(newPlatformParams(6, "PC")), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}

	details := getGame(t, store, 100)
	if details.Game.DlcsFetchedAt == nil {
		t.Errorf("DlcsFetchedAt = nil, want still set")
	}
}
