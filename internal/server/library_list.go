package server

import (
	"cmp"
	"slices"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

// librarySort is the type for sorting the library.
type librarySort string

// The possible values for librarySort for sorting the library.
const (
	librarySortTitle    librarySort = "title"
	librarySortStarted  librarySort = "started"
	librarySortFinished librarySort = "finished"
	librarySortAdded    librarySort = "added"
)

// librarySorts contains all possible library sorts.
var librarySorts = []librarySort{
	librarySortTitle,
	librarySortStarted,
	librarySortFinished,
	librarySortAdded,
}

// Valid reports whether ls is a known library sort.
func (ls librarySort) Valid() bool {
	return slices.Contains(librarySorts, ls)
}

// librarySortOption represents a single option for sorting the library, with a value and a human-readable label.
type librarySortOption struct {
	Value librarySort
	Label string
}

// librarySortOptions returns all available library sort options, in the order they should be displayed.
func librarySortOptions() []librarySortOption {
	return []librarySortOption{
		{Value: librarySortTitle, Label: "Title"},
		{Value: librarySortStarted, Label: "Last started"},
		{Value: librarySortFinished, Label: "Last finished"},
		{Value: librarySortAdded, Label: "Recently added"},
	}
}

// filterLibrary returns the entries with the given status, or all entries if status is empty. It modifies entries in
// place.
func filterLibrary(entries []database.LibraryEntry, status database.GameStatus) []database.LibraryEntry {
	if status == "" {
		return entries
	}
	entries = slices.DeleteFunc(entries, func(e database.LibraryEntry) bool {
		return e.UserGame.Status == nil ||
			database.GameStatus(*e.UserGame.Status) != status
	})
	return entries
}

// sortLibrary sorts entries in place by order and returns them. Dates sort newest first with undated entries last; ties
// and the title order are decided by title, then IGDB ID.
func sortLibrary(entries []database.LibraryEntry, order librarySort) []database.LibraryEntry {
	var primarySort func(a, b database.LibraryEntry) int
	switch order {
	case librarySortStarted:
		primarySort = func(a, b database.LibraryEntry) int {
			return compareNewestFirstNilLast(a.LastStartedAt, b.LastStartedAt)
		}
	case librarySortFinished:
		primarySort = func(a, b database.LibraryEntry) int {
			return compareNewestFirstNilLast(a.LastFinishedAt, b.LastFinishedAt)
		}
	case librarySortAdded:
		primarySort = func(a, b database.LibraryEntry) int {
			return b.UserGame.AddedAt.Compare(a.UserGame.AddedAt)
		}
	default:
		primarySort = func(_, _ database.LibraryEntry) int {
			return 0
		}
	}

	slices.SortFunc(entries, func(a, b database.LibraryEntry) int {
		return cmp.Or(
			primarySort(a, b),
			cmp.Compare(a.Game.Title, b.Game.Title),
			cmp.Compare(a.Game.IgdbID, b.Game.IgdbID),
		)
	})
	return entries
}

// compareNewestFirstNilLast compares two optional dates for a newest-first sort, with nil dates last.
func compareNewestFirstNilLast(a, b *time.Time) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}
	return b.Compare(*a)
}
