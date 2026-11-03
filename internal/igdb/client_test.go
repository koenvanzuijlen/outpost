package igdb

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

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

// countingHandler counts calls and answers each call with respond(callNumber).
func countingHandler(calls *atomic.Int32, respond func(n int32, w http.ResponseWriter)) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respond(calls.Add(1), w)
	}
}

func TestQueryRetrySucceeds(t *testing.T) {
	var calls atomic.Int32
	client := newTestClient(t, countingHandler(&calls, func(n int32, w http.ResponseWriter) {
		if n == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	client.retryDelay = time.Millisecond

	if _, err := client.SearchGames(t.Context(), "zelda"); err != nil {
		t.Fatalf("SearchGames() error = %v, want nil", err)
	}
	// SearchGames sends two queries. The first query takes two calls because of the retry.
	if got := calls.Load(); got != 3 {
		t.Errorf("calls = %d, want 3", got)
	}
}

func TestQueryRetriesExhausted(t *testing.T) {
	var calls atomic.Int32
	client := newTestClient(t, countingHandler(&calls, func(_ int32, w http.ResponseWriter) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	client.retryDelay = time.Millisecond

	_, err := client.SearchGames(t.Context(), "zelda")
	if err == nil {
		t.Fatal("SearchGames() error = nil, want an error")
	}
	if !errors.Is(err, errRateLimited) {
		t.Errorf("SearchGames() error = %v, want errRateLimited", err)
	}
	if got := calls.Load(); got != maxAttempts {
		t.Errorf("calls = %d, want %d", got, maxAttempts)
	}
}

func TestQueryNoRetryForOtherErrors(t *testing.T) {
	var calls atomic.Int32
	client := newTestClient(t, countingHandler(&calls, func(_ int32, w http.ResponseWriter) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	client.retryDelay = time.Millisecond

	if _, err := client.SearchGames(t.Context(), "zelda"); err == nil {
		t.Fatal("SearchGames() error = nil, want an error")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("calls = %d, want 1", got)
	}
}

func TestQueryContextEndsDuringWait(t *testing.T) {
	var calls atomic.Int32
	client := newTestClient(t, countingHandler(&calls, func(_ int32, w http.ResponseWriter) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	client.retryDelay = time.Second

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := client.SearchGames(ctx, "zelda")
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("SearchGames() error = %v, want context.DeadlineExceeded", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("call took %v, want well under 1s", elapsed)
	}
}
