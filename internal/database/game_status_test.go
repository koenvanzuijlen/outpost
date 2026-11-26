package database

import "testing"

func TestGameStatusValid(t *testing.T) {
	tests := []struct {
		name   string
		status GameStatus
		want   bool
	}{
		{"backlog", GameStatus("backlog"), true},
		{"playing", GameStatus("playing"), true},
		{"completed", GameStatus("completed"), true},
		{"shelved", GameStatus("shelved"), true},
		{"abandoned", GameStatus("abandoned"), true},
		{"retired", GameStatus("retired"), true},
		{"uninterested", GameStatus("uninterested"), true},
		{"unknown", GameStatus("unknown"), false},
		{"empty", GameStatus(""), false},
		{"whitespace", GameStatus(" backlog"), false},
		{"capitalized", GameStatus("Backlog"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.Valid()
			if got != tt.want {
				t.Errorf("Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGameStatusNotStarted(t *testing.T) {
	tests := []struct {
		name   string
		status GameStatus
		want   bool
	}{
		{"backlog", GameStatus("backlog"), true},
		{"uninterested", GameStatus("uninterested"), true},
		{"playing", GameStatus("playing"), false},
		{"completed", GameStatus("completed"), false},
		{"shelved", GameStatus("shelved"), false},
		{"retired", GameStatus("retired"), false},
		{"abandoned", GameStatus("abandoned"), false},
		{"empty", GameStatus(""), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.notStarted()
			if got != tt.want {
				t.Errorf("notStarted() = %v, want %v", got, tt.want)
			}
		})
	}
}
