package database

import (
	"testing"
	"time"
)

func TestPlaythroughOutcomeValid(t *testing.T) {
	tests := []struct {
		name    string
		outcome PlaythroughOutcome
		want    bool
	}{
		{"playing", PlaythroughOutcome("playing"), true},
		{"completed", PlaythroughOutcome("completed"), true},
		{"abandoned", PlaythroughOutcome("abandoned"), true},
		{"retired", PlaythroughOutcome("retired"), true},
		{"backlog", PlaythroughOutcome("backlog"), false},
		{"shelved", PlaythroughOutcome("shelved"), false},
		{"unknown", PlaythroughOutcome("unknown"), false},
		{"empty", PlaythroughOutcome(""), false},
		{"whitespace", PlaythroughOutcome(" playing"), false},
		{"capitalized", PlaythroughOutcome("Playing"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.outcome.Valid()
			if got != tt.want {
				t.Errorf("Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlaythroughOutcomeSubsetOfGameStatus(t *testing.T) {
	for _, o := range PlaythroughOutcomes {
		t.Run(string(o), func(t *testing.T) {
			got := GameStatus(o).Valid()
			if !got {
				t.Errorf("GameStatus().Valid() = %v, want %v", got, true)
			}
		})
	}
}

func TestFinishedAtFor(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	current := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		from PlaythroughOutcome
		to   PlaythroughOutcome
		want *time.Time
	}{
		{"playing to completed", PlaythroughOutcomePlaying, PlaythroughOutcomeCompleted, &now},
		{"playing to abandoned", PlaythroughOutcomePlaying, PlaythroughOutcomeAbandoned, &now},
		{"playing to retired", PlaythroughOutcomePlaying, PlaythroughOutcomeRetired, &now},
		{"completed to playing", PlaythroughOutcomeCompleted, PlaythroughOutcomePlaying, nil},
		{"abandoned to playing", PlaythroughOutcomeAbandoned, PlaythroughOutcomePlaying, nil},
		{"retired to playing", PlaythroughOutcomeRetired, PlaythroughOutcomePlaying, nil},
		{"completed to abandoned", PlaythroughOutcomeCompleted, PlaythroughOutcomeAbandoned, &current},
		{"abandoned to retired", PlaythroughOutcomeAbandoned, PlaythroughOutcomeRetired, &current},
		{"completed to completed", PlaythroughOutcomeCompleted, PlaythroughOutcomeCompleted, &current},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := finishedAtFor(tt.from, tt.to, &current, now)
			switch {
			case tt.want == nil && got != nil:
				t.Errorf("finishedAtFor() = %v, want nil", *got)
			case tt.want != nil && got == nil:
				t.Errorf("finishedAtFor() = nil, want %v", *tt.want)
			case tt.want != nil && !got.Equal(*tt.want):
				t.Errorf("finishedAtFor() = %v, want %v", *got, *tt.want)
			}
		})
	}
}

func TestStatusAfterOutcomeChange(t *testing.T) {
	tests := []struct {
		name         string
		from         PlaythroughOutcome
		to           PlaythroughOutcome
		status       GameStatus
		otherPlaying int64
		wantStatus   GameStatus
		wantChanged  bool
	}{
		{"playing to completed", PlaythroughOutcomePlaying, PlaythroughOutcomeCompleted, GameStatusPlaying, 0, GameStatusCompleted, true},
		{"playing to abandoned", PlaythroughOutcomePlaying, PlaythroughOutcomeAbandoned, GameStatusPlaying, 0, GameStatusAbandoned, true},
		{"playing to retired", PlaythroughOutcomePlaying, PlaythroughOutcomeRetired, GameStatusPlaying, 0, GameStatusRetired, true},
		{"end to end", PlaythroughOutcomeCompleted, PlaythroughOutcomeAbandoned, GameStatusPlaying, 0, GameStatusPlaying, false},
		{"end to playing", PlaythroughOutcomeCompleted, PlaythroughOutcomePlaying, GameStatusPlaying, 0, GameStatusPlaying, false},
		{"status shelved", PlaythroughOutcomePlaying, PlaythroughOutcomeCompleted, GameStatusShelved, 0, GameStatusShelved, false},
		{"one other playing", PlaythroughOutcomePlaying, PlaythroughOutcomeCompleted, GameStatusPlaying, 1, GameStatusPlaying, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotChanged := statusAfterOutcomeChange(tt.from, tt.to, tt.status, tt.otherPlaying)
			if gotStatus != tt.wantStatus || gotChanged != tt.wantChanged {
				t.Errorf("statusAfterOutcomeChange() = (%q, %v), want (%q, %v)", gotStatus, gotChanged, tt.wantStatus, tt.wantChanged)
			}
		})
	}
}
