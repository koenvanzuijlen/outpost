package server

import (
	"cmp"
	"net/url"
	"slices"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// wishlistSort is the type for sorting the wishlist.
type wishlistSort string

// The possible values for wishlistSort.
const (
	wishlistSortReleased wishlistSort = "released"
	wishlistSortTitle    wishlistSort = "title"
	wishlistSortAdded    wishlistSort = "added"
)

// wishlistSorts contains all possible wishlist sorts.
var wishlistSorts = []wishlistSort{
	wishlistSortReleased,
	wishlistSortTitle,
	wishlistSortAdded,
}

// Valid reports whether ws is a known wishlist sort.
func (ws wishlistSort) Valid() bool {
	return slices.Contains(wishlistSorts, ws)
}

// DefaultOrder returns the order for ws when the URL has no order: descending (newest first) for the added date,
// ascending for the release date and the title.
func (ws wishlistSort) DefaultOrder() sortOrder {
	switch ws {
	case wishlistSortReleased, wishlistSortTitle:
		return sortOrderAsc
	case wishlistSortAdded:
		return sortOrderDesc
	default:
		return sortOrderAsc
	}
}

// wishlistSortOption is one option of the wishlist sort select, with a value and a label.
type wishlistSortOption struct {
	Value wishlistSort
	Label string
}

// wishlistSortOptions returns all wishlist sort options, in the order of the select.
func wishlistSortOptions() []wishlistSortOption {
	return []wishlistSortOption{
		{Value: wishlistSortReleased, Label: "Release date"},
		{Value: wishlistSortTitle, Label: "Title"},
		{Value: wishlistSortAdded, Label: "Date added"},
	}
}

// wishlistQuery holds the query values of the wishlist page: the sort and the sort order.
type wishlistQuery struct {
	Sort  wishlistSort
	Order sortOrder
}

// URL returns the wishlist page URL for q.
func (q wishlistQuery) URL() string {
	values := url.Values{}
	values.Set("sort", string(q.Sort))
	values.Set("order", string(q.Order))
	return "/wishlist?" + values.Encode()
}

// OrderToggleURL returns the wishlist page URL for q with the order reversed. q is not changed.
func (q wishlistQuery) OrderToggleURL() string {
	next := q
	next.Order = q.Order.Reverse()
	return next.URL()
}

// sortWishlist sorts games in place by sort in the given order and returns them. Unreleased games are last in both
// orders. Ties are decided by title, then IGDB ID, both ascending.
func sortWishlist(
	games []dbgen.ListWishlistedUserGamesRow,
	sort wishlistSort,
	order sortOrder,
) []dbgen.ListWishlistedUserGamesRow {
	var primarySort func(a, b dbgen.ListWishlistedUserGamesRow) int
	switch sort {
	case wishlistSortReleased:
		primarySort = func(a, b dbgen.ListWishlistedUserGamesRow) int {
			return compareDatesNilLast(a.Game.ReleasedAt, b.Game.ReleasedAt, order)
		}
	case wishlistSortAdded:
		primarySort = func(a, b dbgen.ListWishlistedUserGamesRow) int {
			return order.Apply(a.UserGame.AddedAt.Compare(b.UserGame.AddedAt))
		}
	default:
		primarySort = func(a, b dbgen.ListWishlistedUserGamesRow) int {
			return order.Apply(cmp.Compare(a.Game.Title, b.Game.Title))
		}
	}

	slices.SortFunc(games, func(a, b dbgen.ListWishlistedUserGamesRow) int {
		return cmp.Or(
			primarySort(a, b),
			cmp.Compare(a.Game.Title, b.Game.Title),
			cmp.Compare(a.Game.IgdbID, b.Game.IgdbID),
		)
	})
	return games
}
