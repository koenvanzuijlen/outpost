package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// ErrGameListNotFound is returned when the user has no list with the given ID.
var ErrGameListNotFound = errors.New("game list not found")

// ErrGameListNameTaken is returned when another list of the user has the same name. The comparison ignores the case
// of ASCII letters.
var ErrGameListNameTaken = errors.New("game list name taken")

// GameListDetails are the fields of a list that CreateGameList and UpdateGameList write. A nil Description clears
// the description.
type GameListDetails struct {
	Name        string
	Description *string
}

// CreateGameList creates a list of the user with details and returns it. The caller validates the name and the
// description. Returns ErrGameListNameTaken if another list of the user has the name.
func (s *Store) CreateGameList(ctx context.Context, userID int64, details GameListDetails) (dbgen.GameList, error) {
	row, err := s.queries.CreateGameList(ctx, dbgen.CreateGameListParams{
		UserID:      userID,
		Name:        details.Name,
		Description: details.Description,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return dbgen.GameList{}, ErrGameListNameTaken
		}
		return dbgen.GameList{}, fmt.Errorf("create game list: %w", err)
	}
	return row, nil
}

// GetGameList returns the list of the user with the ID listID. Returns ErrGameListNotFound if the user has no list
// with that ID, also when the list belongs to another user.
func (s *Store) GetGameList(ctx context.Context, userID, listID int64) (dbgen.GameList, error) {
	row, err := s.queries.GetGameList(ctx, dbgen.GetGameListParams{
		UserID: userID,
		ID:     listID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dbgen.GameList{}, ErrGameListNotFound
		}
		return dbgen.GameList{}, fmt.Errorf("get game list: %w", err)
	}
	return row, nil
}

// UpdateGameList overwrites the name and the description of the list of the user with the ID listID. The caller
// validates the name and the description. The current values count as success. Returns ErrGameListNameTaken if
// another list of the user has the name, or ErrGameListNotFound if the user has no list with that ID.
func (s *Store) UpdateGameList(ctx context.Context, userID, listID int64, details GameListDetails) error {
	rows, err := s.queries.UpdateGameList(ctx, dbgen.UpdateGameListParams{
		UserID:      userID,
		ID:          listID,
		Name:        details.Name,
		Description: details.Description,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ErrGameListNameTaken
		}
		return fmt.Errorf("update game list: %w", err)
	}
	if rows == 0 {
		return ErrGameListNotFound
	}
	return nil
}

// DeleteGameList deletes the list of the user with the ID listID and its items. The games stay in the catalog, the
// library and the wishlist. A list that does not exist, or that belongs to another user, counts as success.
func (s *Store) DeleteGameList(ctx context.Context, userID, listID int64) error {
	err := s.queries.DeleteGameList(ctx, dbgen.DeleteGameListParams{
		UserID: userID,
		ID:     listID,
	})
	if err != nil {
		return fmt.Errorf("delete game list: %w", err)
	}
	return nil
}
