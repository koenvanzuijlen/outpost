package server

import "testing"

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
