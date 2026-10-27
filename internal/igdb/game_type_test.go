package igdb

import "testing"

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

func TestGameTypeStandalone(t *testing.T) {
	tests := []struct {
		name     string
		gameType GameType
		want     bool
	}{
		{"main game", GameTypeMain, true},
		{"fork", GameTypeFork, true},
		{"dlc", GameTypeDLCAddon, false},
		{"bundle", GameTypeBundle, false},
		{"update", GameTypeUpdate, false},
		{"episode", GameTypeEpisode, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.gameType.Standalone()
			if got != tt.want {
				t.Errorf("Standalone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGameTypeDLC(t *testing.T) {
	tests := []struct {
		name     string
		gameType GameType
		want     bool
	}{
		{"main game", GameTypeMain, false},
		{"fork", GameTypeFork, false},
		{"dlc", GameTypeDLCAddon, true},
		{"bundle", GameTypeBundle, false},
		{"update", GameTypeUpdate, false},
		{"episode", GameTypeEpisode, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.gameType.DLC()
			if got != tt.want {
				t.Errorf("DLC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGameTypeStandaloneDisjointFromDLC(t *testing.T) {
	// DLCGameTypes and StandaloneGameTypes do not contain same types
	for _, gt := range StandaloneGameTypes {
		for _, dlcGT := range DLCGameTypes {
			if gt == dlcGT {
				t.Errorf("%v is in both StandaloneGameTypes and DLCGameTypes", gt)
			}
		}
	}
}
