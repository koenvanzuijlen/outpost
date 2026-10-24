package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	if !strings.Contains(w.Body.String(), "Halo: Combat Evolved") {
		t.Errorf("body does not contain expected content: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Xbox") {
		t.Errorf("body does not contain expected content: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "name=\"status\"") {
		t.Errorf("body does not contain expected content: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "4.5") {
		t.Errorf("body does not contain expected content: %s", w.Body.String())
	}
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
	if !strings.Contains(w.Body.String(), "Halo: Combat Evolved") {
		t.Errorf("body does not contain expected content: %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "name=\"status\"") {
		t.Errorf("body should not contain update form for wishlisted game: %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "/playthroughs") {
		t.Errorf("body should not contain playthroughs for wishlisted game: %s", w.Body.String())
	}
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
	if !strings.Contains(w.Body.String(), "Hard mode run") {
		t.Errorf("body does not contain expected content: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Platform: Xbox") {
		t.Errorf("body does not contain expected content: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), fmt.Sprintf("/games/740/playthroughs/%d/outcome", playthroughID)) {
		t.Errorf("body does not contain expected content: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), fmt.Sprintf("/games/740/playthroughs/%d/delete", playthroughID)) {
		t.Errorf("body does not contain expected content: %s", w.Body.String())
	}
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
	if !strings.Contains(w.Body.String(), "Hard mode run") {
		t.Errorf("body does not contain expected content: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Platform: Xbox") {
		t.Errorf("body does not contain expected content: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Outcome: abandoned") {
		t.Errorf("body does not contain expected content: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `value="abandoned" aria-pressed="true"`) {
		t.Errorf("body does not contain expected content: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `value="playing" aria-pressed="false"`) {
		t.Errorf("body does not contain expected content: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), fmt.Sprintf("/games/740/playthroughs/%d/outcome", playthroughID)) {
		t.Errorf("body does not contain expected content: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), fmt.Sprintf("/games/740/playthroughs/%d/delete", playthroughID)) {
		t.Errorf("body does not contain expected content: %s", w.Body.String())
	}
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
