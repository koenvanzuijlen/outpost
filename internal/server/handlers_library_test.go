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

func TestHandleRemoveFromLibrary(t *testing.T) {
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

func TestHandleRemoveFromLibraryTwice(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)

	for i := range 2 {
		w := postForm(t, server, "/games/1942/delete", "")
		if w.Code != http.StatusSeeOther {
			t.Errorf("call %d status = %d, want %d", i+1, w.Code, http.StatusSeeOther)
		}
	}
}

func TestHandleRemoveFromLibraryNonNumericID(t *testing.T) {
	server := newTestServer(t)

	w := postForm(t, server, "/games/abc/delete", "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGamePageRemoveLabel(t *testing.T) {
	labels := []string{"Remove from library", "Stop tracking", "Remove from wishlist"}
	tests := []struct {
		name   string
		setup  func(t *testing.T, server *Server)
		igdbID int64
		want   string
	}{
		{"owned game", func(t *testing.T, server *Server) { addTestGame(t, server) }, 1942, "Remove from library"},
		{"tracked dlc", func(t *testing.T, server *Server) {
			addTestGameWithDLCs(t, server)
			if w := postForm(t, server, testTrackDLCPath, ""); w.Code != http.StatusSeeOther {
				t.Fatalf("track status = %d, want %d", w.Code, http.StatusSeeOther)
			}
		}, 12503, "Stop tracking"},
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
