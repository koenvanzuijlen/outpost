package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// ErrUserGameNotFound is returned when a game is not found in the user's library.
var ErrUserGameNotFound = errors.New("game not found in library")

// LibraryGame represents a game in the user's library, including the game details, its DLCs, the user's game entry,
// the platforms the user owns it on, the DLC the user owns per platform, and the playthroughs associated with the game.
type LibraryGame struct {
	Game         dbgen.Game
	UserGame     dbgen.UserGame
	Platforms    []dbgen.Platform
	DLCs         []dbgen.Game
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

	dlcs, err := s.queries.ListDLCGames(ctx, dbgen.ListDLCGamesParams{
		ParentIgdbID: &igdbID,
		GameTypes:    dlcGameTypes(),
	})
	if err != nil {
		return LibraryGame{}, fmt.Errorf("list dlc games: %w", err)
	}

	platforms, err := s.queries.ListPlatformsForUserGameID(ctx, row.UserGame.ID)
	if err != nil {
		return LibraryGame{}, fmt.Errorf("list owned platforms: %w", err)
	}

	playthroughs, err := s.queries.ListPlaythroughsForUserGameID(ctx, row.UserGame.ID)
	if err != nil {
		return LibraryGame{}, fmt.Errorf("list playthroughs: %w", err)
	}

	copyDLCs := make(map[int64][]CopyDLC)
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

	return LibraryGame{
		Game:         row.Game,
		DLCs:         dlcs,
		UserGame:     row.UserGame,
		Platforms:    platforms,
		Playthroughs: playthroughs,
		CopyDLCs:     copyDLCs,
	}, nil
}

// UpdateStatusAndRating updates the status and rating of a game in the user's library. Returns ErrUserGameNotFound if
// the game is not in the library.
func (s *Store) UpdateStatusAndRating(ctx context.Context, userID, igdbID int64, status GameStatus, rating *int64) error {
	if !status.Valid() {
		return ErrInvalidGameStatus
	}
	updatedRows, err := s.queries.UpdateUserGameStatusAndRating(ctx, dbgen.UpdateUserGameStatusAndRatingParams{
		Status: status.StringPtr(),
		Rating: rating,
		UserID: userID,
		IgdbID: igdbID,
	})
	if err != nil {
		return fmt.Errorf("update status and rating: %w", err)
	}
	if updatedRows == 0 {
		return ErrUserGameNotFound
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
