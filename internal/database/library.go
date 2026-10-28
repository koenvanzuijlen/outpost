package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// ErrUserGameNotFound is returned when a game is not found in the user's library.
var ErrUserGameNotFound = errors.New("game not found in library")

// LibraryGame represents a game in the user's library, including the game details, its DLCs, the user's game entry,
// the platforms the user owns it on, the DLC the user owns per platform, and the playthroughs associated with the game.
type LibraryGame struct {
	Game         dbgen.Game
	UserGame     dbgen.UserGame
	Platforms    []dbgen.Platform
	DLCs         []dbgen.ListDLCGamesRow
	Parent       *dbgen.Game
	CopyDLCs     map[int64][]CopyDLC // key: platform IGDB ID
	Playthroughs []dbgen.ListPlaythroughsForUserGameIDRow
}

// GetLibraryGame retrieves a game from the user's library by its IGDB ID. Returns ErrUserGameNotFound if the game is
// not in the library.
func (s *Store) GetLibraryGame(ctx context.Context, userID, igdbID int64) (LibraryGame, error) {
	row, err := s.queries.GetUserGameByIGDBID(ctx, dbgen.GetUserGameByIGDBIDParams{
		UserID: userID,
		IgdbID: igdbID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return LibraryGame{}, ErrUserGameNotFound
		}
		return LibraryGame{}, fmt.Errorf("get user game: %w", err)
	}

	isDLC := igdb.GameType(row.Game.GameType).DLC()

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
			return LibraryGame{}, fmt.Errorf("list dlc games: %w", err)
		}

		platforms, err = s.queries.ListPlatformsForUserGameID(ctx, row.UserGame.ID)
		if err != nil {
			return LibraryGame{}, fmt.Errorf("list owned platforms: %w", err)
		}

		ownedCopyDLCs, err := s.queries.ListOwnedCopyDLCs(ctx, dbgen.ListOwnedCopyDLCsParams{
			UserGameID:   row.UserGame.ID,
			ParentIgdbID: &igdbID,
			GameTypes:    dlcGameTypes(),
		})
		if err != nil {
			return LibraryGame{}, fmt.Errorf("list owned copy dlcs: %w", err)
		}
		for _, ownedCopyDLC := range ownedCopyDLCs {
			copyDLCs[ownedCopyDLC.PlatformIgdbID] = append(copyDLCs[ownedCopyDLC.PlatformIgdbID], CopyDLC{
				Game:  ownedCopyDLC.Game,
				Owned: ownedCopyDLC.Owned,
			})
		}
	} else {
		if row.Game.ParentIgdbID == nil {
			return LibraryGame{}, fmt.Errorf("dlc %d has no parent", row.Game.ID)
		}
		parent, err := s.queries.GetGameByIGDBID(ctx, *row.Game.ParentIgdbID)
		if err != nil {
			return LibraryGame{}, fmt.Errorf("get parent game: %w", err)
		}
		parentGame = &parent

		platforms, err = s.queries.ListPlatformsForOwnedDLC(ctx, dbgen.ListPlatformsForOwnedDLCParams{
			UserID:    userID,
			DlcGameID: row.Game.ID,
		})
		if err != nil {
			return LibraryGame{}, fmt.Errorf("list owned platforms dlc: %w", err)
		}
	}

	playthroughs, err := s.queries.ListPlaythroughsForUserGameID(ctx, row.UserGame.ID)
	if err != nil {
		return LibraryGame{}, fmt.Errorf("list playthroughs: %w", err)
	}

	return LibraryGame{
		Game:         row.Game,
		DLCs:         dlcs,
		UserGame:     row.UserGame,
		Platforms:    platforms,
		Playthroughs: playthroughs,
		CopyDLCs:     copyDLCs,
		Parent:       parentGame,
	}, nil
}

// GameDetails represents the details of a game in the user's library, including its status, rating, and notes.
type GameDetails struct {
	Status GameStatus
	Rating *int64
	Notes  *string
}

// UpdateGameDetails updates the status, rating and notes of a game in the user's library. A nil value clears the field. Returns ErrUserGameNotFound
// if the game is not in the library.
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
		return fmt.Errorf("update user game detailsTestUpdateGameDetails: %w", err)
	}
	if updatedRows == 0 {
		return ErrUserGameNotFound
	}
	return nil
}

// RemoveFromLibrary deletes the user's library entry for a game. What else goes with it depends on the entry:
//   - An owned game: its copies, the DLC ownership on those copies, its playthroughs, and the entries of its tracked
//     DLC (in the same transaction, since no foreign key links them).
//   - A tracked DLC: its status, rating and playthroughs. Ownership of the DLC on the base game's copies is kept.
//   - A wishlisted game: only the wishlist entry.
//
// Catalog data (games, platforms, cached DLC lists) is never deleted, so adding the game again needs no IGDB call for
// its DLC. Returns ErrUserGameNotFound if the game is not in the library.
func (s *Store) RemoveFromLibrary(ctx context.Context, userID, igdbID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	row, err := q.GetUserGameByIGDBID(ctx, dbgen.GetUserGameByIGDBIDParams{
		UserID: userID,
		IgdbID: igdbID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserGameNotFound
		}
		return fmt.Errorf("get user game by igdb id: %w", err)
	}

	if !igdb.GameType(row.Game.GameType).DLC() {
		err = q.DeleteTrackedDLCs(ctx, dbgen.DeleteTrackedDLCsParams{
			UserID:       userID,
			ParentIgdbID: &igdbID,
			DlcGameTypes: dlcGameTypes(),
		})
		if err != nil {
			return fmt.Errorf("delete tracked dlcs: %w", err)
		}
	}

	err = q.DeleteUserGame(ctx, row.UserGame.ID)
	if err != nil {
		return fmt.Errorf("delete user game: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// getOwnedUserGame retrieves a user's owned game by its IGDB ID, excluding wishlisted games. Returns
// ErrUserGameNotFound if the game is not found or is wishlisted.
func getOwnedUserGame(ctx context.Context, q *dbgen.Queries, userID, igdbID int64) (dbgen.GetUserGameByIGDBIDRow, error) {
	row, err := q.GetUserGameByIGDBID(ctx, dbgen.GetUserGameByIGDBIDParams{UserID: userID, IgdbID: igdbID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dbgen.GetUserGameByIGDBIDRow{}, ErrUserGameNotFound
		}
		return dbgen.GetUserGameByIGDBIDRow{}, fmt.Errorf("get user game by igdb id: %w", err)
	}
	if row.UserGame.Wishlisted {
		return dbgen.GetUserGameByIGDBIDRow{}, ErrUserGameNotFound
	}
	return row, nil
}
