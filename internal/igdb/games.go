package igdb

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrGameNotFound is returned when a game cannot be found in the IGDB API.
var ErrGameNotFound = errors.New("game not found")

// gameFields defines the default set of fields to request when querying games from the IGDB API.
const gameFields = "parent_game,name,game_type,cover.url,first_release_date,summary"

// GameType as defined by the IGDB API. See https://api-docs.igdb.com/#game
type GameType int

// GameType as defined by the IGDB API. See https://api-docs.igdb.com/#game
const (
	GameTypeMain                GameType = 0
	GameTypeDLCAddon            GameType = 1
	GameTypeExpansion           GameType = 2
	GameTypeBundle              GameType = 3
	GameTypeStandaloneExpansion GameType = 4
	GameTypeMod                 GameType = 5
	GameTypeEpisode             GameType = 6
	GameTypeSeason              GameType = 7
	GameTypeRemake              GameType = 8
	GameTypeRemaster            GameType = 9
	GameTypeExpandedGame        GameType = 10
	GameTypePort                GameType = 11
	GameTypeFork                GameType = 12
	GameTypePack                GameType = 13
	GameTypeUpdate              GameType = 14
)

// String returns the human readable string representation of the GameType.
func (gt GameType) String() string {
	switch gt {
	case GameTypeMain:
		return "Main"
	case GameTypeDLCAddon:
		return "DLC / Add-on"
	case GameTypeExpansion:
		return "Expansion"
	case GameTypeBundle:
		return "Bundle"
	case GameTypeStandaloneExpansion:
		return "Standalone Expansion"
	case GameTypeMod:
		return "Mod"
	case GameTypeEpisode:
		return "Episode"
	case GameTypeSeason:
		return "Season"
	case GameTypeRemake:
		return "Remake"
	case GameTypeRemaster:
		return "Remaster"
	case GameTypeExpandedGame:
		return "Expanded Game"
	case GameTypePort:
		return "Port"
	case GameTypeFork:
		return "Fork"
	case GameTypePack:
		return "Pack"
	case GameTypeUpdate:
		return "Update"
	default:
		return fmt.Sprintf("Unknown (%d)", int(gt))
	}
}

// Cover represents the cover image of a game retrieved from the IGDB API.
type Cover struct {
	URL string `json:"url"`
}

// Game represents a video game retrieved from the IGDB API.
type Game struct {
	ID               int      `json:"id"`
	ParentGameID     int      `json:"parent_game"`
	Name             string   `json:"name"`
	GameType         GameType `json:"game_type"`
	Cover            Cover    `json:"cover"`
	FirstReleaseDate int      `json:"first_release_date"`
	Summary          string   `json:"summary"`
}

// CoverURL returns the URL of the game's cover image, or an empty string if no cover is available. It replaces the
// default thumbnail size with a larger cover size.
func (g Game) CoverURL() string {
	if g.Cover.URL == "" {
		return ""
	}
	return "https:" + strings.ReplaceAll(g.Cover.URL, "t_thumb", "t_cover_big")
}

// SearchGames searches for games matching the given query string.
func (c *Client) SearchGames(ctx context.Context, query string) ([]Game, error) {
	escaped := strings.ReplaceAll(query, `"`, `\"`)
	body := fmt.Sprintf(`search "%s"; fields %s; limit 10;`, escaped, gameFields)

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
