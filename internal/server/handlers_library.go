package server

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

type libraryPage struct {
	Games           []database.LibraryEntry
	WishlistedGames []dbgen.ListWishlistedUserGamesRow
}

// handleLibrary renders the home page: the library and the wishlist.
func (server *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	games, err := server.store.ListLibrary(ctx, defaultUserID)
	if err != nil {
		server.logger.Error("failed to list library", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	wishlistedGames, err := server.store.ListWishlist(ctx, defaultUserID)
	if err != nil {
		server.logger.Error("failed to list wishlisted games", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	server.render(w, "library.html", libraryPage{
		Games:           games,
		WishlistedGames: wishlistedGames,
	})
}

// handleAddToLibrary adds a game to the library. It reads form values igdb_id and platform_id (optional; without it the
// game is added as not owned), and redirects to the home page. A copy that is already owned counts as success.
func (server *Server) handleAddToLibrary(w http.ResponseWriter, r *http.Request) {
	igdbID, err := strconv.Atoi(r.PostFormValue("igdb_id"))
	if err != nil || igdbID <= 0 {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	platformIGDBID, ok := formOptionalPositiveInt(r, "platform_id")
	if !ok {
		http.Error(w, "Invalid platform", http.StatusBadRequest)
		return
	}

	game, err := server.igdbClient.GameByID(r.Context(), igdbID)
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
	var platform *dbgen.UpsertPlatformParams
	if platformIGDBID != nil {
		i := slices.IndexFunc(game.Platforms, func(p igdb.Platform) bool { return int64(p.ID) == *platformIGDBID })
		if i == -1 {
			http.Error(w, "Game not available for this platform", http.StatusBadRequest)
			return
		}
		platform = new(upsertPlatformParamsFromIGDB(game.Platforms[i], now))
	}

	err = server.store.AddToLibrary(r.Context(), upsertGameParamsFromIGDB(game, now), platform, defaultUserID, now)
	if err != nil && !errors.Is(err, database.ErrAlreadyOwnedOnPlatform) {
		server.logger.Error("failed to add to library", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleRemoveUserGame removes a game from the library or wishlist, see database.Store.RemoveUserGame for what goes
// with it. It reads path value igdbID and redirects to the home page. A game that is already gone counts as success.
func (server *Server) handleRemoveUserGame(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		http.Error(w, "Invalid IGDB ID", http.StatusBadRequest)
		return
	}

	err := server.store.RemoveUserGame(r.Context(), defaultUserID, igdbID)
	if err != nil && !errors.Is(err, database.ErrUserGameNotFound) {
		server.logger.Error("failed to remove from library", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
