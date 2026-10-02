package igdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Cover represents the cover image of a game retrieved from the IGDB API.
type Cover struct {
	URL string `json:"url"`
}

// Game represents a video game retrieved from the IGDB API.
type Game struct {
	ID               int    `json:"id"`
	ParentGameID     int    `json:"parent_game"`
	Name             string `json:"name"`
	GameType         int    `json:"game_type"`
	Cover            Cover  `json:"cover"`
	FirstReleaseDate int    `json:"first_release_date"`
	Summary          string `json:"summary"`
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
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	token, err := c.accessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	escaped := strings.ReplaceAll(query, `"`, `\"`)
	body := fmt.Sprintf(`search "%s"; fields parent_game,name,game_type,cover.url,first_release_date,summary; limit 10;`, escaped)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/games", strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Client-ID", c.clientID)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}
	var games []Game
	if err := json.NewDecoder(resp.Body).Decode(&games); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return games, nil
}
