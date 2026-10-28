package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
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

			details := getTestUserGame(t, server)
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

func TestHandleLibraryRendersLibraryAndWishlist(t *testing.T) {
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

	// Reuse the same server to check the index page reflects the wishlist and collection
	w = httptest.NewRecorder()
	server.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleLibraryRendersUnownedDLC(t *testing.T) {
	server := newTestServer(t)
	addTestGameWithDLCs(t, server)
	if w := postForm(t, server, testAddDLCPath, ""); w.Code != http.StatusSeeOther {
		t.Fatalf("track status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w := httptest.NewRecorder()
	server.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	assertBodyContains(t, w, `<a href="/games/12503">`)
	assertBodyContains(t, w, "not owned")
}

func TestHandleRemoveUserGame(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)

	w := postForm(t, server, "/games/1942/delete", "")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/")

	w = httptest.NewRecorder()
	server.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/games/1942", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("game page status = %d, want %d", w.Code, http.StatusNotFound)
	}
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
