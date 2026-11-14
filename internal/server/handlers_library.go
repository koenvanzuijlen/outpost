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

// libraryPage represents the data needed to render the library page: the filtered and sorted entries, the active
// statuses with a toggle link for each status, the current and available sorts, and the order with the link that
// reverses it.
type libraryPage struct {
	Games          []database.LibraryEntry
	Statuses       []database.GameStatus
	StatusToggles  []libraryStatusToggle
	Sort           librarySort
	SortOptions    []librarySortOption
	Order          sortOrder
	OrderToggleURL string
}

// handleLibrary renders the library page. It reads query values status (repeatable; none means all statuses, empty
// values are skipped), sort (default title) and order (asc or desc; default from librarySort.DefaultOrder), and
// returns 400 for an unknown value.
func (server *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	statusQuery := r.URL.Query()["status"]
	selectedStatus := make([]database.GameStatus, 0, len(statusQuery))
	for _, s := range statusQuery {
		if s == "" {
			continue
		}
		if !database.GameStatus(s).Valid() {
			server.renderError(w, r, http.StatusBadRequest, "Invalid game status")
			return
		}
		selectedStatus = append(selectedStatus, database.GameStatus(s))
	}

	sort := librarySort(r.URL.Query().Get("sort"))
	if sort == "" {
		sort = librarySortTitle
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

	games, err := server.store.ListLibrary(ctx, defaultUserID)
	if err != nil {
		server.logger.Error("failed to list library", "error", err)
		server.renderServerError(w, r)
		return
	}

	query := libraryQuery{
		Statuses: selectedStatus,
		Sort:     sort,
		Order:    order,
	}

	games = filterLibrary(games, selectedStatus)
	games = sortLibrary(games, sort, order)

	server.renderComponent(w, r, http.StatusOK, libraryView(libraryPage{
		Games:          games,
		Statuses:       selectedStatus,
		StatusToggles:  query.StatusToggles(),
		Sort:           sort,
		SortOptions:    librarySortOptions(),
		Order:          order,
		OrderToggleURL: query.OrderToggleURL(),
	}))
}

// handleAddToLibrary adds a game or a DLC to the library. It reads path value igdbID and, for a game, form value
// platform_id (optional; without it the game is added as not owned). A DLC is added separately from its base game, see
// database.Store.AddDLCToLibrary; it takes no platform_id. Other game types, such as bundles, get 400. It redirects to
// the game's page. A copy that is already owned counts as success.
func (server *Server) handleAddToLibrary(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid IGDB ID")
		return
	}

	platformIGDBID, ok := formOptionalPositiveInt(r, "platform_id")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid platform")
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

	isStandalone := game.GameType.Standalone()
	isDLC := game.GameType.DLC()
	if !isStandalone && !isDLC {
		server.renderError(w, r, http.StatusBadRequest, "This game type can't be added to the library")
		return
	}

	now := time.Now().UTC()
	if isStandalone {
		var platform *dbgen.UpsertPlatformParams
		if platformIGDBID != nil {
			i := slices.IndexFunc(game.Platforms, func(p igdb.Platform) bool { return int64(p.ID) == *platformIGDBID })
			if i == -1 {
				server.renderError(w, r, http.StatusBadRequest, "Game not available for this platform")
				return
			}
			platform = new(upsertPlatformParamsFromIGDB(game.Platforms[i], now))
		}

		err = server.store.AddToLibrary(r.Context(), upsertGameParamsFromIGDB(game, now), platform, defaultUserID, now)
		if err != nil && !errors.Is(err, database.ErrAlreadyOwnedOnPlatform) {
			server.logger.Error("failed to add to library", "error", err)
			server.renderServerError(w, r)
			return
		}
	} else {
		if game.ParentGameID == 0 {
			server.renderError(w, r, http.StatusBadRequest, "DLC has no base game")
			return
		}
		if platformIGDBID != nil {
			server.renderError(w, r, http.StatusBadRequest, "Invalid platform")
			return
		}

		err := server.store.AddDLCToLibrary(r.Context(), defaultUserID, int64(game.ParentGameID), int64(game.ID), now)
		if err != nil {
			if errors.Is(err, database.ErrUserGameNotFound) {
				server.renderError(w, r, http.StatusNotFound, "Base game not in library")
			} else if errors.Is(err, database.ErrDLCNotFound) {
				server.renderError(w, r, http.StatusNotFound, "DLC not found")
			} else {
				server.logger.Error("failed to add DLC to library", "error", err)
				server.renderServerError(w, r)
			}
			return
		}
	}
	http.Redirect(w, r, "/games/"+strconv.FormatInt(igdbID, 10), http.StatusSeeOther)

}

// handleRemoveFromWishlist removes a game from the wishlist. It reads path value igdbID and redirects to the home page.
// A game that isn't on the wishlist counts as success and is left as it is.
func (server *Server) handleRemoveFromWishlist(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid IGDB ID")
		return
	}

	err := server.store.RemoveUserGame(r.Context(), defaultUserID, igdbID, true)
	if err != nil && !errors.Is(err, database.ErrUserGameNotFound) {
		server.logger.Error("failed to remove from wishlist", "error", err)
		server.renderServerError(w, r)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleRemoveFromLibrary removes a game from the library, see database.Store.RemoveUserGame for what goes with it. It
// reads path value igdbID and redirects to the home page. A game that isn't in the library counts as success and is
// left as it is.
func (server *Server) handleRemoveFromLibrary(w http.ResponseWriter, r *http.Request) {
	igdbID, ok := pathID(r, "igdbID")
	if !ok {
		server.renderError(w, r, http.StatusBadRequest, "Invalid IGDB ID")
		return
	}

	err := server.store.RemoveUserGame(r.Context(), defaultUserID, igdbID, false)
	if err != nil && !errors.Is(err, database.ErrUserGameNotFound) {
		server.logger.Error("failed to remove from library", "error", err)
		server.renderServerError(w, r)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
