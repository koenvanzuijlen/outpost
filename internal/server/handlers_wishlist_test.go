package server

import (
	"net/http"
	"testing"
)

func TestHandleAddToWishlistValidation(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"invalid id", "/games/abc/wishlist"},
		{"negative id", "/games/-1/wishlist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			w := postForm(t, server, tt.path, "")

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleAddToWishlist(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}
	w := postForm(t, server, "/games/1942/wishlist", "")

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/games/1942")
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

func TestHandleRemoveFromWishlist(t *testing.T) {
	server := newTestServer(t)
	wishlistTestGame(t, server)

	w := postForm(t, server, "/games/1942/wishlist/remove", "")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/")

	w = getPage(t, server, "/wishlist")
	assertBodyNotContains(t, w, `<a href="/games/1942">`)
}

func TestHandleRemoveFromWishlistKeepsLibraryGame(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)

	w := postForm(t, server, "/games/1942/wishlist/remove", "")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = getPage(t, server, "/library")
	assertBodyContains(t, w, `<a href="/games/1942">`)
}

func TestHandleAddToWishlistDLC(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id_dlc.json")[0]}
	w := postForm(t, server, "/games/13166/wishlist", "")

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
