package server

import (
	"slices"
	"strings"
	"testing"
)

// TestRatingStarFills checks the fill of each star for ratings at the boundaries of empty, half and full stars.
func TestRatingStarFills(t *testing.T) {
	const (
		F = ratingStarFull
		H = ratingStarHalf
		E = ratingStarEmpty
	)

	tests := []struct {
		name   string
		rating int64
		want   []ratingStarFill
	}{
		{"unrated", 0, []ratingStarFill{E, E, E, E, E}},
		{"half star", 1, []ratingStarFill{H, E, E, E, E}},
		{"one star", 2, []ratingStarFill{F, E, E, E, E}},
		{"one and a half", 3, []ratingStarFill{F, H, E, E, E}},
		{"four stars", 8, []ratingStarFill{F, F, F, F, E}},
		{"four and a half", 9, []ratingStarFill{F, F, F, F, H}},
		{"maximum", 10, []ratingStarFill{F, F, F, F, F}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ratingStarFills(tt.rating); !slices.Equal(got, tt.want) {
				t.Errorf("ratingStarFills(%d) = %v, want %v", tt.rating, got, tt.want)
			}
		})
	}
}

// TestRatingInputOptions checks the number of rating input options and the value, label and side of the options at the
// start and end of the range.
func TestRatingInputOptions(t *testing.T) {
	got := ratingInputOptions()
	if len(got) != 10 {
		t.Fatalf("len(ratingInputOptions()) = %d, want 10", len(got))
	}

	tests := []struct {
		index int
		want  ratingInputOption
	}{
		{0, ratingInputOption{1, "0.5 out of 5", ratingStarLeft}},
		{1, ratingInputOption{2, "1 out of 5", ratingStarRight}},
		{9, ratingInputOption{10, "5 out of 5", ratingStarRight}},
	}

	for _, tt := range tests {
		if got[tt.index] != tt.want {
			t.Errorf("ratingInputOptions()[%d] = %+v, want %+v", tt.index, got[tt.index], tt.want)
		}
	}
}

// TestRatingStars checks that the full variant has the star class of a half star and that the compact variant shows
// the score.
func TestRatingStars(t *testing.T) {
	tests := []struct {
		name    string
		rating  int64
		compact bool
		want    string
	}{
		{"full variant half star class", 3, false, "rating-stars__star--half"},
		{"compact variant score", 9, true, "4.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sb strings.Builder
			if err := ratingStars(tt.rating, tt.compact).Render(t.Context(), &sb); err != nil {
				t.Fatalf("ratingStars(%d, %v).Render() error = %v, want nil", tt.rating, tt.compact, err)
			}
			if got := sb.String(); !strings.Contains(got, tt.want) {
				t.Errorf("ratingStars(%d, %v) = %q, want it to contain %q", tt.rating, tt.compact, got, tt.want)
			}
		})
	}
}
