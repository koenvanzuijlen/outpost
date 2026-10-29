package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

func TestHandleAddToLibraryValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"missing body", ""},
		{"invalid id", "igdb_id=abc&platform_id=6"},
		{"negative id", "igdb_id=-1&platform_id=6"},
		{"invalid platform", "igdb_id=1942&platform_id=abc"},
		{"negative platform", "igdb_id=1942&platform_id=-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			w := postForm(t, server, "/library", tt.body)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleAddToLibrary(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		fake          *fakeIGDB
		wantStatus    int
		wantLocation  string
		wantPlatforms []int64
	}{
		{"owned", "igdb_id=1942&platform_id=6", &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}, http.StatusSeeOther, "/", []int64{6}},
		{"not owned", "igdb_id=1942&platform_id=", &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}, http.StatusSeeOther, "/", nil},
		{"no platform field", "igdb_id=1942", &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}, http.StatusSeeOther, "/", nil},
		{"dlc game type", "igdb_id=13166&platform_id=6", &fakeIGDB{game: loadIGDBGames(t, "game_by_id_dlc.json")[0]}, http.StatusBadRequest, "", nil},
		{"dlc game type not owned", "igdb_id=13166", &fakeIGDB{game: loadIGDBGames(t, "game_by_id_dlc.json")[0]}, http.StatusBadRequest, "", nil},
		{"game not on platform", "igdb_id=1942&platform_id=999", &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}, http.StatusBadRequest, "", nil},
		{"not found", "igdb_id=1942&platform_id=6", &fakeIGDB{gameErr: igdb.ErrGameNotFound}, http.StatusNotFound, "", nil},
		{"igdb error", "igdb_id=1942&platform_id=6", &fakeIGDB{gameErr: errors.New("IGDB failure")}, http.StatusInternalServerError, "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			server.igdbClient = tt.fake
			w := postForm(t, server, "/library", tt.body)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			assertLocation(t, w, tt.wantLocation)

			games, err := server.store.ListLibrary(t.Context(), defaultUserID)
			if err != nil {
				t.Fatalf("ListLibrary() failed: %v", err)
			}
			if tt.wantStatus != http.StatusSeeOther {
				if len(games) != 0 {
					t.Errorf("number of games = %d, want 0", len(games))
				}
				return
			}

			details := getTestGame(t, server)
			if len(details.OwnedPlatforms) != len(tt.wantPlatforms) {
				t.Fatalf("len(Platforms) = %d, want %d", len(details.OwnedPlatforms), len(tt.wantPlatforms))
			}
			for i, want := range tt.wantPlatforms {
				if got := details.OwnedPlatforms[i].IgdbID; got != want {
					t.Errorf("Platforms[%d].IgdbID = %d, want %d", i, got, want)
				}
			}
		})
	}
}

func TestHandleAddToLibraryTwice(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}

	for i := range 2 {
		if w := postForm(t, server, "/library", "igdb_id=1942&platform_id=6"); w.Code != http.StatusSeeOther {
			t.Errorf("call %d status = %d, want %d", i+1, w.Code, http.StatusSeeOther)
		}
	}

	games, err := server.store.ListLibrary(t.Context(), defaultUserID)
	if err != nil {
		t.Fatalf("ListLibrary() failed: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("number of games = %d, want %d", len(games), 1)
	}
	if games[0].Game.Title != "The Witcher 3: Wild Hunt" {
		t.Errorf("game title = %q, want %q", games[0].Game.Title, "The Witcher 3: Wild Hunt")
	}
}

func TestHandleAddToLibraryDatabaseFailure(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}

	// Simulate a database failure by closing the database connection.
	_ = server.store.Close()

	w := postForm(t, server, "/library", "igdb_id=1942&platform_id=6")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	assertLocation(t, w, "")
}

func TestHandleHomeRendersLibraryAndWishlist(t *testing.T) {
	searchResponse := loadIGDBGames(t, "games_search.json")
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: searchResponse[0]}
	w := postForm(t, server, "/wishlist", "igdb_id=194662")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("wishlist status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	server.igdbClient = &fakeIGDB{game: searchResponse[1]}
	w = postForm(t, server, "/library", "igdb_id=1942&platform_id=6")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("add to library status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = getPage(t, server, "/")
	assertBodyContains(t, w, `<a href="/games/194662">`)
	assertBodyContains(t, w, `<a href="/games/1942">`)
}

func TestHandleHomeShowsPlayingGames(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)
	if w := postForm(t, server, "/games/1942", "status=playing"); w.Code != http.StatusSeeOther {
		t.Fatalf("update game status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "games_search.json")[0]}
	if w := postForm(t, server, "/library", "igdb_id=194662&platform_id=6"); w.Code != http.StatusSeeOther {
		t.Fatalf("add to library status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w := getPage(t, server, "/")
	body := collapseSpace(w.Body.String())
	_, rest, ok := strings.Cut(body, "<h2>Playing</h2>")
	if !ok {
		t.Fatalf("body missing playing section:\n%s", body)
	}
	playing, _, ok := strings.Cut(rest, "<h2>Backlog")
	if !ok {
		t.Fatalf("body missing backlog section:\n%s", body)
	}

	if !strings.Contains(playing, `<a href="/games/1942">`) {
		t.Errorf("playing section missing playing game 1942:\n%s", playing)
	}
	if strings.Contains(playing, `<a href="/games/194662">`) {
		t.Errorf("playing section contains backlog game 194662:\n%s", playing)
	}
}

func TestHandleHomeLinks(t *testing.T) {
	server := newTestServer(t)
	w := getPage(t, server, "/")

	// html/template only escapes action output, so the literal & in the template stays unescaped.
	assertBodyContains(t, w, `href="/library?status=playing&sort=started"`)
	assertBodyContains(t, w, `href="/library?status=backlog"`)
	assertBodyContains(t, w, `href="/wishlist"`)
}

func TestHandleHomeRendersUnownedDLC(t *testing.T) {
	server := newTestServer(t)
	addTestGameWithDLCs(t, server)
	if w := postForm(t, server, testAddDLCPath, ""); w.Code != http.StatusSeeOther {
		t.Fatalf("add dlc to library status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w := getPage(t, server, "/")
	assertBodyContains(t, w, `<a href="/games/12503">`)
	assertBodyContains(t, w, "not owned")
}

// addBacklogAndCompletedGames adds The Witcher IV (194662) with status backlog and The Witcher 3 (1942) with status
// completed to the library.
func addBacklogAndCompletedGames(t *testing.T, server *Server) {
	t.Helper()
	addTestGame(t, server)
	if w := postForm(t, server, "/games/1942", "status=completed"); w.Code != http.StatusSeeOther {
		t.Fatalf("update game status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "games_search.json")[0]}
	if w := postForm(t, server, "/library", "igdb_id=194662&platform_id=6"); w.Code != http.StatusSeeOther {
		t.Fatalf("add to library status = %d, want %d", w.Code, http.StatusSeeOther)
	}
}

func TestHandleLibraryFiltersByStatus(t *testing.T) {
	server := newTestServer(t)
	addBacklogAndCompletedGames(t, server)

	w := getPage(t, server, "/library?status=backlog")
	assertBodyContains(t, w, `<a href="/games/194662">`)
	assertBodyNotContains(t, w, `<a href="/games/1942">`)
}

func TestHandleLibraryInvalidQuery(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"invalid sort", "/library?sort=bogus"},
		{"invalid status", "/library?status=bogus"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleLibrarySelectedOptions(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantSelected []string
		wantNot      []string
	}{
		{
			"defaults",
			"/library",
			[]string{`<option value="title" selected>`},
			[]string{`<option value="backlog" selected>`, `<option value="added" selected>`},
		},
		{
			"status and sort",
			"/library?status=backlog&sort=added",
			[]string{`<option value="backlog" selected>`, `<option value="added" selected>`},
			[]string{`<option value="completed" selected>`, `<option value="title" selected>`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			w := getPage(t, server, tt.path)

			for _, want := range tt.wantSelected {
				assertBodyContains(t, w, want)
			}
			for _, unwanted := range tt.wantNot {
				assertBodyNotContains(t, w, unwanted)
			}
		})
	}
}

func TestHandleRemoveUserGame(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)

	w := postForm(t, server, "/games/1942/delete", "")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/")

	w = getTestGamePage(t, server)
	assertBodyContains(t, w, "Add to library")
}

func TestHandleRemoveUserGameTwice(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)

	for i := range 2 {
		w := postForm(t, server, "/games/1942/delete", "")
		if w.Code != http.StatusSeeOther {
			t.Errorf("call %d status = %d, want %d", i+1, w.Code, http.StatusSeeOther)
		}
	}
}

func TestHandleRemoveUserGameNonNumericID(t *testing.T) {
	server := newTestServer(t)

	w := postForm(t, server, "/games/abc/delete", "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGamePageRemoveLabel(t *testing.T) {
	labels := []string{"Remove from library", "Remove from wishlist"}
	tests := []struct {
		name   string
		setup  func(t *testing.T, server *Server)
		igdbID int64
		want   string
	}{
		{"library game", func(t *testing.T, server *Server) { addTestGame(t, server) }, 1942, "Remove from library"},
		{"library dlc", func(t *testing.T, server *Server) {
			addTestGameWithDLCs(t, server)
			if w := postForm(t, server, testAddDLCPath, ""); w.Code != http.StatusSeeOther {
				t.Fatalf("add dlc to library status = %d, want %d", w.Code, http.StatusSeeOther)
			}
		}, 12503, "Remove from library"},
		{"wishlisted game", func(t *testing.T, server *Server) { wishlistTestGame(t, server) }, 1942, "Remove from wishlist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			tt.setup(t, server)

			w := getGamePage(t, server, tt.igdbID)
			for _, label := range labels {
				if label == tt.want {
					assertBodyContains(t, w, label)
				} else {
					assertBodyNotContains(t, w, label)
				}
			}
		})
	}
}
