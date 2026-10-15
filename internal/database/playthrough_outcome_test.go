package database

import "testing"

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
