package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleGameValidation(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"zero id", "/games/0"},
		{"negative id", "/games/-1"},
		{"string id", "/games/abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleGameNotInLibrary(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/games/999999", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleGameInLibrary(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}

	w := postForm(t, server, "/collection", "igdb_id=740&platform_id=11")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = postForm(t, server, "/games/740", "status=completed&rating=9")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("update games status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	req := httptest.NewRequest(http.MethodGet, "/games/740", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	assertBodyContains(t, w, "Halo: Combat Evolved")
	assertBodyContains(t, w, "Xbox")
	assertBodyContains(t, w, `name="status"`)
	assertBodyContains(t, w, "4.5")
}

func TestHandleGameInLibraryWishlisted(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}
	w := postForm(t, server, "/wishlist", "igdb_id=740")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	req := httptest.NewRequest(http.MethodGet, "/games/740", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	assertBodyContains(t, w, "Halo: Combat Evolved")
	assertBodyNotContains(t, w, `name="status"`)
	assertBodyNotContains(t, w, "/playthroughs")
}

func TestHandleGameActivePlaythrough(t *testing.T) {
	server := newTestServer(t)
	playthroughID := startTestPlaythrough(t, server, "Hard mode run", "11")

	req := httptest.NewRequest(http.MethodGet, "/games/740", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	assertBodyContains(t, w, "Hard mode run")
	assertBodyContains(t, w, "Platform: Xbox")
	assertBodyContains(t, w, fmt.Sprintf("/games/740/playthroughs/%d/outcome", playthroughID))
	assertBodyContains(t, w, fmt.Sprintf("/games/740/playthroughs/%d/delete", playthroughID))
}

func TestHandleGameoutcomeedPlaythrough(t *testing.T) {
	server := newTestServer(t)
	playthroughID := startTestPlaythrough(t, server, "Hard mode run", "11")

	w := postForm(t, server, fmt.Sprintf("/games/740/playthroughs/%d/outcome", playthroughID), "outcome=abandoned")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("set outcome status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	req := httptest.NewRequest(http.MethodGet, "/games/740", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	assertBodyContains(t, w, "Hard mode run")
	assertBodyContains(t, w, "Platform: Xbox")
	assertBodyContains(t, w, "Outcome: abandoned")
	assertBodyContains(t, w, `value="abandoned" aria-pressed="true"`)
	assertBodyContains(t, w, `value="playing" aria-pressed="false"`)
	assertBodyContains(t, w, fmt.Sprintf("/games/740/playthroughs/%d/outcome", playthroughID))
	assertBodyContains(t, w, fmt.Sprintf("/games/740/playthroughs/%d/delete", playthroughID))
}

func TestHandleUpdateGameValidation(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{"bad id", "/games/abc", "status=backlog&rating=10"},
		{"no status", "/games/740", "rating=10"},
		{"invalid status", "/games/740", "status=invalid&rating=10"},
		{"low rating", "/games/740", "status=backlog&rating=0"},
		{"high rating", "/games/740", "status=backlog&rating=11"},
		{"invalid rating", "/games/740", "status=backlog&rating=abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			w := postForm(t, server, tt.path, tt.body)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleUpdateGame(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}
	w := postForm(t, server, "/collection", "igdb_id=740&platform_id=11")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("collection status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = postForm(t, server, "/games/740", "status=completed&rating=9")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("update games status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	location := w.Header().Get("Location")
	if location != "/games/740" {
		t.Errorf("Location header = %q, want %q", location, "/games/740")
	}
	libraryGame, err := server.store.GetLibraryGame(t.Context(), 1, 740)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if libraryGame.UserGame.Status == nil || *libraryGame.UserGame.Status != "completed" {
		t.Errorf("status = %v, want %q", derefOrNil(libraryGame.UserGame.Status), "completed")
	}
	if libraryGame.UserGame.Rating == nil || *libraryGame.UserGame.Rating != 9 {
		t.Errorf("rating = %v, want %v", derefOrNil(libraryGame.UserGame.Rating), 9)
	}
}

func TestHandleUpdateGameNoRating(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}
	w := postForm(t, server, "/collection", "igdb_id=740&platform_id=11")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("collection status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = postForm(t, server, "/games/740", "status=completed&rating=")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("update games status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	location := w.Header().Get("Location")
	if location != "/games/740" {
		t.Errorf("Location header = %q, want %q", location, "/games/740")
	}
	libraryGame, err := server.store.GetLibraryGame(t.Context(), 1, 740)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if libraryGame.UserGame.Status == nil || *libraryGame.UserGame.Status != "completed" {
		t.Errorf("status = %v, want %q", derefOrNil(libraryGame.UserGame.Status), "completed")
	}
	if libraryGame.UserGame.Rating != nil {
		t.Errorf("rating = %v, want %v", derefOrNil(libraryGame.UserGame.Rating), nil)
	}
}

func TestHandleUpdateGameWishlisted(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}
	w := postForm(t, server, "/wishlist", "igdb_id=740")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("wishlist status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = postForm(t, server, "/games/740", "status=completed&rating=9")
	if w.Code != http.StatusNotFound {
		t.Fatalf("update games status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
