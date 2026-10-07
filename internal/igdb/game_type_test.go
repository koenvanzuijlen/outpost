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
