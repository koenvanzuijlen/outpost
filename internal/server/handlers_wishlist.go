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

// wishlistPage represents the data needed to render the wishlist page.
type wishlistPage struct {
	Games []dbgen.ListWishlistedUserGamesRow
}

// handleWishlist renders the wishlist page: all wishlisted games, oldest release first and unreleased games last.
func (server *Server) handleWishlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	wishlistedGames, err := server.store.ListWishlist(ctx, defaultUserID)
	if err != nil {
		server.logger.Error("failed to list wishlisted games", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	server.renderComponent(w, r, wishlistView(wishlistPage{
		Games: wishlistedGames,
	}))
}

// handleAddToWishlist adds a game to the wishlist. It reads path value igdbID and redirects to the game's page. A game
// that is already in the library or on the wishlist counts as success.
func (server *Server) handleAddToWishlist(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	game, err := server.igdbClient.GameByID(r.Context(), int(igdbID))
	if err != nil {
		if errors.Is(err, igdb.ErrGameNotFound) {
			http.Error(w, "Game not found", http.StatusNotFound)
		} else {
			server.logger.Error("failed to fetch game", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	if !game.GameType.Standalone() {
		http.Error(w, "This game type can't be added to the library", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	err = server.store.AddToWishlist(r.Context(), upsertGameParamsFromIGDB(game, now), defaultUserID, now)
	if err != nil && !errors.Is(err, database.ErrAlreadyAdded) {
		server.logger.Error("failed to add to wishlist", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/games/"+strconv.FormatInt(igdbID, 10), http.StatusSeeOther)
}
