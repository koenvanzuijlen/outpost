package database

import "testing"

func TestValid(t *testing.T) {
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
