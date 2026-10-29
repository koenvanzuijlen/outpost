package server

import (
	"slices"
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// entryBuilder returns a function that builds library entries with IGDB IDs counting up from 1.
func entryBuilder() func(title string, status database.GameStatus, started, finished *time.Time) database.LibraryEntry {
	var nextID int64
	return func(title string, status database.GameStatus, started, finished *time.Time) database.LibraryEntry {
		nextID++
		return database.LibraryEntry{
			Game:           dbgen.Game{Title: title, IgdbID: nextID},
			UserGame:       dbgen.UserGame{Status: ptrIfNonZero(string(status))},
			LastStartedAt:  started,
			LastFinishedAt: finished,
		}
	}
}

// day returns a pointer to the given date in 2026.
func day(month time.Month, d int) *time.Time {
	t := time.Date(2026, month, d, 0, 0, 0, 0, time.UTC)
	return &t
}

// addedOn sets the added date of entry to added and returns it.
func addedOn(entry database.LibraryEntry, added *time.Time) database.LibraryEntry {
	entry.UserGame.AddedAt = *added
	return entry
}

// titles returns the titles of entries in order.
func titles(entries []database.LibraryEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Game.Title)
	}
	return out
}

func TestFilterLibrary(t *testing.T) {
	entries := func() []database.LibraryEntry {
		e := entryBuilder()
		return []database.LibraryEntry{
			e("A", database.GameStatusPlaying, nil, nil),
			e("B", database.GameStatusBacklog, nil, nil),
			e("C", database.GameStatusPlaying, nil, nil),
			e("D", "", nil, nil),
		}
	}

	tests := []struct {
		name   string
		status database.GameStatus
		want   []string
	}{
		{name: "matching status", status: database.GameStatusPlaying, want: []string{"A", "C"}},
		{name: "empty status returns all", status: "", want: []string{"A", "B", "C", "D"}},
		{name: "status nobody has", status: database.GameStatusRetired, want: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := titles(filterLibrary(entries(), tt.status))
			if !slices.Equal(got, tt.want) {
				t.Errorf("filterLibrary(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestSortLibrary(t *testing.T) {
	e := entryBuilder()
	tests := []struct {
		name    string
		order   librarySort
		entries []database.LibraryEntry
		want    []string
	}{
		{
			name:  "started newest first, undated last",
			order: librarySortStarted,
			entries: []database.LibraryEntry{
				e("B", "", day(3, 1), nil),
				e("A", "", nil, nil),
				e("C", "", day(1, 1), nil),
			},
			want: []string{"B", "C", "A"},
		},
		{
			name:  "finished newest first, undated last",
			order: librarySortFinished,
			entries: []database.LibraryEntry{
				e("B", "", nil, day(3, 1)),
				e("A", "", nil, nil),
				e("C", "", nil, day(1, 1)),
			},
			want: []string{"B", "C", "A"},
		},
		{
			name:  "same date ties sort by title",
			order: librarySortStarted,
			entries: []database.LibraryEntry{
				e("B", "", day(2, 1), nil),
				e("A", "", day(2, 1), nil),
			},
			want: []string{"A", "B"},
		},
		{
			name:  "added newest first, ties by title",
			order: librarySortAdded,
			entries: []database.LibraryEntry{
				addedOn(e("C", "", nil, nil), day(1, 1)),
				addedOn(e("B", "", nil, nil), day(3, 1)),
				addedOn(e("A", "", nil, nil), day(3, 1)),
				addedOn(e("D", "", nil, nil), day(2, 1)),
			},
			want: []string{"A", "B", "D", "C"},
		},
		{
			name:  "title",
			order: librarySortTitle,
			entries: []database.LibraryEntry{
				e("C", "", nil, nil),
				e("A", "", nil, nil),
				e("D", "", nil, nil),
				e("B", "", nil, nil),
			},
			want: []string{"A", "B", "C", "D"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := titles(sortLibrary(tt.entries, tt.order))
			if !slices.Equal(got, tt.want) {
				t.Errorf("sortLibrary(%q) = %v, want %v", tt.order, got, tt.want)
			}
		})
	}
}
