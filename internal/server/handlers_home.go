package server

import (
	"net/http"
	"slices"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// homeSampleSize is the number of random games shown in the backlog and wishlist sections of the home page.
const homeSampleSize = 10

// homePage represents the data needed to render the home page: all playing games, a random sample of the backlog and
// the wishlist, and the total size of both.
type homePage struct {
	PlayingGames  []database.LibraryEntry
	BacklogGames  []database.LibraryEntry
	BacklogCount  int
	WishlistGames []dbgen.ListWishlistedUserGamesRow
	WishlistCount int
}

// handleHome renders the home page: playing games by last started, and a random sample of the backlog and wishlist.
func (server *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	games, err := server.store.ListLibrary(ctx, defaultUserID)
	if err != nil {
		server.logger.Error("failed to list library", "error", err)
		server.renderServerError(w, r)
		return
	}
	wishlistedGames, err := server.store.ListWishlist(ctx, defaultUserID)
	if err != nil {
		server.logger.Error("failed to list wishlisted games", "error", err)
		server.renderServerError(w, r)
		return
	}

	playing := sortLibrary(
		filterLibrary(slices.Clone(games), []database.GameStatus{database.GameStatusPlaying}),
		librarySortStarted,
		sortOrderDesc,
	)

	backlog := filterLibrary(slices.Clone(games), []database.GameStatus{database.GameStatusBacklog})
	backlogCount := len(backlog)
	backlog = randomSample(backlog, homeSampleSize)

	wishlistCount := len(wishlistedGames)
	wishlistedGames = randomSample(wishlistedGames, homeSampleSize)

	server.renderComponent(w, r, http.StatusOK, homeView(homePage{
		PlayingGames:  playing,
		BacklogGames:  backlog,
		BacklogCount:  backlogCount,
		WishlistGames: wishlistedGames,
		WishlistCount: wishlistCount,
	}))
}
