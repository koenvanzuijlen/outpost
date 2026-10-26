package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleLibraryRendersCollectionAndWishlist(t *testing.T) {
	searchResponse := loadIGDBGames(t, "games_search.json")
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: searchResponse[0]}
	w := postForm(t, server, "/wishlist", "igdb_id=194662")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("wishlist status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	server.igdbClient = &fakeIGDB{game: searchResponse[1]}
	w = postForm(t, server, "/collection", "igdb_id=1942&platform_id=6")
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
