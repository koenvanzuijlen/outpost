package server

import (
	"errors"
	"net/http"
	"testing"

	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

func TestHandleAddToCollectionValidation(t *testing.T) {
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
			w := postForm(t, server, "/collection", tt.body)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleAddToCollection(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		fake         *fakeIGDB
		wantStatus   int
		wantLocation string
	}{
		{"added", "igdb_id=1942&platform_id=6", &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}, http.StatusSeeOther, "/"},
		{"dlc game type", "igdb_id=13166&platform_id=6", &fakeIGDB{game: loadIGDBGames(t, "game_by_id_dlc.json")[0]}, http.StatusBadRequest, ""},
		{"game not on platform", "igdb_id=1942&platform_id=999", &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}, http.StatusBadRequest, ""},
		{"not found", "igdb_id=1942&platform_id=6", &fakeIGDB{gameErr: igdb.ErrGameNotFound}, http.StatusNotFound, ""},
		{"igdb error", "igdb_id=1942&platform_id=6", &fakeIGDB{gameErr: errors.New("IGDB failure")}, http.StatusInternalServerError, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			server.igdbClient = tt.fake
			w := postForm(t, server, "/collection", tt.body)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			assertLocation(t, w, tt.wantLocation)

			games, err := server.store.ListGames(t.Context(), defaultUserID)
			if err != nil {
				t.Fatalf("ListGames() failed: %v", err)
			}
			wantCount := 0
			if tt.wantStatus == http.StatusSeeOther {
				wantCount = 1
			}
			if len(games) != wantCount {
				t.Errorf("number of games = %d, want %d", len(games), wantCount)
			}
		})
	}
}

func TestHandleAddToCollectionTwice(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}
	w := postForm(t, server, "/collection", "igdb_id=1942&platform_id=6")

	if w.Code != http.StatusSeeOther {
		t.Errorf("first status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = postForm(t, server, "/collection", "igdb_id=1942&platform_id=6")

	if w.Code != http.StatusSeeOther {
		t.Errorf("second status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	games, err := server.store.ListGames(t.Context(), defaultUserID)
	if err != nil {
		t.Fatalf("ListGames() failed: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("number of games = %d, want %d", len(games), 1)
	}
	if games[0].Game.Title != "The Witcher 3: Wild Hunt" {
		t.Errorf("game title = %q, want %q", games[0].Game.Title, "The Witcher 3: Wild Hunt")
	}
}

func TestHandleAddToCollectionDatabaseFailure(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}

	// Simulate a database failure by closing the database connection.
	_ = server.store.Close()

	w := postForm(t, server, "/collection", "igdb_id=1942&platform_id=6")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	assertLocation(t, w, "")
}

const (
	testRemovePCPath  = "/games/1942/platforms/6/delete"
	testRemovePS4Path = "/games/1942/platforms/48/delete"
)

// addTestGamePS4 adds The Witcher 3 on PC and PS4 (48).
func addTestGamePS4(t *testing.T, server *Server) {
	t.Helper()
	addTestGame(t, server)
	if w := postForm(t, server, "/collection", "igdb_id=1942&platform_id=48"); w.Code != http.StatusSeeOther {
		t.Fatalf("collection status = %d, want %d", w.Code, http.StatusSeeOther)
	}
}

func TestHandleRemoveFromCollection(t *testing.T) {
	server := newTestServer(t)
	addTestGamePS4(t, server)

	w := getTestGamePage(t, server)
	assertBodyContains(t, w, `action="`+testRemovePCPath+`"`)

	w = postForm(t, server, testRemovePCPath, "")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/games/1942")

	w = getTestGamePage(t, server)
	assertBodyNotContains(t, w, `action="`+testRemovePCPath+`"`)
	assertBodyContains(t, w, `action="`+testRemovePS4Path+`"`)
}

func TestHandleRemoveFromCollectionTwice(t *testing.T) {
	server := newTestServer(t)
	addTestGamePS4(t, server)

	for i := range 2 {
		w := postForm(t, server, testRemovePCPath, "")
		if w.Code != http.StatusSeeOther {
			t.Errorf("call %d status = %d, want %d", i+1, w.Code, http.StatusSeeOther)
		}
	}
}

func TestHandleRemoveFromCollectionValidation(t *testing.T) {
	tests := []struct {
		name string
		path string
		want int
	}{
		{"non-numeric game id", "/games/abc/platforms/6/delete", http.StatusBadRequest},
		{"non-numeric platform id", "/games/1942/platforms/abc/delete", http.StatusBadRequest},
		{"game not in library", "/games/1020/platforms/6/delete", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			addTestGame(t, server)

			w := postForm(t, server, tt.path, "")
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}
