package database

import (
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// TestCreateGameList checks that a new list stores its name, its description and its owner.
func TestCreateGameList(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	list, err := store.CreateGameList(ctx, 1, GameListDetails{Name: "RPGs", Description: new("Long games")})
	if err != nil {
		t.Fatalf("CreateGameList() error = %v", err)
	}
	if list.ID <= 0 {
		t.Errorf("CreateGameList() ID = %d, want > 0", list.ID)
	}
	if list.UserID != 1 {
		t.Errorf("CreateGameList() UserID = %d, want %d", list.UserID, 1)
	}
	if list.Name != "RPGs" {
		t.Errorf("CreateGameList() Name = %q, want %q", list.Name, "RPGs")
	}
	if got := derefOrNil(list.Description); got != "Long games" {
		t.Errorf("CreateGameList() Description = %v, want %q", got, "Long games")
	}

	got, err := store.GetGameList(ctx, 1, list.ID)
	if err != nil {
		t.Fatalf("GetGameList() error = %v", err)
	}
	if got.Name != "RPGs" {
		t.Errorf("GetGameList() Name = %q, want %q", got.Name, "RPGs")
	}
	if desc := derefOrNil(got.Description); desc != "Long games" {
		t.Errorf("GetGameList() Description = %v, want %q", desc, "Long games")
	}
}

// TestCreateGameListNameTaken checks that the names of the lists of one user are unique without case, and that other
// users can use the same name.
func TestCreateGameListNameTaken(t *testing.T) {
	tests := []struct {
		name    string
		userID  int64
		newName string
		wantErr error
	}{
		{name: "same name", userID: 1, newName: "RPGs", wantErr: ErrGameListNameTaken},
		{name: "other case", userID: 1, newName: "rpgs", wantErr: ErrGameListNameTaken},
		{name: "other user", userID: 2, newName: "RPGs", wantErr: nil},
		{name: "other name", userID: 1, newName: "Co-op", wantErr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTwoUserStore(t)
			createGameList(t, store, 1, "RPGs")

			_, err := store.CreateGameList(t.Context(), tt.userID, GameListDetails{Name: tt.newName})
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("CreateGameList(%q) error = %v, want %v", tt.newName, err, tt.wantErr)
			}
		})
	}
}

// TestGetGameListNotFound checks that a list of another user and an unknown ID both give ErrGameListNotFound.
func TestGetGameListNotFound(t *testing.T) {
	store := newTwoUserStore(t)
	list := createGameList(t, store, 1, "RPGs")
	if list.ID != 1 {
		t.Fatalf("createGameList() ID = %d, want %d", list.ID, 1)
	}

	tests := []struct {
		name    string
		userID  int64
		listID  int64
		wantErr error
	}{
		{name: "other user", userID: 2, listID: 1, wantErr: ErrGameListNotFound},
		{name: "unknown ID", userID: 1, listID: 99, wantErr: ErrGameListNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.GetGameList(t.Context(), tt.userID, tt.listID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("GetGameList(%d, %d) error = %v, want %v", tt.userID, tt.listID, err, tt.wantErr)
			}
		})
	}
}

// TestUpdateGameList checks which updates succeed, and that a failed update keeps the stored values.
func TestUpdateGameList(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		listID   int64
		details  GameListDetails
		wantErr  error
		wantName string
		wantDesc any
	}{
		{
			name:     "rename",
			userID:   1,
			listID:   1,
			details:  GameListDetails{Name: "Role-playing", Description: new("Long games")},
			wantErr:  nil,
			wantName: "Role-playing",
			wantDesc: "Long games",
		},
		{
			name:     "change case",
			userID:   1,
			listID:   1,
			details:  GameListDetails{Name: "rpgs"},
			wantErr:  nil,
			wantName: "rpgs",
			wantDesc: nil,
		},
		{
			name:     "name taken",
			userID:   1,
			listID:   1,
			details:  GameListDetails{Name: "co-op"},
			wantErr:  ErrGameListNameTaken,
			wantName: "RPGs",
			wantDesc: nil,
		},
		{
			name:     "other user",
			userID:   2,
			listID:   1,
			details:  GameListDetails{Name: "Mine"},
			wantErr:  ErrGameListNotFound,
			wantName: "RPGs",
			wantDesc: nil,
		},
		{
			name:     "unknown ID",
			userID:   1,
			listID:   99,
			details:  GameListDetails{Name: "Mine"},
			wantErr:  ErrGameListNotFound,
			wantName: "RPGs",
			wantDesc: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTwoUserStore(t)
			ctx := t.Context()
			if list := createGameList(t, store, 1, "RPGs"); list.ID != 1 {
				t.Fatalf("createGameList(RPGs) ID = %d, want %d", list.ID, 1)
			}
			if list := createGameList(t, store, 1, "Co-op"); list.ID != 2 {
				t.Fatalf("createGameList(Co-op) ID = %d, want %d", list.ID, 2)
			}

			err := store.UpdateGameList(ctx, tt.userID, tt.listID, tt.details)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("UpdateGameList() error = %v, want %v", err, tt.wantErr)
			}

			got, err := store.GetGameList(ctx, 1, 1)
			if err != nil {
				t.Fatalf("GetGameList() error = %v", err)
			}
			if got.Name != tt.wantName {
				t.Errorf("GetGameList() Name = %q, want %q", got.Name, tt.wantName)
			}
			if desc := derefOrNil(got.Description); desc != tt.wantDesc {
				t.Errorf("GetGameList() Description = %v, want %v", desc, tt.wantDesc)
			}
		})
	}
}

// TestDeleteGameList checks that a delete removes the list, that a repeated delete counts as success, and that a
// delete never removes a list of another user.
func TestDeleteGameList(t *testing.T) {
	store := newTwoUserStore(t)
	ctx := t.Context()
	rpgs := createGameList(t, store, 1, "RPGs")
	mine := createGameList(t, store, 2, "Mine")

	if err := store.DeleteGameList(ctx, 1, rpgs.ID); err != nil {
		t.Fatalf("DeleteGameList(rpgs) error = %v", err)
	}
	if _, err := store.GetGameList(ctx, 1, rpgs.ID); !errors.Is(err, ErrGameListNotFound) {
		t.Errorf("GetGameList(rpgs) error = %v, want %v", err, ErrGameListNotFound)
	}

	if err := store.DeleteGameList(ctx, 1, rpgs.ID); err != nil {
		t.Errorf("DeleteGameList(rpgs) again error = %v, want nil", err)
	}

	if err := store.DeleteGameList(ctx, 1, mine.ID); err != nil {
		t.Errorf("DeleteGameList(mine) error = %v, want nil", err)
	}
	if _, err := store.GetGameList(ctx, 2, mine.ID); err != nil {
		t.Errorf("GetGameList(mine) error = %v, want nil", err)
	}
}

// gameListT1 is the first date of a list item in the tests. Later dates add hours to it.
var gameListT1 = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

// newCoverGameParams returns UpsertGame parameters for a game with the given IGDB ID and title, and a cover URL that
// contains the IGDB ID.
func newCoverGameParams(igdbID int64, title string) dbgen.UpsertGameParams {
	p := newGameParams(igdbID, title)
	p.CoverUrl = new(fmt.Sprintf("https://example.com/%d.jpg", igdbID))
	return p
}

// addToGameList adds game to the list of the user at addedAt. It fails the test if the game cannot be added.
func addToGameList(
	t *testing.T,
	store *Store,
	userID, listID int64,
	game dbgen.UpsertGameParams,
	addedAt time.Time,
) {
	t.Helper()
	if err := store.AddToGameList(t.Context(), userID, listID, game, addedAt); err != nil {
		t.Fatalf("AddToGameList(%d, %d) error = %v", listID, game.IgdbID, err)
	}
}

// gameListIGDBIDs returns the IGDB IDs of the games in the list of the user, in the order of ListGameListEntries. It
// fails the test if the entries cannot be read.
func gameListIGDBIDs(t *testing.T, store *Store, userID, listID int64) []int64 {
	t.Helper()
	entries, err := store.ListGameListEntries(t.Context(), userID, listID)
	if err != nil {
		t.Fatalf("ListGameListEntries() error = %v", err)
	}
	ids := []int64{}
	for _, e := range entries {
		ids = append(ids, e.Game.IgdbID)
	}
	return ids
}

// TestAddToGameList checks that an add stores the game in the list, and that a repeated add keeps the first date.
func TestAddToGameList(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	list := createGameList(t, store, 1, "RPGs")

	addToGameList(t, store, 1, list.ID, newGameParams(100, "Halo"), gameListT1)
	addToGameList(t, store, 1, list.ID, newGameParams(100, "Halo"), gameListT1.Add(time.Hour))

	entries, err := store.ListGameListEntries(ctx, 1, list.ID)
	if err != nil {
		t.Fatalf("ListGameListEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(ListGameListEntries()) = %d, want 1", len(entries))
	}
	got := entries[0]
	if got.Game.IgdbID != 100 {
		t.Errorf("ListGameListEntries() IgdbID = %d, want %d", got.Game.IgdbID, 100)
	}
	if got.Game.Title != "Halo" {
		t.Errorf("ListGameListEntries() Title = %q, want %q", got.Game.Title, "Halo")
	}
	if !got.AddedAt.Equal(gameListT1) {
		t.Errorf("ListGameListEntries() AddedAt = %v, want %v", got.AddedAt, gameListT1)
	}
}

// TestAddToGameListErrors checks that an add to a list that the user does not have fails, and leaves the catalog
// unchanged.
func TestAddToGameListErrors(t *testing.T) {
	tests := []struct {
		name    string
		userID  int64
		listID  int64
		wantErr error
	}{
		{name: "other user", userID: 2, listID: 1, wantErr: ErrGameListNotFound},
		{name: "unknown list", userID: 1, listID: 99, wantErr: ErrGameListNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTwoUserStore(t)
			ctx := t.Context()
			if list := createGameList(t, store, 1, "RPGs"); list.ID != 1 {
				t.Fatalf("createGameList() ID = %d, want %d", list.ID, 1)
			}

			err := store.AddToGameList(ctx, tt.userID, tt.listID, newGameParams(100, "Halo"), gameListT1)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("AddToGameList() error = %v, want %v", err, tt.wantErr)
			}
			if _, err := store.GetGame(ctx, 1, 100); !errors.Is(err, ErrGameNotInCatalog) {
				t.Errorf("GetGame(100) error = %v, want %v", err, ErrGameNotInCatalog)
			}
		})
	}
}

// TestListGameListEntriesStatus checks that the status and the rating come only from the library of the owner of the
// list, and stay nil for a wishlisted game and a game outside the library.
func TestListGameListEntriesStatus(t *testing.T) {
	store := newTwoUserStore(t)
	ctx := t.Context()
	list := createGameList(t, store, 1, "Mixed")

	if err := store.AddToLibrary(ctx, newGameParams(100, "A"), new(newPlatformParams(11, "Xbox")), 1, gameListT1); err != nil {
		t.Fatalf("AddToLibrary(100) error = %v", err)
	}
	details := GameDetails{Status: GameStatusCompleted, Rating: new(int64(8))}
	if err := store.UpdateGameDetails(ctx, 1, 100, details); err != nil {
		t.Fatalf("UpdateGameDetails(100) error = %v", err)
	}
	if err := store.AddToWishlist(ctx, newGameParams(200, "B"), 1, gameListT1); err != nil {
		t.Fatalf("AddToWishlist(200) error = %v", err)
	}
	if err := store.AddToLibrary(ctx, newGameParams(400, "D"), nil, 2, gameListT1); err != nil {
		t.Fatalf("AddToLibrary(400) error = %v", err)
	}
	for _, game := range []dbgen.UpsertGameParams{
		newGameParams(100, "A"),
		newGameParams(200, "B"),
		newGameParams(300, "C"),
		newGameParams(400, "D"),
	} {
		addToGameList(t, store, 1, list.ID, game, gameListT1)
	}

	entries, err := store.ListGameListEntries(ctx, 1, list.ID)
	if err != nil {
		t.Fatalf("ListGameListEntries() error = %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("len(ListGameListEntries()) = %d, want 4", len(entries))
	}

	tests := []struct {
		igdbID     int64
		wantStatus any
		wantRating any
	}{
		{igdbID: 100, wantStatus: "completed", wantRating: int64(8)},
		{igdbID: 200, wantStatus: nil, wantRating: nil},
		{igdbID: 300, wantStatus: nil, wantRating: nil},
		{igdbID: 400, wantStatus: nil, wantRating: nil},
	}
	for i, tt := range tests {
		got := entries[i]
		if got.Game.IgdbID != tt.igdbID {
			t.Errorf("entry %d IgdbID = %d, want %d", i, got.Game.IgdbID, tt.igdbID)
			continue
		}
		if status := derefOrNil(got.Status); status != tt.wantStatus {
			t.Errorf("game %d Status = %v, want %v", tt.igdbID, status, tt.wantStatus)
		}
		if rating := derefOrNil(got.Rating); rating != tt.wantRating {
			t.Errorf("game %d Rating = %v, want %v", tt.igdbID, rating, tt.wantRating)
		}
	}
}

// TestListGameListEntriesOtherUser checks that another user reads no games from a list of the user.
func TestListGameListEntriesOtherUser(t *testing.T) {
	store := newTwoUserStore(t)
	list := createGameList(t, store, 1, "RPGs")
	addToGameList(t, store, 1, list.ID, newGameParams(100, "Halo"), gameListT1)

	entries, err := store.ListGameListEntries(t.Context(), 2, list.ID)
	if err != nil {
		t.Fatalf("ListGameListEntries() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len(ListGameListEntries()) = %d, want 0", len(entries))
	}
}

// TestRemoveFromGameList checks that a remove takes one game out of the list, that a repeated remove and an unknown
// game count as success, and that another user cannot remove a game.
func TestRemoveFromGameList(t *testing.T) {
	store := newTwoUserStore(t)
	ctx := t.Context()
	list := createGameList(t, store, 1, "RPGs")
	addToGameList(t, store, 1, list.ID, newGameParams(100, "A"), gameListT1)
	addToGameList(t, store, 1, list.ID, newGameParams(200, "B"), gameListT1)

	steps := []struct {
		name   string
		userID int64
		igdbID int64
		want   []int64
	}{
		{name: "other user", userID: 2, igdbID: 100, want: []int64{100, 200}},
		{name: "remove", userID: 1, igdbID: 100, want: []int64{200}},
		{name: "again", userID: 1, igdbID: 100, want: []int64{200}},
		{name: "unknown game", userID: 1, igdbID: 999, want: []int64{200}},
	}
	for _, step := range steps {
		if err := store.RemoveFromGameList(ctx, step.userID, list.ID, step.igdbID); err != nil {
			t.Errorf("%s: RemoveFromGameList() error = %v, want nil", step.name, err)
		}
		if got := gameListIGDBIDs(t, store, 1, list.ID); !slices.Equal(got, step.want) {
			t.Errorf("%s: games in list = %v, want %v", step.name, got, step.want)
		}
	}
}

// TestRemoveUserGameKeepsGameListItem checks that a game removed from the library stays in its lists, without a
// status.
func TestRemoveUserGameKeepsGameListItem(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	addOwnedGame(t, store, 100)
	list := createGameList(t, store, 1, "RPGs")
	addToGameList(t, store, 1, list.ID, newGameParams(100, "Test Game"), gameListT1)

	if err := store.RemoveUserGame(ctx, 1, 100, false); err != nil {
		t.Fatalf("RemoveUserGame() error = %v", err)
	}

	entries, err := store.ListGameListEntries(ctx, 1, list.ID)
	if err != nil {
		t.Fatalf("ListGameListEntries() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Game.IgdbID != 100 {
		t.Fatalf("ListGameListEntries() = %v, want game 100", entries)
	}
	if status := derefOrNil(entries[0].Status); status != nil {
		t.Errorf("ListGameListEntries() Status = %v, want nil", status)
	}
}

// TestListGameLists checks the order of the lists, the game count, and the covers: the newest first, at most 4, and
// only games with a cover. A list of another user does not appear.
func TestListGameLists(t *testing.T) {
	store := newTwoUserStore(t)
	rpgs := createGameList(t, store, 1, "RPGs")
	createGameList(t, store, 1, "co-op")
	createGameList(t, store, 1, "Arcade")
	mine := createGameList(t, store, 2, "Mine")
	addToGameList(t, store, 2, mine.ID, newCoverGameParams(900, "Other"), gameListT1)

	// Games 101 to 106, added 1 to 6 hours after gameListT1. Game 103 has no cover, so 5 games have one.
	for i := int64(1); i <= 6; i++ {
		igdbID := 100 + i
		game := newCoverGameParams(igdbID, fmt.Sprintf("Game %d", igdbID))
		if igdbID == 103 {
			game.CoverUrl = nil
		}
		addToGameList(t, store, 1, rpgs.ID, game, gameListT1.Add(time.Duration(i)*time.Hour))
	}

	got, err := store.ListGameLists(t.Context(), 1)
	if err != nil {
		t.Fatalf("ListGameLists() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(ListGameLists()) = %d, want 3", len(got))
	}

	tests := []struct {
		name       string
		gameCount  int
		wantCovers []string
	}{
		{name: "Arcade", gameCount: 0, wantCovers: []string{}},
		{name: "co-op", gameCount: 0, wantCovers: []string{}},
		{
			name:      "RPGs",
			gameCount: 6,
			wantCovers: []string{
				"https://example.com/106.jpg",
				"https://example.com/105.jpg",
				"https://example.com/104.jpg",
				"https://example.com/102.jpg",
			},
		},
	}
	for i, tt := range tests {
		summary := got[i]
		if summary.List.Name != tt.name {
			t.Errorf("ListGameLists()[%d] Name = %q, want %q", i, summary.List.Name, tt.name)
			continue
		}
		if summary.GameCount != tt.gameCount {
			t.Errorf("%s GameCount = %d, want %d", tt.name, summary.GameCount, tt.gameCount)
		}
		if !slices.Equal(summary.Covers, tt.wantCovers) {
			t.Errorf("%s Covers = %v, want %v", tt.name, summary.Covers, tt.wantCovers)
		}
	}
}

// TestListGameListsForGame checks that every list of the user appears, marked when it holds the game, and that the
// lists of another user do not appear.
func TestListGameListsForGame(t *testing.T) {
	store := newTwoUserStore(t)
	a := createGameList(t, store, 1, "A")
	createGameList(t, store, 1, "B")
	c := createGameList(t, store, 2, "C")
	addToGameList(t, store, 1, a.ID, newGameParams(100, "Halo"), gameListT1)
	addToGameList(t, store, 2, c.ID, newGameParams(100, "Halo"), gameListT1)
	gameID := getGame(t, store, 100).Game.ID

	got, err := store.ListGameListsForGame(t.Context(), 1, gameID)
	if err != nil {
		t.Fatalf("ListGameListsForGame() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(ListGameListsForGame()) = %d, want 2", len(got))
	}

	tests := []struct {
		wantName    string
		wantHasGame bool
	}{
		{wantName: "A", wantHasGame: true},
		{wantName: "B", wantHasGame: false},
	}
	for i, tt := range tests {
		if got[i].GameList.Name != tt.wantName {
			t.Errorf("ListGameListsForGame()[%d] Name = %q, want %q", i, got[i].GameList.Name, tt.wantName)
		}
		if got[i].HasGame != tt.wantHasGame {
			t.Errorf("ListGameListsForGame()[%d] HasGame = %v, want %v", i, got[i].HasGame, tt.wantHasGame)
		}
	}
}
