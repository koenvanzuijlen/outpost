package database

import (
	"errors"
	"slices"
	"time"
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

// finishedAtFor determines the appropriate finished_at timestamp based on the transition of playthrough outcomes.
func finishedAtFor(from, to PlaythroughOutcome, current *time.Time, now time.Time) *time.Time {
	if from == PlaythroughOutcomePlaying && to != PlaythroughOutcomePlaying {
		return &now
	}
	if from != PlaythroughOutcomePlaying && to == PlaythroughOutcomePlaying {
		return nil
	}
	return current
}

// statusAfterOutcomeChange determines the new game status after ending a playthrough, which can be changed only when it
// ends the last playing playthrough of a game that is playing. Returns the new game status and a boolean indicating if
// it should change.
func statusAfterOutcomeChange(from, to PlaythroughOutcome, status GameStatus, otherPlaying int64) (GameStatus, bool) {
	if from != PlaythroughOutcomePlaying || to == PlaythroughOutcomePlaying || status != GameStatusPlaying || otherPlaying > 0 {
		return status, false
	}
	return GameStatus(to), true
}
