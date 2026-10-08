package igdb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrGameNotFound is returned when a game cannot be found in the IGDB API.
var ErrGameNotFound = errors.New("game not found")

// gameFields defines the default set of fields to request when querying games from the IGDB API.
const gameFields = "parent_game,name,game_type,cover.url,first_release_date,summary,platforms.name"

// Cover represents the cover image of a game retrieved from the IGDB API.
type Cover struct {
	URL string `json:"url"`
}

// Game represents a video game retrieved from the IGDB API.
type Game struct {
	ID               int        `json:"id"`
	ParentGameID     int        `json:"parent_game"`
	Name             string     `json:"name"`
	GameType         GameType   `json:"game_type"`
	Cover            Cover      `json:"cover"`
	FirstReleaseDate int        `json:"first_release_date"`
	Summary          string     `json:"summary"`
	Platforms        []Platform `json:"platforms"`
}

// CoverURL returns the URL of the game's cover image, or an empty string if no cover is available. It replaces the
// default thumbnail size with a larger cover size.
func (g Game) CoverURL() string {
	if g.Cover.URL == "" {
		return ""
	}
	return "https:" + strings.ReplaceAll(g.Cover.URL, "t_thumb", "t_cover_big")
}

// ReleaseYear returns the year the game was first released, or 0 if the release date is not available.
func (g Game) ReleaseYear() int {
	if g.FirstReleaseDate == 0 {
		return 0
	}
	return time.Unix(int64(g.FirstReleaseDate), 0).UTC().Year()
}

// SearchGames searches for games matching the given query string.
func (c *Client) SearchGames(ctx context.Context, query string) ([]Game, error) {
	escaped := strings.ReplaceAll(query, `"`, `\"`)
	body := fmt.Sprintf(`search "%s"; where platforms != null; fields %s; limit 10;`, escaped, gameFields)

	var games []Game
	if err := c.query(ctx, "/games", body, &games); err != nil {
		return nil, fmt.Errorf("query games: %w", err)
	}

	return games, nil
}

// GameByID retrieves a game by its unique ID from the IGDB API.
func (c *Client) GameByID(ctx context.Context, id int) (Game, error) {
	body := fmt.Sprintf(`where id = %d; fields %s;`, id, gameFields)

	var games []Game
	if err := c.query(ctx, "/games", body, &games); err != nil {
		return Game{}, fmt.Errorf("query games: %w", err)
	}
	if len(games) == 0 {
		return Game{}, fmt.Errorf("game %d: %w", id, ErrGameNotFound)
	}
	return games[0], nil
}
