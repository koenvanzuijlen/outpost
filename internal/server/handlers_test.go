package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// postForm sends a POST request with the given form body to the specified URL and returns the response recorder.
func postForm(t *testing.T, server *Server, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

func TestHandleAddToCollectionValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"missing body", ""},
		{"invalid id", "igdb_id=abc&platform=pc"},
		{"negative id", "igdb_id=-1&platform=pc"},
		{"missing platform", "igdb_id=700&platform=%20"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			w := postForm(t, server, "/collection", tt.body)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleAddToCollection(t *testing.T) {
	tests := []struct {
		name         string
		fake         *fakeIGDB
		wantStatus   int
		wantLocation string
	}{
		{"added", &fakeIGDB{game: loadIGDBGames(t, "game_by_id_response.json")[0]}, http.StatusSeeOther, "/"},
		{"not found", &fakeIGDB{gameErr: igdb.ErrGameNotFound}, http.StatusNotFound, ""},
		{"igdb error", &fakeIGDB{gameErr: errors.New("IGDB failure")}, http.StatusInternalServerError, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			server.igdbClient = tt.fake
			w := postForm(t, server, "/collection", "igdb_id=740&platform=pc")

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			location := w.Header().Get("Location")
			if location != tt.wantLocation {
				t.Errorf("location = %q, want %q", location, tt.wantLocation)
			}
		})
	}
}

func TestHandleAddToCollectionTwice(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "game_by_id_response.json")[0]}
	w := postForm(t, server, "/collection", "igdb_id=740&platform=pc")

	if w.Code != http.StatusSeeOther {
		t.Errorf("first status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = postForm(t, server, "/collection", "igdb_id=740&platform=pc")

	if w.Code != http.StatusSeeOther {
		t.Errorf("second status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	items, err := server.queries.ListCollectionItemsWithGames(t.Context(), defaultUserID)
	if err != nil {
		t.Fatalf("ListCollectionItemsWithGames() failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("number of collection items = %d, want %d", len(items), 1)
	}
	if items[0].Game.Title != "Halo: Combat Evolved" {
		t.Errorf("collection item title = %q, want %q", items[0].Game.Title, "Halo: Combat Evolved")
	}
}
