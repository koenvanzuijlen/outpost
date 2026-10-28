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

// handleLibrary serves the user's library page, displaying library games and wishlisted games.
func (server *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	games, err := server.store.ListLibrary(ctx, defaultUserID)
	if err != nil {
		server.logger.Error("failed to list games", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	wishlistedGames, err := server.store.ListWishlist(ctx, defaultUserID)
	if err != nil {
		server.logger.Error("failed to list wishlisted games", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	server.render(w, "library.html", libraryPage{
		Games:           games,
		WishlistedGames: wishlistedGames,
	})
}

// handleAddToLibrary handles adding a game to the user's library. It expects the game's IGDB ID as form value igdb_id,
// and optionally platform_id: the IGDB ID of the platform the user owns the game on. Without a platform the game is
// added without being owned. A copy that is already owned counts as success.
func (server *Server) handleAddToLibrary(w http.ResponseWriter, r *http.Request) {
	igdbID, err := strconv.Atoi(r.PostFormValue("igdb_id"))
	if err != nil || igdbID <= 0 {
		http.Error(w, "Invalid igdb_id", http.StatusBadRequest)
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
			http.Error(w, "Failed to fetch game", http.StatusInternalServerError)
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

// handleRemoveUserGame handles removing a game from the user's library or wishlist, including its copies, DLC
// ownership, playthroughs and its DLC's library entries (see database.Store.RemoveUserGame). It expects the IGDB ID of
// the game as a path parameter. It is idempotent: a game that is already gone counts as success.
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
