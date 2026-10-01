package igdb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func newTestTokenClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := New("id", "secret")
	client.tokenURL = srv.URL
	return client
}

func TestAccessTokenOK(t *testing.T) {
	testToken := "fake"
	client := newTestTokenClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"` + testToken + `","expires_in":3600}`))
	})

	token, err := client.accessToken(t.Context())
	if err != nil {
		t.Fatalf("accessToken() failed: %v", err)
	}
	if token != testToken {
		t.Errorf("accessToken() = %q, want %q", token, testToken)
	}
}

func TestAccessToken401(t *testing.T) {
	errorMsg := `{"status":401,"message":"invalid client secret"}`
	client := newTestTokenClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(errorMsg))
	})

	tok, err := client.accessToken(t.Context())
	if err == nil {
		t.Fatal("accessToken() error = nil, want error")
	}
	if !strings.Contains(err.Error(), errorMsg) {
		t.Errorf("accessToken() error = %v, want it to contain %q", err, errorMsg)
	}
	if tok != "" {
		t.Errorf("accessToken() = %q, want %q", tok, "")
	}
}

func TestAccessTokenBadJSON(t *testing.T) {
	badJSON := "token: abc"
	client := newTestTokenClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(badJSON))
	})

	token, err := client.accessToken(t.Context())
	if err == nil {
		t.Fatal("accessToken() error = nil, want error")
	}
	if token != "" {
		t.Errorf("accessToken() = %q, want %q", token, "")
	}
}

func TestAccessTokenCaching(t *testing.T) {
	hits := 0
	client := newTestTokenClient(t, func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte(`{"access_token":"fake","expires_in":3600}`))
	})

	_, err := client.accessToken(t.Context())
	if err != nil {
		t.Fatalf("accessToken() failed: %v", err)
	}
	_, err = client.accessToken(t.Context())
	if err != nil {
		t.Fatalf("accessToken() failed: %v", err)
	}
	if hits != 1 {
		t.Errorf("token endpoint hits = %d, want 1", hits)
	}
}

func TestAccessTokenRefresh(t *testing.T) {
	hits := 0
	client := newTestTokenClient(t, func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte(`{"access_token":"fake","expires_in":30}`))
	})

	_, err := client.accessToken(t.Context())
	if err != nil {
		t.Fatalf("accessToken() failed: %v", err)
	}
	_, err = client.accessToken(t.Context())
	if err != nil {
		t.Fatalf("accessToken() failed: %v", err)
	}
	if hits != 2 {
		t.Errorf("token endpoint hits = %d, want 2", hits)
	}
}

func TestAccessTokenConcurrency(t *testing.T) {
	hits := 0
	client := newTestTokenClient(t, func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte(`{"access_token":"fake","expires_in":3600}`))
	})

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() { _, _ = client.accessToken(t.Context()) })
	}
	wg.Wait()

	if hits != 1 {
		t.Errorf("token endpoint hits = %d, want 1", hits)
	}
}
