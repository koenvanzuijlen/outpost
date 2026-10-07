// Package igdb provides a client for interacting with the IGDB API via Twitch authentication.
package igdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client represents an IGDB API client authenticated via Twitch.
type Client struct {
	baseURL      string
	tokenURL     string
	httpClient   *http.Client
	clientID     string
	clientSecret string
	tokenMu      sync.Mutex
	token        string
	tokenExpiry  time.Time
}

// New creates a new IGDB API client with the provided Twitch client ID and client secret.
func New(clientID, clientSecret string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      "https://api.igdb.com/v4",
		tokenURL:     "https://id.twitch.tv/oauth2/token",
	}
}

// accessToken retrieves a valid access token for the IGDB API, refreshing it if necessary.
func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExpiry.Add(-1*time.Minute)) {
		return c.token, nil
	}

	queryParams := url.Values{}
	queryParams.Set("client_id", c.clientID)
	queryParams.Set("client_secret", c.clientSecret)
	queryParams.Set("grant_type", "client_credentials")
	endpoint := c.tokenURL + "?" + queryParams.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	type tokenResponse struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if tr.AccessToken == "" {
		return "", errors.New("token response contained no access_token")
	}

	c.token = tr.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	return c.token, nil
}

// query dispatches a request to the specified IGDB API endpoint with the given request body and decodes the response
// into the provided output parameter.
func (c *Client) query(ctx context.Context, endpoint, body string, out any) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	token, err := c.accessToken(ctx)
	if err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+endpoint, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Client-ID", c.clientID)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}
