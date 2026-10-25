package server

import (
	"net/http"
	"testing"
)

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
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}
	w := postForm(t, server, "/wishlist", "igdb_id=740")

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	location := w.Header().Get("Location")
	if location != "/" {
		t.Errorf("location header = %q, want %q", location, "/")
	}
}

func TestHandleAddToWishlistDLC(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id_dlc.json")[0]}
	w := postForm(t, server, "/wishlist", "igdb_id=13166")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	location := w.Header().Get("Location")
	if location != "" {
		t.Errorf("location header = %q, want empty", location)
	}

	wishlist, err := server.store.ListWishlist(t.Context(), defaultUserID)
	if err != nil {
		t.Fatalf("ListWishlist() failed: %v", err)
	}
	if len(wishlist) != 0 {
		t.Errorf("number of wishlisted games = %d, want 0", len(wishlist))
	}
}
