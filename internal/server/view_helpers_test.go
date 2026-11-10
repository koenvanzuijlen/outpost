package server

import "testing"

func TestDLCTitle(t *testing.T) {
	tests := []struct {
		name        string
		parentTitle string
		title       string
		want        string
	}{
		{"dash", "The Witcher 3: Wild Hunt", "The Witcher 3: Wild Hunt - Hearts of Stone", "Hearts of Stone"},
		{"colon", "Elden Ring", "Elden Ring: Shadow of the Erdtree", "Shadow of the Erdtree"},
		{"en dash", "Hades", "Hades – Bonus", "Bonus"},
		{"no prefix", "The Witcher 3: Wild Hunt", "Thronebreaker", "Thronebreaker"},
		{"same title", "Doom", "Doom", "Doom"},
		{"longer word", "Doom", "Doomsday", "Doomsday"},
		{"separator only", "Doom", "Doom -", "Doom -"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dlcTitle(tt.parentTitle, tt.title); got != tt.want {
				t.Errorf("dlcTitle(%q, %q) = %q, want %q", tt.parentTitle, tt.title, got, tt.want)
			}
		})
	}
}

func TestRatingText(t *testing.T) {
	tests := []struct {
		name   string
		rating int64
		want   string
	}{
		{name: "half star", rating: 1, want: "0.5 ⭐"},
		{name: "whole star", rating: 2, want: "1 ⭐"},
		{name: "odd rating", rating: 7, want: "3.5 ⭐"},
		{name: "maximum", rating: 10, want: "5 ⭐"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ratingText(tt.rating); got != tt.want {
				t.Errorf("ratingText(%d) = %q, want %q", tt.rating, got, tt.want)
			}
		})
	}
}

// TestRatingScore checks that the score shows half stars and drops the trailing zeros of whole stars.
func TestRatingScore(t *testing.T) {
	tests := []struct {
		rating int64
		want   string
	}{
		{1, "0.5"},
		{2, "1"},
		{7, "3.5"},
		{10, "5"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := ratingScore(tt.rating); got != tt.want {
				t.Errorf("ratingScore(%d) = %q, want %q", tt.rating, got, tt.want)
			}
		})
	}
}

func TestOutcomeLabel(t *testing.T) {
	tests := []struct {
		outcome string
		want    string
	}{
		{"playing", "in progress"},
		{"completed", "completed"},
		{"abandoned", "abandoned"},
		{"retired", "retired"},
	}

	for _, tt := range tests {
		t.Run(tt.outcome, func(t *testing.T) {
			if got := outcomeLabel(tt.outcome); got != tt.want {
				t.Errorf("outcomeLabel(%q) = %q, want %q", tt.outcome, got, tt.want)
			}
		})
	}
}

func TestRatingLabel(t *testing.T) {
	tests := []struct {
		name   string
		rating int64
		want   string
	}{
		{name: "half star", rating: 1, want: "0.5 out of 5"},
		{name: "whole star", rating: 2, want: "1 out of 5"},
		{name: "odd rating", rating: 7, want: "3.5 out of 5"},
		{name: "maximum", rating: 10, want: "5 out of 5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ratingLabel(tt.rating); got != tt.want {
				t.Errorf("ratingLabel(%d) = %q, want %q", tt.rating, got, tt.want)
			}
		})
	}
}
