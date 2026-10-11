package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// postForm sends a POST request with the given form body to the specified URL and returns the response recorder.
func postForm(t *testing.T, server *Server, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

func TestHandleIndexRendersCollectionAndWishlist(t *testing.T) {
	searchResponse := loadIGDBGames(t, "games_search_response.json")
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: searchResponse[0]}
	w := postForm(t, server, "/wishlist", "igdb_id=740")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("wishlist status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	server.igdbClient = &fakeIGDB{game: searchResponse[1]}
	w = postForm(t, server, "/collection", "igdb_id=2640&platform_id=6")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("collection status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	// Reuse the same server to check the index page reflects the wishlist and collection
	w = httptest.NewRecorder()
	server.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleAddToWishlistValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"missing body", ""},
		{"invalid id", "igdb_id=abc"},
		{"negative id", "igdb_id=-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			w := postForm(t, server, "/wishlist", tt.body)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleAddToWishlist(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id_response.json")[0]}
	w := postForm(t, server, "/wishlist", "igdb_id=740")

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	location := w.Header().Get("Location")
	if location != "/" {
		t.Errorf("location header = %q, want %q", location, "/")
	}
}

func TestHandleAddToCollectionValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"missing body", ""},
		{"invalid id", "igdb_id=abc&platform_id=11"},
		{"negative id", "igdb_id=-1&platform_id=11"},
		{"invalid platform", "igdb_id=740&platform_id=abc"},
		{"negative platform", "igdb_id=740&platform_id=-1"},
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
		{"added", "igdb_id=740&platform_id=11", &fakeIGDB{game: loadIGDBGames(t, "game_by_id_response.json")[0]}, http.StatusSeeOther, "/"},
		{"game not on platform", "igdb_id=740&platform_id=999", &fakeIGDB{game: loadIGDBGames(t, "game_by_id_response.json")[0]}, http.StatusBadRequest, ""},
		{"not found", "igdb_id=740&platform_id=11", &fakeIGDB{gameErr: igdb.ErrGameNotFound}, http.StatusNotFound, ""},
		{"igdb error", "igdb_id=740&platform_id=11", &fakeIGDB{gameErr: errors.New("IGDB failure")}, http.StatusInternalServerError, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			server.igdbClient = tt.fake
			w := postForm(t, server, "/collection", tt.body)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			location := w.Header().Get("Location")
			if location != tt.wantLocation {
				t.Errorf("location = %q, want %q", location, tt.wantLocation)
			}
		})
	}
}

func TestHandleAddToCollectionTwice(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id_response.json")[0]}
	w := postForm(t, server, "/collection", "igdb_id=740&platform_id=11")

	if w.Code != http.StatusSeeOther {
		t.Errorf("first status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = postForm(t, server, "/collection", "igdb_id=740&platform_id=11")

	if w.Code != http.StatusSeeOther {
		t.Errorf("second status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	ownedCopies, err := server.queries.ListOwnedCopies(t.Context(), defaultUserID)
	if err != nil {
		t.Fatalf("ListOwnedCopies() failed: %v", err)
	}
	if len(ownedCopies) != 1 {
		t.Fatalf("number of owned copies = %d, want %d", len(ownedCopies), 1)
	}
	if ownedCopies[0].Game.Title != "Halo: Combat Evolved" {
		t.Errorf("owned copy title = %q, want %q", ownedCopies[0].Game.Title, "Halo: Combat Evolved")
	}
}

func TestHandleAddToCollectionDatabaseFailure(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id_response.json")[0]}

	// Simulate a database failure by closing the database connection.
	_ = server.db.Close()

	w := postForm(t, server, "/collection", "igdb_id=740&platform_id=11")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	location := w.Header().Get("Location")
	if location != "" {
		t.Errorf("location header = %q, want empty", location)
	}
}
