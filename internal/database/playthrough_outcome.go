package database

import (
	"errors"
	"slices"
)

// ErrInvalidPlaythroughOutcome is returned when an invalid playthrough outcome is provided.
var ErrInvalidPlaythroughOutcome = errors.New("invalid playthrough outcome")

// PlaythroughOutcome has the possible values for the outcome of a playthrough.
type PlaythroughOutcome string

// PlaythroughOutcome has the possible values for the outcome of a playthrough.
const (
	PlaythroughOutcomePlaying   PlaythroughOutcome = "playing"
	PlaythroughOutcomeCompleted PlaythroughOutcome = "completed"
	PlaythroughOutcomeAbandoned PlaythroughOutcome = "abandoned"
	PlaythroughOutcomeRetired   PlaythroughOutcome = "retired"
)

// PlaythroughOutcomes contains all possible playthrough outcomes, in the order they should be displayed.
var PlaythroughOutcomes = []PlaythroughOutcome{
	PlaythroughOutcomePlaying,
	PlaythroughOutcomeCompleted,
	PlaythroughOutcomeAbandoned,
	PlaythroughOutcomeRetired,
}

// Valid checks if the given outcome is a valid playthrough outcome.
func (o PlaythroughOutcome) Valid() bool {
	return slices.Contains(PlaythroughOutcomes, o)
}
