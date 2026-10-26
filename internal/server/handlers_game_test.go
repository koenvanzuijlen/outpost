package server

import (
	"errors"
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
	addTestGame(t, server)

	w := postForm(t, server, "/games/1942", "status=completed&rating=9")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("update games status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = getTestGamePage(t, server)
	assertBodyContains(t, w, "The Witcher 3: Wild Hunt")
	assertBodyContains(t, w, "PC (Microsoft Windows)")
	assertBodyContains(t, w, `name="status"`)
	assertBodyContains(t, w, "4.5")
}

func TestHandleGameInLibraryWishlisted(t *testing.T) {
	server := newTestServer(t)
	wishlistTestGame(t, server)

	w := getTestGamePage(t, server)
	assertBodyContains(t, w, "The Witcher 3: Wild Hunt")
	assertBodyNotContains(t, w, `name="status"`)
	assertBodyNotContains(t, w, "/playthroughs")
}

func TestHandleGameActivePlaythrough(t *testing.T) {
	server := newTestServer(t)
	startTestPlaythrough(t, server, "Hard mode run", "6")

	libraryGame := getTestLibraryGame(t, server)
	if len(libraryGame.Playthroughs) != 1 {
		t.Fatalf("len(Playthroughs) = %d, want %d", len(libraryGame.Playthroughs), 1)
	}
	playthrough := libraryGame.Playthroughs[0].Playthrough
	if playthrough.StartedAt == nil {
		t.Fatal("StartedAt = nil, want a date")
	}
	playthroughID := playthrough.ID

	w := getTestGamePage(t, server)
	assertBodyContains(t, w, "Hard mode run")
	assertBodyContains(t, w, "Platform: PC (Microsoft Windows)")
	assertBodyContains(t, w, fmt.Sprintf("/games/1942/playthroughs/%d/outcome", playthroughID))
	assertBodyContains(t, w, fmt.Sprintf("/games/1942/playthroughs/%d/delete", playthroughID))
	assertBodyContains(t, w, fmt.Sprintf(`action="/games/1942/playthroughs/%d"`, playthroughID))
	assertBodyContains(t, w, `<option value="6" selected>PC (Microsoft Windows)</option>`)
	assertBodyContains(t, w, fmt.Sprintf(`value="%s"`, playthrough.StartedAt.Format("2006-01-02")))
	assertBodyNotContains(t, w, `name="finished_at"`)
}

func TestHandleGameCompletedPlaythrough(t *testing.T) {
	server := newTestServer(t)
	playthroughID := startTestPlaythrough(t, server, "Hard mode run", "6")

	w := postForm(t, server, fmt.Sprintf("/games/1942/playthroughs/%d/outcome", playthroughID), "outcome=abandoned")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("set outcome status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = getTestGamePage(t, server)
	assertBodyContains(t, w, "Hard mode run")
	assertBodyContains(t, w, "Platform: PC (Microsoft Windows)")
	assertBodyContains(t, w, "Outcome: abandoned")
	assertBodyContains(t, w, `value="abandoned" aria-pressed="true"`)
	assertBodyContains(t, w, `value="playing" aria-pressed="false"`)
	assertBodyContains(t, w, fmt.Sprintf("/games/1942/playthroughs/%d/outcome", playthroughID))
	assertBodyContains(t, w, fmt.Sprintf("/games/1942/playthroughs/%d/delete", playthroughID))
	assertBodyContains(t, w, `name="finished_at"`)
}

func TestHandleGameDLCs(t *testing.T) {
	server := newTestServer(t)
	fake := addTestGame(t, server)
	fake.dlcs = loadIGDBGames(t, "dlcs.json")

	w := getTestGamePage(t, server)
	assertBodyContains(t, w, "Hearts of Stone (12 October 2015) - Expansion")
	assertBodyNotContains(t, w, "couldn't be loaded")
	if fake.dlcsCalls != 1 {
		t.Errorf("dlcsCalls after first view = %d, want %d", fake.dlcsCalls, 1)
	}

	w = getTestGamePage(t, server)
	assertBodyContains(t, w, "Hearts of Stone")
	if fake.dlcsCalls != 1 {
		t.Errorf("dlcsCalls after second view = %d, want %d", fake.dlcsCalls, 1)
	}
}

func TestHandleGameCopyDLCs(t *testing.T) {
	server := newTestServer(t)
	fake := addTestGame(t, server)
	fake.dlcs = loadIGDBGames(t, "dlcs.json")

	w := getTestGamePage(t, server)
	// Hearts of Stone (12503) is available on PC (6), the platform of the test copy.
	assertBodyContains(t, w, "The Witcher 3: Wild Hunt - Hearts of Stone - Not owned")
}

func TestHandleGameDLCsError(t *testing.T) {
	server := newTestServer(t)
	fake := addTestGame(t, server)
	fake.dlcsErr = errors.New("igdb unavailable")

	w := getTestGamePage(t, server)
	assertBodyContains(t, w, "couldn't be loaded")
	if fake.dlcsCalls != 1 {
		t.Errorf("dlcsCalls after first view = %d, want %d", fake.dlcsCalls, 1)
	}

	getTestGamePage(t, server)
	if fake.dlcsCalls != 2 {
		t.Errorf("dlcsCalls after second view = %d, want %d", fake.dlcsCalls, 2)
	}
}

func TestHandleGameNoDLCs(t *testing.T) {
	server := newTestServer(t)
	fake := addTestGame(t, server)

	w := getTestGamePage(t, server)
	assertBodyContains(t, w, "No DLC")
	if fake.dlcsCalls != 1 {
		t.Errorf("dlcsCalls after first view = %d, want %d", fake.dlcsCalls, 1)
	}

	getTestGamePage(t, server)
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
		{"no status", "/games/1942", "rating=10"},
		{"invalid status", "/games/1942", "status=invalid&rating=10"},
		{"low rating", "/games/1942", "status=backlog&rating=0"},
		{"high rating", "/games/1942", "status=backlog&rating=11"},
		{"invalid rating", "/games/1942", "status=backlog&rating=abc"},
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
	addTestGame(t, server)

	w := postForm(t, server, "/games/1942", "status=completed&rating=9")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("update games status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/games/1942")
	libraryGame := getTestLibraryGame(t, server)
	if libraryGame.UserGame.Status == nil || *libraryGame.UserGame.Status != "completed" {
		t.Errorf("status = %v, want %q", derefOrNil(libraryGame.UserGame.Status), "completed")
	}
	if libraryGame.UserGame.Rating == nil || *libraryGame.UserGame.Rating != 9 {
		t.Errorf("rating = %v, want %v", derefOrNil(libraryGame.UserGame.Rating), 9)
	}
}

func TestHandleUpdateGameNoRating(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)

	w := postForm(t, server, "/games/1942", "status=completed&rating=")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("update games status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/games/1942")
	libraryGame := getTestLibraryGame(t, server)
	if libraryGame.UserGame.Status == nil || *libraryGame.UserGame.Status != "completed" {
		t.Errorf("status = %v, want %q", derefOrNil(libraryGame.UserGame.Status), "completed")
	}
	if libraryGame.UserGame.Rating != nil {
		t.Errorf("rating = %v, want %v", derefOrNil(libraryGame.UserGame.Rating), nil)
	}
}

func TestHandleUpdateGameWishlisted(t *testing.T) {
	server := newTestServer(t)
	wishlistTestGame(t, server)

	w := postForm(t, server, "/games/1942", "status=completed&rating=9")
	if w.Code != http.StatusNotFound {
		t.Fatalf("update games status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
