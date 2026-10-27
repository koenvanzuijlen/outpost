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

func TestHandleLibraryRendersTrackedUnownedDLC(t *testing.T) {
	server := newTestServer(t)
	addTestGameWithDLCs(t, server)
	if w := postForm(t, server, testTrackDLCPath, ""); w.Code != http.StatusSeeOther {
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
