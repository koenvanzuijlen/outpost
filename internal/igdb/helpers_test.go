package igdb

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestTokenClient returns a Client whose Twitch token endpoint is served by handler.
func newTestTokenClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := New("id", "secret")
	client.tokenURL = srv.URL
	return client
}

// newTestClient returns a Client with a valid token whose IGDB API endpoint is served by handler.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	client := newTestTokenClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"fake","expires_in":3600}`))
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client.baseURL = srv.URL
	return client
}
