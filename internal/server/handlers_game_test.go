package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/koenvanzuijlen/outpost/internal/igdb"
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
	startTestPlaythrough(t, server, "Hard mode run", "11")

	libraryGame, err := server.store.GetLibraryGame(t.Context(), 1, 740)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if len(libraryGame.Playthroughs) != 1 {
		t.Fatalf("len(Playthroughs) = %d, want %d", len(libraryGame.Playthroughs), 1)
	}
	playthrough := libraryGame.Playthroughs[0].Playthrough
	if playthrough.StartedAt == nil {
		t.Fatal("StartedAt = nil, want a date")
	}
	playthroughID := playthrough.ID

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
	assertBodyContains(t, w, fmt.Sprintf(`action="/games/740/playthroughs/%d"`, playthroughID))
	assertBodyContains(t, w, `<option value="11" selected>Xbox</option>`)
	assertBodyContains(t, w, fmt.Sprintf(`value="%s"`, playthrough.StartedAt.Format("2006-01-02")))
	assertBodyNotContains(t, w, `name="finished_at"`)
}

func TestHandleGameCompletedPlaythrough(t *testing.T) {
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
	assertBodyContains(t, w, `name="finished_at"`)
}

// addWitcher3 adds The Witcher 3 to the collection with the given DLCs and returns the fake IGDB client.
func addWitcher3(t *testing.T, server *Server, dlcs []igdb.Game) *fakeIGDB {
	t.Helper()
	fake := &fakeIGDB{game: loadIGDBGames(t, "game_by_id_witcher3.json")[0], dlcs: dlcs}
	server.igdbClient = fake
	w := postForm(t, server, "/collection", "igdb_id=1942&platform_id=6")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("collection status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	return fake
}

// getWitcher3Page requests the game page for The Witcher 3 and asserts a 200 response.
func getWitcher3Page(t *testing.T, server *Server) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/games/1942", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	return w
}

func TestHandleGameDLCs(t *testing.T) {
	server := newTestServer(t)
	fake := addWitcher3(t, server, loadIGDBGames(t, "dlcs.json"))

	w := getWitcher3Page(t, server)
	assertBodyContains(t, w, "Hearts of Stone (12 October 2015) - Expansion")
	assertBodyNotContains(t, w, "couldn't be loaded")
	if fake.dlcsCalls != 1 {
		t.Errorf("dlcsCalls after first view = %d, want %d", fake.dlcsCalls, 1)
	}

	w = getWitcher3Page(t, server)
	assertBodyContains(t, w, "Hearts of Stone")
	if fake.dlcsCalls != 1 {
		t.Errorf("dlcsCalls after second view = %d, want %d", fake.dlcsCalls, 1)
	}
}

func TestHandleGameDLCsError(t *testing.T) {
	server := newTestServer(t)
	fake := addWitcher3(t, server, nil)
	fake.dlcsErr = errors.New("igdb unavailable")

	w := getWitcher3Page(t, server)
	assertBodyContains(t, w, "couldn't be loaded")
	if fake.dlcsCalls != 1 {
		t.Errorf("dlcsCalls after first view = %d, want %d", fake.dlcsCalls, 1)
	}

	getWitcher3Page(t, server)
	if fake.dlcsCalls != 2 {
		t.Errorf("dlcsCalls after second view = %d, want %d", fake.dlcsCalls, 2)
	}
}

func TestHandleGameNoDLCs(t *testing.T) {
	server := newTestServer(t)
	fake := addWitcher3(t, server, nil)

	w := getWitcher3Page(t, server)
	assertBodyContains(t, w, "No DLC")
	if fake.dlcsCalls != 1 {
		t.Errorf("dlcsCalls after first view = %d, want %d", fake.dlcsCalls, 1)
	}

	getWitcher3Page(t, server)
	if fake.dlcsCalls != 1 {
		t.Errorf("dlcsCalls after second view = %d, want %d", fake.dlcsCalls, 1)
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
