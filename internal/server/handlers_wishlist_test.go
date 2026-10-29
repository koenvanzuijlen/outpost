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
	w := postForm(t, server, "/wishlist", "igdb_id=1942")

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/")
}

func TestHandleWishlistsGames(t *testing.T) {
	server := newTestServer(t)
	wishlistTestGame(t, server)
	w := getPage(t, server, "/wishlist")

	assertBodyContains(t, w, `<a href="/games/1942">`)
}

func TestHandleWishlistEmpty(t *testing.T) {
	server := newTestServer(t)
	w := getPage(t, server, "/wishlist")

	assertBodyContains(t, w, "Your wishlist is empty.")
}

func TestHandleAddToWishlistDLC(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id_dlc.json")[0]}
	w := postForm(t, server, "/wishlist", "igdb_id=13166")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	assertLocation(t, w, "")

	wishlist, err := server.store.ListWishlist(t.Context(), defaultUserID)
	if err != nil {
		t.Fatalf("ListWishlist() failed: %v", err)
	}
	if len(wishlist) != 0 {
		t.Errorf("number of wishlisted games = %d, want 0", len(wishlist))
	}
}
