package server

import (
	"slices"
	"testing"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

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

func TestCopySummary(t *testing.T) {
	tests := []struct {
		name string
		dlcs []database.CopyDLC
		want string
	}{
		{"no DLC", nil, "PC"},
		{"none owned", []database.CopyDLC{{Owned: false}, {Owned: false}}, "PC"},
		{"one owned", []database.CopyDLC{{Owned: true}, {Owned: false}}, "PC (1 DLC owned)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := copySummary("PC", tt.dlcs); got != tt.want {
				t.Errorf("copySummary(%q, %v) = %q, want %q", "PC", tt.dlcs, got, tt.want)
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

// The test platforms have different internal IDs and IGDB IDs, so a test fails if the code mixes up the two IDs.
var (
	testPlatformPC     = dbgen.Platform{ID: 1, IgdbID: 6, Name: "PC"}
	testPlatformSwitch = dbgen.Platform{ID: 2, IgdbID: 130, Name: "Switch"}
	testPlatformXbox   = dbgen.Platform{ID: 3, IgdbID: 49, Name: "Xbox"}
)

// platformEntry returns a playthrough row that points at the given platform.
func platformEntry(p dbgen.Platform) dbgen.ListPlaythroughsForUserGameIDRow {
	return dbgen.ListPlaythroughsForUserGameIDRow{
		Playthrough:    dbgen.Playthrough{PlatformID: new(p.ID)},
		PlatformName:   new(p.Name),
		PlatformIgdbID: new(p.IgdbID),
	}
}

// TestPlaythroughPlatformOptions checks that the platform options are the owned platforms, plus the platform of the
// playthrough if it is not owned.
func TestPlaythroughPlatformOptions(t *testing.T) {
	tests := []struct {
		name  string
		owned []dbgen.Platform
		entry dbgen.ListPlaythroughsForUserGameIDRow
		want  []int64
	}{
		{
			name:  "no platform",
			owned: []dbgen.Platform{testPlatformPC},
			entry: dbgen.ListPlaythroughsForUserGameIDRow{},
			want:  []int64{6},
		},
		{
			name:  "owned platform",
			owned: []dbgen.Platform{testPlatformPC, testPlatformSwitch},
			entry: platformEntry(testPlatformSwitch),
			want:  []int64{6, 130},
		},
		{
			name:  "platform not owned",
			owned: []dbgen.Platform{testPlatformPC},
			entry: platformEntry(testPlatformSwitch),
			want:  []int64{6, 130},
		},
		{
			name:  "nothing owned",
			owned: nil,
			entry: platformEntry(testPlatformSwitch),
			want:  []int64{130},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []int64
			for _, p := range playthroughPlatformOptions(tt.owned, tt.entry) {
				got = append(got, p.IgdbID)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("playthroughPlatformOptions() IGDB IDs = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestPlaythroughPlatformOptionsKeepsOwned checks that a call does not change the result of an earlier call when owned
// has spare capacity. Without a copy, both results share the backing array of owned.
func TestPlaythroughPlatformOptionsKeepsOwned(t *testing.T) {
	owned := make([]dbgen.Platform, 1, 2)
	owned[0] = testPlatformPC

	first := playthroughPlatformOptions(owned, platformEntry(testPlatformSwitch))
	playthroughPlatformOptions(owned, platformEntry(testPlatformXbox))

	if len(first) != 2 {
		t.Fatalf("len(first) = %d, want 2", len(first))
	}
	if first[1].IgdbID != 130 {
		t.Errorf("first[1].IgdbID = %d, want %d", first[1].IgdbID, 130)
	}
}
