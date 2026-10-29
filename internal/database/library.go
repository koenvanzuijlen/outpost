package database

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// ErrUserGameNotFound is returned when a game is not in the user's library or wishlist, or when an action needs a
// library entry and the game is only on the wishlist.
var ErrUserGameNotFound = errors.New("game not found in library")

// GameOverview is a game in the user's library or wishlist with everything the game page shows: the game, the
// user's entry, the platforms it is owned on, its DLC (with library state and ownership per copy), its parent for DLC,
// and its playthroughs.
type GameOverview struct {
	Game           dbgen.Game
	UserGame       dbgen.UserGame
	OwnedPlatforms []dbgen.Platform
	DLCs           []dbgen.ListDLCGamesRow
	Parent         *dbgen.Game
	CopyDLCs       map[int64][]CopyDLC // key: platform IGDB ID
	Playthroughs   []dbgen.ListPlaythroughsForUserGameIDRow
}

// GetGame retrieves a game from the user's library or wishlist by its IGDB ID. Returns ErrUserGameNotFound if the
// game is in neither.
func (s *Store) GetGame(ctx context.Context, userID, igdbID int64) (GameOverview, error) {
	entry, err := s.queries.GetUserGameByIGDBID(ctx, dbgen.GetUserGameByIGDBIDParams{
		UserID: userID,
		IgdbID: igdbID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GameOverview{}, ErrUserGameNotFound
		}
		return GameOverview{}, fmt.Errorf("get user game: %w", err)
	}

	isDLC := igdb.GameType(entry.Game.GameType).DLC()

	var dlcs []dbgen.ListDLCGamesRow
	copyDLCs := make(map[int64][]CopyDLC)
	var parentGame *dbgen.Game
	var platforms []dbgen.Platform
	if !isDLC {
		dlcs, err = s.queries.ListDLCGames(ctx, dbgen.ListDLCGamesParams{
			UserID:       userID,
			ParentIgdbID: &igdbID,
			GameTypes:    dlcGameTypes(),
		})
		if err != nil {
			return GameOverview{}, fmt.Errorf("list dlc games: %w", err)
		}

		platforms, err = s.queries.ListPlatformsForUserGameID(ctx, entry.UserGame.ID)
		if err != nil {
			return GameOverview{}, fmt.Errorf("list owned platforms: %w", err)
		}

		ownedCopyDLCs, err := s.queries.ListOwnedCopyDLCs(ctx, dbgen.ListOwnedCopyDLCsParams{
			UserGameID:   entry.UserGame.ID,
			ParentIgdbID: &igdbID,
			GameTypes:    dlcGameTypes(),
		})
		if err != nil {
			return GameOverview{}, fmt.Errorf("list owned copy dlcs: %w", err)
		}
		for _, ownedCopyDLC := range ownedCopyDLCs {
			copyDLCs[ownedCopyDLC.PlatformIgdbID] = append(copyDLCs[ownedCopyDLC.PlatformIgdbID], CopyDLC{
				Game:  ownedCopyDLC.Game,
				Owned: ownedCopyDLC.Owned,
			})
		}
	} else {
		if entry.Game.ParentIgdbID == nil {
			return GameOverview{}, fmt.Errorf("dlc %d has no parent", entry.Game.ID)
		}
		parent, err := s.queries.GetGameByIGDBID(ctx, *entry.Game.ParentIgdbID)
		if err != nil {
			return GameOverview{}, fmt.Errorf("get parent game: %w", err)
		}
		parentGame = &parent

		platforms, err = s.queries.ListPlatformsForOwnedDLC(ctx, dbgen.ListPlatformsForOwnedDLCParams{
			UserID:    userID,
			DlcGameID: entry.Game.ID,
		})
		if err != nil {
			return GameOverview{}, fmt.Errorf("list owned platforms dlc: %w", err)
		}
	}

	playthroughs, err := s.queries.ListPlaythroughsForUserGameID(ctx, entry.UserGame.ID)
	if err != nil {
		return GameOverview{}, fmt.Errorf("list playthroughs: %w", err)
	}

	return GameOverview{
		Game:           entry.Game,
		DLCs:           dlcs,
		UserGame:       entry.UserGame,
		OwnedPlatforms: platforms,
		Playthroughs:   playthroughs,
		CopyDLCs:       copyDLCs,
		Parent:         parentGame,
	}, nil
}

// LibraryEntry is one game in the user's library, with all platforms it is owned on (for DLC: the base game's copies it
// is owned on). LastStartedAt and LastFinishedAt are the latest playthrough dates, or nil if there are none.
type LibraryEntry struct {
	Game           dbgen.Game
	UserGame       dbgen.UserGame
	Platforms      []string
	LastStartedAt  *time.Time
	LastFinishedAt *time.Time
}

// ListLibrary returns the user's library (games and DLC, owned or not, excluding the wishlist), sorted by title. A game
// owned on several platforms appears once with a list of platforms, and the platform list can be empty.
func (s *Store) ListLibrary(ctx context.Context, userID int64) ([]LibraryEntry, error) {
	games, err := s.queries.ListLibraryGames(ctx, dbgen.ListLibraryGamesParams{
		UserID:       userID,
		DlcGameTypes: dlcGameTypes(),
	})
	if err != nil {
		return nil, fmt.Errorf("list library games: %w", err)
	}

	dlcs, err := s.queries.ListLibraryDLCs(ctx, dbgen.ListLibraryDLCsParams{
		UserID:    userID,
		GameTypes: dlcGameTypes(),
	})
	if err != nil {
		return nil, fmt.Errorf("list library dlcs: %w", err)
	}

	// Group library games by user game ID, so that each game appears once with all its platforms aggregated.
	library := make([]LibraryEntry, 0, len(games)+len(dlcs))
	indexByUserGame := make(map[int64]int)
	add := func(ug dbgen.UserGame, g dbgen.Game, platform *string) {
		idx, ok := indexByUserGame[ug.ID]
		if !ok {
			idx = len(library)
			library = append(library, LibraryEntry{
				Game:      g,
				UserGame:  ug,
				Platforms: []string{},
			})
			indexByUserGame[ug.ID] = idx
		}
		if platform != nil {
			library[idx].Platforms = append(library[idx].Platforms, *platform)
		}
	}

	for _, game := range games {
		add(game.UserGame, game.Game, game.PlatformName)
	}
	for _, dlc := range dlcs {
		add(dlc.UserGame, dlc.Game, dlc.PlatformName)
	}

	for _, entry := range library {
		slices.Sort(entry.Platforms)
	}

	// Fill in the last started and finished dates for each library entry based on playthroughs.
	playthroughDates, err := s.queries.ListPlaythroughDates(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list playthrough dates: %w", err)
	}

	for _, pt := range playthroughDates {
		idx, ok := indexByUserGame[pt.UserGameID]
		if !ok {
			continue
		}
		entry := &library[idx]
		entry.LastStartedAt = latest(entry.LastStartedAt, pt.StartedAt)
		entry.LastFinishedAt = latest(entry.LastFinishedAt, pt.FinishedAt)
	}

	slices.SortFunc(library, func(a, b LibraryEntry) int {
		return cmp.Or(
			cmp.Compare(a.Game.Title, b.Game.Title),
			cmp.Compare(a.Game.IgdbID, b.Game.IgdbID),
		)
	})

	return library, nil
}

// AddToLibrary adds a game to the user's library, storing it in the catalog if it isn't there yet. A new library entry
// gets status backlog; an existing one keeps its status and is taken off the wishlist. If platform is not nil, the
// user's copy on that platform is added too. Without a platform the game is in the library without being owned, and
// adding it again is a no-op. Returns ErrAlreadyOwnedOnPlatform if the copy already exists.
func (s *Store) AddToLibrary(
	ctx context.Context,
	game dbgen.UpsertGameParams,
	platform *dbgen.UpsertPlatformParams,
	userID int64,
	addedAt time.Time,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	storedGame, err := q.UpsertGame(ctx, game)
	if err != nil {
		return fmt.Errorf("upsert game: %w", err)
	}

	userGame, err := q.UpsertLibraryUserGame(ctx, dbgen.UpsertLibraryUserGameParams{
		UserID:  userID,
		GameID:  storedGame.ID,
		Status:  GameStatusBacklog.StringPtr(),
		AddedAt: addedAt,
	})
	if err != nil {
		return fmt.Errorf("upsert user game: %w", err)
	}

	if platform != nil {
		storedPlatform, err := q.UpsertPlatform(ctx, *platform)
		if err != nil {
			return fmt.Errorf("upsert platform: %w", err)
		}

		_, err = q.CreateOwnedCopy(ctx, dbgen.CreateOwnedCopyParams{
			UserGameID: userGame.ID,
			PlatformID: storedPlatform.ID,
		})
		if err != nil {
			var sqliteErr *sqlite.Error
			if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
				return ErrAlreadyOwnedOnPlatform
			}
			return fmt.Errorf("create owned copy: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// GameDetails are the fields of a library game that UpdateGameDetails overwrites. A nil field is cleared.
type GameDetails struct {
	Status GameStatus
	Rating *int64
	Notes  *string
}

// UpdateGameDetails updates the status, rating and notes of a library game. Every field is overwritten; nil clears it.
// Returns ErrInvalidGameStatus if the status is unknown, or ErrUserGameNotFound if the game isn't in the library.
func (s *Store) UpdateGameDetails(ctx context.Context, userID, igdbID int64, details GameDetails) error {
	if !details.Status.Valid() {
		return ErrInvalidGameStatus
	}
	updatedRows, err := s.queries.UpdateUserGameDetails(ctx, dbgen.UpdateUserGameDetailsParams{
		Status: details.Status.StringPtr(),
		Rating: details.Rating,
		Notes:  details.Notes,
		UserID: userID,
		IgdbID: igdbID,
	})
	if err != nil {
		return fmt.Errorf("update user game details: %w", err)
	}
	if updatedRows == 0 {
		return ErrUserGameNotFound
	}
	return nil
}

// RemoveUserGame removes a game from the user's library or wishlist. What else goes with it depends on the entry:
//   - A library game: its copies, the DLC ownership on those copies, its playthroughs, and the library entries of its
//     DLC (in the same transaction, since no foreign key links them).
//   - A library DLC: its status, rating and playthroughs. Ownership of the DLC on the base game's copies is kept.
//   - A wishlisted game: only the wishlist entry.
//
// Catalog data (games, platforms, cached DLC lists) is never deleted, so adding the game again needs no IGDB call for
// its DLC. Returns ErrUserGameNotFound if the game is in neither the library nor the wishlist.
func (s *Store) RemoveUserGame(ctx context.Context, userID, igdbID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	entry, err := q.GetUserGameByIGDBID(ctx, dbgen.GetUserGameByIGDBIDParams{
		UserID: userID,
		IgdbID: igdbID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserGameNotFound
		}
		return fmt.Errorf("get user game by igdb id: %w", err)
	}

	if !igdb.GameType(entry.Game.GameType).DLC() {
		err = q.DeleteLibraryDLCs(ctx, dbgen.DeleteLibraryDLCsParams{
			UserID:       userID,
			ParentIgdbID: &igdbID,
			DlcGameTypes: dlcGameTypes(),
		})
		if err != nil {
			return fmt.Errorf("delete library dlcs: %w", err)
		}
	}

	err = q.DeleteUserGame(ctx, entry.UserGame.ID)
	if err != nil {
		return fmt.Errorf("delete user game: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// getLibraryUserGame retrieves a user's library entry by its IGDB ID, owned or not. Returns ErrUserGameNotFound if the
// game is not in the library, including when it is only on the wishlist.
func getLibraryUserGame(ctx context.Context, q *dbgen.Queries, userID, igdbID int64) (dbgen.UserGame, error) {
	entry, err := q.GetUserGameByIGDBID(ctx, dbgen.GetUserGameByIGDBIDParams{UserID: userID, IgdbID: igdbID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dbgen.UserGame{}, ErrUserGameNotFound
		}
		return dbgen.UserGame{}, fmt.Errorf("get user game by igdb id: %w", err)
	}
	if entry.UserGame.Wishlisted {
		return dbgen.UserGame{}, ErrUserGameNotFound
	}
	return entry.UserGame, nil
}
