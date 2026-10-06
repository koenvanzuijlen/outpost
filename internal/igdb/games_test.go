package igdb

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestGameCoverURL(t *testing.T) {
	tests := []struct {
		name         string
		retrievedURL string
		expectedURL  string
	}{
		{"empty", "", ""},
		{"filled", "//images.igdb.com/t_thumb/cover.jpg", "https://images.igdb.com/t_cover_big/cover.jpg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := Game{Cover: Cover{URL: tt.retrievedURL}}
			got := game.CoverURL()
			if got != tt.expectedURL {
				t.Errorf("CoverURL() = %v, want %v", got, tt.expectedURL)
			}
		})
	}
}

func TestGameTypeString(t *testing.T) {
	tests := []struct {
		name     string
		gameType GameType
		want     string
	}{
		{"main game", GameTypeMain, "Main"},
		{"dlc", GameTypeDLCAddon, "DLC / Add-on"},
		{"unknown", GameType(99), "Unknown (99)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.gameType.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSearchGames(t *testing.T) {
	searchResponseFixture, err := os.ReadFile("testdata/games_search_response.json")
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
