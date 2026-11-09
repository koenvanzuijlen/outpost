package server

import (
	"slices"
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// wishlistRowBuilder returns a function that builds wishlist rows with IGDB IDs counting up from 1.
// A nil added gives the zero time.
func wishlistRowBuilder() func(title string, released, added *time.Time) dbgen.ListWishlistedUserGamesRow {
	var nextID int64
	return func(title string, released, added *time.Time) dbgen.ListWishlistedUserGamesRow {
		nextID++
		row := dbgen.ListWishlistedUserGamesRow{
			Game: dbgen.Game{Title: title, IgdbID: nextID, ReleasedAt: released},
		}
		if added != nil {
			row.UserGame.AddedAt = *added
		}
		return row
	}
}

// wishlistTitles returns the titles of rows in order.
func wishlistTitles(rows []dbgen.ListWishlistedUserGamesRow) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Game.Title)
	}
	return out
}

func TestSortWishlist(t *testing.T) {
	released := func() []dbgen.ListWishlistedUserGamesRow {
		r := wishlistRowBuilder()
		return []dbgen.ListWishlistedUserGamesRow{
			r("B", day(time.March, 1), nil),
			r("A", nil, nil),
			r("C", day(time.January, 1), nil),
		}
	}
	added := func() []dbgen.ListWishlistedUserGamesRow {
		r := wishlistRowBuilder()
		return []dbgen.ListWishlistedUserGamesRow{
			r("C", nil, day(time.January, 1)),
			r("B", nil, day(time.March, 1)),
			r("A", nil, day(time.March, 1)),
			r("D", nil, day(time.February, 1)),
		}
	}

	tests := []struct {
		name  string
		sort  wishlistSort
		order sortOrder
		games []dbgen.ListWishlistedUserGamesRow
		want  []string
	}{
		{
			name:  "released oldest first, unreleased last",
			sort:  wishlistSortReleased,
			order: sortOrderAsc,
			games: released(),
			want:  []string{"C", "B", "A"},
		},
		{
			name:  "released newest first, unreleased last",
			sort:  wishlistSortReleased,
			order: sortOrderDesc,
			games: released(),
			want:  []string{"B", "C", "A"},
		},
		{
			name:  "same release date ties by title",
			sort:  wishlistSortReleased,
			order: sortOrderAsc,
			games: func() []dbgen.ListWishlistedUserGamesRow {
				r := wishlistRowBuilder()
				return []dbgen.ListWishlistedUserGamesRow{
					r("B", day(time.February, 1), nil),
					r("A", day(time.February, 1), nil),
				}
			}(),
			want: []string{"A", "B"},
		},
		{
			name:  "added newest first, ties by title",
			sort:  wishlistSortAdded,
			order: sortOrderDesc,
			games: added(),
			want:  []string{"A", "B", "D", "C"},
		},
		{
			name:  "added oldest first",
			sort:  wishlistSortAdded,
			order: sortOrderAsc,
			games: added(),
			want:  []string{"C", "D", "A", "B"},
		},
		{
			name:  "title descending",
			sort:  wishlistSortTitle,
			order: sortOrderDesc,
			games: func() []dbgen.ListWishlistedUserGamesRow {
				r := wishlistRowBuilder()
				return []dbgen.ListWishlistedUserGamesRow{
					r("C", nil, nil),
					r("A", nil, nil),
					r("D", nil, nil),
					r("B", nil, nil),
				}
			}(),
			want: []string{"D", "C", "B", "A"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wishlistTitles(sortWishlist(tt.games, tt.sort, tt.order))
			if !slices.Equal(got, tt.want) {
				t.Errorf("sortWishlist(%q, %q) = %v, want %v", tt.sort, tt.order, got, tt.want)
			}
		})
	}
}
