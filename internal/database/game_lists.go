package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

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

// gameListCoverCount is the number of covers that a GameListSummary holds at most.
const gameListCoverCount = 4

// GameListSummary is one list on the overview of the lists: the list, the number of games in it, and the cover URLs
// of the newest games that have a cover, at most gameListCoverCount.
type GameListSummary struct {
	List      dbgen.GameList
	GameCount int
	Covers    []string
}

// ListGameLists returns the lists of the user, sorted by name, with the game count and the covers of each list.
func (s *Store) ListGameLists(ctx context.Context, userID int64) ([]GameListSummary, error) {
	lists, err := s.queries.ListGameListsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list game lists: %w", err)
	}

	covers, err := s.queries.ListGameListItemCoversForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list game list item covers: %w", err)
	}

	listIndexes := make(map[int64]int, len(lists))
	for idx, list := range lists {
		listIndexes[list.ID] = idx
	}

	summaries := make([]GameListSummary, 0, len(lists))
	for _, list := range lists {
		summaries = append(summaries, GameListSummary{List: list})
	}

	for _, cover := range covers {
		idx, ok := listIndexes[cover.GameListID]
		if !ok {
			continue
		}
		summaries[idx].GameCount++
		if cover.CoverUrl != nil && len(summaries[idx].Covers) < gameListCoverCount {
			summaries[idx].Covers = append(summaries[idx].Covers, *cover.CoverUrl)
		}
	}

	return summaries, nil
}

// ListGameListEntries returns the games in the list of the user with the ID listID, sorted by title, with the date
// each game was added. Status and Rating come from the library entry of the user, and are nil for a game outside
// the library or on the wishlist. A list that does not exist, or that belongs to another user, gives an empty
// slice.
func (s *Store) ListGameListEntries(ctx context.Context, userID, listID int64) ([]dbgen.ListGameListEntriesRow, error) {
	rows, err := s.queries.ListGameListEntries(ctx, dbgen.ListGameListEntriesParams{
		UserID:     userID,
		GameListID: listID,
	})
	if err != nil {
		return nil, fmt.Errorf("list game list entries: %w", err)
	}
	return rows, nil
}

// ListGameListsForGame returns all lists of the user, sorted by name, and reports per list whether it holds the
// game with the internal ID gameID.
func (s *Store) ListGameListsForGame(ctx context.Context, userID, gameID int64) ([]dbgen.ListGameListsForGameRow, error) {
	rows, err := s.queries.ListGameListsForGame(ctx, dbgen.ListGameListsForGameParams{
		UserID: userID,
		GameID: gameID,
	})
	if err != nil {
		return nil, fmt.Errorf("list game lists for game: %w", err)
	}
	return rows, nil
}

// AddToGameList adds a game to the list of the user with the ID listID, storing the game in the catalog if it isn't
// there yet. A game that is already in the list counts as success and keeps its first date. Returns
// ErrGameListNotFound if the user has no list with that ID; the catalog is not changed then.
func (s *Store) AddToGameList(
	ctx context.Context,
	userID,
	listID int64,
	game dbgen.UpsertGameParams,
	addedAt time.Time,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	_, err = q.GetGameList(ctx, dbgen.GetGameListParams{ID: listID, UserID: userID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrGameListNotFound
		}
		return fmt.Errorf("get game list: %w", err)
	}

	row, err := q.UpsertGame(ctx, game)
	if err != nil {
		return fmt.Errorf("upsert game: %w", err)
	}

	err = q.CreateGameListItem(ctx, dbgen.CreateGameListItemParams{
		GameListID: listID,
		GameID:     row.ID,
		AddedAt:    addedAt,
	})
	if err != nil {
		return fmt.Errorf("create game list item: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// RemoveFromGameList removes the game with igdbID from the list of the user with the ID listID. A game that is not
// in the list, a list that does not exist, and a list of another user count as success.
func (s *Store) RemoveFromGameList(ctx context.Context, userID, listID, igdbID int64) error {
	err := s.queries.DeleteGameListItem(ctx, dbgen.DeleteGameListItemParams{
		UserID:     userID,
		GameListID: listID,
		IgdbID:     igdbID,
	})
	if err != nil {
		return fmt.Errorf("delete game list item: %w", err)
	}
	return nil
}
