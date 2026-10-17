package server

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestHandleStartPlaythrough(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}
	w := postForm(t, server, "/collection", "igdb_id=740&platform_id=11")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("collection status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = postForm(t, server, "/games/740/playthroughs", "platform=11&name=First+run")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("start playthrough status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	location := w.Header().Get("Location")
	if location != "/games/740" {
		t.Errorf("Location header = %q, want %q", location, "/games/740")
	}

	libraryGame, err := server.store.GetLibraryGame(t.Context(), 1, 740)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if len(libraryGame.Playthroughs) != 1 {
		t.Fatalf("len(Playthroughs) = %d, want %d", len(libraryGame.Playthroughs), 1)
	}
	playthrough := libraryGame.Playthroughs[0]
	if playthrough.Playthrough.Name == nil || *playthrough.Playthrough.Name != "First run" {
		t.Errorf("name = %v, want %q", derefOrNil(playthrough.Playthrough.Name), "First run")
	}
	if playthrough.PlatformName == nil || *playthrough.PlatformName != "Xbox" {
		t.Errorf("platform name = %v, want %q", derefOrNil(playthrough.PlatformName), "Xbox")
	}
	if libraryGame.UserGame.Status == nil || *libraryGame.UserGame.Status != "playing" {
		t.Errorf("status = %v, want %q", derefOrNil(libraryGame.UserGame.Status), "playing")
	}
}

func TestHandleStartPlaythroughBlankName(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}
	w := postForm(t, server, "/collection", "igdb_id=740&platform_id=11")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("collection status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = postForm(t, server, "/games/740/playthroughs", "name=++")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("start playthrough status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	libraryGame, err := server.store.GetLibraryGame(t.Context(), 1, 740)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if len(libraryGame.Playthroughs) != 1 {
		t.Fatalf("len(Playthroughs) = %d, want %d", len(libraryGame.Playthroughs), 1)
	}
	if name := libraryGame.Playthroughs[0].Playthrough.Name; name != nil {
		t.Errorf("name = %v, want %v", derefOrNil(name), nil)
	}
}

func TestHandleStartPlaythroughValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"non-numeric platform", "platform=abc"},
		{"unknown platform", "platform=6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}
			w := postForm(t, server, "/collection", "igdb_id=740&platform_id=11")
			if w.Code != http.StatusSeeOther {
				t.Fatalf("collection status = %d, want %d", w.Code, http.StatusSeeOther)
			}

			w = postForm(t, server, "/games/740/playthroughs", tt.body)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleStartPlaythroughNotInLibrary(t *testing.T) {
	server := newTestServer(t)
	w := postForm(t, server, "/games/740/playthroughs", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleStartPlaythroughWishlisted(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}
	w := postForm(t, server, "/wishlist", "igdb_id=740")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("wishlist status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = postForm(t, server, "/games/740/playthroughs", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// startTestPlaythrough adds game 740 to the collection, starts a playthrough for it and returns the playthrough ID.
func startTestPlaythrough(t *testing.T, server *Server) int64 {
	t.Helper()
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}
	w := postForm(t, server, "/collection", "igdb_id=740&platform_id=11")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("collection status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = postForm(t, server, "/games/740/playthroughs", "")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("start playthrough status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	libraryGame, err := server.store.GetLibraryGame(t.Context(), 1, 740)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	if len(libraryGame.Playthroughs) != 1 {
		t.Fatalf("len(Playthroughs) = %d, want %d", len(libraryGame.Playthroughs), 1)
	}
	return libraryGame.Playthroughs[0].Playthrough.ID
}

func TestHandleFinishPlaythrough(t *testing.T) {
	tests := []string{"completed", "abandoned", "retired"}

	for _, outcome := range tests {
		t.Run(outcome, func(t *testing.T) {
			server := newTestServer(t)
			playthroughID := startTestPlaythrough(t, server)

			path := "/games/740/playthroughs/" + strconv.FormatInt(playthroughID, 10) + "/finish"
			w := postForm(t, server, path, "outcome="+outcome)
			if w.Code != http.StatusSeeOther {
				t.Fatalf("finish playthrough status = %d, want %d", w.Code, http.StatusSeeOther)
			}
			location := w.Header().Get("Location")
			if location != "/games/740" {
				t.Errorf("Location header = %q, want %q", location, "/games/740")
			}

			libraryGame, err := server.store.GetLibraryGame(t.Context(), 1, 740)
			if err != nil {
				t.Fatalf("GetLibraryGame() failed: %v", err)
			}
			if libraryGame.UserGame.Status == nil || *libraryGame.UserGame.Status != outcome {
				t.Errorf("status = %v, want %q", derefOrNil(libraryGame.UserGame.Status), outcome)
			}
			if got := libraryGame.Playthroughs[0].Playthrough.Outcome; got != outcome {
				t.Errorf("outcome = %q, want %q", got, outcome)
			}
		})
	}
}

func TestHandleFinishPlaythroughValidation(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{"playing outcome", "/games/740/playthroughs/{id}/finish", "outcome=playing"},
		{"invalid outcome", "/games/740/playthroughs/{id}/finish", "outcome=invalid"},
		{"non-numeric playthrough id", "/games/740/playthroughs/abc/finish", "outcome=completed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			playthroughID := startTestPlaythrough(t, server)

			path := strings.ReplaceAll(tt.path, "{id}", strconv.FormatInt(playthroughID, 10))
			w := postForm(t, server, path, tt.body)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleFinishPlaythroughUnknownPlaythrough(t *testing.T) {
	server := newTestServer(t)
	playthroughID := startTestPlaythrough(t, server)

	path := "/games/740/playthroughs/" + strconv.FormatInt(playthroughID+1, 10) + "/finish"
	w := postForm(t, server, path, "outcome=completed")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
