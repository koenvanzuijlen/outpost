package server

import (
	"cmp"
	"math/rand/v2"
	"net/url"
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

// DefaultOrder returns the order for ls when the URL has no order: ascending for the title, descending (newest first)
// for the dates.
func (ls librarySort) DefaultOrder() sortOrder {
	switch ls {
	case librarySortTitle:
		return sortOrderAsc
	case librarySortStarted, librarySortFinished, librarySortAdded:
		return sortOrderDesc
	default:
		return sortOrderAsc
	}
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
		{Value: librarySortStarted, Label: "Date started"},
		{Value: librarySortFinished, Label: "Date finished"},
		{Value: librarySortAdded, Label: "Date added"},
	}
}

// libraryQuery holds the query values of the library page: the active statuses, the sort and the sort order.
type libraryQuery struct {
	Statuses []database.GameStatus
	Sort     librarySort
	Order    sortOrder
}

// URL returns the library page URL for q. The statuses are in the order of database.GameStatuses, each status once.
func (q libraryQuery) URL() string {
	values := url.Values{}
	for _, status := range database.GameStatuses {
		if slices.Contains(q.Statuses, status) {
			values.Add("status", string(status))
		}
	}
	values.Set("sort", string(q.Sort))
	values.Set("order", string(q.Order))
	return "/library?" + values.Encode()
}

// libraryStatusToggle is a status filter link on the library page. Href is the library URL with Status switched on or
// off.
type libraryStatusToggle struct {
	Status database.GameStatus
	Active bool
	Href   string
}

// StatusToggles returns one toggle for each status in database.GameStatuses. An active toggle links to q without its
// status. An inactive toggle links to q with its status added. q is not changed.
func (q libraryQuery) StatusToggles() []libraryStatusToggle {
	toggles := make([]libraryStatusToggle, 0, len(database.GameStatuses))
	for _, status := range database.GameStatuses {
		active := slices.Contains(q.Statuses, status)

		next := q
		if active {
			next.Statuses = slices.DeleteFunc(slices.Clone(q.Statuses), func(s database.GameStatus) bool {
				return s == status
			})
		} else {
			next.Statuses = append(slices.Clone(q.Statuses), status)
		}

		toggles = append(toggles, libraryStatusToggle{
			Status: status,
			Active: active,
			Href:   next.URL(),
		})
	}
	return toggles
}

// OrderToggleURL returns the library page URL for q with the order reversed. q is not changed.
func (q libraryQuery) OrderToggleURL() string {
	next := q
	next.Order = q.Order.Reverse()
	return next.URL()
}

// filterLibrary returns the entries with one of the given statuses, or all entries if statuses is empty. It modifies
// entries in place.
func filterLibrary(entries []database.LibraryEntry, statuses []database.GameStatus) []database.LibraryEntry {
	if len(statuses) == 0 {
		return entries
	}
	entries = slices.DeleteFunc(entries, func(e database.LibraryEntry) bool {
		return e.UserGame.Status == nil ||
			!slices.Contains(statuses, database.GameStatus(*e.UserGame.Status))
	})
	return entries
}

// sortLibrary sorts entries in place by sort in the given order and returns them. Undated entries are last in both
// orders. Ties are decided by title, then IGDB ID, both ascending.
func sortLibrary(entries []database.LibraryEntry, sort librarySort, order sortOrder) []database.LibraryEntry {
	var primarySort func(a, b database.LibraryEntry) int
	switch sort {
	case librarySortStarted:
		primarySort = func(a, b database.LibraryEntry) int {
			return compareDatesNilLast(a.LastStartedAt, b.LastStartedAt, order)
		}
	case librarySortFinished:
		primarySort = func(a, b database.LibraryEntry) int {
			return compareDatesNilLast(a.LastFinishedAt, b.LastFinishedAt, order)
		}
	case librarySortAdded:
		primarySort = func(a, b database.LibraryEntry) int {
			return order.Apply(a.UserGame.AddedAt.Compare(b.UserGame.AddedAt))
		}
	default:
		primarySort = func(a, b database.LibraryEntry) int {
			return order.Apply(cmp.Compare(a.Game.Title, b.Game.Title))
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

// compareDatesNilLast compares two optional dates in the given order. Nil dates are last in both orders.
func compareDatesNilLast(a, b *time.Time, order sortOrder) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}
	return order.Apply(a.Compare(*b))
}

// randomSample returns a random sample of n entries from the given slice. If n is greater than or equal to the number
// of entries, it returns all entries. The original slice is shuffled in place.
func randomSample[T any](entries []T, n int) []T {
	rand.Shuffle(len(entries), func(i, j int) {
		entries[i], entries[j] = entries[j], entries[i]
	})
	if n >= len(entries) {
		return entries
	}
	return entries[:n]
}
