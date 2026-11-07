package server

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// TestGameCardZeroValues checks that the status badge, the rating and the cover show only for non-zero values.
func TestGameCardZeroValues(t *testing.T) {
	cover := "https://example.com/cover.jpg"
	released := time.Date(2015, 5, 19, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		game       dbgen.Game
		status     database.GameStatus
		rating     int64
		wantBadge  bool
		wantRating bool
		wantCover  bool
		wantYear   bool
	}{
		{"all set", dbgen.Game{IgdbID: 1942, Title: "A", CoverUrl: &cover, ReleasedAt: &released}, "playing", 4, true, true, true, true},
		{"zero values", dbgen.Game{IgdbID: 1942, Title: "A"}, "", 0, false, false, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := gameCard(tt.game, tt.status, tt.rating).Render(t.Context(), &buf); err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			body := buf.String()
			checks := []struct {
				substr string
				want   bool
			}{
				{"game-card__status", tt.wantBadge},
				{"out of 5", tt.wantRating},
				{"<img", tt.wantCover},
				{"2015", tt.wantYear},
			}
			for _, c := range checks {
				if got := strings.Contains(body, c.substr); got != c.want {
					t.Errorf("contains %q = %v, want %v", c.substr, got, c.want)
				}
			}
		})
	}
}
