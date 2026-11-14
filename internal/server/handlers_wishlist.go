package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// wishlistPage represents the data needed to render the wishlist page: the sorted games, the current and available
// sorts, and the order with the link that reverses it.
type wishlistPage struct {
	Games          []dbgen.ListWishlistedUserGamesRow
	Sort           wishlistSort
	SortOptions    []wishlistSortOption
	Order          sortOrder
	OrderToggleURL string
}

// handleWishlist renders the wishlist page with all wishlisted games. It reads query values sort (default released) and
// order (asc or desc; default from wishlistSort.DefaultOrder), and returns 400 for an unknown value.
func (server *Server) handleWishlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sort := wishlistSort(r.URL.Query().Get("sort"))
	if sort == "" {
		sort = wishlistSortReleased
	}
	if !sort.Valid() {
		server.renderError(w, r, http.StatusBadRequest, "Invalid sort")
		return
	}

	order := sortOrder(r.URL.Query().Get("order"))
	if order == "" {
		order = sort.DefaultOrder()
	}
	if !order.Valid() {
		server.renderError(w, r, http.StatusBadRequest, "Invalid order")
		return
	}

	wishlistedGames, err := server.store.ListWishlist(ctx, defaultUserID)
	if err != nil {
		server.logger.Error("failed to list wishlisted games", "error", err)
		server.renderServerError(w, r)
		return
	}

	query := wishlistQuery{
		Sort:  sort,
		Order: order,
	}

	wishlistedGames = sortWishlist(wishlistedGames, sort, order)

	server.renderComponent(w, r, http.StatusOK, wishlistView(wishlistPage{
		Games:          wishlistedGames,
		Sort:           sort,
		SortOptions:    wishlistSortOptions(),
		Order:          order,
		OrderToggleURL: query.OrderToggleURL(),
	}))
}

// handleAddToWishlist adds a game to the wishlist. It reads path value igdbID and redirects to the game's page. A game
// that is already in the library or on the wishlist counts as success.
func (server *Server) handleAddToWishlist(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid IGDB ID")
		return
	}

	game, err := server.igdbClient.GameByID(r.Context(), int(igdbID))
	if err != nil {
		if errors.Is(err, igdb.ErrGameNotFound) {
			server.renderError(w, r, http.StatusNotFound, "Game not found")
		} else {
			server.logger.Error("failed to fetch game", "error", err)
			server.renderServerError(w, r)
		}
		return
	}

	if !game.GameType.Standalone() {
		server.renderError(w, r, http.StatusBadRequest, "This game type can't be added to the library")
		return
	}

	now := time.Now().UTC()
	err = server.store.AddToWishlist(r.Context(), upsertGameParamsFromIGDB(game, now), defaultUserID, now)
	if err != nil && !errors.Is(err, database.ErrAlreadyAdded) {
		server.logger.Error("failed to add to wishlist", "error", err)
		server.renderServerError(w, r)
		return
	}
	http.Redirect(w, r, "/games/"+strconv.FormatInt(igdbID, 10), http.StatusSeeOther)
}
