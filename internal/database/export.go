package database

import (
	"context"
	"fmt"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// ExportCopy is one owned copy: its platform and the DLC owned on it, sorted by title.
type ExportCopy struct {
	Platform dbgen.Platform
	OwnedDLC []dbgen.Game
}

// ExportGame is one library or wishlist entry with its catalog game, its copies and its playthroughs.
// Copies are sorted by platform name. Playthroughs are sorted by start date, oldest first, with undated
// playthroughs last. A wishlisted game and a library DLC have no copies, and a wishlisted game has no playthroughs.
type ExportGame struct {
	Game         dbgen.Game
	UserGame     dbgen.UserGame
	Copies       []ExportCopy
	Playthroughs []dbgen.ListPlaythroughsForUserIDRow
}

// ExportData is all data of one user, as the export writes it.
// Library and Wishlist are sorted by title, then by IGDB ID.
type ExportData struct {
	Library  []ExportGame
	Wishlist []ExportGame
}

// UserExportData is the export data of one user, with the user.
type UserExportData struct {
	User User
	Data ExportData
}

// Export returns all library and wishlist entries of the user, with their copies, the DLC owned on each copy and
// their playthroughs.
func (s *Store) Export(ctx context.Context, userID int64) (ExportData, error) {
	userGames, err := s.queries.ListUserGamesForUserID(ctx, userID)
	if err != nil {
		return ExportData{}, fmt.Errorf("list user games: %w", err)
	}

	ownedCopies, err := s.queries.ListOwnedCopiesForUserID(ctx, userID)
	if err != nil {
		return ExportData{}, fmt.Errorf("list owned copies: %w", err)
	}

	ownedDLC, err := s.queries.ListOwnedDLCForUserID(ctx, userID)
	if err != nil {
		return ExportData{}, fmt.Errorf("list owned dlc: %w", err)
	}

	playthroughs, err := s.queries.ListPlaythroughsForUserID(ctx, userID)
	if err != nil {
		return ExportData{}, fmt.Errorf("list playthroughs: %w", err)
	}

	dlcByCopy := make(map[int64][]dbgen.Game)
	for _, dlc := range ownedDLC {
		dlcByCopy[dlc.OwnedCopyID] = append(dlcByCopy[dlc.OwnedCopyID], dlc.Game)
	}

	copiesByUserGame := make(map[int64][]ExportCopy)
	for _, ownedCopy := range ownedCopies {
		copiesByUserGame[ownedCopy.UserGameID] = append(copiesByUserGame[ownedCopy.UserGameID], ExportCopy{
			Platform: ownedCopy.Platform,
			OwnedDLC: dlcByCopy[ownedCopy.ID],
		})
	}

	playthroughsByUserGame := make(map[int64][]dbgen.ListPlaythroughsForUserIDRow)
	for _, pt := range playthroughs {
		playthroughsByUserGame[pt.Playthrough.UserGameID] = append(playthroughsByUserGame[pt.Playthrough.UserGameID], pt)
	}

	var library []ExportGame
	var wishlist []ExportGame
	for _, ug := range userGames {
		game := ExportGame{
			Game:         ug.Game,
			UserGame:     ug.UserGame,
			Copies:       copiesByUserGame[ug.UserGame.ID],
			Playthroughs: playthroughsByUserGame[ug.UserGame.ID],
		}
		if ug.UserGame.Wishlisted {
			wishlist = append(wishlist, game)
		} else {
			library = append(library, game)
		}
	}

	return ExportData{
		Library:  library,
		Wishlist: wishlist,
	}, nil
}

// ExportAll returns the export data of every user, sorted by username. It reads each user separately, so a change
// during the export can show in one user and not in another.
func (s *Store) ExportAll(ctx context.Context) ([]UserExportData, error) {
	users, err := s.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("read users: %w", err)
	}

	result := make([]UserExportData, 0, len(users))
	for _, user := range users {
		data, err := s.Export(ctx, user.ID)
		if err != nil {
			return nil, fmt.Errorf("export user %d: %w", user.ID, err)
		}
		result = append(result, UserExportData{
			User: user,
			Data: data,
		})
	}

	return result, nil
}
