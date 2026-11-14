package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// checkHealth sends a GET request to url and reports whether the server is healthy. The server is healthy when it
// answers 200. Returns an error if the request fails or the status is not 200.
func checkHealth(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request health: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

// runHealthcheck checks the health endpoint of the server on this host, at the port from PORT. The check times out
// after 3 seconds. The Docker HEALTHCHECK runs it through "outpost healthcheck". Returns an error if the server is not
// healthy.
func runHealthcheck() error {
	url := "http://127.0.0.1:" + envOrDefault("PORT", "8080") + "/health"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return checkHealth(ctx, url)
}
