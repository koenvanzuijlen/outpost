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
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	wishlistedGames, err := server.store.ListWishlist(ctx, defaultUserID)
	if err != nil {
		server.logger.Error("failed to list wishlisted games", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	playing := sortLibrary(
		filterLibrary(slices.Clone(games), database.GameStatusPlaying),
		librarySortStarted,
	)

	backlog := filterLibrary(slices.Clone(games), database.GameStatusBacklog)
	backlogCount := len(backlog)
	backlog = randomSample(backlog, homeSampleSize)

	wishlistCount := len(wishlistedGames)
	wishlistedGames = randomSample(wishlistedGames, homeSampleSize)

	server.render(w, "home.html", homePage{
		PlayingGames:  playing,
		BacklogGames:  backlog,
		BacklogCount:  backlogCount,
		WishlistGames: wishlistedGames,
		WishlistCount: wishlistCount,
	})
}

// libraryPage represents the data needed to render the library page: the filtered and sorted entries, and the current
// and available values of the status and sort selects.
type libraryPage struct {
	Games         []database.LibraryEntry
	Status        database.GameStatus
	StatusOptions []database.GameStatus
	Sort          librarySort
	SortOptions   []librarySortOption
}

// handleLibrary renders the library page. It reads query values status (empty for all statuses) and sort (default title),
// and returns 400 for an unknown value.
func (server *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status := database.GameStatus(r.URL.Query().Get("status"))
	if status != "" && !status.Valid() {
		http.Error(w, "Invalid game status", http.StatusBadRequest)
		return
	}

	sort := librarySort(r.URL.Query().Get("sort"))
	if sort == "" {
		sort = librarySortTitle
	}
	if !sort.Valid() {
		http.Error(w, "Invalid sort", http.StatusBadRequest)
		return
	}

	games, err := server.store.ListLibrary(ctx, defaultUserID)
	if err != nil {
		server.logger.Error("failed to list library", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	games = filterLibrary(games, status)
	games = sortLibrary(games, sort)

	server.render(w, "library.html", libraryPage{
		Games:         games,
		Status:        status,
		StatusOptions: database.GameStatuses,
		Sort:          sort,
		SortOptions:   librarySortOptions(),
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
