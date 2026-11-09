package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// addTwoWishlistGames adds The Witcher 3 (1942, released 2015) and The Witcher IV (194662, released 2029) to the
// wishlist.
func addTwoWishlistGames(t *testing.T, server *Server) {
	t.Helper()
	wishlistTestGame(t, server)

	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "games_search.json")[0]}
	if w := postForm(t, server, "/games/194662/wishlist", ""); w.Code != http.StatusSeeOther {
		t.Fatalf("wishlist status = %d, want %d", w.Code, http.StatusSeeOther)
	}
}

func TestHandleWishlistSortOrder(t *testing.T) {
	server := newTestServer(t)
	addTwoWishlistGames(t, server)

	tests := []struct {
		path       string
		wantFirst  string
		wantSecond string
	}{
		{"/wishlist", "1942", "194662"},
		{"/wishlist?order=desc", "194662", "1942"},
		{"/wishlist?sort=title&order=desc", "194662", "1942"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			w := getPage(t, server, tt.path)
			body := w.Body.String()
			first := strings.Index(body, `href="/games/`+tt.wantFirst+`"`)
			second := strings.Index(body, `href="/games/`+tt.wantSecond+`"`)
			if first == -1 || second == -1 || first > second {
				t.Errorf("index of %s = %d, want before %s at %d", tt.wantFirst, first, tt.wantSecond, second)
			}
		})
	}
}

func TestHandleWishlistInvalidQuery(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"invalid sort", "/wishlist?sort=bogus"},
		{"invalid order", "/wishlist?order=bogus"},
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

func TestHandleWishlistSelectedOptions(t *testing.T) {
	tests := []struct {
		path         string
		wantSelected string
		wantToggle   string
	}{
		{"/wishlist", `<option value="released" selected>`, `href="/wishlist?order=desc&amp;sort=released"`},
		{"/wishlist?sort=title", `<option value="title" selected>`, `href="/wishlist?order=desc&amp;sort=title"`},
		{"/wishlist?sort=added", `<option value="added" selected>`, `href="/wishlist?order=asc&amp;sort=added"`},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			server := newTestServer(t)
			w := getPage(t, server, tt.path)

			assertBodyContains(t, w, tt.wantSelected)
			assertBodyContains(t, w, tt.wantToggle)
		})
	}
}

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

	assertBodyContains(t, w, `href="/games/1942"`)
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
	assertBodyNotContains(t, w, `href="/games/1942"`)
}

func TestHandleRemoveFromWishlistKeepsLibraryGame(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)

	w := postForm(t, server, "/games/1942/wishlist/remove", "")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = getPage(t, server, "/library")
	assertBodyContains(t, w, `href="/games/1942"`)
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
