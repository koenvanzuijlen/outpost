package database

import (
	"slices"
	"testing"
	"time"
)

// setupExport extends setupLibraryDLCs with two playthroughs of game 100, a wishlisted game 500 for user 1, and a
// library game 600 for user 2.
func setupExport(t *testing.T, store *Store) {
	t.Helper()
	ctx := t.Context()
	// User 2 adds game 300 first, so the user game IDs of user 1 differ from the game IDs.
	if _, err := store.db.ExecContext(ctx, "INSERT INTO users(id, username, password_hash) VALUES (2, 'other', '!')"); err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	if err := store.AddToLibrary(ctx, newGameParams(300, "Test Game"), nil, 2, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary(300, user 2) failed: %v", err)
	}
	setupLibraryDLCs(t, store)

	pc := int64(6)
	if _, err := store.StartPlaythrough(ctx, 1, 100, &pc, nil, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("StartPlaythrough(platform) failed: %v", err)
	}
	second := "Second"
	if _, err := store.StartPlaythrough(ctx, 1, 100, nil, &second, time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("StartPlaythrough(name) failed: %v", err)
	}

	if err := store.AddToWishlist(ctx, newGameParams(500, "Wish Game"), 1, time.Now().UTC()); err != nil {
		t.Fatalf("AddToWishlist() failed: %v", err)
	}

	pcParams := newPlatformParams(6, "PC")
	if err := store.AddToLibrary(ctx, newGameParams(600, "Other User Game"), &pcParams, 2, time.Now().UTC()); err != nil {
		t.Fatalf("AddToLibrary(600) failed: %v", err)
	}
}

// exportIGDBIDs returns the IGDB IDs of games, in order.
func exportIGDBIDs(games []ExportGame) []int64 {
	ids := make([]int64, 0, len(games))
	for _, g := range games {
		ids = append(ids, g.Game.IgdbID)
	}
	return ids
}

// findExportGame returns the game with the given IGDB ID from games, failing the test if it is missing.
func findExportGame(t *testing.T, games []ExportGame, igdbID int64) ExportGame {
	t.Helper()
	i := slices.IndexFunc(games, func(g ExportGame) bool { return g.Game.IgdbID == igdbID })
	if i == -1 {
		t.Fatalf("game %d not found in export", igdbID)
	}
	return games[i]
}

func TestExportLists(t *testing.T) {
	store := newTestStore(t)
	setupExport(t, store)

	data, err := store.Export(t.Context(), 1)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	if got, want := exportIGDBIDs(data.Library), []int64{201, 202, 100, 300}; !slices.Equal(got, want) {
		t.Errorf("Export() library IGDB IDs = %v, want %v", got, want)
	}
	if got, want := exportIGDBIDs(data.Wishlist), []int64{500}; !slices.Equal(got, want) {
		t.Errorf("Export() wishlist IGDB IDs = %v, want %v", got, want)
	}
	if slices.Contains(exportIGDBIDs(data.Library), 600) || slices.Contains(exportIGDBIDs(data.Wishlist), 600) {
		t.Errorf("Export() contains game 600 of user 2, want it missing")
	}
}

func TestExportCopies(t *testing.T) {
	store := newTestStore(t)
	setupExport(t, store)

	data, err := store.Export(t.Context(), 1)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	tests := []struct {
		igdbID int64
		want   map[int64][]int64
	}{
		{100, map[int64][]int64{6: {201}, 48: {}}},
		{300, map[int64][]int64{6: {}}},
		{201, map[int64][]int64{}},
		{202, map[int64][]int64{}},
	}
	for _, tt := range tests {
		game := findExportGame(t, data.Library, tt.igdbID)
		got := make(map[int64][]int64)
		for _, c := range game.Copies {
			dlcIDs := []int64{}
			for _, d := range c.OwnedDLC {
				dlcIDs = append(dlcIDs, d.IgdbID)
			}
			got[c.Platform.IgdbID] = dlcIDs
		}
		if len(got) != len(tt.want) {
			t.Errorf("Export() game %d copies = %v, want %v", tt.igdbID, got, tt.want)
			continue
		}
		for platformID, wantDLC := range tt.want {
			if gotDLC, ok := got[platformID]; !ok || !slices.Equal(gotDLC, wantDLC) {
				t.Errorf("Export() game %d platform %d owned DLC IDs = %v, want %v", tt.igdbID, platformID, gotDLC, wantDLC)
			}
		}
	}
}

func TestExportPlaythroughs(t *testing.T) {
	store := newTestStore(t)
	setupExport(t, store)

	data, err := store.Export(t.Context(), 1)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	game := findExportGame(t, data.Library, 100)
	if len(game.Playthroughs) != 2 {
		t.Fatalf("len(Export() game 100 playthroughs) = %d, want 2", len(game.Playthroughs))
	}

	wants := []struct {
		startedAt time.Time
		name      any
		platform  any
	}{
		{time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), "Second", nil},
		{time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), nil, int64(6)},
	}
	for i, want := range wants {
		got := game.Playthroughs[i]
		if got.Playthrough.StartedAt == nil || !got.Playthrough.StartedAt.Equal(want.startedAt) {
			t.Errorf("Export() playthrough %d StartedAt = %v, want %v", i, derefOrNil(got.Playthrough.StartedAt), want.startedAt)
		}
		if gotName := derefOrNil(got.Playthrough.Name); gotName != want.name {
			t.Errorf("Export() playthrough %d Name = %v, want %v", i, gotName, want.name)
		}
		if gotPlatform := derefOrNil(got.PlatformIgdbID); gotPlatform != want.platform {
			t.Errorf("Export() playthrough %d platform IGDB ID = %v, want %v", i, gotPlatform, want.platform)
		}
	}

	if got := findExportGame(t, data.Library, 300).Playthroughs; len(got) != 0 {
		t.Errorf("len(Export() game 300 playthroughs) = %d, want 0", len(got))
	}
}

// exportTitles returns the titles of games, in order.
func exportTitles(games []ExportGame) []string {
	titles := make([]string, 0, len(games))
	for _, g := range games {
		titles = append(titles, g.Game.Title)
	}
	return titles
}

// TestExportAll checks that each user gets the own library and wishlist, and that the users are sorted by username.
func TestExportAll(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()

	koen, err := store.CompleteSetup(ctx, "koen", "hash")
	if err != nil {
		t.Fatalf("CompleteSetup() failed: %v", err)
	}
	anna, err := store.CreateUser(ctx, "anna", "hash", UserRoleUser)
	if err != nil {
		t.Fatalf("CreateUser() failed: %v", err)
	}
	if err := store.AddToLibrary(ctx, newGameParams(100, "Koen Game"), nil, koen.ID, now); err != nil {
		t.Fatalf("AddToLibrary() failed: %v", err)
	}
	if err := store.AddToWishlist(ctx, newGameParams(200, "Anna Game"), anna.ID, now); err != nil {
		t.Fatalf("AddToWishlist() failed: %v", err)
	}

	got, err := store.ExportAll(ctx)
	if err != nil {
		t.Fatalf("ExportAll() failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(ExportAll()) = %d, want %d", len(got), 2)
	}

	wants := []struct {
		username string
		library  []string
		wishlist []string
	}{
		{"anna", []string{}, []string{"Anna Game"}},
		{"koen", []string{"Koen Game"}, []string{}},
	}
	for i, want := range wants {
		if got[i].User.Username != want.username {
			t.Errorf("ExportAll()[%d] username = %q, want %q", i, got[i].User.Username, want.username)
		}
		if titles := exportTitles(got[i].Data.Library); !slices.Equal(titles, want.library) {
			t.Errorf("ExportAll()[%d] library titles = %v, want %v", i, titles, want.library)
		}
		if titles := exportTitles(got[i].Data.Wishlist); !slices.Equal(titles, want.wishlist) {
			t.Errorf("ExportAll()[%d] wishlist titles = %v, want %v", i, titles, want.wishlist)
		}
	}
}
