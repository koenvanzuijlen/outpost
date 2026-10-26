package igdb

import (
	"errors"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestGameCoverURL(t *testing.T) {
	tests := []struct {
		name         string
		retrievedURL string
		want         string
	}{
		{"empty", "", ""},
		{"filled", "//images.igdb.com/t_thumb/cover.jpg", "https://images.igdb.com/t_cover_big/cover.jpg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := Game{Cover: Cover{URL: tt.retrievedURL}}
			got := game.CoverURL()
			if got != tt.want {
				t.Errorf("CoverURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGameReleaseYear(t *testing.T) {
	tests := []struct {
		name         string
		firstRelease int
		want         int
	}{
		{"unknown", 0, 0},
		{"known", 915148800, 1999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := Game{FirstReleaseDate: tt.firstRelease}
			got := game.ReleaseYear()
			if got != tt.want {
				t.Errorf("ReleaseYear() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSearchGames(t *testing.T) {
	searchResponseFixture, err := os.ReadFile("testdata/games_search.json")
	if err != nil {
		t.Fatalf("os.ReadFile() failed: %v", err)
	}
	var gotBody string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write(searchResponseFixture)
	})
	games, err := client.SearchGames(t.Context(), "halo")
	if err != nil {
		t.Fatalf("SearchGames() failed: %v", err)
	}
	if !strings.Contains(gotBody, "halo") {
		t.Errorf("SearchGames() request body = %v, want it to contain %q", gotBody, "halo")
	}
	if !strings.Contains(gotBody, "platforms != null") {
		t.Errorf("SearchGames() request body = %v, want it to contain %q", gotBody, "platform != null")
	}
	if !strings.Contains(gotBody, "game_type = (0,4,8,9,10,11,12)") {
		t.Errorf("SearchGames() request body = %v, want it to contain %q", gotBody, "game_type = (0,4,8,9,10,11,12)")
	}
	if len(games) == 0 {
		t.Fatalf("SearchGames() returned no games, want at least one")
	}
	if games[0].Name != "Halo: Combat Evolved" {
		t.Errorf("SearchGames() first game = %v, want %v", games[0].Name, "Halo: Combat Evolved")
	}
	if games[0].Cover.URL == "" {
		t.Errorf("SearchGames() first game cover URL = %v, want it to be non-empty", games[0].Cover.URL)
	}

}

func TestGameByID(t *testing.T) {
	gameByIDResponseFixture, err := os.ReadFile("testdata/game_by_id.json")
	if err != nil {
		t.Fatalf("os.ReadFile() failed: %v", err)
	}
	var gotBody string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write(gameByIDResponseFixture)
	})

	game, err := client.GameByID(t.Context(), 740)
	if err != nil {
		t.Fatalf("GameByID() failed: %v", err)
	}
	if !strings.Contains(gotBody, "where id = 740") {
		t.Errorf("GameByID() request body = %v, want it to contain %q", gotBody, "where id = 740")
	}
	if game.Name != "Halo: Combat Evolved" {
		t.Errorf("GameByID() = %v, want %v", game.Name, "Halo: Combat Evolved")
	}
	if game.Cover.URL == "" {
		t.Errorf("GameByID() cover URL = %v, want it to be non-empty", game.Cover.URL)
	}
	if len(game.Platforms) == 0 {
		t.Fatalf("GameByID() platforms = %v, want it to be non-empty", game.Platforms)
	}
	if game.Platforms[0].ID != 11 {
		t.Errorf("GameByID() first platform ID = %v, want %v", game.Platforms[0].ID, 11)
	}
	if game.Platforms[0].Name != "Xbox" {
		t.Errorf("GameByID() first platform Name = %v, want %v", game.Platforms[0].Name, "Xbox")
	}
}

func TestGameByIDEmpty(t *testing.T) {
	emptyResponseFixture, err := os.ReadFile("testdata/empty.json")
	if err != nil {
		t.Fatalf("os.ReadFile() failed: %v", err)
	}
	var gotBody string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write(emptyResponseFixture)
	})

	_, err = client.GameByID(t.Context(), 740)
	if !errors.Is(err, ErrGameNotFound) {
		t.Errorf("GameByID() error = %v, want %v", err, ErrGameNotFound)
	}
	if !strings.Contains(gotBody, "where id = 740") {
		t.Errorf("GameByID() request body = %v, want it to contain %q", gotBody, "where id = 740")
	}
}

func TestDLCs(t *testing.T) {
	dlcsResponseFixture, err := os.ReadFile("testdata/dlcs.json")
	if err != nil {
		t.Fatalf("os.ReadFile() failed: %v", err)
	}
	var gotBody string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write(dlcsResponseFixture)
	})

	dlcs, err := client.DLCs(t.Context(), 1942)
	if err != nil {
		t.Fatalf("DLCs() failed: %v", err)
	}
	if !strings.Contains(gotBody, "where parent_game = 1942") {
		t.Errorf("DLCs() request body = %v, want it to contain %q", gotBody, "where parent_game = 1942")
	}
	if !strings.Contains(gotBody, "game_type = (1,2,6)") {
		t.Errorf("DLCs() request body = %v, want it to contain %q", gotBody, "game_type = (1,2,6)")
	}
	if len(dlcs) != 8 {
		t.Fatalf("DLCs() returned %d DLCs, want %d", len(dlcs), 8)
	}
	const wantName = "The Witcher 3: Wild Hunt - Blood and Wine"
	i := slices.IndexFunc(dlcs, func(g Game) bool { return g.Name == wantName })
	if i == -1 {
		t.Fatalf("DLCs() = %v, want it to contain %q", dlcs, wantName)
	}
	if dlcs[i].ParentGameID != 1942 {
		t.Errorf("DLCs() %q ParentGameID = %v, want %v", wantName, dlcs[i].ParentGameID, 1942)
	}
}

func TestDLCsEmpty(t *testing.T) {
	emptyResponseFixture, err := os.ReadFile("testdata/empty.json")
	if err != nil {
		t.Fatalf("os.ReadFile() failed: %v", err)
	}
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(emptyResponseFixture)
	})

	dlcs, err := client.DLCs(t.Context(), 1942)
	if err != nil {
		t.Fatalf("DLCs() failed: %v", err)
	}
	if len(dlcs) != 0 {
		t.Errorf("DLCs() returned %d DLCs, want 0", len(dlcs))
	}
}
