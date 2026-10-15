package database

import (
	"errors"
	"slices"
)

// ErrInvalidGameStatus is returned when an invalid game status is provided.
var ErrInvalidGameStatus = errors.New("invalid game status")

// GameStatus has the possible values for the status of a game.
type GameStatus string

// GameStatus has the possible values for the status of a game.
const (
	GameStatusBacklog   GameStatus = "backlog"
	GameStatusPlaying   GameStatus = "playing"
	GameStatusCompleted GameStatus = "completed"
	GameStatusShelved   GameStatus = "shelved"
	GameStatusAbandoned GameStatus = "abandoned"
	GameStatusRetired   GameStatus = "retired"
)

// GameStatuses contains all possible game statuses, in the order they should be displayed.
var GameStatuses = []GameStatus{
	GameStatusBacklog,
	GameStatusPlaying,
	GameStatusCompleted,
	GameStatusShelved,
	GameStatusAbandoned,
	GameStatusRetired,
}

// Valid checks if the given status is a valid game status.
func (s GameStatus) Valid() bool {
	return slices.Contains(GameStatuses, s)
}

// StringPtr gives the string pointer value for a game status.
func (s GameStatus) StringPtr() *string {
	str := string(s)
	return &str
}
