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

			ownedCopies, err := server.store.ListCollection(t.Context(), defaultUserID)
			if err != nil {
				t.Fatalf("ListCollection() failed: %v", err)
			}
			wantCount := 0
			if tt.wantStatus == http.StatusSeeOther {
				wantCount = 1
			}
			if len(ownedCopies) != wantCount {
				t.Errorf("number of owned copies = %d, want %d", len(ownedCopies), wantCount)
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

	ownedCopies, err := server.store.ListCollection(t.Context(), defaultUserID)
	if err != nil {
		t.Fatalf("ListCollection() failed: %v", err)
	}
	if len(ownedCopies) != 1 {
		t.Fatalf("number of owned copies = %d, want %d", len(ownedCopies), 1)
	}
	if ownedCopies[0].Game.Title != "The Witcher 3: Wild Hunt" {
		t.Errorf("owned copy title = %q, want %q", ownedCopies[0].Game.Title, "The Witcher 3: Wild Hunt")
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
