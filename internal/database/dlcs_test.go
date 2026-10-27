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
	i := slices.IndexFunc(game.DLCs, func(g dbgen.ListDLCGamesRow) bool { return g.Game.IgdbID == 201 })
	if i == -1 {
		t.Fatalf("DLC 201 not found in DLCs")
	}
	if game.DLCs[i].Game.Title != "DLC A Renamed" {
		t.Errorf("DLC 201 Title = %q, want %q", game.DLCs[i].Game.Title, "DLC A Renamed")
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
	if slices.ContainsFunc(game.DLCs, func(g dbgen.ListDLCGamesRow) bool { return g.Game.IgdbID == 300 }) {
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

// findDLC returns the DLC row with the given IGDB ID from the library game, failing the test if it's missing.
func findDLC(t *testing.T, game LibraryGame, igdbID int64) dbgen.ListDLCGamesRow {
	t.Helper()
	i := slices.IndexFunc(game.DLCs, func(g dbgen.ListDLCGamesRow) bool { return g.Game.IgdbID == igdbID })
	if i == -1 {
		t.Fatalf("DLC %d not found in DLCs", igdbID)
	}
	return game.DLCs[i]
}

// assertDLCStatus checks that the DLC row is tracked with the given status.
func assertDLCStatus(t *testing.T, dlc dbgen.ListDLCGamesRow, want GameStatus) {
	t.Helper()
	if !dlc.Tracked {
		t.Errorf("DLC %d Tracked = false, want true", dlc.Game.IgdbID)
	}
	if dlc.Status == nil || *dlc.Status != string(want) {
		t.Errorf("DLC %d Status = %v, want %q", dlc.Game.IgdbID, derefOrNil(dlc.Status), want)
	}
}

func TestTrackDLC(t *testing.T) {
	store := newTestStore(t)
	setupDLCOwnership(t, store)

	if err := store.TrackDLC(t.Context(), 1, 100, 201, time.Now().UTC()); err != nil {
		t.Fatalf("TrackDLC() failed: %v", err)
	}

	game := getLibraryGame(t, store, 100)
	assertDLCStatus(t, findDLC(t, game, 201), GameStatusBacklog)
	if findDLC(t, game, 202).Tracked {
		t.Errorf("DLC 202 Tracked = true, want false")
	}
}

func TestTrackDLCTwice(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupDLCOwnership(t, store)

	if err := store.TrackDLC(ctx, 1, 100, 201, time.Now().UTC()); err != nil {
		t.Fatalf("TrackDLC() first failed: %v", err)
	}
	if err := store.UpdateStatusAndRating(ctx, 1, 201, GameStatusCompleted, nil); err != nil {
		t.Fatalf("UpdateStatusAndRating() failed: %v", err)
	}
	if err := store.TrackDLC(ctx, 1, 100, 201, time.Now().UTC()); err != nil {
		t.Fatalf("TrackDLC() second failed: %v", err)
	}

	assertDLCStatus(t, findDLC(t, getLibraryGame(t, store, 100), 201), GameStatusCompleted)
}

func TestTrackDLCKeepsOwnership(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupDLCOwnership(t, store)

	if err := store.SetDLCOwned(ctx, 1, 100, 6, 201, true); err != nil {
		t.Fatalf("SetDLCOwned() failed: %v", err)
	}
	if err := store.TrackDLC(ctx, 1, 100, 201, time.Now().UTC()); err != nil {
		t.Fatalf("TrackDLC() failed: %v", err)
	}

	assertCopyDLCs(t, getLibraryGame(t, store, 100).CopyDLCs[6], map[int64]bool{201: true, 202: false})
}

func TestTrackDLCNotFound(t *testing.T) {
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

			err := store.TrackDLC(ctx, 1, 100, tt.dlcIGDBID, time.Now().UTC())
			if !errors.Is(err, ErrDLCNotFound) {
				t.Errorf("TrackDLC() error = %v, want %v", err, ErrDLCNotFound)
			}
		})
	}
}

func TestTrackDLCUserGameNotFound(t *testing.T) {
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

			err := store.TrackDLC(t.Context(), 1, tt.igdbID, 201, time.Now().UTC())
			if !errors.Is(err, ErrUserGameNotFound) {
				t.Errorf("TrackDLC() error = %v, want %v", err, ErrUserGameNotFound)
			}
		})
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

// setupTrackedDLCs extends setupDLCOwnership by owning DLC 201 on PC and tracking DLCs 201 and 202.
func setupTrackedDLCs(t *testing.T, store *Store) {
	t.Helper()
	ctx := t.Context()
	setupDLCOwnership(t, store)
	if err := store.SetDLCOwned(ctx, 1, 100, 6, 201, true); err != nil {
		t.Fatalf("SetDLCOwned() failed: %v", err)
	}
	for _, dlcIGDBID := range []int64{201, 202} {
		if err := store.TrackDLC(ctx, 1, 100, dlcIGDBID, time.Now().UTC()); err != nil {
			t.Fatalf("TrackDLC(%d) failed: %v", dlcIGDBID, err)
		}
	}
}

func TestListGamesTrackedDLCs(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupTrackedDLCs(t, store)
	if err := store.SetDLCOwned(ctx, 1, 100, 48, 201, true); err != nil {
		t.Fatalf("SetDLCOwned(201, PS4) failed: %v", err)
	}
	if err := store.SetDLCOwned(ctx, 1, 300, 6, 401, true); err != nil {
		t.Fatalf("SetDLCOwned(401, PC) failed: %v", err)
	}
	// Both main games are titled "Test Game"; rename 300 so their PC copies don't tie on (title, platform).
	if _, err := store.queries.UpsertGame(ctx, newGameParams(300, "Other Game")); err != nil {
		t.Fatalf("UpsertGame(300) failed: %v", err)
	}

	games, err := store.ListGames(ctx, 1)
	if err != nil {
		t.Fatalf("ListGames() failed: %v", err)
	}

	type entry struct {
		igdbID   int64
		platform string
	}
	got := make([]entry, 0, len(games))
	for _, g := range games {
		e := entry{igdbID: g.Game.IgdbID, platform: "<nil>"}
		if g.PlatformName != nil {
			e.platform = *g.PlatformName
		}
		got = append(got, e)
	}
	want := []entry{
		{201, "PC"},
		{201, "PS4"},
		{202, "<nil>"},
		{300, "PC"},
		{100, "PC"},
		{100, "PS4"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("ListGames() = %v, want %v", got, want)
	}
}

func TestGetLibraryGameOwnedDLC(t *testing.T) {
	store := newTestStore(t)
	setupTrackedDLCs(t, store)

	game := getLibraryGame(t, store, 201)
	if game.Parent == nil || game.Parent.IgdbID != 100 {
		t.Errorf("Parent = %v, want game 100", derefOrNil(game.Parent))
	}
	assertPlatformIDs(t, game.Platforms, 6)
}

func TestGetLibraryGameTrackedUnownedDLC(t *testing.T) {
	store := newTestStore(t)
	setupTrackedDLCs(t, store)

	game := getLibraryGame(t, store, 202)
	if game.Parent == nil || game.Parent.IgdbID != 100 {
		t.Errorf("Parent = %v, want game 100", derefOrNil(game.Parent))
	}
	assertPlatformIDs(t, game.Platforms)
}

func TestGetLibraryGameParentOfTrackedDLC(t *testing.T) {
	store := newTestStore(t)
	setupTrackedDLCs(t, store)

	game := getLibraryGame(t, store, 100)
	if game.Parent != nil {
		t.Errorf("Parent = %v, want nil", *game.Parent)
	}
	assertPlatformIDs(t, game.Platforms, 6, 48)
}

func TestGetLibraryGameStandaloneExpansion(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	setupTrackedDLCs(t, store)

	standalone := newDLC(501, 100, "Standalone Expansion", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)).Game
	standalone.GameType = int64(igdb.GameTypeStandaloneExpansion)
	if _, err := store.AddToCollection(ctx, standalone, newPlatformParams(130, "Switch"), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToCollection() failed: %v", err)
	}

	game := getLibraryGame(t, store, 501)
	if game.Parent != nil {
		t.Errorf("Parent = %v, want nil", *game.Parent)
	}
	assertPlatformIDs(t, game.Platforms, 130)
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
