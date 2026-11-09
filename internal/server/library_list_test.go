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
		name     string
		statuses []database.GameStatus
		want     []string
	}{
		{"matching status", []database.GameStatus{database.GameStatusPlaying}, []string{"A", "C"}},
		{"no statuses returns all", nil, []string{"A", "B", "C", "D"}},
		{"status nobody has", []database.GameStatus{database.GameStatusRetired}, []string{}},
		{
			"several statuses",
			[]database.GameStatus{database.GameStatusPlaying, database.GameStatusBacklog},
			[]string{"A", "B", "C"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := titles(filterLibrary(entries(), tt.statuses))
			if !slices.Equal(got, tt.want) {
				t.Errorf("filterLibrary(%v) = %v, want %v", tt.statuses, got, tt.want)
			}
		})
	}
}

func TestLibraryQueryStatusToggles(t *testing.T) {
	tests := []struct {
		name       string
		query      libraryQuery
		status     database.GameStatus
		wantActive bool
		wantHref   string
	}{
		{
			name:     "no filter adds the status",
			query:    libraryQuery{Sort: librarySortTitle, Order: sortOrderAsc},
			status:   database.GameStatusShelved,
			wantHref: "/library?order=asc&sort=title&status=shelved",
		},
		{
			name: "adds the status in the fixed order",
			query: libraryQuery{
				Statuses: []database.GameStatus{database.GameStatusShelved},
				Sort:     librarySortAdded,
				Order:    sortOrderDesc,
			},
			status:   database.GameStatusBacklog,
			wantHref: "/library?order=desc&sort=added&status=backlog&status=shelved",
		},
		{
			name: "removes an active status",
			query: libraryQuery{
				Statuses: []database.GameStatus{database.GameStatusShelved, database.GameStatusBacklog},
				Sort:     librarySortTitle,
				Order:    sortOrderAsc,
			},
			status:     database.GameStatusShelved,
			wantActive: true,
			wantHref:   "/library?order=asc&sort=title&status=backlog",
		},
		{
			name: "removing the last status shows all",
			query: libraryQuery{
				Statuses: []database.GameStatus{database.GameStatusPlaying},
				Sort:     librarySortTitle,
				Order:    sortOrderAsc,
			},
			status:     database.GameStatusPlaying,
			wantActive: true,
			wantHref:   "/library?order=asc&sort=title",
		},
		{
			name: "duplicate statuses appear once",
			query: libraryQuery{
				Statuses: []database.GameStatus{database.GameStatusPlaying, database.GameStatusPlaying},
				Sort:     librarySortTitle,
				Order:    sortOrderAsc,
			},
			status:   database.GameStatusBacklog,
			wantHref: "/library?order=asc&sort=title&status=backlog&status=playing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toggles := tt.query.StatusToggles()
			i := slices.IndexFunc(toggles, func(toggle libraryStatusToggle) bool { return toggle.Status == tt.status })
			if i == -1 {
				t.Fatalf("StatusToggles() has no toggle for %q", tt.status)
			}
			if toggles[i].Active != tt.wantActive {
				t.Errorf("StatusToggles()[%q].Active = %v, want %v", tt.status, toggles[i].Active, tt.wantActive)
			}
			if toggles[i].Href != tt.wantHref {
				t.Errorf("StatusToggles()[%q].Href = %q, want %q", tt.status, toggles[i].Href, tt.wantHref)
			}
		})
	}
}

func TestLibraryQueryOrderToggleURL(t *testing.T) {
	tests := []struct {
		name  string
		query libraryQuery
		want  string
	}{
		{
			name:  "ascending becomes descending",
			query: libraryQuery{Sort: librarySortTitle, Order: sortOrderAsc},
			want:  "/library?order=desc&sort=title",
		},
		{
			name: "descending becomes ascending and keeps statuses",
			query: libraryQuery{
				Statuses: []database.GameStatus{database.GameStatusBacklog},
				Sort:     librarySortStarted,
				Order:    sortOrderDesc,
			},
			want: "/library?order=asc&sort=started&status=backlog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.query.OrderToggleURL(); got != tt.want {
				t.Errorf("OrderToggleURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLibraryQueryStatusTogglesKeepsQuery(t *testing.T) {
	// The query slice has room for one more status, so an append without a copy writes into backing[1].
	backing := []database.GameStatus{database.GameStatusShelved, database.GameStatusCompleted}
	libraryQuery{Statuses: backing[:1], Sort: librarySortTitle}.StatusToggles()

	want := []database.GameStatus{database.GameStatusShelved, database.GameStatusCompleted}
	if !slices.Equal(backing, want) {
		t.Errorf("statuses after StatusToggles() = %v, want %v", backing, want)
	}
}

func TestRandomSample(t *testing.T) {
	tests := []struct {
		name    string
		entries []int
		n       int
		wantLen int
	}{
		{name: "n bigger than input returns all", entries: []int{1, 2, 3}, n: 10, wantLen: 3},
		{name: "n equal to input returns all", entries: []int{1, 2, 3}, n: 3, wantLen: 3},
		{name: "n smaller than input returns n", entries: []int{1, 2, 3, 4, 5}, n: 2, wantLen: 2},
		{name: "empty input", entries: []int{}, n: 10, wantLen: 0},
		{name: "nil input", entries: nil, n: 10, wantLen: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := slices.Clone(tt.entries)
			got := randomSample(input, tt.n)

			if len(got) != tt.wantLen {
				t.Fatalf("len(randomSample(%v, %d)) = %d, want %d", tt.entries, tt.n, len(got), tt.wantLen)
			}
			seen := make(map[int]bool, len(got))
			for _, v := range got {
				if !slices.Contains(tt.entries, v) {
					t.Errorf("randomSample(%v, %d) returned %d, not in input", tt.entries, tt.n, v)
				}
				if seen[v] {
					t.Errorf("randomSample(%v, %d) returned %d more than once", tt.entries, tt.n, v)
				}
				seen[v] = true
			}
		})
	}
}

func TestSortLibrary(t *testing.T) {
	e := entryBuilder()
	tests := []struct {
		name    string
		sort    librarySort
		order   sortOrder
		entries []database.LibraryEntry
		want    []string
	}{
		{
			name:  "started newest first, undated last",
			sort:  librarySortStarted,
			order: sortOrderDesc,
			entries: []database.LibraryEntry{
				e("B", "", day(3, 1), nil),
				e("A", "", nil, nil),
				e("C", "", day(1, 1), nil),
			},
			want: []string{"B", "C", "A"},
		},
		{
			name:  "finished newest first, undated last",
			sort:  librarySortFinished,
			order: sortOrderDesc,
			entries: []database.LibraryEntry{
				e("B", "", nil, day(3, 1)),
				e("A", "", nil, nil),
				e("C", "", nil, day(1, 1)),
			},
			want: []string{"B", "C", "A"},
		},
		{
			name:  "same date ties sort by title",
			sort:  librarySortStarted,
			order: sortOrderDesc,
			entries: []database.LibraryEntry{
				e("B", "", day(2, 1), nil),
				e("A", "", day(2, 1), nil),
			},
			want: []string{"A", "B"},
		},
		{
			name:  "added newest first, ties by title",
			sort:  librarySortAdded,
			order: sortOrderDesc,
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
			sort:  librarySortTitle,
			order: sortOrderAsc,
			entries: []database.LibraryEntry{
				e("C", "", nil, nil),
				e("A", "", nil, nil),
				e("D", "", nil, nil),
				e("B", "", nil, nil),
			},
			want: []string{"A", "B", "C", "D"},
		},
		{
			name:  "started oldest first, undated last",
			sort:  librarySortStarted,
			order: sortOrderAsc,
			entries: []database.LibraryEntry{
				e("B", "", day(3, 1), nil),
				e("A", "", nil, nil),
				e("C", "", day(1, 1), nil),
			},
			want: []string{"C", "B", "A"},
		},
		{
			name:  "finished oldest first, undated last",
			sort:  librarySortFinished,
			order: sortOrderAsc,
			entries: []database.LibraryEntry{
				e("B", "", nil, day(3, 1)),
				e("A", "", nil, nil),
				e("C", "", nil, day(1, 1)),
			},
			want: []string{"C", "B", "A"},
		},
		{
			name:  "ascending date ties sort by title",
			sort:  librarySortStarted,
			order: sortOrderAsc,
			entries: []database.LibraryEntry{
				e("B", "", day(2, 1), nil),
				e("A", "", day(2, 1), nil),
			},
			want: []string{"A", "B"},
		},
		{
			name:  "added oldest first, ties by title",
			sort:  librarySortAdded,
			order: sortOrderAsc,
			entries: []database.LibraryEntry{
				addedOn(e("C", "", nil, nil), day(1, 1)),
				addedOn(e("B", "", nil, nil), day(3, 1)),
				addedOn(e("A", "", nil, nil), day(3, 1)),
				addedOn(e("D", "", nil, nil), day(2, 1)),
			},
			want: []string{"C", "D", "A", "B"},
		},
		{
			name:  "title descending",
			sort:  librarySortTitle,
			order: sortOrderDesc,
			entries: []database.LibraryEntry{
				e("C", "", nil, nil),
				e("A", "", nil, nil),
				e("D", "", nil, nil),
				e("B", "", nil, nil),
			},
			want: []string{"D", "C", "B", "A"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := titles(sortLibrary(tt.entries, tt.sort, tt.order))
			if !slices.Equal(got, tt.want) {
				t.Errorf("sortLibrary(%q, %q) = %v, want %v", tt.sort, tt.order, got, tt.want)
			}
		})
	}
}
