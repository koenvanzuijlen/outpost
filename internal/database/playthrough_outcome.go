package database

import (
	"errors"
	"slices"
	"time"
)

// ErrInvalidPlaythroughOutcome is returned when an invalid playthrough outcome is provided.
var ErrInvalidPlaythroughOutcome = errors.New("invalid playthrough outcome")

// PlaythroughOutcome is the outcome of a playthrough, one of the PlaythroughOutcome constants.
type PlaythroughOutcome string

// The playthrough outcomes. Values are stored as text and validated in Go (Valid).
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

// Valid reports whether o is a known playthrough outcome.
func (o PlaythroughOutcome) Valid() bool {
	return slices.Contains(PlaythroughOutcomes, o)
}

// finishedAtFor returns finished_at after a playthrough's outcome changes from `from` to `to`: now when it leaves
// playing, nil when it goes back to playing, and current otherwise.
func finishedAtFor(from, to PlaythroughOutcome, current *time.Time, now time.Time) *time.Time {
	if from == PlaythroughOutcomePlaying && to != PlaythroughOutcomePlaying {
		return &now
	}
	if from != PlaythroughOutcomePlaying && to == PlaythroughOutcomePlaying {
		return nil
	}
	return current
}

// statusAfterOutcomeChange returns the game status after a playthrough's outcome changes, and whether it changes at
// all. It only changes when the last playing playthrough of a game that is playing ends; the new status is then the
// outcome.
func statusAfterOutcomeChange(from, to PlaythroughOutcome, status GameStatus, otherPlaying int64) (GameStatus, bool) {
	if from != PlaythroughOutcomePlaying || to == PlaythroughOutcomePlaying || status != GameStatusPlaying || otherPlaying > 0 {
		return status, false
	}
	return GameStatus(to), true
}
