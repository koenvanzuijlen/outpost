package database

import (
	"errors"
	"slices"
)

// ErrInvalidGameStatus is returned when an invalid game status is provided.
var ErrInvalidGameStatus = errors.New("invalid game status")

// GameStatus is the status of a library game, one of the GameStatus constants.
type GameStatus string

// The game statuses. Values are stored as text and validated in Go (Valid).
const (
	GameStatusBacklog      GameStatus = "backlog"
	GameStatusPlaying      GameStatus = "playing"
	GameStatusCompleted    GameStatus = "completed"
	GameStatusShelved      GameStatus = "shelved"
	GameStatusAbandoned    GameStatus = "abandoned"
	GameStatusRetired      GameStatus = "retired"
	GameStatusUninterested GameStatus = "uninterested"
)

// GameStatuses contains all possible game statuses, in the order they should be displayed.
var GameStatuses = []GameStatus{
	GameStatusBacklog,
	GameStatusPlaying,
	GameStatusCompleted,
	GameStatusShelved,
	GameStatusRetired,
	GameStatusAbandoned,
	GameStatusUninterested,
}

// Valid reports whether s is a known game status.
func (s GameStatus) Valid() bool {
	return slices.Contains(GameStatuses, s)
}

// notStarted reports whether s means that the game has not been started: backlog or uninterested.
func (s GameStatus) notStarted() bool {
	switch s {
	case GameStatusBacklog, GameStatusUninterested:
		return true
	}
	return false
}

// StringPtr returns s as a *string, for the nullable status column.
func (s GameStatus) StringPtr() *string {
	str := string(s)
	return &str
}

// gameStatusFrom converts the nullable status column to a GameStatus. A nil status becomes the empty GameStatus, which
// matches no status.
func gameStatusFrom(s *string) GameStatus {
	if s == nil {
		return ""
	}
	return GameStatus(*s)
}
