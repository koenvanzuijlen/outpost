package server

import (
	"strings"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

// gameStatusIcon returns the name of the Lucide icon for status. It returns an empty string for an unknown status.
func gameStatusIcon(status database.GameStatus) string {
	switch status {
	case database.GameStatusBacklog:
		return "list-todo"
	case database.GameStatusPlaying:
		return "gamepad-2"
	case database.GameStatusCompleted:
		return "trophy"
	case database.GameStatusShelved:
		return "pause"
	case database.GameStatusAbandoned:
		return "circle-x"
	case database.GameStatusRetired:
		return "archive"
	}
	return ""
}

// gameStatusLabel returns the display text for status: the stored value with a capital first letter. The status values
// are ASCII, so the first byte is the first letter.
func gameStatusLabel(status database.GameStatus) string {
	str := string(status)
	if str == "" {
		return ""
	}
	return strings.ToUpper(str[:1]) + str[1:]
}
