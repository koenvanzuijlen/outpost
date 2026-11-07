package server

import (
	"testing"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

// TestGameStatusIconFiles checks that every status has an icon file.
func TestGameStatusIconFiles(t *testing.T) {
	for _, status := range database.GameStatuses {
		name := gameStatusIcon(status)
		path := "icons/" + name + ".svg"
		if _, err := iconFiles.ReadFile(path); err != nil {
			t.Errorf("ReadFile(%q) error = %v, want nil (status %q)", path, err, status)
		}
	}
}

// TestGameStatusIconUnknown checks that an unknown status has no icon.
func TestGameStatusIconUnknown(t *testing.T) {
	if got := gameStatusIcon(database.GameStatus("unknown")); got != "" {
		t.Errorf("gameStatusIcon(unknown) = %q, want %q", got, "")
	}
}

// TestGameStatusLabel checks that the label starts with a capital letter and that an empty status does not panic.
func TestGameStatusLabel(t *testing.T) {
	tests := []struct {
		status database.GameStatus
		want   string
	}{
		{"playing", "Playing"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := gameStatusLabel(tt.status); got != tt.want {
			t.Errorf("gameStatusLabel(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}
